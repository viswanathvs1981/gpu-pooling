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

package agents

import (
	"context"
	"fmt"

	"github.com/NexusGPU/tensor-fusion/internal/gpuallocator"
	"github.com/NexusGPU/tensor-fusion/internal/mcp"
)

// ResourceAgent manages GPU resources
type ResourceAgent struct {
	*Agent
	MessageBus *MessageBus
	Allocator  *gpuallocator.GpuAllocator
}

// NewResourceAgent creates a resource agent
func NewResourceAgent(mcpGateway *mcp.Gateway, messageBus *MessageBus, allocator *gpuallocator.GpuAllocator) *ResourceAgent {
	return &ResourceAgent{
		Agent: NewAgent("resource-agent", "Resource Agent",
			"Manages GPU resource allocation", mcpGateway),
		MessageBus: messageBus,
		Allocator:  allocator,
	}
}

// CheckCapacity checks if sufficient resources are available
func (ra *ResourceAgent) CheckCapacity(ctx context.Context, requiredVGPU float64) (bool, error) {
	// Use MCP tool to query resource availability
	resp, err := ra.UseTool(ctx, "platform", "get_available_resources", map[string]interface{}{
		"required_vgpu": requiredVGPU,
	})

	if err != nil {
		return false, err
	}

	if !resp.Success {
		return false, fmt.Errorf(resp.Error)
	}

	available, ok := resp.Result.(map[string]interface{})["available"].(bool)
	return available && ok, nil
}

// AllocateResources allocates GPU resources
func (ra *ResourceAgent) AllocateResources(ctx context.Context, workloadName string, vgpu float64) error {
	resp, err := ra.UseTool(ctx, "platform", "allocate_gpu", map[string]interface{}{
		"workload_name": workloadName,
		"vgpu_size":     vgpu,
	})

	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("allocation failed: %s", resp.Error)
	}

	ra.UpdateState("last_allocation", map[string]interface{}{
		"workload": workloadName,
		"vgpu":     vgpu,
	})

	return nil
}

// Start starts the resource agent
func (ra *ResourceAgent) Start(ctx context.Context) error {
	return ra.MessageBus.Subscribe(ctx, ra.ID, func(msg *Message) {
		var result interface{}
		var err error

		switch msg.Method {
		case "check_capacity":
			requiredVGPU := msg.Params["required_vgpu"].(float64)
			available, checkErr := ra.CheckCapacity(ctx, requiredVGPU)
			if checkErr != nil {
				err = checkErr
			} else {
				result = map[string]interface{}{
					"available":     available,
					"available_vgpu": 2.5, // Mock value
				}
			}

		case "allocate_resources":
			workloadName := msg.Params["workload_name"].(string)
			vgpu := msg.Params["vgpu"].(float64)
			err = ra.AllocateResources(ctx, workloadName, vgpu)
			result = map[string]string{"status": "allocated"}

		default:
			err = fmt.Errorf("unknown method: %s", msg.Method)
		}

		ra.MessageBus.SendResponse(ctx, msg, result, err)
	})
}



