package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TrainingAgent handles model training operations
type TrainingAgent struct {
	redisClient  *redis.Client
	mcpServerURL string
}

// TrainingRequest represents a training request
type TrainingRequest struct {
	RequestID   string                 `json:"request_id"`
	DatasetPath string                 `json:"dataset_path"`
	BaseModel   string                 `json:"base_model"`
	LoRAConfig  map[string]interface{} `json:"lora_config"`
}

// NewTrainingAgent creates a new training agent
func NewTrainingAgent(redisAddr, mcpServerURL string) (*TrainingAgent, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &TrainingAgent{
		redisClient:  client,
		mcpServerURL: mcpServerURL,
	}, nil
}

// Start starts the training agent
func (t *TrainingAgent) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting Training Agent", "channel", "agent.training")

	pubsub := t.redisClient.Subscribe(ctx, "agent.training")
	defer pubsub.Close()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Training Agent shutting down")
			return nil
		case msg := <-pubsub.Channel():
			if msg == nil {
				continue
			}

			logger.Info("Received training request", "payload", msg.Payload)

			var req TrainingRequest
			if err := json.Unmarshal([]byte(msg.Payload), &req); err != nil {
				logger.Error(err, "Failed to parse training request")
				continue
			}

			// Handle training asynchronously
			go t.handleTraining(context.Background(), &req)
		}
	}
}

// handleTraining handles a training request
func (t *TrainingAgent) handleTraining(ctx context.Context, req *TrainingRequest) {
	logger := log.Log.WithName("training-agent")
	logger.Info("Handling training", "dataset", req.DatasetPath, "base_model", req.BaseModel)

	// Call MCP start_training tool
	params := map[string]interface{}{
		"dataset_path": req.DatasetPath,
		"base_model":   req.BaseModel,
		"lora_config":  req.LoRAConfig,
	}

	result, err := callMCPTool(ctx, t.mcpServerURL, "start_training", params)
	if err != nil {
		logger.Error(err, "Training failed")
		t.sendResponse(ctx, req.RequestID, "failed", nil, err.Error())
		return
	}

	// In real implementation, monitor training job until completion
	// For now, simulate monitoring
	logger.Info("Training started, monitoring progress...")
	time.Sleep(10 * time.Second) // Simulate training time

	logger.Info("Training completed", "result", result)
	t.sendResponse(ctx, req.RequestID, "success", result, "")
}

// sendResponse sends a response back via Redis
func (t *TrainingAgent) sendResponse(ctx context.Context, requestID, status string, result interface{}, errorMsg string) {
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

	channel := fmt.Sprintf("agent.training.response.%s", requestID)
	if err := t.redisClient.Publish(ctx, channel, data).Err(); err != nil {
		log.Log.Error(err, "Failed to publish response")
	}
}

