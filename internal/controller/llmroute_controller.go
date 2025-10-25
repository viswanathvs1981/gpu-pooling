package controller

import (
	"context"
	"time"

	tfv1 "github.com/NexusGPU/tensor-fusion/api/v1"
	"github.com/NexusGPU/tensor-fusion/internal/llmgateway/portkey"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// LLMRouteReconciler reconciles an LLMRoute object
type LLMRouteReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	RoutingController *portkey.RoutingController
}

// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=llmroutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=llmroutes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=llmroutes/finalizers,verbs=update

// Reconcile handles LLMRoute reconciliation
func (r *LLMRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the LLMRoute instance
	route := &tfv1.LLMRoute{}
	if err := r.Get(ctx, req.NamespacedName, route); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Initialize RoutingController if needed
	if r.RoutingController == nil {
		// Get Portkey configuration from environment or config
		portkeyURL := "http://portkey-gateway.portkey.svc.cluster.local:8787"
		portkeyKey := "default-key"
		r.RoutingController = portkey.NewRoutingController(portkeyURL, portkeyKey)
	}

	// Convert TensorFusion route to Portkey format
	portkeyConfig := &portkey.RouteConfig{
		Name:     route.Name,
		Strategy: route.Spec.Strategy,
		Targets:  convertTargets(route.Spec.Targets),
	}

	// Create or update route in Portkey
	if err := r.RoutingController.CreateRoute(ctx, portkeyConfig); err != nil {
		log.Error(err, "failed to create/update route in Portkey")
		route.Status.Phase = "Failed"
		route.Status.Reason = err.Error()
	} else {
		route.Status.Phase = "Active"
		route.Status.Reason = "Route configured successfully"
	}

	// Update status
	route.Status.LastUpdated = metav1.Now()
	if err := r.Status().Update(ctx, route); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue after 5 minutes to sync
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LLMRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tfv1.LLMRoute{}).
		Named("llmroute").
		Complete(r)
}

func convertTargets(tfTargets []tfv1.LLMTarget) []portkey.TargetConfig {
	targets := make([]portkey.TargetConfig, len(tfTargets))
	for i, t := range tfTargets {
		targets[i] = portkey.TargetConfig{
			Provider:   t.Provider,
			Weight:     int(t.Weight),
			VirtualKey: t.VirtualKey,
		}
	}
	return targets
}
