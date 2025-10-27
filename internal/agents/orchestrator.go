package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// OrchestratorAgent coordinates multi-agent workflows
type OrchestratorAgent struct {
	httpServer    *http.Server
	workflows     map[string]*WorkflowExecution
	workflowsLock sync.RWMutex
	mcpServerURL  string
	port          int
}

// Request represents a user request
type Request struct {
	ID          string                 `json:"id"`
	Intent      string                 `json:"intent"`
	RawRequest  string                 `json:"raw_request"`
	Parameters  map[string]interface{} `json:"parameters"`
	Status      string                 `json:"status"` // pending, running, completed, failed
	CreatedAt   time.Time              `json:"created_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// WorkflowExecution tracks workflow execution
type WorkflowExecution struct {
	Request      *Request
	Workflow     *Workflow
	CurrentStep  int
	State        map[string]interface{}
	StartedAt    time.Time
	CompletedAt  *time.Time
}

// NewOrchestratorAgent creates a new orchestrator
func NewOrchestratorAgent(port int, mcpServerURL string) *OrchestratorAgent {
	return &OrchestratorAgent{
		workflows:    make(map[string]*WorkflowExecution),
		mcpServerURL: mcpServerURL,
		port:         port,
	}
}

// Start starts the orchestrator HTTP API
func (o *OrchestratorAgent) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/requests", o.handleRequests)
	mux.HandleFunc("/api/v1/requests/", o.handleGetRequest)
	mux.HandleFunc("/api/v1/workflows", o.handleListWorkflows)
	mux.HandleFunc("/health", o.handleHealth)

	o.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", o.port),
		Handler: mux,
	}

	logger.Info("Starting Orchestrator Agent", "port", o.port)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := o.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error(err, "Error shutting down orchestrator")
		}
	}()

	if err := o.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("orchestrator server error: %w", err)
	}

	return nil
}

// handleRequests handles POST /api/v1/requests
func (o *OrchestratorAgent) handleRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Request string                 `json:"request"`
		Params  map[string]interface{} `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse the request to determine intent
	intent, params := parseRequest(input.Request, input.Params)

	// Create request
	req := &Request{
		ID:         uuid.New().String(),
		Intent:     intent,
		RawRequest: input.Request,
		Parameters: params,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	// Execute workflow asynchronously
	go o.executeWorkflow(r.Context(), req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(req)
}

// handleGetRequest handles GET /api/v1/requests/:id
func (o *OrchestratorAgent) handleGetRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	requestID := r.URL.Path[len("/api/v1/requests/"):]

	o.workflowsLock.RLock()
	exec, exists := o.workflows[requestID]
	o.workflowsLock.RUnlock()

	if !exists {
		http.Error(w, "Request not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exec.Request)
}

// handleListWorkflows handles GET /api/v1/workflows
func (o *OrchestratorAgent) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	o.workflowsLock.RLock()
	workflows := make([]*Request, 0, len(o.workflows))
	for _, exec := range o.workflows {
		workflows = append(workflows, exec.Request)
	}
	o.workflowsLock.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflows": workflows,
		"count":     len(workflows),
	})
}

// handleHealth handles GET /health
func (o *OrchestratorAgent) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

// executeWorkflow executes a workflow for a request
func (o *OrchestratorAgent) executeWorkflow(ctx context.Context, req *Request) {
	logger := log.FromContext(ctx)
	logger.Info("Executing workflow", "intent", req.Intent, "request_id", req.ID)

	req.Status = "running"

	// Select workflow based on intent
	var workflow *Workflow
	switch req.Intent {
	case "deploy_model":
		workflow = NewDeployModelWorkflow()
	case "train_and_deploy":
		workflow = NewTrainAndDeployWorkflow()
	case "optimize_costs":
		workflow = NewOptimizeCostsWorkflow()
	default:
		req.Status = "failed"
		req.Error = fmt.Sprintf("Unknown intent: %s", req.Intent)
		now := time.Now()
		req.CompletedAt = &now
		return
	}

	// Create workflow execution
	exec := &WorkflowExecution{
		Request:     req,
		Workflow:    workflow,
		CurrentStep: 0,
		State:       make(map[string]interface{}),
		StartedAt:   time.Now(),
	}

	o.workflowsLock.Lock()
	o.workflows[req.ID] = exec
	o.workflowsLock.Unlock()

	// Execute workflow steps
	for i, step := range workflow.Steps {
		exec.CurrentStep = i
		logger.Info("Executing step", "step", i, "name", step.Name)

		result, err := step.Execute(ctx, o.mcpServerURL, req.Parameters, exec.State)
		if err != nil {
			req.Status = "failed"
			req.Error = fmt.Sprintf("Step %d (%s) failed: %v", i, step.Name, err)
			now := time.Now()
			req.CompletedAt = &now
			logger.Error(err, "Workflow step failed", "step", step.Name)
			return
		}

		// Store result in state
		exec.State[step.Name] = result
		logger.Info("Step completed", "step", step.Name, "result", result)
	}

	// Workflow completed successfully
	req.Status = "completed"
	req.Result = exec.State
	now := time.Now()
	req.CompletedAt = &now
	exec.CompletedAt = &now

	logger.Info("Workflow completed", "request_id", req.ID, "duration", time.Since(exec.StartedAt))
}

// parseRequest parses a natural language request
func parseRequest(rawRequest string, params map[string]interface{}) (string, map[string]interface{}) {
	if params == nil {
		params = make(map[string]interface{})
	}

	// Simple pattern matching (in production, use NLP or LLM)
	intent := "unknown"

	// Check for deploy patterns
	if contains(rawRequest, []string{"deploy", "deployment", "serve"}) {
		intent = "deploy_model"
	}

	// Check for training patterns
	if contains(rawRequest, []string{"train", "training", "fine-tune", "lora"}) {
		intent = "train_and_deploy"
	}

	// Check for optimization patterns
	if contains(rawRequest, []string{"optimize", "cost", "savings", "cheaper"}) {
		intent = "optimize_costs"
	}

	return intent, params
}

// contains checks if text contains any of the keywords
func contains(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if len(text) >= len(keyword) {
			for i := 0; i <= len(text)-len(keyword); i++ {
				if text[i:i+len(keyword)] == keyword {
					return true
				}
			}
		}
	}
	return false
}

