package portkey

import (
	"context"
	"fmt"

	tfv1 "github.com/NexusGPU/tensor-fusion/api/v1"
	"github.com/NexusGPU/tensor-fusion/internal/azure/broker"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RoutingController manages Portkey routing configurations based on TensorFusion state
type RoutingController struct {
	client        client.Client
	portkeyClient *Client
	sourceSelector *broker.SourceSelector
	logger        klog.Logger
}

// NewRoutingController creates a new routing controller
func NewRoutingController(k8sClient client.Client, portkeyClient *Client, sourceSelector *broker.SourceSelector) *RoutingController {
	return &RoutingController{
		client:         k8sClient,
		portkeyClient:  portkeyClient,
		sourceSelector: sourceSelector,
		logger:         klog.NewKlogr().WithName("routing-controller"),
	}
}

// SyncRoute synchronizes an LLMRoute with Portkey
func (rc *RoutingController) SyncRoute(ctx context.Context, route *tfv1.LLMRoute) error {
	rc.logger.Info("Syncing LLMRoute to Portkey", "route", route.Name)

	// Build Portkey config from LLMRoute
	config, err := rc.buildPortkeyConfig(ctx, route)
	if err != nil {
		return fmt.Errorf("failed to build Portkey config: %w", err)
	}

	// Create or update config in Portkey
	if route.Spec.PortkeyConfigID != "" {
		// Update existing config
		if err := rc.portkeyClient.UpdateConfig(ctx, route.Spec.PortkeyConfigID, config); err != nil {
			return fmt.Errorf("failed to update Portkey config: %w", err)
		}
	} else {
		// Create new config
		created, err := rc.portkeyClient.CreateConfig(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to create Portkey config: %w", err)
		}
		
		// Update LLMRoute with config ID
		route.Spec.PortkeyConfigID = created.ID
		if err := rc.client.Update(ctx, route); err != nil {
			return fmt.Errorf("failed to update LLMRoute with config ID: %w", err)
		}
	}

	rc.logger.Info("Successfully synced LLMRoute", "route", route.Name, "configID", route.Spec.PortkeyConfigID)
	return nil
}

// buildPortkeyConfig converts an LLMRoute to a Portkey Config
func (rc *RoutingController) buildPortkeyConfig(ctx context.Context, route *tfv1.LLMRoute) (*Config, error) {
	config := &Config{
		Name:     route.Name,
		Strategy: rc.mapStrategy(route.Spec.Strategy),
		Targets:  make([]Target, 0, len(route.Spec.Targets)),
	}

	// Configure caching
	if route.Spec.Caching != nil && route.Spec.Caching.Enabled {
		ttlSeconds := 3600 // Default 1 hour
		config.Cache = &CacheConfig{
			Mode:   "semantic",
			MaxAge: ttlSeconds,
		}
	}

	// Configure retry
	if route.Spec.Retry != nil {
		config.Retry = &RetryConfig{
			Attempts: int(route.Spec.Retry.MaxRetries),
		}
	}

	// Build targets with dynamic GPU source selection
	for _, target := range route.Spec.Targets {
		if !target.Enabled {
			continue
		}

		portkeyTarget, err := rc.buildTarget(ctx, &target)
		if err != nil {
			rc.logger.Error(err, "Failed to build target", "target", target.Name)
			continue
		}

		config.Targets = append(config.Targets, *portkeyTarget)
	}

	return config, nil
}

// buildTarget builds a Portkey target from an LLM target spec
func (rc *RoutingController) buildTarget(ctx context.Context, target *tfv1.LLMTarget) (*Target, error) {
	portkeyTarget := &Target{
		Name:     target.Name,
		Provider: target.Provider,
		Model:    target.Model,
		Weight:   int(target.Weight),
	}

	// If AzureGPUSource specified, resolve the endpoint
	if target.AzureGPUSource != "" {
		endpoint, err := rc.resolveAzureGPUSource(ctx, target.AzureGPUSource)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve Azure GPU source: %w", err)
		}
		
		portkeyTarget.Override = map[string]interface{}{
			"base_url": endpoint,
		}
	}

	// If endpoint override specified
	if target.Endpoint != "" {
		if portkeyTarget.Override == nil {
			portkeyTarget.Override = make(map[string]interface{})
		}
		portkeyTarget.Override["base_url"] = target.Endpoint
	}

	// Add additional parameters
	for k, v := range target.Parameters {
		if portkeyTarget.Override == nil {
			portkeyTarget.Override = make(map[string]interface{})
		}
		portkeyTarget.Override[k] = v
	}

	return portkeyTarget, nil
}

