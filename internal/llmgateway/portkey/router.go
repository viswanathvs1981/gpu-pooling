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

package portkey

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RoutingController manages Portkey routing configurations
type RoutingController struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// RouteConfig represents a routing configuration
type RouteConfig struct {
	Name     string           `json:"name"`
	Strategy string           `json:"strategy"` // "loadbalance", "fallback", "cost-based"
	Targets  []TargetConfig   `json:"targets"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TargetConfig represents a target endpoint
type TargetConfig struct {
	Provider    string  `json:"provider"` // "azure", "openai", "self-hosted"
	VirtualKey  string  `json:"virtual_key,omitempty"`
	Weight      int     `json:"weight,omitempty"`
	RetryConfig *RetryConfig `json:"retry,omitempty"`
	CostPerToken float64 `json:"cost_per_token,omitempty"`
}

// RetryConfig contains retry settings
type RetryConfig struct {
	Attempts int `json:"attempts"`
	OnStatusCodes []int `json:"on_status_codes,omitempty"`
}

// NewRoutingController creates a new Portkey routing controller
func NewRoutingController(baseURL, apiKey string) *RoutingController {
	return &RoutingController{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateRoute creates a new routing configuration
func (rc *RoutingController) CreateRoute(ctx context.Context, config *RouteConfig) error {
	jsonData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", 
		fmt.Sprintf("%s/v1/routes", rc.BaseURL), 
		bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-portkey-api-key", rc.APIKey)

	resp, err := rc.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("portkey returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateRoute updates an existing routing configuration
func (rc *RoutingController) UpdateRoute(ctx context.Context, name string, config *RouteConfig) error {
	jsonData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", 
		fmt.Sprintf("%s/v1/routes/%s", rc.BaseURL, name), 
		bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-portkey-api-key", rc.APIKey)

	resp, err := rc.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("portkey returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteRoute deletes a routing configuration
func (rc *RoutingController) DeleteRoute(ctx context.Context, name string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", 
		fmt.Sprintf("%s/v1/routes/%s", rc.BaseURL, name), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-portkey-api-key", rc.APIKey)

	resp, err := rc.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("portkey returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetMetrics retrieves routing metrics
func (rc *RoutingController) GetMetrics(ctx context.Context, routeName string, startTime, endTime time.Time) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", 
		fmt.Sprintf("%s/v1/analytics/routes/%s", rc.BaseURL, routeName), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("start_time", startTime.Format(time.RFC3339))
	q.Add("end_time", endTime.Format(time.RFC3339))
	req.URL.RawQuery = q.Encode()

	req.Header.Set("x-portkey-api-key", rc.APIKey)

	resp, err := rc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("portkey returned status %d", resp.StatusCode)
	}

	var metrics map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return metrics, nil
}



