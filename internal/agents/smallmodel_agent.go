package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SmallModelAgent handles small model training operations
type SmallModelAgent struct {
	redisClient       *redis.Client
	mcpServerURL      string
	catalogServiceURL string
}

// SmallModelRequest represents a training request for small models
type SmallModelRequest struct {
	RequestID   string                 `json:"request_id"`
	Task        string                 `json:"task"`
	DatasetPath string                 `json:"dataset_path"`
	DatasetSize string                 `json:"dataset_size"`
	Budget      string                 `json:"budget,omitempty"`
	AutoDeploy  bool                   `json:"auto_deploy,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// NewSmallModelAgent creates a new small model training agent
func NewSmallModelAgent(redisAddr, mcpServerURL, catalogServiceURL string) (*SmallModelAgent, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &SmallModelAgent{
		redisClient:       client,
		mcpServerURL:      mcpServerURL,
		catalogServiceURL: catalogServiceURL,
	}, nil
}

// Start starts the small model agent
func (sma *SmallModelAgent) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting Small Model Agent", "channel", "agent.smallmodel")

	pubsub := sma.redisClient.Subscribe(ctx, "agent.smallmodel")
	defer pubsub.Close()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Small Model Agent shutting down")
			return nil
		case msg := <-pubsub.Channel():
			if msg == nil {
				continue
			}

			logger.Info("Received small model request", "payload", msg.Payload)

			var req SmallModelRequest
			if err := json.Unmarshal([]byte(msg.Payload), &req); err != nil {
				logger.Error(err, "Failed to parse request")
				continue
			}

			// Handle request asynchronously
			go sma.handleTraining(context.Background(), &req)
		}
	}
}

// handleTraining processes a small model training request
func (sma *SmallModelAgent) handleTraining(ctx context.Context, req *SmallModelRequest) {
	logger := log.Log.WithName("smallmodel-agent")
	logger.Info("Handling small model training", "task", req.Task, "dataset", req.DatasetPath)

	// Step 1: Get model recommendation from catalog
	recommendation, err := sma.getRecommendation(ctx, req)
	if err != nil {
		logger.Error(err, "Failed to get recommendation")
		sma.sendResponse(ctx, req.RequestID, "failed", nil, err.Error())
		return
	}

	logger.Info("Model recommended", "model", recommendation["recommended_model"])

	// Step 2: Prepare training configuration
	trainingParams := map[string]interface{}{
		"dataset_path": req.DatasetPath,
		"base_model":   recommendation["recommended_model"],
		"lora_config": map[string]interface{}{
			"rank":  16,
			"alpha": 32,
		},
		"task": req.Task,
	}

	// Step 3: Start training via MCP
	result, err := callMCPTool(ctx, sma.mcpServerURL, "start_training", trainingParams)
	if err != nil {
		logger.Error(err, "Training failed")
		sma.sendResponse(ctx, req.RequestID, "failed", nil, err.Error())
		return
	}

	// Step 4: Return results
	response := map[string]interface{}{
		"status":         "success",
		"recommendation": recommendation,
		"training":       result,
	}

	logger.Info("Small model training initiated", "result", result)
	sma.sendResponse(ctx, req.RequestID, "success", response, "")
}

// getRecommendation gets model recommendation from catalog service
func (sma *SmallModelAgent) getRecommendation(ctx context.Context, req *SmallModelRequest) (map[string]interface{}, error) {
	// In production, call catalog service HTTP API
	// For now, return a default recommendation

	return map[string]interface{}{
		"recommended_model":     "phi-2",
		"reasoning":             "Best for " + req.Task,
		"training_cost":         "$15",
		"inference_cost_per_1M": "$1",
	}, nil
}

// sendResponse sends a response back via Redis
func (sma *SmallModelAgent) sendResponse(ctx context.Context, requestID, status string, result interface{}, errorMsg string) {
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

	channel := fmt.Sprintf("agent.smallmodel.response.%s", requestID)
	if err := sma.redisClient.Publish(ctx, channel, data).Err(); err != nil {
		log.Log.Error(err, "Failed to publish response")
	}
}

