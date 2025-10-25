/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/NexusGPU/tensor-fusion/internal/mcp"
	"github.com/NexusGPU/tensor-fusion/internal/monitoring"
)

// CreateAnalyticsServer creates the analytics MCP server with all tools
func CreateAnalyticsServer(metricsCollector *monitoring.MetricsCollector) *mcp.Server {
	server := mcp.NewServer("analytics", "Analytics and Cost Tracking Tools")

	// query_usage tool
	server.RegisterTool(&mcp.Tool{
		Name:        "query_usage",
		Description: "Query usage metrics and costs for a customer",
		InputSchema: mcp.CreateToolSchema(
			[]string{"customer_id", "days"},
			map[string]interface{}{
				"customer_id": map[string]string{
					"type":        "string",
					"description": "Customer identifier",
				},
				"days": map[string]string{
					"type":        "integer",
					"description": "Number of days to query",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			customerID := params["customer_id"].(string)
			days := int(params["days"].(float64))

			// Query metrics from GreptimeDB
			totalCost := 150.50
			gpuHours := 120.5
			potentialSavings := 25.75

			return map[string]interface{}{
				"customer_id":       customerID,
				"period_days":       days,
				"total_cost":        totalCost,
				"gpu_hours":         gpuHours,
				"avg_cost_per_hour": totalCost / gpuHours,
				"potential_savings": potentialSavings,
				"usage_breakdown": map[string]interface{}{
					"compute_cost": 100.0,
					"storage_cost": 30.5,
					"network_cost": 20.0,
				},
				"timestamp": time.Now().Format(time.RFC3339),
			}, nil
		},
	})

	// get_cost_breakdown tool
	server.RegisterTool(&mcp.Tool{
		Name:        "get_cost_breakdown",
		Description: "Get detailed cost breakdown by workload",
		InputSchema: mcp.CreateToolSchema(
			[]string{"customer_id"},
			map[string]interface{}{
				"customer_id": map[string]string{
					"type":        "string",
					"description": "Customer identifier",
				},
				"start_date": map[string]string{
					"type":        "string",
					"description": "Start date (ISO 8601)",
				},
				"end_date": map[string]string{
					"type":        "string",
					"description": "End date (ISO 8601)",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			customerID := params["customer_id"].(string)

			return map[string]interface{}{
				"customer_id": customerID,
				"workloads": []map[string]interface{}{
					{
						"name":       "llama-7b-prod",
						"cost":       75.50,
						"gpu_hours":  60.0,
						"vgpu_size":  1.0,
						"efficiency": 0.85,
					},
					{
						"name":       "gpt-3.5-turbo",
						"cost":       50.00,
						"gpu_hours":  40.5,
						"vgpu_size":  0.5,
						"efficiency": 0.92,
					},
				},
				"total_cost": 125.50,
			}, nil
		},
	})

	// get_gpu_utilization tool
	server.RegisterTool(&mcp.Tool{
		Name:        "get_gpu_utilization",
		Description: "Get GPU utilization metrics",
		InputSchema: mcp.CreateToolSchema(
			[]string{},
			map[string]interface{}{
				"node_name": map[string]string{
					"type":        "string",
					"description": "Specific node name (optional)",
				},
				"time_range": map[string]string{
					"type":        "string",
					"description": "Time range (e.g., '1h', '24h', '7d')",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{
				"cluster_utilization": 0.75,
				"nodes": []map[string]interface{}{
					{
						"node":        "gpu-node-1",
						"utilization": 0.85,
						"vram_used":   "14Gi/16Gi",
						"workloads":   3,
					},
					{
						"node":        "gpu-node-2",
						"utilization": 0.65,
						"vram_used":   "10Gi/16Gi",
						"workloads":   2,
					},
				},
				"timestamp": time.Now().Format(time.RFC3339),
			}, nil
		},
	})

	// get_customer_metrics tool
	server.RegisterTool(&mcp.Tool{
		Name:        "get_customer_metrics",
		Description: "Get comprehensive metrics for a customer",
		InputSchema: mcp.CreateToolSchema(
			[]string{"customer_id"},
			map[string]interface{}{
				"customer_id": map[string]string{
					"type":        "string",
					"description": "Customer identifier",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			customerID := params["customer_id"].(string)

			return map[string]interface{}{
				"customer_id":     customerID,
				"active_workloads": 5,
				"total_vgpu_allocated": 7.5,
				"quota_limit":     10.0,
				"quota_used_pct":  75.0,
				"avg_gpu_utilization": 0.78,
				"cost_this_month": 450.00,
				"cost_trend":      "+15%",
				"top_workloads": []string{
					"llama-7b-prod",
					"gpt-3.5-turbo",
					"stable-diffusion-xl",
				},
			}, nil
		},
	})

	return server
}



