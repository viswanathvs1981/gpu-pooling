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
	"time"

	"github.com/NexusGPU/tensor-fusion/internal/mcp"
)

// OrchestratorAgent coordinates multiple agents
type OrchestratorAgent struct {
	*Agent
	MessageBus *MessageBus
	Agents     map[string]string // agentID -> agentType
}

// NewOrchestratorAgent creates an orchestrator agent
func NewOrchestratorAgent(mcpGateway *mcp.Gateway, messageBus *MessageBus) *OrchestratorAgent {
	return &OrchestratorAgent{
		Agent: NewAgent("orchestrator", "Orchestrator Agent",
			"Coordinates multi-agent workflows", mcpGateway),
		MessageBus: messageBus,
		Agents:     make(map[string]string),
	}
}

// RegisterAgent registers an agent with the orchestrator
func (oa *OrchestratorAgent) RegisterAgent(agentID, agentType string) {
	oa.Agents[agentID] = agentType
}

// ExecuteWorkflow executes a multi-agent workflow
func (oa *OrchestratorAgent) ExecuteWorkflow(ctx context.Context, workflowType string, params map[string]interface{}) error {
	switch workflowType {
	case "deploy_model":
		return oa.deployModelWorkflow(ctx, params)
	case "optimize_costs":
		return oa.optimizeCostsWorkflow(ctx, params)
	case "train_and_deploy":
		return oa.trainAndDeployWorkflow(ctx, params)
	default:
		return fmt.Errorf("unknown workflow type: %s", workflowType)
	}
}

// deployModelWorkflow orchestrates model deployment
func (oa *OrchestratorAgent) deployModelWorkflow(ctx context.Context, params map[string]interface{}) error {
	modelID := params["model_id"].(string)
	customerID := params["customer_id"].(string)

	// Step 1: Check resources with Resource Agent
	resourceMsg := &Message{
		From:      oa.ID,
		To:        "resource-agent",
		Type:      "request",
		Method:    "check_capacity",
		Params:    map[string]interface{}{"required_vgpu": 1.0},
		Timestamp: time.Now(),
	}

	resourceResp, err := oa.MessageBus.Request(ctx, resourceMsg, 10*time.Second)
	if err != nil {
		return fmt.Errorf("resource check failed: %w", err)
	}

	if resourceResp.Error != "" {
		return fmt.Errorf("insufficient resources: %s", resourceResp.Error)
	}

	// Step 2: Deploy with Deployment Agent
	deployMsg := &Message{
		From:   oa.ID,
		To:     "deployment-agent",
		Type:   "request",
		Method: "deploy_model",
		Params: map[string]interface{}{
			"model_id":    modelID,
			"customer_id": customerID,
			"config":      params,
		},
		Timestamp: time.Now(),
	}

	deployResp, err := oa.MessageBus.Request(ctx, deployMsg, 60*time.Second)
	if err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	if deployResp.Error != "" {
		return fmt.Errorf("deployment error: %s", deployResp.Error)
	}

	// Step 3: Update routing with Router Agent
	routeMsg := &Message{
		From:   oa.ID,
		To:     "router-agent",
		Type:   "request",
		Method: "add_route",
		Params: map[string]interface{}{
			"model_id":    modelID,
			"customer_id": customerID,
			"endpoint":    deployResp.Result,
		},
		Timestamp: time.Now(),
	}

	_, err = oa.MessageBus.Request(ctx, routeMsg, 10*time.Second)
	if err != nil {
		return fmt.Errorf("routing update failed: %w", err)
	}

	// Step 4: Notify via Slack (MCP tool)
	_, err = oa.UseTool(ctx, "slack", "send_message", map[string]interface{}{
		"channel": "#deployments",
		"message": fmt.Sprintf("✅ Model %s deployed for %s", modelID, customerID),
	})

	return err
}

// optimizeCostsWorkflow orchestrates cost optimization
func (oa *OrchestratorAgent) optimizeCostsWorkflow(ctx context.Context, params map[string]interface{}) error {
	customerID := params["customer_id"].(string)

	// Step 1: Analyze costs with Cost Agent
	costMsg := &Message{
		From:   oa.ID,
		To:     "cost-agent",
		Type:   "request",
		Method: "analyze_costs",
		Params: map[string]interface{}{
			"customer_id": customerID,
			"days":        7,
		},
		Timestamp: time.Now(),
	}

	costResp, err := oa.MessageBus.Request(ctx, costMsg, 30*time.Second)
	if err != nil {
		return fmt.Errorf("cost analysis failed: %w", err)
	}

	analysis, ok := costResp.Result.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid cost analysis response")
	}

	// Step 2: Get optimization recommendations
	savings, ok := analysis["potential_savings"].(float64)
	if !ok || savings < 10.0 {
		return nil // Not worth optimizing
	}

	// Step 3: Apply routing changes with Router Agent
	routeMsg := &Message{
		From:   oa.ID,
		To:     "router-agent",
		Type:   "request",
		Method: "optimize_routing",
		Params: map[string]interface{}{
			"customer_id": customerID,
			"strategy":    "cost-based",
		},
		Timestamp: time.Now(),
	}

	routeResp, err := oa.MessageBus.Request(ctx, routeMsg, 10*time.Second)
	if err != nil {
		return fmt.Errorf("routing optimization failed: %w", err)
	}

	// Step 4: Notify user
	_, err = oa.UseTool(ctx, "slack", "send_message", map[string]interface{}{
		"channel": customerID,
		"message": fmt.Sprintf("💰 Optimized routing to save $%.2f/week", savings),
	})

	return err
}

// trainAndDeployWorkflow orchestrates training and deployment
func (oa *OrchestratorAgent) trainAndDeployWorkflow(ctx context.Context, params map[string]interface{}) error {
	modelName := params["model_name"].(string)
	datasetPath := params["dataset_path"].(string)

	// Step 1: Start training with Training Agent
	trainMsg := &Message{
		From:   oa.ID,
		To:     "training-agent",
		Type:   "request",
		Method: "start_training",
		Params: map[string]interface{}{
			"model_name":   modelName,
			"dataset_path": datasetPath,
			"base_model":   params["base_model"],
		},
		Timestamp: time.Now(),
	}

	trainResp, err := oa.MessageBus.Request(ctx, trainMsg, 120*time.Second)
	if err != nil {
		return fmt.Errorf("training failed: %w", err)
	}

	// Step 2: Deploy trained model
	return oa.deployModelWorkflow(ctx, map[string]interface{}{
		"model_id":    trainResp.Result.(map[string]interface{})["model_id"],
		"customer_id": params["customer_id"],
	})
}

// Start starts the orchestrator and begins listening for requests
func (oa *OrchestratorAgent) Start(ctx context.Context) error {
	return oa.MessageBus.Subscribe(ctx, oa.ID, func(msg *Message) {
		// Handle incoming messages
		var result interface{}
		var err error

		switch msg.Method {
		case "execute_workflow":
			workflowType := msg.Params["workflow_type"].(string)
			err = oa.ExecuteWorkflow(ctx, workflowType, msg.Params)
			result = map[string]string{"status": "success"}
		default:
			err = fmt.Errorf("unknown method: %s", msg.Method)
		}

		// Send response
		oa.MessageBus.SendResponse(ctx, msg, result, err)
	})
}



