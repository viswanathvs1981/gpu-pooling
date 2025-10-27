package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Workflow represents a multi-step workflow
type Workflow struct {
	Name  string
	Steps []WorkflowStep
}

// WorkflowStep represents a single step in a workflow
type WorkflowStep struct {
	Name    string
	Execute func(ctx context.Context, mcpURL string, params map[string]interface{}, state map[string]interface{}) (interface{}, error)
}

// NewDeployModelWorkflow creates a model deployment workflow
func NewDeployModelWorkflow() *Workflow {
	return &Workflow{
		Name: "deploy_model",
		Steps: []WorkflowStep{
			{
				Name: "validate_customer",
				Execute: func(ctx context.Context, mcpURL string, params map[string]interface{}, state map[string]interface{}) (interface{}, error) {
					// In real implementation, validate customer exists
					customerID, ok := params["customer_id"].(string)
					if !ok {
						return nil, fmt.Errorf("customer_id is required")
					}
					return map[string]interface{}{
						"valid":       true,
						"customer_id": customerID,
					}, nil
				},
			},
			{
				Name: "allocate_gpu",
				Execute: func(ctx context.Context, mcpURL string, params map[string]interface{}, state map[string]interface{}) (interface{}, error) {
					vgpu := 1.0
					if v, ok := params["vgpu"].(float64); ok {
						vgpu = v
					}

					return callMCPTool(ctx, mcpURL, "allocate_gpu", map[string]interface{}{
						"vgpu_size": vgpu,
						"duration":  "24h",
						"pool_name": "default-pool",
					})
				},
			},
			{
				Name: "deploy_model",
				Execute: func(ctx context.Context, mcpURL string, params map[string]interface{}, state map[string]interface{}) (interface{}, error) {
					return callMCPTool(ctx, mcpURL, "deploy_model", params)
				},
			},
			{
				Name: "validate_deployment",
				Execute: func(ctx context.Context, mcpURL string, params map[string]interface{}, state map[string]interface{}) (interface{}, error) {
					// Wait a bit for deployment to stabilize
					time.Sleep(5 * time.Second)

					// In real implementation, check deployment health
					deployResult, ok := state["deploy_model"].(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("invalid deploy_model result")
					}

					return map[string]interface{}{
						"healthy":      true,
						"endpoint_url": deployResult["endpoint_url"],
					}, nil
				},
			},
		},
	}
}

// NewTrainAndDeployWorkflow creates a training + deployment workflow
func NewTrainAndDeployWorkflow() *Workflow {
	return &Workflow{
		Name: "train_and_deploy",
		Steps: []WorkflowStep{
			{
				Name: "validate_dataset",
				Execute: func(ctx context.Context, mcpURL string, params map[string]interface{}, state map[string]interface{}) (interface{}, error) {
					datasetPath, ok := params["dataset_path"].(string)
					if !ok {
						return nil, fmt.Errorf("dataset_path is required")
					}

					return map[string]interface{}{
						"valid":        true,
						"dataset_path": datasetPath,
					}, nil
				},
			},
			{
				Name: "start_training",
				Execute: func(ctx context.Context, mcpURL string, params map[string]interface{}, state map[string]interface{}) (interface{}, error) {
					return callMCPTool(ctx, mcpURL, "start_training", params)
				},
			},
			{
				Name: "monitor_training",
				Execute: func(ctx context.Context, mcpURL string, params map[string]interface{}, state map[string]interface{}) (interface{}, error) {
					// In real implementation, poll job status until complete
					// For now, simulate waiting
					time.Sleep(10 * time.Second)

					return map[string]interface{}{
						"status":     "completed",
						"adapter_id": "adapter-" + fmt.Sprint(time.Now().Unix()),
					}, nil
				},
			},
			{
				Name: "validate_adapter",
				Execute: func(ctx context.Context, mcpURL string, params map[string]interface{}, state map[string]interface{}) (interface{}, error) {
					// In real implementation, run quality checks
					return map[string]interface{}{
						"valid":   true,
						"quality": "good",
					}, nil
				},
			},
			{
				Name: "deploy_model",
				Execute: func(ctx context.Context, mcpURL string, params map[string]interface{}, state map[string]interface{}) (interface{}, error) {
					// Add adapter to params
					adapterResult, ok := state["monitor_training"].(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("invalid monitor_training result")
					}

					deployParams := make(map[string]interface{})
					for k, v := range params {
						deployParams[k] = v
					}
					deployParams["adapter_id"] = adapterResult["adapter_id"]

					return callMCPTool(ctx, mcpURL, "deploy_model", deployParams)
				},
			},
		},
	}
}

// NewOptimizeCostsWorkflow creates a cost optimization workflow
func NewOptimizeCostsWorkflow() *Workflow {
	return &Workflow{
		Name: "optimize_costs",
		Steps: []WorkflowStep{
			{
				Name: "query_usage",
				Execute: func(ctx context.Context, mcpURL string, params map[string]interface{}, state map[string]interface{}) (interface{}, error) {
					customerID, ok := params["customer_id"].(string)
					if !ok {
						return nil, fmt.Errorf("customer_id is required")
					}

					return callMCPTool(ctx, mcpURL, "query_usage", map[string]interface{}{
						"filters": map[string]interface{}{
							"customer":   customerID,
							"time_range": "7d",
						},
					})
				},
			},
			{
				Name: "recommend_optimization",
				Execute: func(ctx context.Context, mcpURL string, params map[string]interface{}, state map[string]interface{}) (interface{}, error) {
					return callMCPTool(ctx, mcpURL, "recommend_optimization", params)
				},
			},
			{
				Name: "present_recommendations",
				Execute: func(ctx context.Context, mcpURL string, params map[string]interface{}, state map[string]interface{}) (interface{}, error) {
					recommendations, ok := state["recommend_optimization"].(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("invalid recommend_optimization result")
					}

					// In real implementation, this would wait for user approval
					// For now, auto-approve if savings > $100/month
					totalSavings, ok := recommendations["total_potential_savings"].(float64)
					if ok && totalSavings > 100 {
						return map[string]interface{}{
							"approved":        true,
							"total_savings":   totalSavings,
							"recommendations": recommendations,
						}, nil
					}

					return map[string]interface{}{
						"approved":        false,
						"reason":          "Savings too low to auto-approve",
						"recommendations": recommendations,
					}, nil
				},
			},
			{
				Name: "apply_optimizations",
				Execute: func(ctx context.Context, mcpURL string, params map[string]interface{}, state map[string]interface{}) (interface{}, error) {
					approval, ok := state["present_recommendations"].(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("invalid present_recommendations result")
					}

					approved, ok := approval["approved"].(bool)
					if !ok || !approved {
						return map[string]interface{}{
							"applied": false,
							"reason":  "Not approved",
						}, nil
					}

					// In real implementation, apply routing changes
					// For now, simulate
					return map[string]interface{}{
						"applied":       true,
						"changes_made":  []string{"updated_routing", "scaled_down_off_peak"},
						"expected_savings": approval["total_savings"],
					}, nil
				},
			},
		},
	}
}

// callMCPTool calls an MCP tool via HTTP
func callMCPTool(ctx context.Context, mcpURL string, toolName string, params map[string]interface{}) (interface{}, error) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  toolName,
		"params":  params,
		"id":      1,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/execute", mcpURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call MCP tool: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		JSONRPC string      `json:"jsonrpc"`
		Result  interface{} `json:"result"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    string `json:"data"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("MCP error: %s (%s)", result.Error.Message, result.Error.Data)
	}

	return result.Result, nil
}

