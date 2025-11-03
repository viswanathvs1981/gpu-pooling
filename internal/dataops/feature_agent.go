package dataops

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
)

// FeatureEngineeringAgent handles automated feature generation
type FeatureEngineeringAgent struct {
	generator *FeatureGenerator
	selector  *FeatureSelector
}

// FeatureGenerator generates features
type FeatureGenerator struct{}

// FeatureSelector selects important features
type FeatureSelector struct{}

// FeatureRequest represents feature engineering request
type FeatureRequest struct {
	Data       []map[string]interface{} `json:"data"`
	TaskType   string                   `json:"task_type"`   // classification, regression, etc.
	TargetCol  string                   `json:"target_col,omitempty"`
}

// FeatureResponse represents generated features
type FeatureResponse struct {
	Features      []string  `json:"features"`
	Importances   map[string]float64 `json:"importances,omitempty"`
	TotalGenerated int      `json:"total_generated"`
	Selected      int       `json:"selected"`
}

// NewFeatureEngineeringAgent creates a new feature engineering agent
func NewFeatureEngineeringAgent() *FeatureEngineeringAgent {
	return &FeatureEngineeringAgent{
		generator: &FeatureGenerator{},
		selector:  &FeatureSelector{},
	}
}

// GenerateFeatures generates new features from existing ones
func (g *FeatureGenerator) GenerateFeatures(data []map[string]interface{}) []string {
	if len(data) == 0 {
		return nil
	}

	generatedFeatures := []string{}
	
	// Get numeric columns
	numericCols := g.getNumericColumns(data)
	
	// 1. Polynomial features (x^2, x^3)
	for _, col := range numericCols {
		generatedFeatures = append(generatedFeatures, fmt.Sprintf("%s_squared", col))
		generatedFeatures = append(generatedFeatures, fmt.Sprintf("%s_cubed", col))
	}
	
	// 2. Interaction features (x1 * x2)
	for i, col1 := range numericCols {
		for j := i + 1; j < len(numericCols) && j < i+3; j++ { // Limit combinations
			col2 := numericCols[j]
			generatedFeatures = append(generatedFeatures, fmt.Sprintf("%s_x_%s", col1, col2))
		}
	}
	
	// 3. Ratio features (x1 / x2)
	for i, col1 := range numericCols {
		for j := i + 1; j < len(numericCols) && j < i+2; j++ {
			col2 := numericCols[j]
			generatedFeatures = append(generatedFeatures, fmt.Sprintf("%s_div_%s", col1, col2))
		}
	}
	
	// 4. Aggregation features (for time-series)
	if g.hasTimeColumn(data) {
		for _, col := range numericCols {
			generatedFeatures = append(generatedFeatures, 
				fmt.Sprintf("%s_rolling_mean_3", col),
				fmt.Sprintf("%s_rolling_std_3", col),
				fmt.Sprintf("%s_diff", col),
			)
		}
	}
	
	// 5. Categorical encoding (if categorical columns exist)
	categoricalCols := g.getCategoricalColumns(data)
	for _, col := range categoricalCols {
		generatedFeatures = append(generatedFeatures, fmt.Sprintf("%s_encoded", col))
		generatedFeatures = append(generatedFeatures, fmt.Sprintf("%s_frequency", col))
	}

	return generatedFeatures
}

// getNumericColumns identifies numeric columns
func (g *FeatureGenerator) getNumericColumns(data []map[string]interface{}) []string {
	if len(data) == 0 {
		return nil
	}

	numericCols := []string{}
	for key, value := range data[0] {
		switch value.(type) {
		case int, int64, float64, float32:
			numericCols = append(numericCols, key)
		}
	}
	
	return numericCols
}

// getCategoricalColumns identifies categorical columns
func (g *FeatureGenerator) getCategoricalColumns(data []map[string]interface{}) []string {
	if len(data) == 0 {
		return nil
	}

	categoricalCols := []string{}
	for key, value := range data[0] {
		switch value.(type) {
		case string:
			categoricalCols = append(categoricalCols, key)
		}
	}
	
	return categoricalCols
}

// hasTimeColumn checks if data has time/date column
func (g *FeatureGenerator) hasTimeColumn(data []map[string]interface{}) bool {
	if len(data) == 0 {
		return false
	}

	for key := range data[0] {
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "time") || 
		   strings.Contains(lowerKey, "date") ||
		   strings.Contains(lowerKey, "timestamp") {
			return true
		}
	}
	
	return false
}

// SelectFeatures selects most important features
func (s *FeatureSelector) SelectFeatures(features []string, maxFeatures int) ([]string, map[string]float64) {
	// Simulate feature importance calculation
	importances := make(map[string]float64)
	
	for i, feature := range features {
		// Assign importance based on feature type (simplified)
		importance := 1.0 / float64(i+1) // Higher rank = higher importance
		
		// Boost importance for certain patterns
		if strings.Contains(feature, "squared") {
			importance *= 1.2
		}
		if strings.Contains(feature, "rolling") || strings.Contains(feature, "diff") {
			importance *= 1.3
		}
		if strings.Contains(feature, "_x_") {
			importance *= 1.1
		}
		
		importances[feature] = importance
	}
	
	// Sort by importance
	type featureImportance struct {
		Name       string
		Importance float64
	}
	
	sortedFeatures := []featureImportance{}
	for name, imp := range importances {
		sortedFeatures = append(sortedFeatures, featureImportance{name, imp})
	}
	
	sort.Slice(sortedFeatures, func(i, j int) bool {
		return sortedFeatures[i].Importance > sortedFeatures[j].Importance
	})
	
	// Select top features
	selectedCount := maxFeatures
	if selectedCount > len(sortedFeatures) {
		selectedCount = len(sortedFeatures)
	}
	
	selected := []string{}
	selectedImportances := make(map[string]float64)
	
	for i := 0; i < selectedCount; i++ {
		selected = append(selected, sortedFeatures[i].Name)
		selectedImportances[sortedFeatures[i].Name] = sortedFeatures[i].Importance
	}
	
	return selected, selectedImportances
}

// HTTPHandler provides HTTP endpoints
func (a *FeatureEngineeringAgent) HTTPHandler() http.Handler {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/v1/generate-features", a.handleGenerateFeatures)
	mux.HandleFunc("/v1/select-features", a.handleSelectFeatures)
	mux.HandleFunc("/health", a.handleHealth)
	
	return mux
}

func (a *FeatureEngineeringAgent) handleGenerateFeatures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req FeatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Generate features
	allFeatures := a.generator.GenerateFeatures(req.Data)
	
	// Select top features (default: 50)
	maxFeatures := 50
	selectedFeatures, importances := a.selector.SelectFeatures(allFeatures, maxFeatures)

	resp := FeatureResponse{
		Features:       selectedFeatures,
		Importances:    importances,
		TotalGenerated: len(allFeatures),
		Selected:       len(selectedFeatures),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *FeatureEngineeringAgent) handleSelectFeatures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Features    []string `json:"features"`
		MaxFeatures int      `json:"max_features"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	selected, importances := a.selector.SelectFeatures(req.Features, req.MaxFeatures)

	resp := FeatureResponse{
		Features:    selected,
		Importances: importances,
		Selected:    len(selected),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *FeatureEngineeringAgent) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Start starts the feature engineering agent
func (a *FeatureEngineeringAgent) Start(ctx context.Context, addr string) error {
	server := &http.Server{
		Addr:    addr,
		Handler: a.HTTPHandler(),
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	log.Printf("Feature Engineering Agent listening on %s", addr)
	return server.ListenAndServe()
}

