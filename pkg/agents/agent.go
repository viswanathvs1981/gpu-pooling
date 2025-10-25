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
	"encoding/json"
	"fmt"
	"time"

	"github.com/NexusGPU/tensor-fusion/internal/mcp"
)

// Agent represents an autonomous agent
type Agent struct {
	ID          string
	Name        string
	Description string
	MCPGateway  *mcp.Gateway
	MessageBus  *MessageBus
	State       map[string]interface{}
}

// Message represents a message between agents
type Message struct {
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Type      string                 `json:"type"` // "request", "response", "event"
	Method    string                 `json:"method,omitempty"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Result    interface{}            `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewAgent creates a new agent
func NewAgent(id, name, description string, mcpGateway *mcp.Gateway, messageBus *MessageBus) *Agent {
	return &Agent{
		ID:          id,
		Name:        name,
		Description: description,
		MCPGateway:  mcpGateway,
		MessageBus:  messageBus,
		State:       make(map[string]interface{}),
	}
}

// SendMessage sends a message to another agent
func (a *Agent) SendMessage(ctx context.Context, to string, method string, params map[string]interface{}) (*Message, error) {
	msg := &Message{
		From:      a.ID,
		To:        to,
		Type:      "request",
		Method:    method,
		Params:    params,
		Timestamp: time.Now(),
	}

	// Send via Redis message bus
	if a.MessageBus != nil {
		return msg, a.MessageBus.Publish(ctx, msg)
	}

	// Fallback for testing without message bus
	return msg, nil
}

// SendRequest sends a request and waits for response
func (a *Agent) SendRequest(ctx context.Context, to string, method string, params map[string]interface{}, timeout time.Duration) (*Message, error) {
	msg := &Message{
		From:      a.ID,
		To:        to,
		Type:      "request",
		Method:    method,
		Params:    params,
		Timestamp: time.Now(),
	}

	if a.MessageBus != nil {
		return a.MessageBus.Request(ctx, msg, timeout)
	}

	return nil, fmt.Errorf("message bus not configured")
}

// UseTool calls an MCP tool
func (a *Agent) UseTool(ctx context.Context, serverName, toolName string, params map[string]interface{}) (*mcp.ToolResponse, error) {
	req := &mcp.ToolRequest{
		ToolName: toolName,
		Params:   params,
	}

	return a.MCPGateway.ExecuteTool(ctx, serverName, req)
}

// UpdateState updates agent state
func (a *Agent) UpdateState(key string, value interface{}) {
	a.State[key] = value
}

// GetState retrieves a state value
func (a *Agent) GetState(key string) (interface{}, bool) {
	val, exists := a.State[key]
	return val, exists
}

// ToJSON converts agent to JSON
func (a *Agent) ToJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"id":          a.ID,
		"name":        a.Name,
		"description": a.Description,
		"state":       a.State,
	})
}

// DeploymentAgent handles model deployments
type DeploymentAgent struct {
	*Agent
}

// NewDeploymentAgent creates a deployment agent
func NewDeploymentAgent(mcpGateway *mcp.Gateway, messageBus *MessageBus) *DeploymentAgent {
	return &DeploymentAgent{
		Agent: NewAgent("deployment-agent", "Deployment Agent", 
			"Manages model deployments and lifecycle", mcpGateway, messageBus),
	}
}

// Start starts the deployment agent
func (da *DeploymentAgent) Start(ctx context.Context) error {
	if da.MessageBus == nil {
		return fmt.Errorf("message bus not configured")
	}

	return da.MessageBus.Subscribe(ctx, da.ID, func(msg *Message) {
		var result interface{}
		var err error

		switch msg.Method {
		case "deploy_model":
			modelID := msg.Params["model_id"].(string)
			customerID := msg.Params["customer_id"].(string)
			config := msg.Params["config"].(map[string]interface{})
			err = da.DeployModel(ctx, modelID, customerID, config)
			result = map[string]string{"status": "deployed", "model_id": modelID}
		default:
			err = fmt.Errorf("unknown method: %s", msg.Method)
		}

		da.MessageBus.SendResponse(ctx, msg, result, err)
	})
}

// DeployModel handles model deployment
func (da *DeploymentAgent) DeployModel(ctx context.Context, modelID, customerID string, config map[string]interface{}) error {
	// Use platform MCP tool to deploy
	resp, err := da.UseTool(ctx, "platform", "deploy_model", map[string]interface{}{
		"model_id":    modelID,
		"customer_id": customerID,
		"config":      config,
	})

	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("deployment failed: %s", resp.Error)
	}

	da.UpdateState("last_deployment", map[string]interface{}{
		"model_id":    modelID,
		"customer_id": customerID,
		"timestamp":   time.Now(),
		"status":      "success",
	})

	return nil
}

// CostAgent handles cost optimization
type CostAgent struct {
	*Agent
}

// NewCostAgent creates a cost agent
func NewCostAgent(mcpGateway *mcp.Gateway, messageBus *MessageBus) *CostAgent {
	return &CostAgent{
		Agent: NewAgent("cost-agent", "Cost Agent", 
			"Monitors and optimizes costs", mcpGateway, messageBus),
	}
}

// Start starts the cost agent
func (ca *CostAgent) Start(ctx context.Context) error {
	if ca.MessageBus == nil {
		return fmt.Errorf("message bus not configured")
	}

	return ca.MessageBus.Subscribe(ctx, ca.ID, func(msg *Message) {
		var result interface{}
		var err error

		switch msg.Method {
		case "analyze_costs":
			customerID := msg.Params["customer_id"].(string)
			days := int(msg.Params["days"].(float64))
			result, err = ca.AnalyzeCosts(ctx, customerID, days)

		case "get_usage_patterns":
			// Return mock usage patterns
			result = map[string]interface{}{
				"avg_tokens_per_request": 1500,
				"requests_per_day":       1000,
				"peak_hours":             []int{9, 10, 11, 14, 15, 16},
				"potential_savings":      125.50,
			}

		default:
			err = fmt.Errorf("unknown method: %s", msg.Method)
		}

		ca.MessageBus.SendResponse(ctx, msg, result, err)
	})
}

// AnalyzeCosts analyzes cost patterns
func (ca *CostAgent) AnalyzeCosts(ctx context.Context, customerID string, days int) (map[string]interface{}, error) {
	resp, err := ca.UseTool(ctx, "analytics", "query_usage", map[string]interface{}{
		"customer_id": customerID,
		"days":        days,
	})

	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("cost analysis failed: %s", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type")
	}

	return result, nil
}

