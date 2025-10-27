package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DeploymentAgent handles model deployment operations
type DeploymentAgent struct {
	redisClient  *redis.Client
	mcpServerURL string
}

// DeploymentRequest represents a deployment request
type DeploymentRequest struct {
	RequestID  string                 `json:"request_id"`
	ModelID    string                 `json:"model_id"`
	CustomerID string                 `json:"customer_id"`
	Config     map[string]interface{} `json:"config"`
}

// NewDeploymentAgent creates a new deployment agent
func NewDeploymentAgent(redisAddr, mcpServerURL string) (*DeploymentAgent, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &DeploymentAgent{
		redisClient:  client,
		mcpServerURL: mcpServerURL,
	}, nil
}

// Start starts the deployment agent
func (d *DeploymentAgent) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting Deployment Agent", "channel", "agent.deployment")

	pubsub := d.redisClient.Subscribe(ctx, "agent.deployment")
	defer pubsub.Close()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Deployment Agent shutting down")
			return nil
		case msg := <-pubsub.Channel():
			if msg == nil {
				continue
			}

			logger.Info("Received deployment request", "payload", msg.Payload)

			var req DeploymentRequest
			if err := json.Unmarshal([]byte(msg.Payload), &req); err != nil {
				logger.Error(err, "Failed to parse deployment request")
				continue
			}

			// Handle deployment asynchronously
			go d.handleDeployment(context.Background(), &req)
		}
	}
}

// handleDeployment handles a deployment request
func (d *DeploymentAgent) handleDeployment(ctx context.Context, req *DeploymentRequest) {
	logger := log.Log.WithName("deployment-agent")
	logger.Info("Handling deployment", "model_id", req.ModelID, "customer_id", req.CustomerID)

	// Call MCP deploy_model tool
	params := map[string]interface{}{
		"model_id":    req.ModelID,
		"customer_id": req.CustomerID,
		"config":      req.Config,
	}

	result, err := callMCPTool(ctx, d.mcpServerURL, "deploy_model", params)
	if err != nil {
		logger.Error(err, "Deployment failed")
		d.sendResponse(ctx, req.RequestID, "failed", nil, err.Error())
		return
	}

	logger.Info("Deployment successful", "result", result)
	d.sendResponse(ctx, req.RequestID, "success", result, "")
}

// sendResponse sends a response back via Redis
func (d *DeploymentAgent) sendResponse(ctx context.Context, requestID, status string, result interface{}, errorMsg string) {
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

	channel := fmt.Sprintf("agent.deployment.response.%s", requestID)
	if err := d.redisClient.Publish(ctx, channel, data).Err(); err != nil {
		log.Log.Error(err, "Failed to publish response")
	}
}

