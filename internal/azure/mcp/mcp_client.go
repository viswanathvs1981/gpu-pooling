package mcp

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

// MCPClient implements the Model Context Protocol (MCP) for Azure
type MCPClient struct {
	httpClient *http.Client
	auth       AuthConfig
	baseURL    string
	logger     klog.Logger
}

// AuthConfig contains authentication configuration for Azure
type AuthConfig struct {
	TenantID       string
	ClientID       string
	ClientSecret   string
	SubscriptionID string
	UseManagedIdentity bool
}

// MCPRequest represents an MCP protocol request
type MCPRequest struct {
	Method     string
	Path       string
	APIVersion string
	Body       interface{}
	Headers    map[string]string
}

// MCPResponse represents an MCP protocol response
type MCPResponse struct {
	StatusCode int
	Data       map[string]interface{}
	RawBody    []byte
}

// NewMCPClient creates a new MCP client for Azure operations
func NewMCPClient(auth AuthConfig) (*MCPClient, error) {
	return &MCPClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		auth:    auth,
		baseURL: "https://management.azure.com",
		logger:  klog.NewKlogr().WithName("mcp-client"),
	}, nil
}

// Send sends an MCP request to Azure
func (c *MCPClient) Send(ctx context.Context, req *MCPRequest) (*MCPResponse, error) {
	// Build full URL
	url := c.baseURL + req.Path
	if req.APIVersion != "" {
		url += "?api-version=" + req.APIVersion
	}

	c.logger.V(2).Info("Sending MCP request", "method", req.Method, "url", url)

	// Marshal body if present
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Get and set auth token
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)

	// Send request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	var data map[string]interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &data); err != nil {
			c.logger.V(2).Info("Failed to parse response as JSON", "error", err, "body", string(respBody))
			// Not all responses are JSON, continue without parsing
		}
	}

	response := &MCPResponse{
		StatusCode: httpResp.StatusCode,
		Data:       data,
		RawBody:    respBody,
	}

	// Check for errors
	if httpResp.StatusCode >= 400 {
		errorMsg := fmt.Sprintf("request failed with status %d", httpResp.StatusCode)
		if errData, ok := data["error"].(map[string]interface{}); ok {
			if msg, ok := errData["message"].(string); ok {
				errorMsg += ": " + msg
			}
		}
		return response, fmt.Errorf(errorMsg)
	}

	c.logger.V(2).Info("MCP request successful", "statusCode", httpResp.StatusCode)
	return response, nil
}

// getAccessToken retrieves an Azure access token
func (c *MCPClient) getAccessToken(ctx context.Context) (string, error) {
	if c.auth.UseManagedIdentity {
		return c.getManagedIdentityToken(ctx)
	}
	return c.getServicePrincipalToken(ctx)
}

// getServicePrincipalToken gets a token using service principal credentials
func (c *MCPClient) getServicePrincipalToken(ctx context.Context) (string, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.auth.TenantID)

	data := fmt.Sprintf("client_id=%s&scope=https://management.azure.com/.default&client_secret=%s&grant_type=client_credentials",
		c.auth.ClientID, c.auth.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewBufferString(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get token: %s", string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.AccessToken, nil
}

// getManagedIdentityToken gets a token using Azure Managed Identity
func (c *MCPClient) getManagedIdentityToken(ctx context.Context) (string, error) {
	// Azure Managed Identity endpoint
	endpoint := "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com/"

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Metadata", "true")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get managed identity token: %s", string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.AccessToken, nil
}

// Close closes the MCP client
func (c *MCPClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}


