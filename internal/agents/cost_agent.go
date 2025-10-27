package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CostAgent monitors costs and provides optimization recommendations
type CostAgent struct {
	redisClient  *redis.Client
	mcpServerURL string
}

// CostRequest represents a cost analysis request
type CostRequest struct {
	RequestID  string `json:"request_id"`
	CustomerID string `json:"customer_id"`
	Action     string `json:"action"` // get_costs, forecast, optimize
}

// NewCostAgent creates a new cost agent
func NewCostAgent(redisAddr, mcpServerURL string) (*CostAgent, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &CostAgent{
		redisClient:  client,
		mcpServerURL: mcpServerURL,
	}, nil
}

// Start starts the cost agent
func (c *CostAgent) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting Cost Agent")

	// Start periodic cost monitoring
	go c.periodicMonitoring(ctx)

	// Listen for ad-hoc requests
	pubsub := c.redisClient.Subscribe(ctx, "agent.cost")
	defer pubsub.Close()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Cost Agent shutting down")
			return nil
		case msg := <-pubsub.Channel():
			if msg == nil {
				continue
			}

			logger.Info("Received cost request", "payload", msg.Payload)

			var req CostRequest
			if err := json.Unmarshal([]byte(msg.Payload), &req); err != nil {
				logger.Error(err, "Failed to parse cost request")
				continue
			}

			// Handle request asynchronously
			go c.handleRequest(context.Background(), &req)
		}
	}
}

// periodicMonitoring runs periodic cost monitoring
func (c *CostAgent) periodicMonitoring(ctx context.Context) {
	logger := log.Log.WithName("cost-agent-monitor")
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logger.Info("Running periodic cost analysis...")
			c.analyzeCosts(ctx)
		}
	}
}

// analyzeCosts analyzes costs and detects optimization opportunities
func (c *CostAgent) analyzeCosts(ctx context.Context) {
	logger := log.Log.WithName("cost-agent")

	// In real implementation, query all customers
	// For now, demonstrate with example customer
	customerID := "default"

	// Get current costs
	params := map[string]interface{}{
		"customer_id": customerID,
		"period": map[string]interface{}{
			"start": time.Now().AddDate(0, 0, -7).Format(time.RFC3339),
			"end":   time.Now().Format(time.RFC3339),
		},
	}

	costs, err := callMCPTool(ctx, c.mcpServerURL, "get_costs", params)
	if err != nil {
		logger.Error(err, "Failed to get costs")
		return
	}

	logger.Info("Current costs retrieved", "customer", customerID, "costs", costs)

	// Get optimization recommendations
	optParams := map[string]interface{}{
		"customer_id":         customerID,
		"optimization_target": "cost",
	}

	recommendations, err := callMCPTool(ctx, c.mcpServerURL, "recommend_optimization", optParams)
	if err != nil {
		logger.Error(err, "Failed to get recommendations")
		return
	}

	recMap, ok := recommendations.(map[string]interface{})
	if !ok {
		return
	}

	// Check if there are significant savings
	totalSavings, ok := recMap["total_potential_savings"].(float64)
	if ok && totalSavings > 100 {
		logger.Info("Optimization opportunity detected",
			"customer", customerID,
			"potential_savings", totalSavings,
		)

		// Publish notification
		notification := map[string]interface{}{
			"type":                "cost_optimization",
			"customer_id":         customerID,
			"potential_savings":   totalSavings,
			"recommendations":     recommendations,
			"timestamp":           time.Now().Format(time.RFC3339),
		}

		data, _ := json.Marshal(notification)
		c.redisClient.Publish(ctx, "notifications.cost", data)
	}
}

// handleRequest handles an ad-hoc cost request
func (c *CostAgent) handleRequest(ctx context.Context, req *CostRequest) {
	logger := log.Log.WithName("cost-agent")
	logger.Info("Handling cost request", "action", req.Action, "customer", req.CustomerID)

	var result interface{}
	var err error

	switch req.Action {
	case "get_costs":
		params := map[string]interface{}{
			"customer_id": req.CustomerID,
			"period": map[string]interface{}{
				"start": time.Now().AddDate(0, 0, -30).Format(time.RFC3339),
				"end":   time.Now().Format(time.RFC3339),
			},
		}
		result, err = callMCPTool(ctx, c.mcpServerURL, "get_costs", params)

	case "forecast":
		params := map[string]interface{}{
			"customer_id":   req.CustomerID,
			"forecast_days": 30,
		}
		result, err = callMCPTool(ctx, c.mcpServerURL, "forecast_costs", params)

	case "optimize":
		params := map[string]interface{}{
			"customer_id":         req.CustomerID,
			"optimization_target": "cost",
		}
		result, err = callMCPTool(ctx, c.mcpServerURL, "recommend_optimization", params)

	default:
		err = fmt.Errorf("unknown action: %s", req.Action)
	}

	if err != nil {
		logger.Error(err, "Request failed")
		c.sendResponse(ctx, req.RequestID, "failed", nil, err.Error())
		return
	}

	logger.Info("Request completed", "result", result)
	c.sendResponse(ctx, req.RequestID, "success", result, "")
}

// sendResponse sends a response back via Redis
func (c *CostAgent) sendResponse(ctx context.Context, requestID, status string, result interface{}, errorMsg string) {
	response := map[string]interface{}{
		"request_id": requestID,
		"status":     status,
		"result":     result,
		"error":      errorMsg,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(response)
	if err != nil {
		log.Log.Error(err, "Failed to marshal response")
		return
	}

	channel := fmt.Sprintf("agent.cost.response.%s", requestID)
	if err := c.redisClient.Publish(ctx, channel, data).Err(); err != nil {
		log.Log.Error(err, "Failed to publish response")
	}
}

