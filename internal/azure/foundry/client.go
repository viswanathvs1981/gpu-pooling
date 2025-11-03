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

package foundry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	tfv1 "github.com/NexusGPU/tensor-fusion/api/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Client handles Azure AI Foundry API interactions
type Client struct {
	Endpoint   string
	APIKey     string
	APIVersion string
	HTTPClient *http.Client
}

// CompletionRequest represents Azure OpenAI request
type CompletionRequest struct {
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionResponse represents Azure OpenAI response
type CompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModelDeployment represents an Azure model deployment
type ModelDeployment struct {
	Name              string            `json:"name"`
	Model             string            `json:"model"`
	Status            string            `json:"status"`
	Scale             map[string]int    `json:"scale,omitempty"`
	Properties        map[string]string `json:"properties,omitempty"`
	
	// Additional fields for model selection
	EstimatedTFlops   float64           `json:"estimatedTFlops,omitempty"`
	EstimatedVRAM     float64           `json:"estimatedVRAM,omitempty"`
	InputCostPer1K    float64           `json:"inputCostPer1K,omitempty"`
	OutputCostPer1K   float64           `json:"outputCostPer1K,omitempty"`
	Region            string            `json:"region,omitempty"`
	ModelFamily       string            `json:"modelFamily,omitempty"`
	AverageLatencyMs  float64           `json:"averageLatencyMs,omitempty"`
}

// Type aliases for compatibility
type FoundryClient = Client
type FoundryModel = ModelDeployment
type ModelMetrics struct {
	RequestsPerSecond float64
	TokensPerSecond   float64
	ErrorRate         float64
	AverageLatency    time.Duration
	AverageLatencyMs  float64
	QueueDepth        int
}

// NewClient creates a new Azure AI Foundry client
func NewClient(endpoint, apiKey, apiVersion string) *Client {
	return &Client{
		Endpoint:   endpoint,
		APIKey:     apiKey,
		APIVersion: apiVersion,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// NewFoundryClient is an alias for NewClient
func NewFoundryClient(endpoint, apiKey, apiVersion string) *FoundryClient {
	return NewClient(endpoint, apiKey, apiVersion)
}

// CreateChatCompletion sends a chat completion request
func (c *Client) CreateChatCompletion(ctx context.Context, deploymentName string, req *CompletionRequest) (*CompletionResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s", 
		c.Endpoint, deploymentName, c.APIVersion)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("azure returned status %d: %s", resp.StatusCode, string(body))
	}

	var completion CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &completion, nil
}

// ListDeployments lists all model deployments
func (c *Client) ListDeployments(ctx context.Context) ([]ModelDeployment, error) {
	url := fmt.Sprintf("%s/openai/deployments?api-version=%s", c.Endpoint, c.APIVersion)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("api-key", c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("azure returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []ModelDeployment `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data, nil
}

// GetDeployment retrieves a specific deployment
func (c *Client) GetDeployment(ctx context.Context, deploymentName string) (*ModelDeployment, error) {
	url := fmt.Sprintf("%s/openai/deployments/%s?api-version=%s", c.Endpoint, deploymentName, c.APIVersion)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("api-key", c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("azure returned status %d", resp.StatusCode)
	}

	var deployment ModelDeployment
	if err := json.NewDecoder(resp.Body).Decode(&deployment); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &deployment, nil
}

// ListAvailableModels lists all available models in the Foundry catalog
func (c *Client) ListAvailableModels(ctx context.Context) ([]FoundryModel, error) {
	// Placeholder: In production, this would query Azure AI Foundry model catalog
	return c.ListDeployments(ctx)
}

// CheckModelAvailability checks if a specific model is available
func (c *Client) CheckModelAvailability(ctx context.Context, modelName string) (bool, error) {
	// Placeholder: In production, this would check model availability in Foundry
	deployment, err := c.GetDeployment(ctx, modelName)
	if err != nil {
		return false, nil
	}
	return deployment.Status == "Succeeded", nil
}

// GetModelMetrics retrieves metrics for a specific model deployment
func (c *Client) GetModelMetrics(ctx context.Context, modelName string) (*ModelMetrics, error) {
	// Placeholder: In production, this would fetch real metrics from Azure Monitor
	return &ModelMetrics{
		RequestsPerSecond: 10.0,
		TokensPerSecond:   500.0,
		ErrorRate:         0.01,
		AverageLatency:    100 * time.Millisecond,
		AverageLatencyMs:  100.0,
		QueueDepth:        5,
	}, nil
}

// ConvertToAzureGPUModel converts a Foundry model to AzureGPUModel
func ConvertToAzureGPUModel(model FoundryModel) tfv1.AzureGPUModel {
	return tfv1.AzureGPUModel{
		ModelName: model.Name,
		SKU:       model.Model,
		TFlops:    resource.MustParse("100"),    // Placeholder
		VRAM:      resource.MustParse("24Gi"),   // Placeholder
	}
}
