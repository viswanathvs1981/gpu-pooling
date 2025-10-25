package controller

import (
	"context"
	"time"

	tfv1 "github.com/NexusGPU/tensor-fusion/api/v1"
	"github.com/NexusGPU/tensor-fusion/internal/azure/foundry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AzureGPUSourceReconciler reconciles an AzureGPUSource object
type AzureGPUSourceReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	FoundryClient *foundry.Client
}

// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=azuregpusources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=azuregpusources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=azuregpusources/finalizers,verbs=update

// Reconcile handles AzureGPUSource reconciliation
func (r *AzureGPUSourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the AzureGPUSource instance
	source := &tfv1.AzureGPUSource{}
	if err := r.Get(ctx, req.NamespacedName, source); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Initialize Foundry client if needed
	if r.FoundryClient == nil && source.Spec.Enabled {
		// Get credentials from secret if specified
		endpoint := source.Spec.Endpoint
		apiKey := "default-key" // Should be retrieved from secret
		apiVersion := "2024-02-15-preview"

		r.FoundryClient = foundry.NewClient(endpoint, apiKey, apiVersion)
	}

	if !source.Spec.Enabled {
		source.Status.Phase = "Disabled"
		source.Status.AvailableModels = []string{}
		if err := r.Status().Update(ctx, source); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// List available deployments
	if r.FoundryClient != nil {
		deployments, err := r.FoundryClient.ListDeployments(ctx)
		if err != nil {
			log.Error(err, "failed to list Azure deployments")
			source.Status.Phase = "Error"
			source.Status.LastSyncError = err.Error()
		} else {
			source.Status.Phase = "Active"
			source.Status.AvailableModels = make([]string, len(deployments))
			for i, d := range deployments {
				source.Status.AvailableModels[i] = d.Name
			}
			source.Status.LastSyncTime = metav1.Now()
			source.Status.LastSyncError = ""
		}
	}

	// Update status
	if err := r.Status().Update(ctx, source); err != nil {
		return ctrl.Result{}, err
	}

	// Parse sync interval
	syncInterval, err := time.ParseDuration(source.Spec.SyncInterval)
	if err != nil {
		syncInterval = 5 * time.Minute
	}

	return ctrl.Result{RequeueAfter: syncInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AzureGPUSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tfv1.AzureGPUSource{}).
		Named("azuregpusource").
		Complete(r)
}
