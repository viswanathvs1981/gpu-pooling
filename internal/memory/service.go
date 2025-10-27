package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	tfv1 "github.com/NexusGPU/tensor-fusion/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// MemoryService provides memory provisioning and access APIs
type MemoryService struct {
	httpServer     *http.Server
	k8sClient      client.Client
	semantic       *SemanticMemory
	episodic       *EpisodicMemory
	longterm       *LongtermMemory
	port           int
	serviceBaseURL string
}

// AgentProvisionRequest represents a request to provision memory for an agent
type AgentProvisionRequest struct {
	AgentID     string   `json:"agent_id"`
	MemoryTypes []string `json:"memory_types"`
	Retention   string   `json:"retention,omitempty"`
	MaxSize     string   `json:"max_size,omitempty"`
}

// AgentProvisionResponse represents the response with memory URLs
type AgentProvisionResponse struct {
	AgentID    string            `json:"agent_id"`
	MemoryURLs map[string]string `json:"memory_urls"`
	Status     string            `json:"status"`
}

// NewMemoryService creates a new memory service
func NewMemoryService(port int, k8sClient client.Client, qdrantURL, greptimeURL string) (*MemoryService, error) {
	semantic := NewSemanticMemory(qdrantURL)
	episodic := NewEpisodicMemory(greptimeURL)
	longterm := NewLongtermMemory(qdrantURL) // Long-term also uses vector DB

	return &MemoryService{
		k8sClient:      k8sClient,
		semantic:       semantic,
		episodic:       episodic,
		longterm:       longterm,
		port:           port,
		serviceBaseURL: fmt.Sprintf("http://tensor-fusion-memory-service.tensor-fusion-sys.svc.cluster.local:%d", port),
	}, nil
}

// Start starts the memory service HTTP server
func (ms *MemoryService) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/agents", ms.handleProvisionAgent)
	mux.HandleFunc("/api/v1/memory/", ms.handleMemoryOperations)
	mux.HandleFunc("/health", ms.handleHealth)

	ms.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", ms.port),
		Handler: mux,
	}

	logger.Info("Starting Memory Service", "port", ms.port)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := ms.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error(err, "Error shutting down memory service")
		}
	}()

	if err := ms.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("memory service error: %w", err)
	}

	return nil
}

// handleProvisionAgent handles agent memory provisioning
func (ms *MemoryService) handleProvisionAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AgentProvisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create AgentMemory CR
	agentMemory := &tfv1.AgentMemory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("agent-memory-%s", req.AgentID),
			Namespace: "tensor-fusion-sys",
		},
		Spec: tfv1.AgentMemorySpec{
			AgentID:     req.AgentID,
			MemoryTypes: req.MemoryTypes,
			Retention:   req.Retention,
			MaxSize:     req.MaxSize,
		},
	}

	if err := ms.k8sClient.Create(r.Context(), agentMemory); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create AgentMemory: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate memory URLs
	memoryURLs := make(map[string]string)
	for _, memType := range req.MemoryTypes {
		memoryURLs[memType] = fmt.Sprintf("%s/api/v1/memory/%s/%s", ms.serviceBaseURL, req.AgentID, memType)
	}

	// Update status
	agentMemory.Status.Phase = "Active"
	agentMemory.Status.SemanticURL = memoryURLs["semantic"]
	agentMemory.Status.EpisodicURL = memoryURLs["episodic"]
	agentMemory.Status.LongtermURL = memoryURLs["longterm"]
	now := metav1.Now()
	agentMemory.Status.LastAccessTime = &now

	if err := ms.k8sClient.Status().Update(r.Context(), agentMemory); err != nil {
		log.Log.Error(err, "Failed to update AgentMemory status")
	}

	// Respond
	resp := AgentProvisionResponse{
		AgentID:    req.AgentID,
		MemoryURLs: memoryURLs,
		Status:     "provisioned",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleMemoryOperations routes memory operations to appropriate backend
func (ms *MemoryService) handleMemoryOperations(w http.ResponseWriter, r *http.Request) {
	// Parse URL: /api/v1/memory/{agent_id}/{memory_type}/{operation}
	// For now, delegate to specific handlers based on memory type
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Memory operation endpoint",
		"status":  "active",
	})
}

// handleHealth returns service health
func (ms *MemoryService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

