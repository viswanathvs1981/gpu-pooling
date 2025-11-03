package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CatalogService provides model catalog and recommendation APIs
type CatalogService struct {
	httpServer *http.Server
	k8sClient  client.Client
	models     map[string]*ModelDefinition
	port       int
}

// ModelDefinition represents a small model in the catalog
type ModelDefinition struct {
	Name            string                 `json:"name"`
	Parameters      string                 `json:"parameters"`
	ContextLength   int                    `json:"context_length"`
	Architecture    string                 `json:"architecture"`
	GPURequirement  float64                `json:"gpu_requirement"`
	TrainingTime    string                 `json:"training_time"`
	BestFor         []string               `json:"best_for"`
	BaseModelURL    string                 `json:"base_model_url"`
	TrainingConfig  map[string]interface{} `json:"training_config"`
}

// RecommendationRequest represents a model recommendation request
type RecommendationRequest struct {
	Task                string `json:"task"`
	DatasetSize         string `json:"dataset_size"`
	LatencyRequirement  string `json:"latency_requirement,omitempty"`
	Budget              string `json:"budget,omitempty"`
}

// RecommendationResponse contains the recommended model
type RecommendationResponse struct {
	RecommendedModel     string  `json:"recommended_model"`
	Reasoning            string  `json:"reasoning"`
	TrainingCost         string  `json:"training_cost"`
	InferenceCostPerMillion string  `json:"inference_cost_per_1M"`
}

// NewCatalogService creates a new model catalog service
func NewCatalogService(port int, k8sClient client.Client) *CatalogService {
	service := &CatalogService{
		k8sClient: k8sClient,
		models:    make(map[string]*ModelDefinition),
		port:      port,
	}

	// Initialize catalog with pre-defined models
	service.initializeCatalog()

	return service
}

// Start starts the catalog service
func (cs *CatalogService) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/models", cs.handleListModels)
	mux.HandleFunc("/api/v1/recommend", cs.handleRecommend)
	mux.HandleFunc("/health", cs.handleHealth)

	cs.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", cs.port),
		Handler: mux,
	}

	logger.Info("Starting Model Catalog Service", "port", cs.port)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := cs.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error(err, "Error shutting down catalog service")
		}
	}()

	if err := cs.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("catalog service error: %w", err)
	}

	return nil
}

// handleListModels returns all available models
func (cs *CatalogService) handleListModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	models := make([]*ModelDefinition, 0, len(cs.models))
	for _, model := range cs.models {
		models = append(models, model)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"models": models,
		"count":  len(models),
	})
}

