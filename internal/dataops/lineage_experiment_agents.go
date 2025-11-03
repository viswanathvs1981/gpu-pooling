package dataops

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// LineageAgent tracks data lineage
type LineageAgent struct {
	graph        map[string][]string  // entity -> dependencies
	metadata     map[string]map[string]string
	piiDetector  *PIIDetector
}

// PIIDetector detects PII in data
type PIIDetector struct{}

// LineageRequest represents lineage query request
type LineageRequest struct {
	Entity string `json:"entity"` // dataset, feature, model
}

// LineageResponse represents lineage information
type LineageResponse struct {
	Entity       string              `json:"entity"`
	Dependencies []string            `json:"dependencies"`
	UsedBy       []string            `json:"used_by"`
	Metadata     map[string]string   `json:"metadata"`
	ContainsPII  bool                `json:"contains_pii"`
	PIIFields    []string            `json:"pii_fields,omitempty"`
}

// ExperimentAgent tracks ML experiments
type ExperimentAgent struct {
	experiments  map[string]*Experiment
	insights     *InsightGenerator
}

// Experiment represents an ML experiment
type Experiment struct {
	ID          string
	Name        string
	Parameters  map[string]interface{}
	Metrics     map[string]float64
	Timestamp   time.Time
	Status      string
}

// InsightGenerator generates insights from experiments
type InsightGenerator struct{}

// ExperimentRequest represents experiment logging request
type ExperimentRequest struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
	Metrics    map[string]float64     `json:"metrics"`
}

// ExperimentResponse represents experiment logging response
type ExperimentResponse struct {
	ExperimentID string `json:"experiment_id"`
	Status       string `json:"status"`
}

// InsightRequest represents insight generation request
type InsightRequest struct {
	ExperimentIDs []string `json:"experiment_ids"`
}

// InsightResponse represents generated insights
type InsightResponse struct {
	Insights      []string           `json:"insights"`
	BestExperiment string            `json:"best_experiment"`
	Recommendations []string          `json:"recommendations"`
}

// NewLineageAgent creates a new lineage agent
func NewLineageAgent() *LineageAgent {
	return &LineageAgent{
		graph:       make(map[string][]string),
		metadata:    make(map[string]map[string]string),
		piiDetector: &PIIDetector{},
	}
}

// NewExperimentAgent creates a new experiment agent
func NewExperimentAgent() *ExperimentAgent {
	return &ExperimentAgent{
		experiments: make(map[string]*Experiment),
		insights:    &InsightGenerator{},
	}
}

// TrackLineage records lineage relationship
func (a *LineageAgent) TrackLineage(entity string, dependencies []string, metadata map[string]string) {
	a.graph[entity] = dependencies
	a.metadata[entity] = metadata
	
	log.Printf("Lineage tracked: %s depends on %v", entity, dependencies)
}

// GetLineage retrieves lineage for an entity
func (a *LineageAgent) GetLineage(entity string) *LineageResponse {
	dependencies := a.graph[entity]
	metadata := a.metadata[entity]
	
	// Find what uses this entity
	usedBy := []string{}
	for ent, deps := range a.graph {
		for _, dep := range deps {
			if dep == entity {
				usedBy = append(usedBy, ent)
				break
			}
		}
	}
	
	// Check for PII
	containsPII, piiFields := a.piiDetector.DetectPII(entity, metadata)
	
	return &LineageResponse{
		Entity:       entity,
		Dependencies: dependencies,
		UsedBy:       usedBy,
		Metadata:     metadata,
		ContainsPII:  containsPII,
		PIIFields:    piiFields,
	}
}

// DetectPII detects personally identifiable information
func (p *PIIDetector) DetectPII(entity string, metadata map[string]string) (bool, []string) {
	piiPatterns := []string{
		"email", "phone", "ssn", "credit_card", "address",
		"name", "dob", "birth", "passport", "license",
	}
	
	piiFields := []string{}
	entityLower := strings.ToLower(entity)
	
	// Check entity name
	for _, pattern := range piiPatterns {
		if strings.Contains(entityLower, pattern) {
			piiFields = append(piiFields, entity)
			break
		}
	}
	
	// Check metadata
	for key, value := range metadata {
		keyLower := strings.ToLower(key)
		valueLower := strings.ToLower(value)
		
		for _, pattern := range piiPatterns {
			if strings.Contains(keyLower, pattern) || strings.Contains(valueLower, pattern) {
				piiFields = append(piiFields, key)
				break
			}
		}
	}
	
	return len(piiFields) > 0, piiFields
}

