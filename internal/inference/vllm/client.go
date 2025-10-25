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

package vllm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// VLLMClient handles communication with vLLM inference endpoints
type VLLMClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// CompletionRequest represents a request to vLLM
type CompletionRequest struct {
	Model       string                 `json:"model"`
	Prompt      string                 `json:"prompt,omitempty"`
	Messages    []Message              `json:"messages,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	TopP        float64                `json:"top_p,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	LoraAdapter string                 `json:"lora_adapter,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionResponse represents a response from vLLM
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
	Message      Message `json:"message,omitempty"`
	Text         string  `json:"text,omitempty"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModelInfo represents information about a loaded model
type ModelInfo struct {
	ID             string   `json:"id"`
	Object         string   `json:"object"`
	Created        int64    `json:"created"`
	OwnedBy        string   `json:"owned_by"`
	LoraAdapters   []string `json:"lora_adapters,omitempty"`
	MaxModelLength int      `json:"max_model_len,omitempty"`
}

// NewVLLMClient creates a new vLLM client
func NewVLLMClient(baseURL string) *VLLMClient {
	return &VLLMClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// CreateCompletion sends a completion request to vLLM
func (c *VLLMClient) CreateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/v1/completions", c.BaseURL)
	if len(req.Messages) > 0 {
		endpoint = fmt.Sprintf("%s/v1/chat/completions", c.BaseURL)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vLLM returned status %d: %s", resp.StatusCode, string(body))
	}

	var completion CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &completion, nil
}

// ListModels retrieves available models from vLLM
func (c *VLLMClient) ListModels(ctx context.Context) ([]ModelInfo, error) {
	endpoint := fmt.Sprintf("%s/v1/models", c.BaseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vLLM returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []ModelInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data, nil
}

// HealthCheck verifies vLLM is healthy
func (c *VLLMClient) HealthCheck(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/health", c.BaseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("vLLM health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// LoadLoraAdapter loads a LoRA adapter into vLLM
func (c *VLLMClient) LoadLoraAdapter(ctx context.Context, adapterName, adapterPath string) error {
	endpoint := fmt.Sprintf("%s/v1/lora/load", c.BaseURL)

	payload := map[string]string{
		"name": adapterName,
		"path": adapterPath,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to load LoRA adapter: status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UnloadLoraAdapter unloads a LoRA adapter from vLLM
func (c *VLLMClient) UnloadLoraAdapter(ctx context.Context, adapterName string) error {
	endpoint := fmt.Sprintf("%s/v1/lora/unload", c.BaseURL)

	payload := map[string]string{
		"name": adapterName,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to unload LoRA adapter: status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}



