package aisafety

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"
)

// EvaluationAgent handles model evaluation and benchmarking
type EvaluationAgent struct {
	benchmarker *Benchmarker
	validator   *OutputValidator
	abTester    *ABTester
}

// Benchmarker runs standardized benchmarks
type Benchmarker struct{}

// OutputValidator validates model outputs
type OutputValidator struct{}

// ABTester performs A/B testing
type ABTester struct {
	experiments map[string]*ABExperiment
}

// ABExperiment represents an A/B test
type ABExperiment struct {
	ID            string
	ModelA        string
	ModelB        string
	MetricName    string
	SamplesA      []float64
	SamplesB      []float64
	StartTime     time.Time
}

// BenchmarkRequest represents a benchmark request
type BenchmarkRequest struct {
	ModelID       string   `json:"model_id"`
	BenchmarkType string   `json:"benchmark_type"` // "mmlu", "truthfulqa", "hellaswag", etc.
	SampleSize    int      `json:"sample_size,omitempty"`
}

// BenchmarkResponse represents benchmark results
type BenchmarkResponse struct {
	ModelID       string             `json:"model_id"`
	BenchmarkType string             `json:"benchmark_type"`
	Score         float64            `json:"score"`
	Details       map[string]float64 `json:"details,omitempty"`
	Timestamp     time.Time          `json:"timestamp"`
}

// ValidationRequest represents output validation request
type ValidationRequest struct {
	Output      string   `json:"output"`
	Rules       []string `json:"rules"`
	Constraints map[string]interface{} `json:"constraints,omitempty"`
}

// ValidationResponse represents validation results
type ValidationResponse struct {
	Valid        bool     `json:"valid"`
	Violations   []string `json:"violations,omitempty"`
	Score        float64  `json:"score"`
	Confidence   float64  `json:"confidence"`
}

// ABTestRequest represents A/B test request
type ABTestRequest struct {
	ExperimentID string  `json:"experiment_id"`
	ModelA       string  `json:"model_a"`
	ModelB       string  `json:"model_b"`
	MetricName   string  `json:"metric_name"`
	MetricValue  float64 `json:"metric_value"`
	Model        string  `json:"model"` // Which model produced this result
}

// ABTestResponse represents A/B test results
type ABTestResponse struct {
	ExperimentID  string  `json:"experiment_id"`
	WinningModel  string  `json:"winning_model"`
	Confidence    float64 `json:"confidence"`
	PValue        float64 `json:"p_value"`
	Effect        string  `json:"effect"` // "significant", "marginal", "none"
	Recommendation string  `json:"recommendation"`
}

// NewEvaluationAgent creates a new evaluation agent
func NewEvaluationAgent() *EvaluationAgent {
	return &EvaluationAgent{
		benchmarker: &Benchmarker{},
		validator:   &OutputValidator{},
		abTester:    &ABTester{experiments: make(map[string]*ABExperiment)},
	}
}

// RunBenchmark executes a standardized benchmark
func (b *Benchmarker) RunBenchmark(req *BenchmarkRequest) (*BenchmarkResponse, error) {
	// Simulate benchmark execution (in production, run actual benchmarks)
	score := 0.0
	details := make(map[string]float64)
	
	switch req.BenchmarkType {
	case "mmlu":
		// Massive Multitask Language Understanding
		score = 0.72 + (float64(time.Now().Unix()%100) / 1000.0) // Simulated: 72-73%
		details["humanities"] = 0.68
		details["social_sciences"] = 0.74
		details["stem"] = 0.71
		details["other"] = 0.75
		
	case "truthfulqa":
		// TruthfulQA benchmark
		score = 0.58 + (float64(time.Now().Unix()%100) / 1000.0) // Simulated: 58-59%
		details["truthful"] = score
		details["informative"] = 0.82
		
	case "hellaswag":
		// HellaSwag (commonsense reasoning)
		score = 0.85 + (float64(time.Now().Unix()%100) / 1000.0) // Simulated: 85-86%
		
	case "humaneval":
		// HumanEval (code generation)
		score = 0.48 + (float64(time.Now().Unix()%100) / 1000.0) // Simulated: 48-49%
		details["pass@1"] = score
		details["pass@10"] = 0.72
		
	default:
		return nil, fmt.Errorf("unknown benchmark type: %s", req.BenchmarkType)
	}
	
	return &BenchmarkResponse{
		ModelID:       req.ModelID,
		BenchmarkType: req.BenchmarkType,
		Score:         score,
		Details:       details,
		Timestamp:     time.Now(),
	}, nil
}

// ValidateOutput validates model output against rules
func (v *OutputValidator) ValidateOutput(req *ValidationRequest) (*ValidationResponse, error) {
	violations := []string{}
	score := 1.0
	
	// Check each rule
	for _, rule := range req.Rules {
		switch rule {
		case "no_pii":
			if containsPII(req.Output) {
				violations = append(violations, "Output contains PII")
				score -= 0.3
			}
		case "max_length":
			if maxLen, ok := req.Constraints["max_length"].(float64); ok {
				if len(req.Output) > int(maxLen) {
					violations = append(violations, fmt.Sprintf("Output exceeds max length (%d > %d)", len(req.Output), int(maxLen)))
					score -= 0.2
				}
			}
		case "no_code":
			if containsCode(req.Output) {
				violations = append(violations, "Output contains code")
				score -= 0.2
			}
		case "professional_tone":
			if !isProfessional(req.Output) {
				violations = append(violations, "Output lacks professional tone")
				score -= 0.15
			}
		}
	}
	
	if score < 0 {
		score = 0
	}
	
	valid := len(violations) == 0
	confidence := 0.85 // Simplified confidence
	
	return &ValidationResponse{
		Valid:      valid,
		Violations: violations,
		Score:      score,
		Confidence: confidence,
	}, nil
}