// LogExperiment logs a new experiment
func (a *ExperimentAgent) LogExperiment(req *ExperimentRequest) string {
	experimentID := fmt.Sprintf("exp-%d", time.Now().Unix())
	
	experiment := &Experiment{
		ID:          experimentID,
		Name:        req.Name,
		Parameters:  req.Parameters,
		Metrics:     req.Metrics,
		Timestamp:   time.Now(),
		Status:      "completed",
	}
	
	a.experiments[experimentID] = experiment
	
	log.Printf("Experiment logged: %s (ID: %s)", req.Name, experimentID)
	
	return experimentID
}

// GenerateInsights generates insights from experiments
func (g *InsightGenerator) GenerateInsights(experiments map[string]*Experiment, experimentIDs []string) *InsightResponse {
	if len(experimentIDs) == 0 {
		return &InsightResponse{
			Insights:        []string{"No experiments provided"},
			Recommendations: []string{},
		}
	}
	
	// Filter experiments
	targetExperiments := make(map[string]*Experiment)
	for _, id := range experimentIDs {
		if exp, ok := experiments[id]; ok {
			targetExperiments[id] = exp
		}
	}
	
	if len(targetExperiments) == 0 {
		return &InsightResponse{
			Insights:        []string{"No valid experiments found"},
			Recommendations: []string{},
		}
	}
	
	// Find best experiment (highest accuracy)
	bestID := ""
	bestAccuracy := 0.0
	
	for id, exp := range targetExperiments {
		if accuracy, ok := exp.Metrics["accuracy"]; ok {
			if accuracy > bestAccuracy {
				bestAccuracy = accuracy
				bestID = id
			}
		}
	}
	
	// Generate insights
	insights := []string{
		fmt.Sprintf("Analyzed %d experiments", len(targetExperiments)),
	}
	
	if bestID != "" {
		insights = append(insights, fmt.Sprintf("Best performing experiment: %s (accuracy: %.2f%%)", bestID, bestAccuracy*100))
	}
	
	// Generate recommendations
	recommendations := []string{
		"Try increasing learning_rate by 2× for faster convergence",
		"Consider using XGBoost for tabular data (typically 3× faster than neural networks)",
		"Enable early stopping to prevent overfitting",
	}
	
	return &InsightResponse{
		Insights:        insights,
		BestExperiment:  bestID,
		Recommendations: recommendations,
	}
}

// HTTP Handlers for Lineage Agent
func (a *LineageAgent) HTTPHandler() http.Handler {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/v1/track-lineage", a.handleTrackLineage)
	mux.HandleFunc("/v1/get-lineage", a.handleGetLineage)
	mux.HandleFunc("/health", a.handleHealth)
	
	return mux
}

func (a *LineageAgent) handleTrackLineage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Entity       string            `json:"entity"`
		Dependencies []string          `json:"dependencies"`
		Metadata     map[string]string `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	a.TrackLineage(req.Entity, req.Dependencies, req.Metadata)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (a *LineageAgent) handleGetLineage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LineageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	resp := a.GetLineage(req.Entity)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *LineageAgent) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Start starts the lineage agent
func (a *LineageAgent) Start(ctx context.Context, addr string) error {
	server := &http.Server{
		Addr:    addr,
		Handler: a.HTTPHandler(),
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	log.Printf("Lineage Agent listening on %s", addr)
	return server.ListenAndServe()
}

// HTTP Handlers for Experiment Agent
func (a *ExperimentAgent) HTTPHandler() http.Handler {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/v1/log-experiment", a.handleLogExperiment)
	mux.HandleFunc("/v1/generate-insights", a.handleGenerateInsights)
	mux.HandleFunc("/health", a.handleHealth)
	
	return mux
}

func (a *ExperimentAgent) handleLogExperiment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	experimentID := a.LogExperiment(&req)

	resp := ExperimentResponse{
		ExperimentID: experimentID,
		Status:       "success",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *ExperimentAgent) handleGenerateInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req InsightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	resp := a.insights.GenerateInsights(a.experiments, req.ExperimentIDs)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *ExperimentAgent) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Start starts the experiment agent
func (a *ExperimentAgent) Start(ctx context.Context, addr string) error {
	server := &http.Server{
		Addr:    addr,
		Handler: a.HTTPHandler(),
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	log.Printf("Experiment Agent listening on %s", addr)
	return server.ListenAndServe()
}

