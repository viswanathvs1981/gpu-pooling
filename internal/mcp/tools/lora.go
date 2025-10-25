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

	"github.com/NexusGPU/tensor-fusion/internal/inference/lora"
	"github.com/NexusGPU/tensor-fusion/internal/mcp"
	"github.com/NexusGPU/tensor-fusion/internal/training"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateLoRAServer creates the LoRA MCP server with all tools
func CreateLoRAServer(loraRegistry *lora.LoRAAdapterRegistry, trainingManager *training.TrainingJobManager) *mcp.Server {
	server := mcp.NewServer("lora", "LoRA Fine-tuning and Adapter Management Tools")

	// register_adapter tool
	server.RegisterTool(&mcp.Tool{
		Name:        "register_adapter",
		Description: "Register a LoRA adapter in the registry",
		InputSchema: mcp.CreateToolSchema(
			[]string{"name", "base_model", "path"},
			map[string]interface{}{
				"name": map[string]string{
					"type":        "string",
					"description": "Adapter name",
				},
				"base_model": map[string]string{
					"type":        "string",
					"description": "Base model name",
				},
				"path": map[string]string{
					"type":        "string",
					"description": "Path to adapter weights",
				},
				"size": map[string]string{
					"type":        "string",
					"description": "Adapter size (e.g., '100MB')",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			name := params["name"].(string)
			baseModel := params["base_model"].(string)
			path := params["path"].(string)
			size := "100MB"
			if s, ok := params["size"].(string); ok {
				size = s
			}

			adapter := lora.LoRAAdapter{
				Name:      name,
				BaseModel: baseModel,
				Path:      path,
				Size:      size,
				Created:   metav1.Now(),
				Labels: map[string]string{
					"app": "lora-adapter",
				},
			}

			err := loraRegistry.RegisterAdapter(ctx, adapter)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"name":       name,
				"base_model": baseModel,
				"path":       path,
				"status":     "registered",
			}, nil
		},
	})

	// start_training tool
	server.RegisterTool(&mcp.Tool{
		Name:        "start_training",
		Description: "Start a LoRA fine-tuning job",
		InputSchema: mcp.CreateToolSchema(
			[]string{"name", "base_model_name", "dataset_pvc", "output_pvc"},
			map[string]interface{}{
				"name": map[string]string{
					"type":        "string",
					"description": "Training job name",
				},
				"base_model_name": map[string]string{
					"type":        "string",
					"description": "Base model to fine-tune",
				},
				"dataset_pvc": map[string]string{
					"type":        "string",
					"description": "PVC containing training dataset",
				},
				"output_pvc": map[string]string{
					"type":        "string",
					"description": "PVC for output adapter",
				},
				"gpu_number": map[string]string{
					"type":        "integer",
					"description": "Number of GPUs",
				},
				"lora_rank": map[string]string{
					"type":        "integer",
					"description": "LoRA rank (default: 8)",
				},
				"learning_rate": map[string]string{
					"type":        "number",
					"description": "Learning rate (default: 0.0001)",
				},
				"epochs": map[string]string{
					"type":        "integer",
					"description": "Number of epochs (default: 3)",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			config := training.TrainingJobConfig{
				Name:          params["name"].(string),
				BaseModelName: params["base_model_name"].(string),
				DatasetPVC:    params["dataset_pvc"].(string),
				OutputPVC:     params["output_pvc"].(string),
				GPUNumber:     1,
				GPUMemory:     "24Gi",
				LoRARank:      8,
				LearningRate:  0.0001,
				Epochs:        3,
			}

			if gpuNum, ok := params["gpu_number"].(float64); ok {
				config.GPUNumber = int(gpuNum)
			}
			if rank, ok := params["lora_rank"].(float64); ok {
				config.LoRARank = int(rank)
			}
			if lr, ok := params["learning_rate"].(float64); ok {
				config.LearningRate = lr
			}
			if epochs, ok := params["epochs"].(float64); ok {
				config.Epochs = int(epochs)
			}

			err := trainingManager.CreateTrainingJob(ctx, config)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"job_name":   config.Name,
				"base_model": config.BaseModelName,
				"status":     "started",
				"lora_rank":  config.LoRARank,
				"epochs":     config.Epochs,
			}, nil
		},
	})

	// get_training_status tool
	server.RegisterTool(&mcp.Tool{
		Name:        "get_training_status",
		Description: "Get the status of a training job",
		InputSchema: mcp.CreateToolSchema(
			[]string{"job_name"},
			map[string]interface{}{
				"job_name": map[string]string{
					"type":        "string",
					"description": "Training job name",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			jobName := params["job_name"].(string)

			status, err := trainingManager.GetTrainingJobStatus(ctx, jobName)
			if err != nil {
				return nil, err
			}

			if status == nil {
				return map[string]interface{}{
					"job_name": jobName,
					"status":   "not_found",
				}, nil
			}

			return map[string]interface{}{
				"job_name":        jobName,
				"active":          status.Active,
				"succeeded":       status.Succeeded,
				"failed":          status.Failed,
				"completion_time": status.CompletionTime,
			}, nil
		},
	})

	// list_adapters tool
	server.RegisterTool(&mcp.Tool{
		Name:        "list_adapters",
		Description: "List all registered LoRA adapters",
		InputSchema: mcp.CreateToolSchema(
			[]string{},
			map[string]interface{}{
				"base_model": map[string]string{
					"type":        "string",
					"description": "Filter by base model (optional)",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			adapters, err := loraRegistry.ListAdapters(ctx)
			if err != nil {
				return nil, err
			}

			// Filter by base model if specified
			if baseModel, ok := params["base_model"].(string); ok {
				filtered := []lora.LoRAAdapter{}
				for _, adapter := range adapters {
					if adapter.BaseModel == baseModel {
						filtered = append(filtered, adapter)
					}
				}
				adapters = filtered
			}

			return map[string]interface{}{
				"adapters": adapters,
				"count":    len(adapters),
			}, nil
		},
	})

	// delete_adapter tool
	server.RegisterTool(&mcp.Tool{
		Name:        "delete_adapter",
		Description: "Delete a LoRA adapter from the registry",
		InputSchema: mcp.CreateToolSchema(
			[]string{"name"},
			map[string]interface{}{
				"name": map[string]string{
					"type":        "string",
					"description": "Adapter name to delete",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			name := params["name"].(string)

			err := loraRegistry.DeleteAdapter(ctx, name)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"name":   name,
				"status": "deleted",
			}, nil
		},
	})

	return server
}



