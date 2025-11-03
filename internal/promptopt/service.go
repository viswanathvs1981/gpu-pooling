package promptopt

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)

// OptimizationRequest represents a request to optimize a prompt
type OptimizationRequest struct {
	OriginalPrompt string   `json:"original_prompt"`
	TaskType       string   `json:"task_type,omitempty"`       // e.g., "reasoning", "classification", "generation"
	TargetModel    string   `json:"target_model,omitempty"`    // e.g., "gpt-4", "llama-3.1"
	Techniques     []string `json:"techniques,omitempty"`      // Specific techniques to apply
	Context        string   `json:"context,omitempty"`         // Additional context
	MaxTokens      int      `json:"max_tokens,omitempty"`      // Maximum token budget
	OptimizeTokens bool     `json:"optimize_tokens,omitempty"` // Enable token optimization
}

// OptimizationResponse represents the optimized prompt and metadata
type OptimizationResponse struct {
	OptimizedPrompt      string   `json:"optimized_prompt"`
	Confidence           float64  `json:"confidence"`
	TechniquesApplied    []string `json:"techniques_applied"`
	Changes              []string `json:"changes"`
	EstimatedImprovement float64  `json:"estimated_improvement"`
	LatencyMs            int64    `json:"latency_ms"`
	// Token optimization metrics
	OriginalTokens  int     `json:"original_tokens,omitempty"`
	OptimizedTokens int     `json:"optimized_tokens,omitempty"`
	TokensSaved     int     `json:"tokens_saved,omitempty"`
	TokenSavingsPct float64 `json:"token_savings_percent,omitempty"`
}

// Service provides prompt optimization capabilities
type Service struct {
	rewriter     *Rewriter
	safety       *SafetyChecker
	tokenizer    *Tokenizer
	redis        *redis.Client
	stats        *Statistics
	a2aEnabled   bool
}

// Statistics tracks optimization metrics
type Statistics struct {
	TotalOptimizations  int64
	SuccessfulOpts      int64
	TotalLatencyMs      int64
}

// NewService creates a new prompt optimization service
func NewService(redisAddr string, enableA2A bool) (*Service, error) {
	var rdb *redis.Client
	if enableA2A && redisAddr != "" {
		rdb = redis.NewClient(&redis.Options{
			Addr: redisAddr,
		})
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("Warning: Redis connection failed: %v", err)
			rdb = nil
		}
	}

	return &Service{
		rewriter:   NewRewriter(),
		safety:     NewSafetyChecker(),
		tokenizer:  NewTokenizer(),
		redis:      rdb,
		stats:      &Statistics{},
		a2aEnabled: enableA2A && rdb != nil,
	}, nil
}

// Optimize applies prompt optimization techniques
func (s *Service) Optimize(ctx context.Context, req *OptimizationRequest) (*OptimizationResponse, error) {
	startTime := time.Now()
	atomic.AddInt64(&s.stats.TotalOptimizations, 1)

	// Step 1: Safety check
	if unsafe, reason := s.safety.CheckSafety(req.OriginalPrompt); unsafe {
		return nil, fmt.Errorf("unsafe prompt detected: %s", reason)
	}

	// Step 2: Count original tokens
	originalTokens := s.tokenizer.CountTokens(req.OriginalPrompt)

	// Step 3: Apply rewriting techniques
	optimized, techniquesApplied, changes, err := s.rewriter.Rewrite(req)
	if err != nil {
		return nil, fmt.Errorf("rewriting failed: %w", err)
	}

	// Step 4: Apply token optimization if requested
	optimizedTokens := originalTokens
	tokensSaved := 0
	tokenSavingsPct := 0.0

	if req.OptimizeTokens || req.MaxTokens > 0 {
		if req.MaxTokens > 0 {
			// Optimize to fit within token budget
			optimized, optimizedTokens = s.tokenizer.OptimizeToTokenBudget(optimized, req.MaxTokens)
			changes = append(changes, fmt.Sprintf("Compressed to fit %d token budget", req.MaxTokens))
		} else {
			// General compression (30% target)
			optimized = s.tokenizer.CompressText(optimized, 0.30)
			optimizedTokens = s.tokenizer.CountTokens(optimized)
		}
		
		tokensSaved, tokenSavingsPct = s.tokenizer.CalculateTokenSavings(req.OriginalPrompt, optimized)
		
		if tokensSaved > 0 {
			techniquesApplied = append(techniquesApplied, "token-optimization")
			changes = append(changes, fmt.Sprintf("Reduced tokens by %d (%.1f%%)", tokensSaved, tokenSavingsPct*100))
		}
	} else {
		optimizedTokens = s.tokenizer.CountTokens(optimized)
	}

	// Step 5: Calculate confidence and improvement estimate
	confidence := s.calculateConfidence(req, techniquesApplied)
	improvement := s.estimateImprovement(req.TaskType, techniquesApplied)

	latency := time.Since(startTime).Milliseconds()
	atomic.AddInt64(&s.stats.SuccessfulOpts, 1)
	atomic.AddInt64(&s.stats.TotalLatencyMs, latency)

	return &OptimizationResponse{
		OptimizedPrompt:      optimized,
		Confidence:           confidence,
		TechniquesApplied:    techniquesApplied,
		Changes:              changes,
		EstimatedImprovement: improvement,
		LatencyMs:            latency,
		OriginalTokens:       originalTokens,
		OptimizedTokens:      optimizedTokens,
		TokensSaved:          tokensSaved,
		TokenSavingsPct:      tokenSavingsPct,
	}, nil
}

