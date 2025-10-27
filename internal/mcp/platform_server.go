package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/NexusGPU/tensor-fusion/internal/mcp/tools"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PlatformServer provides MCP (Model Context Protocol) tools for platform operations
type PlatformServer struct {
	httpServer *http.Server
	k8sClient  client.Client
	clientset  *kubernetes.Clientset
	toolset    *tools.PlatformTools
	port       int
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	ID      interface{}            `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// NewPlatformServer creates a new MCP server
func NewPlatformServer(port int) (*PlatformServer, error) {
	// Get Kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	// Create controller-runtime client
	k8sClient, err := client.New(config, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	// Create clientset for legacy APIs
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	// Create toolset
	toolset := tools.NewPlatformTools(k8sClient, clientset)

	server := &PlatformServer{
		k8sClient: k8sClient,
		clientset: clientset,
		toolset:   toolset,
		port:      port,
	}

	return server, nil
}

// Start starts the MCP server
func (s *PlatformServer) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/tools", s.handleToolsList)
	mux.HandleFunc("/execute", s.handleExecute)
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      s.loggingMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.Info("Starting MCP Platform Server", "port", s.port)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error(err, "Error shutting down MCP server")
		}
	}()

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("MCP server error: %w", err)
	}

	return nil
}

// handleRoot provides info about the server
func (s *PlatformServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := map[string]interface{}{
		"name":        "TensorFusion Platform MCP Server",
		"version":     "1.0.0",
		"protocol":    "JSON-RPC 2.0",
		"description": "Model Context Protocol server for TensorFusion platform operations",
		"endpoints": map[string]string{
			"/tools":   "List available tools",
			"/execute": "Execute a tool",
			"/health":  "Health check",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// handleToolsList returns list of available tools
func (s *PlatformServer) handleToolsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	toolsList := s.toolset.ListTools()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": toolsList,
	})
}

// handleExecute executes a tool via JSON-RPC
func (s *PlatformServer) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, nil, -32700, "Parse error", err.Error())
		return
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		s.sendError(w, req.ID, -32600, "Invalid Request", "jsonrpc must be 2.0")
		return
	}

	// Execute the tool
	ctx := r.Context()
	result, err := s.toolset.ExecuteTool(ctx, req.Method, req.Params)
	if err != nil {
		s.sendError(w, req.ID, -32603, "Internal error", err.Error())
		return
	}

	// Send success response
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleHealth returns health status
func (s *PlatformServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

// sendError sends a JSON-RPC error response
func (s *PlatformServer) sendError(w http.ResponseWriter, id interface{}, code int, message string, data string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC errors use 200 OK
	json.NewEncoder(w).Encode(resp)
}

// loggingMiddleware logs all requests
func (s *PlatformServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Log.Info("MCP request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
		)
	})
}

