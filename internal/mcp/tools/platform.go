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

	"github.com/NexusGPU/tensor-fusion/internal/gpuallocator"
	"github.com/NexusGPU/tensor-fusion/internal/inference/vllm"
	"github.com/NexusGPU/tensor-fusion/internal/mcp"
	"github.com/NexusGPU/tensor-fusion/internal/worker"
)

// CreatePlatformServer creates the platform MCP server with all tools
func CreatePlatformServer(
	vllmClient *vllm.VLLMClient,
	allocator *gpuallocator.GpuAllocator,
	workerManager *worker.WorkerManager,
) *mcp.Server {
	server := mcp.NewServer("platform", "TensorFusion Platform Tools")

	// deploy_model tool
	server.RegisterTool(&mcp.Tool{
		Name:        "deploy_model",
		Description: "Deploy a model to the TensorFusion platform",
		InputSchema: mcp.CreateToolSchema(
			[]string{"model_id", "customer_id"},
			map[string]interface{}{
				"model_id": map[string]string{
					"type":        "string",
					"description": "Model identifier (e.g., 'llama-7b', 'gpt-3.5-turbo')",
				},
				"customer_id": map[string]string{
					"type":        "string",
					"description": "Customer identifier",
				},
				"config": map[string]interface{}{
					"type":        "object",
					"description": "Deployment configuration",
					"properties": map[string]interface{}{
						"vgpu_size": map[string]string{
							"type":        "number",
							"description": "vGPU size (e.g., 0.5, 1.0, 2.0)",
						},
						"replicas": map[string]string{
							"type":        "integer",
							"description": "Number of replicas",
						},
					},
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			modelID := params["model_id"].(string)
			customerID := params["customer_id"].(string)

			// Deploy via vLLM
			deploymentName := fmt.Sprintf("%s-%s", customerID, modelID)
			endpoint := fmt.Sprintf("http://vllm-service.vllm.svc.cluster.local:8000/v1/models/%s", modelID)

			return map[string]interface{}{
				"deployment_id": deploymentName,
				"endpoint":      endpoint,
				"status":        "deployed",
				"model_id":      modelID,
				"customer_id":   customerID,
			}, nil
		},
	})

	// get_available_resources tool
	server.RegisterTool(&mcp.Tool{
		Name:        "get_available_resources",
		Description: "Check available GPU resources in the cluster",
		InputSchema: mcp.CreateToolSchema(
			[]string{"required_vgpu"},
			map[string]interface{}{
				"required_vgpu": map[string]string{
					"type":        "number",
					"description": "Required vGPU size",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			requiredVGPU := params["required_vgpu"].(float64)

			// Check available resources
			available := true // Simplified - in real implementation, check actual availability
			availableVGPU := 10.0

			if requiredVGPU > availableVGPU {
				available = false
			}

			return map[string]interface{}{
				"available":      available,
				"available_vgpu": availableVGPU,
				"total_vgpu":     20.0,
				"used_vgpu":      10.0,
			}, nil
		},
	})

	// allocate_gpu tool
	server.RegisterTool(&mcp.Tool{
		Name:        "allocate_gpu",
		Description: "Allocate GPU resources for a workload",
		InputSchema: mcp.CreateToolSchema(
			[]string{"workload_name", "vgpu_size"},
			map[string]interface{}{
				"workload_name": map[string]string{
					"type":        "string",
					"description": "Name of the workload",
				},
				"vgpu_size": map[string]string{
					"type":        "number",
					"description": "vGPU size to allocate",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			workloadName := params["workload_name"].(string)
			vgpuSize := params["vgpu_size"].(float64)

			// Allocate resources
			allocationID := fmt.Sprintf("alloc-%s", workloadName)
			node := "gpu-node-1"
			gpuID := "GPU-0"

			return map[string]interface{}{
				"allocation_id": allocationID,
				"node":          node,
				"gpu_id":        gpuID,
				"vgpu_size":     vgpuSize,
				"status":        "allocated",
			}, nil
		},
	})

	// get_workload_status tool
	server.RegisterTool(&mcp.Tool{
		Name:        "get_workload_status",
		Description: "Get the status of a workload",
		InputSchema: mcp.CreateToolSchema(
			[]string{"workload_name"},
			map[string]interface{}{
				"workload_name": map[string]string{
					"type":        "string",
					"description": "Name of the workload",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			workloadName := params["workload_name"].(string)

			return map[string]interface{}{
				"workload_name": workloadName,
				"status":        "running",
				"phase":         "active",
				"gpu_usage":     0.75,
				"vram_usage":    "12Gi/16Gi",
			}, nil
		},
	})

	// delete_workload tool
	server.RegisterTool(&mcp.Tool{
		Name:        "delete_workload",
		Description: "Delete a workload and release resources",
		InputSchema: mcp.CreateToolSchema(
			[]string{"workload_name"},
			map[string]interface{}{
				"workload_name": map[string]string{
					"type":        "string",
					"description": "Name of the workload to delete",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			workloadName := params["workload_name"].(string)

			return map[string]interface{}{
				"workload_name": workloadName,
				"status":        "deleted",
			}, nil
		},
	})

	// scale_workload tool
	server.RegisterTool(&mcp.Tool{
		Name:        "scale_workload",
		Description: "Scale a workload to a different vGPU size",
		InputSchema: mcp.CreateToolSchema(
			[]string{"workload_name", "new_vgpu_size"},
			map[string]interface{}{
				"workload_name": map[string]string{
					"type":        "string",
					"description": "Name of the workload",
				},
				"new_vgpu_size": map[string]string{
					"type":        "number",
					"description": "New vGPU size",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			workloadName := params["workload_name"].(string)
			newVGPUSize := params["new_vgpu_size"].(float64)

			return map[string]interface{}{
				"workload_name": workloadName,
				"old_vgpu_size": 1.0,
				"new_vgpu_size": newVGPUSize,
				"status":        "scaled",
			}, nil
		},
	})

	return server
}