// calculateConfidence estimates optimization confidence
func (s *Service) calculateConfidence(req *OptimizationRequest, techniques []string) float64 {
	baseConfidence := 0.70
	
	// Increase confidence based on number of techniques applied
	techniqueFactor := float64(len(techniques)) * 0.05
	
	// Increase confidence if task type is specified
	if req.TaskType != "" {
		baseConfidence += 0.10
	}
	
	// Increase confidence if context is provided
	if req.Context != "" {
		baseConfidence += 0.10
	}
	
	confidence := baseConfidence + techniqueFactor
	if confidence > 0.95 {
		confidence = 0.95
	}
	
	return confidence
}

// estimateImprovement estimates expected quality improvement
func (s *Service) estimateImprovement(taskType string, techniques []string) float64 {
	// Base improvement per technique
	baseImprovement := 0.15
	
	// Task-specific multipliers (based on empirical data from the presentation)
	taskMultipliers := map[string]float64{
		"reasoning":       1.7,  // 70% improvement for reasoning tasks
		"classification":  1.85, // 85% improvement for classification
		"generation":      1.3,  // 30% improvement for generation
		"factual":         1.6,  // 60% improvement for factual queries
	}
	
	multiplier := 1.0
	if m, ok := taskMultipliers[taskType]; ok {
		multiplier = m
	}
	
	return baseImprovement * float64(len(techniques)) * multiplier
}

// StartA2AListener listens for optimization requests on Redis
func (s *Service) StartA2AListener(ctx context.Context) error {
	if !s.a2aEnabled || s.redis == nil {
		return fmt.Errorf("A2A communication not enabled")
	}

	pubsub := s.redis.Subscribe(ctx, "prompt-optimization")
	defer pubsub.Close()

	log.Println("Prompt Optimizer: A2A listener started on channel 'prompt-optimization'")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-pubsub.Channel():
			s.handleA2AMessage(ctx, msg.Payload)
		}
	}
}

// handleA2AMessage processes A2A optimization requests
func (s *Service) handleA2AMessage(ctx context.Context, payload string) {
	var req OptimizationRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		log.Printf("Error parsing A2A message: %v", err)
		return
	}

	resp, err := s.Optimize(ctx, &req)
	if err != nil {
		log.Printf("Error optimizing prompt: %v", err)
		return
	}

	// Publish response back
	respData, _ := json.Marshal(resp)
	s.redis.Publish(ctx, "prompt-optimization-response", string(respData))
}

// HTTPHandler provides HTTP endpoints for prompt optimization
func (s *Service) HTTPHandler() http.Handler {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/v1/optimize", s.handleOptimize)
	mux.HandleFunc("/v1/stats", s.handleStats)
	mux.HandleFunc("/health", s.handleHealth)
	
	return mux
}

func (s *Service) handleOptimize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OptimizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	resp, err := s.Optimize(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Service) handleStats(w http.ResponseWriter, r *http.Request) {
	total := atomic.LoadInt64(&s.stats.TotalOptimizations)
	successful := atomic.LoadInt64(&s.stats.SuccessfulOpts)
	totalLatency := atomic.LoadInt64(&s.stats.TotalLatencyMs)
	
	avgLatency := float64(0)
	if successful > 0 {
		avgLatency = float64(totalLatency) / float64(successful)
	}
	
	successRate := float64(0)
	if total > 0 {
		successRate = float64(successful) / float64(total)
	}
	
	stats := map[string]interface{}{
		"total_optimizations":  total,
		"successful_optimizations": successful,
		"success_rate":         successRate,
		"average_latency_ms":   avgLatency,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Service) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