// RecordABTest records an A/B test result
func (a *ABTester) RecordABTest(req *ABTestRequest) error {
	experiment, exists := a.experiments[req.ExperimentID]
	if !exists {
		// Create new experiment
		experiment = &ABExperiment{
			ID:         req.ExperimentID,
			ModelA:     req.ModelA,
			ModelB:     req.ModelB,
			MetricName: req.MetricName,
			SamplesA:   []float64{},
			SamplesB:   []float64{},
			StartTime:  time.Now(),
		}
		a.experiments[req.ExperimentID] = experiment
	}
	
	// Record sample
	if req.Model == req.ModelA {
		experiment.SamplesA = append(experiment.SamplesA, req.MetricValue)
	} else if req.Model == req.ModelB {
		experiment.SamplesB = append(experiment.SamplesB, req.MetricValue)
	}
	
	return nil
}

// AnalyzeABTest analyzes A/B test results
func (a *ABTester) AnalyzeABTest(experimentID string) (*ABTestResponse, error) {
	experiment, exists := a.experiments[experimentID]
	if !exists {
		return nil, fmt.Errorf("experiment not found: %s", experimentID)
	}
	
	// Need sufficient samples
	if len(experiment.SamplesA) < 30 || len(experiment.SamplesB) < 30 {
		return &ABTestResponse{
			ExperimentID:   experimentID,
			WinningModel:   "insufficient_data",
			Confidence:     0.0,
			PValue:         1.0,
			Effect:         "none",
			Recommendation: fmt.Sprintf("Need more samples (A: %d, B: %d)", len(experiment.SamplesA), len(experiment.SamplesB)),
		}, nil
	}
	
	// Calculate means
	meanA := calculateMean(experiment.SamplesA)
	meanB := calculateMean(experiment.SamplesB)
	
	// Simple t-test (simplified)
	pValue := calculatePValue(experiment.SamplesA, experiment.SamplesB)
	
	winningModel := experiment.ModelA
	if meanB > meanA {
		winningModel = experiment.ModelB
	}
	
	effect := "none"
	confidence := 1.0 - pValue
	
	if pValue < 0.01 {
		effect = "significant"
	} else if pValue < 0.05 {
		effect = "marginal"
	}
	
	recommendation := ""
	if effect == "significant" {
		recommendation = fmt.Sprintf("Deploy %s (%.1f%% better)", winningModel, math.Abs(meanB-meanA)/meanA*100)
	} else {
		recommendation = "Continue testing - no clear winner"
	}
	
	return &ABTestResponse{
		ExperimentID:   experimentID,
		WinningModel:   winningModel,
		Confidence:     confidence,
		PValue:         pValue,
		Effect:         effect,
		Recommendation: recommendation,
	}, nil
}

// Helper functions
func containsPII(text string) bool {
	// Simplified PII detection
	piiPatterns := []string{"@", "ssn", "credit card", "phone:", "email:"}
	for _, pattern := range piiPatterns {
		if contains(text, pattern) {
			return true
		}
	}
	return false
}

func containsCode(text string) bool {
	codeIndicators := []string{"func ", "def ", "class ", "import ", "```", "function("}
	for _, indicator := range codeIndicators {
		if contains(text, indicator) {
			return true
		}
	}
	return false
}

func isProfessional(text string) bool {
	// Check for casual language
	casualWords := []string{"lol", "omg", "wtf", "gonna", "wanna"}
	for _, word := range casualWords {
		if contains(text, word) {
			return false
		}
	}
	return true
}

func contains(text, substr string) bool {
	return len(text) > 0 && len(substr) > 0 && 
		   (text == substr || len(text) >= len(substr) && 
		   (text[:len(substr)] == substr || containsSubstring(text, substr)))
}

func containsSubstring(text, substr string) bool {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculatePValue(samplesA, samplesB []float64) float64 {
	// Simplified p-value calculation (in production, use proper statistical library)
	meanA := calculateMean(samplesA)
	meanB := calculateMean(samplesB)
	
	diff := math.Abs(meanA - meanB)
	
	// Simplified: larger difference = smaller p-value
	if diff > 0.2 {
		return 0.001
	} else if diff > 0.1 {
		return 0.01
	} else if diff > 0.05 {
		return 0.05
	} else {
		return 0.5
	}
}

// HTTP Handlers
func (a *EvaluationAgent) HTTPHandler() http.Handler {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/v1/benchmark", a.handleBenchmark)
	mux.HandleFunc("/v1/validate", a.handleValidate)
	mux.HandleFunc("/v1/ab-test/record", a.handleABRecord)
	mux.HandleFunc("/v1/ab-test/analyze", a.handleABAnalyze)
	mux.HandleFunc("/health", a.handleHealth)
	
	return mux
}

func (a *EvaluationAgent) handleBenchmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BenchmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	resp, err := a.benchmarker.RunBenchmark(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *EvaluationAgent) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	resp, err := a.validator.ValidateOutput(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *EvaluationAgent) handleABRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ABTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := a.abTester.RecordABTest(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})
}

func (a *EvaluationAgent) handleABAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ExperimentID string `json:"experiment_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	resp, err := a.abTester.AnalyzeABTest(req.ExperimentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *EvaluationAgent) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Start starts the evaluation agent
func (a *EvaluationAgent) Start(ctx context.Context, addr string) error {
	server := &http.Server{
		Addr:    addr,
		Handler: a.HTTPHandler(),
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	log.Printf("Model Evaluation Agent listening on %s", addr)
	return server.ListenAndServe()
}