// handleRecommend provides model recommendations
func (cs *CatalogService) handleRecommend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RecommendationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Simple recommendation logic
	recommendedModel := cs.recommendModel(req)

	resp := RecommendationResponse{
		RecommendedModel:     recommendedModel.Name,
		Reasoning:            cs.generateReasoning(recommendedModel, req),
		TrainingCost:         cs.estimateTrainingCost(recommendedModel),
		InferenceCostPerMillion: cs.estimateInferenceCost(recommendedModel),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleHealth returns service health
func (cs *CatalogService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

// initializeCatalog populates the catalog with pre-defined models
func (cs *CatalogService) initializeCatalog() {
	cs.models["tinyllama-1.1b"] = &ModelDefinition{
		Name:           "TinyLlama-1.1B",
		Parameters:     "1.1B",
		ContextLength:  2048,
		Architecture:   "llama",
		GPURequirement: 0.25,
		TrainingTime:   "2-4 hours",
		BestFor:        []string{"embeddings", "classification", "fast-inference"},
		BaseModelURL:   "TinyLlama/TinyLlama-1.1B-Chat-v1.0",
		TrainingConfig: map[string]interface{}{
			"batch_size":    8,
			"learning_rate": 3e-4,
			"lora_rank":     8,
			"lora_alpha":    16,
		},
	}

	cs.models["phi-2"] = &ModelDefinition{
		Name:           "Phi-2",
		Parameters:     "2.7B",
		ContextLength:  2048,
		Architecture:   "phi",
		GPURequirement: 0.5,
		TrainingTime:   "4-6 hours",
		BestFor:        []string{"reasoning", "qa", "classification"},
		BaseModelURL:   "microsoft/phi-2",
		TrainingConfig: map[string]interface{}{
			"batch_size":    8,
			"learning_rate": 2e-4,
			"lora_rank":     16,
			"lora_alpha":    32,
		},
	}

	cs.models["stablelm-3b"] = &ModelDefinition{
		Name:           "StableLM-3B",
		Parameters:     "3B",
		ContextLength:  4096,
		Architecture:   "stablelm",
		GPURequirement: 0.5,
		TrainingTime:   "6-8 hours",
		BestFor:        []string{"general-purpose", "chat", "generation"},
		BaseModelURL:   "stabilityai/stablelm-3b-4e1t",
		TrainingConfig: map[string]interface{}{
			"batch_size":    6,
			"learning_rate": 2e-4,
			"lora_rank":     16,
			"lora_alpha":    32,
		},
	}

	cs.models["mistral-7b"] = &ModelDefinition{
		Name:           "Mistral-7B",
		Parameters:     "7B",
		ContextLength:  8192,
		Architecture:   "mistral",
		GPURequirement: 1.0,
		TrainingTime:   "8-12 hours",
		BestFor:        []string{"high-quality-generation", "instruction-following", "coding"},
		BaseModelURL:   "mistralai/Mistral-7B-v0.1",
		TrainingConfig: map[string]interface{}{
			"batch_size":    4,
			"learning_rate": 1e-4,
			"lora_rank":     32,
			"lora_alpha":    64,
		},
	}

	cs.models["gemma-2b"] = &ModelDefinition{
		Name:           "Gemma-2B",
		Parameters:     "2B",
		ContextLength:  8192,
		Architecture:   "gemma",
		GPURequirement: 0.5,
		TrainingTime:   "4-6 hours",
		BestFor:        []string{"safety", "instruction-following", "general-purpose"},
		BaseModelURL:   "google/gemma-2b",
		TrainingConfig: map[string]interface{}{
			"batch_size":    8,
			"learning_rate": 2e-4,
			"lora_rank":     16,
			"lora_alpha":    32,
		},
	}
}

// recommendModel selects the best model based on requirements
func (cs *CatalogService) recommendModel(req RecommendationRequest) *ModelDefinition {
	// Simple heuristic-based recommendation
	// In production, this could use ML-based selection

	// Budget-based selection
	if req.Budget == "low" || req.Budget == "very low" {
		return cs.models["tinyllama-1.1b"]
	}

	// Latency-based selection
	if req.LatencyRequirement == "< 50ms" {
		return cs.models["tinyllama-1.1b"]
	} else if req.LatencyRequirement == "< 100ms" {
		return cs.models["phi-2"]
	}

	// Task-based selection
	switch req.Task {
	case "classification", "email-classification", "sentiment-analysis":
		return cs.models["phi-2"]
	case "embeddings":
		return cs.models["tinyllama-1.1b"]
	case "generation", "summarization":
		return cs.models["mistral-7b"]
	case "qa", "question-answering":
		return cs.models["phi-2"]
	case "reasoning":
		return cs.models["phi-2"]
	default:
		return cs.models["phi-2"] // Default recommendation
	}
}

// generateReasoning explains why a model was recommended
func (cs *CatalogService) generateReasoning(model *ModelDefinition, req RecommendationRequest) string {
	return fmt.Sprintf("Best accuracy/cost ratio for %s tasks with %s dataset", req.Task, req.DatasetSize)
}

// estimateTrainingCost calculates estimated training cost
func (cs *CatalogService) estimateTrainingCost(model *ModelDefinition) string {
	// Simple cost estimation based on GPU hours
	costPerGPUHour := 2.40 // $2.40/hour per vGPU
	
	var hours float64
	switch model.Parameters {
	case "1.1B":
		hours = 3
	case "2B", "2.7B":
		hours = 5
	case "3B":
		hours = 7
	case "7B":
		hours = 10
	default:
		hours = 5
	}

	cost := model.GPURequirement * costPerGPUHour * hours
	return fmt.Sprintf("$%.0f", cost)
}

// estimateInferenceCost calculates cost per 1M tokens
func (cs *CatalogService) estimateInferenceCost(model *ModelDefinition) string {
	// Simplified cost model
	var costPer1M float64
	switch model.Parameters {
	case "1.1B":
		costPer1M = 0.5
	case "2B", "2.7B":
		costPer1M = 1.0
	case "3B":
		costPer1M = 1.5
	case "7B":
		costPer1M = 2.5
	default:
		costPer1M = 1.0
	}

	return fmt.Sprintf("$%.1f", costPer1M)
}

