package tools

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PlatformTools provides all platform operation tools
type PlatformTools struct {
	k8sClient client.Client
	clientset *kubernetes.Clientset
	deploy    *DeployTool
	training  *TrainingTool
	metrics   *MetricsTool
	costs     *CostsTool
}

// ToolDefinition describes a tool's interface
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	InputSchema interface{}            `json:"inputSchema,omitempty"`
	Handler     ToolHandler            `json:"-"`
}

// ToolHandler is a function that executes a tool
type ToolHandler func(ctx context.Context, params map[string]interface{}) (interface{}, error)

// NewPlatformTools creates a new toolset
func NewPlatformTools(k8sClient client.Client, clientset *kubernetes.Clientset) *PlatformTools {
	return &PlatformTools{
		k8sClient: k8sClient,
		clientset: clientset,
		deploy:    NewDeployTool(k8sClient, clientset),
		training:  NewTrainingTool(k8sClient, clientset),
		metrics:   NewMetricsTool(k8sClient),
		costs:     NewCostsTool(k8sClient),
	}
}

// ListTools returns all available tools
func (pt *PlatformTools) ListTools() []ToolDefinition {
	tools := []ToolDefinition{
		{
			Name:        "deploy_model",
			Description: "Deploy a model with vLLM to Kubernetes",
			Parameters: map[string]interface{}{
				"model_id":    "string - Model identifier",
				"customer_id": "string - Customer namespace",
				"config": map[string]interface{}{
					"vgpu":     "float - vGPU allocation (e.g., 0.5)",
					"replicas": "int - Number of replicas",
					"image":    "string - vLLM image (optional)",
				},
			},
		},
		{
			Name:        "start_training",
			Description: "Start a LoRA training job",
			Parameters: map[string]interface{}{
				"dataset_path": "string - Path to training dataset",
				"base_model":   "string - Base model name",
				"lora_config": map[string]interface{}{
					"rank":  "int - LoRA rank (default: 32)",
					"alpha": "int - LoRA alpha (default: 64)",
				},
			},
		},
		{
			Name:        "get_metrics",
			Description: "Query metrics for a customer",
			Parameters: map[string]interface{}{
				"customer_id": "string - Customer namespace",
				"time_range":  "string - Time range (e.g., '24h', '7d')",
				"metrics":     "[]string - Metrics to query (latency, throughput)",
			},
		},
		{
			Name:        "allocate_gpu",
			Description: "Allocate GPU resources",
			Parameters: map[string]interface{}{
				"vgpu_size": "float - vGPU size",
				"duration":  "string - Duration (e.g., '2h')",
				"pool_name": "string - GPU pool name",
			},
		},
		{
			Name:        "update_routing",
			Description: "Update LLM routing configuration",
			Parameters: map[string]interface{}{
				"route_name": "string - Route name",
				"policy": map[string]interface{}{
					"conditions": "map - Routing conditions",
					"backends":   "[]string - Backend services",
				},
			},
		},
		{
			Name:        "get_costs",
			Description: "Get cost breakdown for a customer",
			Parameters: map[string]interface{}{
				"customer_id": "string - Customer namespace",
				"period": map[string]interface{}{
					"start": "string - Start time (ISO 8601)",
					"end":   "string - End time (ISO 8601)",
				},
			},
		},
		{
			Name:        "query_usage",
			Description: "Query usage patterns",
			Parameters: map[string]interface{}{
				"filters": map[string]interface{}{
					"customer":   "string - Customer ID",
					"time_range": "string - Time range",
				},
			},
		},
		{
			Name:        "forecast_costs",
			Description: "Forecast future costs",
			Parameters: map[string]interface{}{
				"customer_id":   "string - Customer ID",
				"forecast_days": "int - Days to forecast",
			},
		},
		{
			Name:        "detect_anomalies",
			Description: "Detect anomalies in metrics",
			Parameters: map[string]interface{}{
				"metric_name": "string - Metric to analyze",
				"threshold":   "float - Anomaly threshold",
				"time_window": "string - Time window",
			},
		},
		{
			Name:        "recommend_optimization",
			Description: "Recommend cost/performance optimizations",
			Parameters: map[string]interface{}{
				"customer_id":         "string - Customer ID",
				"optimization_target": "string - cost|latency|throughput",
			},
		},
	}

	// Append new capability tools
	tools = append(tools, RegisterMemoryTools()...)
	tools = append(tools, RegisterSmallModelTools()...)
	tools = append(tools, RegisterDiscoveryTools()...)

	return tools
}

// ExecuteTool executes a tool by name
func (pt *PlatformTools) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	// Check if tool has a handler (new tools)
	allTools := pt.ListTools()
	for _, tool := range allTools {
		if tool.Name == toolName && tool.Handler != nil {
			return tool.Handler(ctx, params)
		}
	}

	// Legacy tool handling
	switch toolName {
	case "deploy_model":
		return pt.deploy.DeployModel(ctx, params)
	case "start_training":
		return pt.training.StartTraining(ctx, params)
	case "get_metrics":
		return pt.metrics.GetMetrics(ctx, params)
	case "allocate_gpu":
		return pt.deploy.AllocateGPU(ctx, params)
	case "update_routing":
		return pt.deploy.UpdateRouting(ctx, params)
	case "get_costs":
		return pt.costs.GetCosts(ctx, params)
	case "query_usage":
		return pt.costs.QueryUsage(ctx, params)
	case "forecast_costs":
		return pt.costs.ForecastCosts(ctx, params)
	case "detect_anomalies":
		return pt.metrics.DetectAnomalies(ctx, params)
	case "recommend_optimization":
		return pt.costs.RecommendOptimization(ctx, params)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

