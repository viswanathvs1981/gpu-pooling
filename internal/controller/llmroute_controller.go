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
		log.Info("RoutingController not initialized, skipping route sync")
		route.Status.Phase = tfv1.LLMRoutePhasePending
		now := metav1.Now()
		route.Status.LastUpdated = &now
		if err := r.Status().Update(ctx, route); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Sync route with Portkey
	if err := r.RoutingController.SyncRoute(ctx, route); err != nil {
		log.Error(err, "failed to sync route with Portkey")
		route.Status.Phase = tfv1.LLMRoutePhaseError
	} else {
		route.Status.Phase = tfv1.LLMRoutePhaseActive
	}

	// Update status
	now := metav1.Now()
	route.Status.LastUpdated = &now
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
