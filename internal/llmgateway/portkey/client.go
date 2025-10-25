package portkey

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"k8s.io/klog/v2"
)

// Client handles communication with Portkey AI Gateway
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	logger     klog.Logger
}

// NewClient creates a new Portkey client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: baseURL,
		apiKey:  apiKey,
		logger:  klog.NewKlogr().WithName("portkey-client"),
	}
}

// Config represents a Portkey routing configuration
type Config struct {
	ID       string    `json:"id,omitempty"`
	Name     string    `json:"name"`
	Strategy string    `json:"strategy"` // loadbalance, fallback, etc.
	Targets  []Target  `json:"targets"`
	Cache    *CacheConfig `json:"cache,omitempty"`
	Retry    *RetryConfig `json:"retry,omitempty"`
}

// Target represents a target LLM endpoint
type Target struct {
	Name         string            `json:"name"`
	Provider     string            `json:"provider"`
	Model        string            `json:"model"`
	VirtualKey   string            `json:"virtual_key,omitempty"`
	Weight       int               `json:"weight,omitempty"`
	Override     map[string]interface{} `json:"override_params,omitempty"`
}

// CacheConfig represents caching configuration
type CacheConfig struct {
	Mode      string `json:"mode"` // simple, semantic
	MaxAge    int    `json:"max_age_seconds"`
}

// RetryConfig represents retry configuration
type RetryConfig struct {
	Attempts int      `json:"attempts"`
	OnStatusCodes []int `json:"on_status_codes,omitempty"`
}

// CreateConfig creates a new routing configuration in Portkey
func (c *Client) CreateConfig(ctx context.Context, config *Config) (*Config, error) {
	c.logger.Info("Creating Portkey config", "name", config.Name)

	body, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/configs", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-portkey-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create config: %s", string(body))
	}

	var result Config
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	c.logger.Info("Created Portkey config", "id", result.ID)
	return &result, nil
}

// UpdateConfig updates an existing configuration
func (c *Client) UpdateConfig(ctx context.Context, configID string, config *Config) error {
	c.logger.Info("Updating Portkey config", "id", configID)

	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", 
		fmt.Sprintf("%s/v1/configs/%s", c.baseURL, configID), 
		bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-portkey-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update config: %s", string(body))
	}

	return nil
}

// GetConfig retrieves a configuration
func (c *Client) GetConfig(ctx context.Context, configID string) (*Config, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", 
		fmt.Sprintf("%s/v1/configs/%s", c.baseURL, configID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-portkey-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get config: status %d", resp.StatusCode)
	}

	var config Config
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// DeleteConfig deletes a configuration
func (c *Client) DeleteConfig(ctx context.Context, configID string) error {
	c.logger.Info("Deleting Portkey config", "id", configID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", 
		fmt.Sprintf("%s/v1/configs/%s", c.baseURL, configID), nil)
	if err != nil {
		return err
	}

	req.Header.Set("x-portkey-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete config: %s", string(body))
	}

	return nil
}

// GetUsageStats retrieves usage statistics for an API key
func (c *Client) GetUsageStats(ctx context.Context, apiKey string, timeRange TimeRange) (*UsageStats, error) {
	url := fmt.Sprintf("%s/v1/usage?api_key=%s&start=%d&end=%d", 
		c.baseURL, apiKey, timeRange.Start.Unix(), timeRange.End.Unix())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-portkey-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get usage stats: status %d", resp.StatusCode)
	}

	var stats UsageStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetRequestLogs retrieves request logs
func (c *Client) GetRequestLogs(ctx context.Context, filters LogFilters) ([]RequestLog, error) {
	url := c.buildLogsURL(filters)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-portkey-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get logs: status %d", resp.StatusCode)
	}

	var result struct {
		Logs []RequestLog `json:"logs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Logs, nil
}

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// UsageStats represents usage statistics
type UsageStats struct {
	TotalRequests     int64   `json:"total_requests"`
	SuccessfulRequests int64  `json:"successful_requests"`
	FailedRequests    int64   `json:"failed_requests"`
	TotalCost         float64 `json:"total_cost"`
	TotalTokens       int64   `json:"total_tokens"`
	PromptTokens      int64   `json:"prompt_tokens"`
	CompletionTokens  int64   `json:"completion_tokens"`
	AverageLatencyMs  float64 `json:"average_latency_ms"`
	P95LatencyMs      float64 `json:"p95_latency_ms"`
	P99LatencyMs      float64 `json:"p99_latency_ms"`
	CacheHits         int64   `json:"cache_hits"`
	CacheMisses       int64   `json:"cache_misses"`
}

// LogFilters represents filters for log queries
type LogFilters struct {
	StartTime   time.Time
	EndTime     time.Time
	APIKey      string
	Model       string
	Status      string
	Limit       int
}

// RequestLog represents a single request log entry
type RequestLog struct {
	ID               string    `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	Model            string    `json:"model"`
	Provider         string    `json:"provider"`
	Status           int       `json:"status"`
	LatencyMs        int       `json:"latency_ms"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	Cost             float64   `json:"cost"`
	CacheHit         bool      `json:"cache_hit"`
	ErrorMessage     string    `json:"error_message,omitempty"`
}

func (c *Client) buildLogsURL(filters LogFilters) string {
	url := fmt.Sprintf("%s/v1/logs?start=%d&end=%d", 
		c.baseURL, filters.StartTime.Unix(), filters.EndTime.Unix())

	if filters.APIKey != "" {
		url += "&api_key=" + filters.APIKey
	}
	if filters.Model != "" {
		url += "&model=" + filters.Model
	}
	if filters.Status != "" {
		url += "&status=" + filters.Status
	}
	if filters.Limit > 0 {
		url += fmt.Sprintf("&limit=%d", filters.Limit)
	}

	return url
}

// CreateVirtualKey creates a virtual API key in Portkey
func (c *Client) CreateVirtualKey(ctx context.Context, name string, config map[string]interface{}) (string, error) {
	c.logger.Info("Creating Portkey virtual key", "name", name)

	payload := map[string]interface{}{
		"name":   name,
		"config": config,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/virtual-keys", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-portkey-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create virtual key: %s", string(body))
	}

	var result struct {
		VirtualKey string `json:"virtual_key"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.VirtualKey, nil
}

// HealthCheck checks if Portkey Gateway is healthy
func (c *Client) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	return nil
}


