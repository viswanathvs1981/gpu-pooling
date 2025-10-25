package controller

import (
	"context"
	"time"

	tfv1 "github.com/NexusGPU/tensor-fusion/api/v1"
	"github.com/NexusGPU/tensor-fusion/internal/intelligence/ml"
	"github.com/NexusGPU/tensor-fusion/internal/intelligence/vectordb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// WorkloadIntelligenceReconciler reconciles a WorkloadIntelligence object
type WorkloadIntelligenceReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	VectorDBClient   vectordb.Client
	WorkloadProfiler *ml.WorkloadProfiler
}

// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=workloadintelligences,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=workloadintelligences/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=workloadintelligences/finalizers,verbs=update

// Reconcile handles WorkloadIntelligence reconciliation
func (r *WorkloadIntelligenceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the WorkloadIntelligence instance
	wi := &tfv1.WorkloadIntelligence{}
	if err := r.Get(ctx, req.NamespacedName, wi); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Initialize clients if needed
	if r.VectorDBClient == nil {
		qdrantURL := "http://qdrant.qdrant.svc.cluster.local:6333"
		var err error
		r.VectorDBClient, err = vectordb.NewQdrantClient(qdrantURL)
		if err != nil {
			log.Error(err, "failed to initialize VectorDB client")
			return ctrl.Result{}, err
		}
	}

	if r.WorkloadProfiler == nil {
		r.WorkloadProfiler = ml.NewWorkloadProfiler(r.VectorDBClient)
	}

	// Analyze workload patterns
	if wi.Spec.Enabled {
		// Get workload history
		workloads := &tfv1.TensorFusionWorkloadList{}
		if err := r.List(ctx, workloads, client.InNamespace(wi.Namespace)); err != nil {
			return ctrl.Result{}, err
		}

		// Perform analysis
		prediction, err := r.WorkloadProfiler.PredictResources(ctx, wi.Spec.WorkloadName)
		if err != nil {
			log.Error(err, "failed to predict resources")
			wi.Status.Phase = "Failed"
			wi.Status.LastError = err.Error()
		} else {
			wi.Status.Phase = "Active"
			wi.Status.Predictions = &tfv1.ResourcePrediction{
				RecommendedVGPU: prediction.VGPU,
				RecommendedVRAM: prediction.VRAM,
				Confidence:      prediction.Confidence,
			}
			wi.Status.LastPrediction = metav1.Now()
		}
	} else {
		wi.Status.Phase = "Disabled"
	}

	// Update status
	if err := r.Status().Update(ctx, wi); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue periodically
	return ctrl.Result{RequeueAfter: time.Duration(wi.Spec.AnalysisInterval) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkloadIntelligenceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tfv1.WorkloadIntelligence{}).
		Named("workloadintelligence").
		Complete(r)
}
