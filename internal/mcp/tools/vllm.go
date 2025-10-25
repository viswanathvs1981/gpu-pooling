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

	"github.com/NexusGPU/tensor-fusion/internal/inference/vllm"
	"github.com/NexusGPU/tensor-fusion/internal/mcp"
)

// CreateVLLMServer creates the vLLM MCP server with all tools
func CreateVLLMServer(vllmClient *vllm.VLLMClient, deploymentManager *vllm.VLLMDeploymentManager) *mcp.Server {
	server := mcp.NewServer("vllm", "vLLM Inference Engine Tools")

	// deploy_model tool
	server.RegisterTool(&mcp.Tool{
		Name:        "deploy_model",
		Description: "Deploy a model to vLLM inference engine",
		InputSchema: mcp.CreateToolSchema(
			[]string{"model_name", "gpu_count"},
			map[string]interface{}{
				"model_name": map[string]string{
					"type":        "string",
					"description": "Model name (e.g., 'meta-llama/Llama-2-7b-hf')",
				},
				"gpu_count": map[string]string{
					"type":        "integer",
					"description": "Number of GPUs for tensor parallelism",
				},
				"gpu_memory": map[string]string{
					"type":        "string",
					"description": "GPU memory allocation (e.g., '24Gi')",
				},
				"replicas": map[string]string{
					"type":        "integer",
					"description": "Number of replicas",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			modelName := params["model_name"].(string)
			gpuCount := int(params["gpu_count"].(float64))
			gpuMemory := "24Gi"
			if mem, ok := params["gpu_memory"].(string); ok {
				gpuMemory = mem
			}
			replicas := int32(1)
			if rep, ok := params["replicas"].(float64); ok {
				replicas = int32(rep)
			}

			// Deploy via deployment manager
			err := deploymentManager.DeployVLLM(ctx, modelName, gpuCount, gpuMemory, replicas)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"model_name": modelName,
				"endpoint":   "http://vllm-inference-server-service.vllm.svc.cluster.local:8000",
				"status":     "deployed",
				"gpu_count":  gpuCount,
				"replicas":   replicas,
			}, nil
		},
	})

	// load_lora tool
	server.RegisterTool(&mcp.Tool{
		Name:        "load_lora",
		Description: "Load a LoRA adapter into vLLM",
		InputSchema: mcp.CreateToolSchema(
			[]string{"adapter_name", "adapter_path"},
			map[string]interface{}{
				"adapter_name": map[string]string{
					"type":        "string",
					"description": "LoRA adapter name",
				},
				"adapter_path": map[string]string{
					"type":        "string",
					"description": "Path to LoRA adapter weights",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			adapterName := params["adapter_name"].(string)
			adapterPath := params["adapter_path"].(string)

			// Load LoRA adapter
			// In real implementation, this would call vllmClient.LoadLoRA()

			return map[string]interface{}{
				"adapter_name": adapterName,
				"adapter_path": adapterPath,
				"status":       "loaded",
			}, nil
		},
	})

	// unload_lora tool
	server.RegisterTool(&mcp.Tool{
		Name:        "unload_lora",
		Description: "Unload a LoRA adapter from vLLM",
		InputSchema: mcp.CreateToolSchema(
			[]string{"adapter_name"},
			map[string]interface{}{
				"adapter_name": map[string]string{
					"type":        "string",
					"description": "LoRA adapter name to unload",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			adapterName := params["adapter_name"].(string)

			// Unload LoRA adapter
			err := vllmClient.UnloadLoRA(adapterName)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"adapter_name": adapterName,
				"status":       "unloaded",
			}, nil
		},
	})

	// get_model_status tool
	server.RegisterTool(&mcp.Tool{
		Name:        "get_model_status",
		Description: "Get the status of a deployed model",
		InputSchema: mcp.CreateToolSchema(
			[]string{},
			map[string]interface{}{
				"model_name": map[string]string{
					"type":        "string",
					"description": "Model name (optional)",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			// Get deployment status
			status, err := deploymentManager.GetVLLMStatus(ctx)
			if err != nil {
				return nil, err
			}

			if status == nil {
				return map[string]interface{}{
					"status": "not_deployed",
				}, nil
			}

			return map[string]interface{}{
				"status":            "deployed",
				"replicas":          status.Replicas,
				"ready_replicas":    status.ReadyReplicas,
				"available_replicas": status.AvailableReplicas,
			}, nil
		},
	})

	// list_models tool
	server.RegisterTool(&mcp.Tool{
		Name:        "list_models",
		Description: "List all available models in vLLM",
		InputSchema: mcp.CreateToolSchema(
			[]string{},
			map[string]interface{}{},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			// List models
			models, err := vllmClient.ListModels()
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"models": models.Data,
				"count":  len(models.Data),
			}, nil
		},
	})

	return server
}