// resolveAzureGPUSource resolves an AzureGPUSource to an endpoint URL
func (rc *RoutingController) resolveAzureGPUSource(ctx context.Context, sourceName string) (string, error) {
	var source tfv1.AzureGPUSource
	if err := rc.client.Get(ctx, client.ObjectKey{Name: sourceName}, &source); err != nil {
		return "", fmt.Errorf("failed to get AzureGPUSource: %w", err)
	}

	switch source.Spec.SourceType {
	case tfv1.AzureGPUSourceTypeFoundry:
		return source.Spec.FoundryEndpoint, nil
	case tfv1.AzureGPUSourceTypeAKS:
		// For AKS, we'd construct an endpoint based on the cluster
		// This would point to the TensorFusion vGPU worker endpoint
		return fmt.Sprintf("http://%s-vgpu-service:8000", source.Spec.AKSClusterName), nil
	default:
		return "", fmt.Errorf("unknown source type: %s", source.Spec.SourceType)
	}
}

// mapStrategy maps TensorFusion routing strategy to Portkey strategy
func (rc *RoutingController) mapStrategy(strategy tfv1.RoutingStrategy) string {
	switch strategy {
	case tfv1.RoutingStrategyCostOptimized:
		return "loadbalance" // Use weighted load balancing
	case tfv1.RoutingStrategyLatencyOptimized:
		return "fallback" // Prioritize lowest latency
	case tfv1.RoutingStrategyRoundRobin:
		return "loadbalance"
	case tfv1.RoutingStrategyWeighted:
		return "loadbalance"
	case tfv1.RoutingStrategyPriority:
		return "fallback"
	case tfv1.RoutingStrategyLoadTest:
		return "loadbalance"
	default:
		return "fallback"
	}
}

// OptimizeRouting dynamically adjusts routing based on current GPU availability
func (rc *RoutingController) OptimizeRouting(ctx context.Context, route *tfv1.LLMRoute) error {
	rc.logger.Info("Optimizing routing", "route", route.Name)

	// Get current GPU availability from all sources
	// This would query AKS pools and Azure Foundry

	// Adjust target weights based on availability and cost
	// For example, if AKS GPUs are cheaper and available, increase their weight

	// Update the route configuration
	return rc.SyncRoute(ctx, route)
}

// UpdateStatistics updates LLMRoute statistics from Portkey
func (rc *RoutingController) UpdateStatistics(ctx context.Context, route *tfv1.LLMRoute) error {
	if route.Spec.PortkeyConfigID == "" {
		return nil
	}

	// Get usage stats from Portkey
	timeRange := TimeRange{
		Start: route.Status.LastUpdated.Time,
		End:   route.Status.LastUpdated.Time,
	}

	stats, err := rc.portkeyClient.GetUsageStats(ctx, route.Spec.PortkeyConfigID, timeRange)
	if err != nil {
		return fmt.Errorf("failed to get usage stats: %w", err)
	}

	// Update route status
	route.Status.Stats = tfv1.LLMRouteStats{
		TotalRequests:      stats.TotalRequests,
		SuccessfulRequests: stats.SuccessfulRequests,
		FailedRequests:     stats.FailedRequests,
		CachedRequests:     stats.CacheHits,
		AverageLatency:     fmt.Sprintf("%.2fms", stats.AverageLatencyMs),
		P95Latency:         fmt.Sprintf("%.2fms", stats.P95LatencyMs),
		P99Latency:         fmt.Sprintf("%.2fms", stats.P99LatencyMs),
	}

	route.Status.CostTracking = tfv1.CostTracking{
		TotalCost:     fmt.Sprintf("$%.4f", stats.TotalCost),
		TotalTokens:   stats.TotalTokens,
	}

	// Update status in Kubernetes
	if err := rc.client.Status().Update(ctx, route); err != nil {
		return fmt.Errorf("failed to update route status: %w", err)
	}

	return nil
}


