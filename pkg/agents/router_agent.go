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

	"github.com/NexusGPU/tensor-fusion/internal/llmgateway/portkey"
	"github.com/NexusGPU/tensor-fusion/internal/mcp"
)

// RouterAgent handles intelligent request routing
type RouterAgent struct {
	*Agent
	MessageBus        *MessageBus
	RoutingController *portkey.RoutingController
}

// NewRouterAgent creates a router agent
func NewRouterAgent(mcpGateway *mcp.Gateway, messageBus *MessageBus, routingController *portkey.RoutingController) *RouterAgent {
	return &RouterAgent{
		Agent: NewAgent("router-agent", "Router Agent",
			"Manages intelligent request routing", mcpGateway),
		MessageBus:        messageBus,
		RoutingController: routingController,
	}
}

// AddRoute adds a new routing configuration
func (ra *RouterAgent) AddRoute(ctx context.Context, modelID, customerID string, endpoint interface{}) error {
	routeConfig := &portkey.RouteConfig{
		Name:     fmt.Sprintf("%s-%s", customerID, modelID),
		Strategy: "cost-based",
		Targets: []portkey.TargetConfig{
			{
				Provider:     "self-hosted",
				VirtualKey:   endpoint.(string),
				Weight:       100,
				CostPerToken: 0.0001,
			},
		},
	}

	return ra.RoutingController.CreateRoute(ctx, routeConfig)
}

// OptimizeRouting optimizes routing for cost or performance
func (ra *RouterAgent) OptimizeRouting(ctx context.Context, customerID, strategy string) error {
	// Get current usage patterns from Cost Agent
	costMsg := &Message{
		From:   ra.ID,
		To:     "cost-agent",
		Type:   "request",
		Method: "get_usage_patterns",
		Params: map[string]interface{}{
			"customer_id": customerID,
		},
	}

	costResp, err := ra.MessageBus.Request(ctx, costMsg, 10)
	if err != nil {
		return fmt.Errorf("failed to get usage patterns: %w", err)
	}

	patterns := costResp.Result.(map[string]interface{})

	// Update routing based on patterns
	var targets []portkey.TargetConfig
	if strategy == "cost-based" {
		// Route long requests to self-hosted (cheaper)
		targets = []portkey.TargetConfig{
			{
				Provider:     "self-hosted",
				Weight:       70,
				CostPerToken: 0.0001,
			},
			{
				Provider:     "azure",
				Weight:       30,
				CostPerToken: 0.0005,
			},
		}
	}

	routeConfig := &portkey.RouteConfig{
		Name:     fmt.Sprintf("%s-optimized", customerID),
		Strategy: strategy,
		Targets:  targets,
		Metadata: patterns,
	}

	return ra.RoutingController.UpdateRoute(ctx, routeConfig.Name, routeConfig)
}

// Start starts the router agent
func (ra *RouterAgent) Start(ctx context.Context) error {
	return ra.MessageBus.Subscribe(ctx, ra.ID, func(msg *Message) {
		var result interface{}
		var err error

		switch msg.Method {
		case "add_route":
			modelID := msg.Params["model_id"].(string)
			customerID := msg.Params["customer_id"].(string)
			endpoint := msg.Params["endpoint"]
			err = ra.AddRoute(ctx, modelID, customerID, endpoint)
			result = map[string]string{"status": "route_added"}

		case "optimize_routing":
			customerID := msg.Params["customer_id"].(string)
			strategy := msg.Params["strategy"].(string)
			err = ra.OptimizeRouting(ctx, customerID, strategy)
			result = map[string]string{"status": "routing_optimized"}

		default:
			err = fmt.Errorf("unknown method: %s", msg.Method)
		}

		ra.MessageBus.SendResponse(ctx, msg, result, err)
	})
}



