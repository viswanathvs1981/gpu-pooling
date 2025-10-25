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

package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/NexusGPU/tensor-fusion/internal/inference/lora"
	"github.com/NexusGPU/tensor-fusion/internal/inference/vllm"
	"github.com/gorilla/mux"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OpenAIAPI implements OpenAI-compatible API endpoints
type OpenAIAPI struct {
	Router          *mux.Router
	Client          client.Client
	VLLMClient      *vllm.VLLMClient
	AdapterRegistry *lora.Registry
}

// ChatCompletionRequest represents OpenAI chat completion request
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	User        string    `json:"user,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse represents OpenAI chat completion response
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int      `json:"index"`
	Message      Message  `json:"message"`
	FinishReason string   `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModelInfo represents model information
type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// NewOpenAIAPI creates a new OpenAI API handler
func NewOpenAIAPI(client client.Client, vllmClient *vllm.VLLMClient, registry *lora.Registry) *OpenAIAPI {
	api := &OpenAIAPI{
		Router:          mux.NewRouter(),
		Client:          client,
		VLLMClient:      vllmClient,
		AdapterRegistry: registry,
	}

	api.setupRoutes()
	return api
}

// setupRoutes configures API routes
func (api *OpenAIAPI) setupRoutes() {
	api.Router.HandleFunc("/v1/chat/completions", api.HandleChatCompletion).Methods("POST")
	api.Router.HandleFunc("/v1/completions", api.HandleCompletion).Methods("POST")
	api.Router.HandleFunc("/v1/models", api.HandleListModels).Methods("GET")
	api.Router.HandleFunc("/v1/models/{model}", api.HandleGetModel).Methods("GET")
	api.Router.HandleFunc("/health", api.HandleHealth).Methods("GET")
}

// HandleChatCompletion handles /v1/chat/completions requests
func (api *OpenAIAPI) HandleChatCompletion(w http.ResponseWriter, r *http.Request) {
	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON in request body")
		return
	}

	// Validate request
	if len(req.Messages) == 0 {
		api.sendError(w, http.StatusBadRequest, "invalid_request_error", "Messages array cannot be empty")
		return
	}

	// Convert to vLLM request
	vllmReq := &vllm.CompletionRequest{
		Model:       req.Model,
		Messages:    convertMessages(req.Messages),
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
	}

	// Check if model is a LoRA adapter
	adapter, err := api.AdapterRegistry.GetAdapter(r.Context(), req.Model)
	if err == nil {
		vllmReq.LoraAdapter = adapter.Name
	}

	// Call vLLM
	resp, err := api.VLLMClient.CreateCompletion(r.Context(), vllmReq)
	if err != nil {
		klog.Errorf("vLLM request failed: %v", err)
		api.sendError(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("Inference failed: %v", err))
		return
	}

	// Convert response
	chatResp := &ChatCompletionResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: resp.Created,
		Model:   resp.Model,
		Choices: convertChoices(resp.Choices),
		Usage:   Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	api.sendJSON(w, http.StatusOK, chatResp)
}

// HandleCompletion handles /v1/completions requests
func (api *OpenAIAPI) HandleCompletion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Model       string  `json:"model"`
		Prompt      string  `json:"prompt"`
		MaxTokens   int     `json:"max_tokens,omitempty"`
		Temperature float64 `json:"temperature,omitempty"`
		TopP        float64 `json:"top_p,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON in request body")
		return
	}

	vllmReq := &vllm.CompletionRequest{
		Model:       req.Model,
		Prompt:      req.Prompt,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	resp, err := api.VLLMClient.CreateCompletion(r.Context(), vllmReq)
	if err != nil {
		api.sendError(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("Inference failed: %v", err))
		return
	}

	api.sendJSON(w, http.StatusOK, resp)
}

// HandleListModels handles /v1/models requests
func (api *OpenAIAPI) HandleListModels(w http.ResponseWriter, r *http.Request) {
	models, err := api.VLLMClient.ListModels(r.Context())
	if err != nil {
		api.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to list models")
		return
	}

	// Add LoRA adapters
	adapters, err := api.AdapterRegistry.ListAdapters(r.Context(), "")
	if err == nil {
		for _, adapter := range adapters {
			models = append(models, vllm.ModelInfo{
				ID:       adapter.Name,
				Object:   "model",
				Created:  adapter.Created.Unix(),
				OwnedBy:  "user",
			})
		}
	}

	response := map[string]interface{}{
		"object": "list",
		"data":   models,
	}

	api.sendJSON(w, http.StatusOK, response)
}

// HandleGetModel handles /v1/models/{model} requests
func (api *OpenAIAPI) HandleGetModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelName := vars["model"]

	// Check if it's a LoRA adapter
	adapter, err := api.AdapterRegistry.GetAdapter(r.Context(), modelName)
	if err == nil {
		modelInfo := ModelInfo{
			ID:      adapter.Name,
			Object:  "model",
			Created: adapter.Created.Unix(),
			OwnedBy: "user",
		}
		api.sendJSON(w, http.StatusOK, modelInfo)
		return
	}

	// Try vLLM models
	models, err := api.VLLMClient.ListModels(r.Context())
	if err != nil {
		api.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to get model")
		return
	}

	for _, model := range models {
		if model.ID == modelName {
			api.sendJSON(w, http.StatusOK, model)
			return
		}
	}

	api.sendError(w, http.StatusNotFound, "model_not_found", "The model does not exist")
}

// HandleHealth handles /health requests
func (api *OpenAIAPI) HandleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Unix(),
	}

	// Check vLLM health
	if err := api.VLLMClient.HealthCheck(context.Background()); err != nil {
		health["status"] = "degraded"
		health["vllm"] = "unhealthy"
	} else {
		health["vllm"] = "healthy"
	}

	api.sendJSON(w, http.StatusOK, health)
}

// sendJSON sends a JSON response
func (api *OpenAIAPI) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteStatus(status)
	json.NewEncoder(w).Encode(data)
}

// sendError sends an error response
func (api *OpenAIAPI) sendError(w http.ResponseWriter, status int, errorType, message string) {
	errResp := ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    errorType,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errResp)
}

// Helper functions
func convertMessages(messages []Message) []vllm.Message {
	result := make([]vllm.Message, len(messages))
	for i, msg := range messages {
		result[i] = vllm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return result
}

func convertChoices(choices []vllm.Choice) []Choice {
	result := make([]Choice, len(choices))
	for i, choice := range choices {
		result[i] = Choice{
			Index: choice.Index,
			Message: Message{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
			},
			FinishReason: choice.FinishReason,
		}
	}
	return result
}



