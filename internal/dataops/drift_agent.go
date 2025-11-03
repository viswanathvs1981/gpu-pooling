package dataops

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"
)

// DriftDetectionAgent monitors model drift
type DriftDetectionAgent struct {
	detector     *DriftDetector
	retrainer    *AutoRetrainer
}

// DriftDetector detects various types of drift
type DriftDetector struct{}

// AutoRetrainer handles automatic retraining
type AutoRetrainer struct{}

// DriftCheckRequest represents drift check request
type DriftCheckRequest struct {
	ModelName       string      `json:"model_name"`
	ReferenceData   []float64   `json:"reference_data"`
	CurrentData     []float64   `json:"current_data"`
	Threshold       float64     `json:"threshold,omitempty"` // Default: 0.05
}

// DriftCheckResponse represents drift detection result
type DriftCheckResponse struct {
	DriftDetected    bool     `json:"drift_detected"`
	DriftScore       float64  `json:"drift_score"`
	DriftType        string   `json:"drift_type"` // input, prediction, concept
	RootCause        string   `json:"root_cause"`
	Recommendation   string   `json:"recommendation"`
	AutoRetrainTriggered bool `json:"auto_retrain_triggered"`
}

// NewDriftDetectionAgent creates a new drift detection agent
func NewDriftDetectionAgent() *DriftDetectionAgent {
	return &DriftDetectionAgent{
		detector:  &DriftDetector{},
		retrainer: &AutoRetrainer{},
	}
}

// CheckDrift checks for data/model drift using KS test
func (d *DriftDetector) CheckDrift(reference, current []float64, threshold float64) (bool, float64, string) {
	if len(reference) == 0 || len(current) == 0 {
		return false, 0.0, "insufficient data"
	}

	// Calculate Population Stability Index (PSI)
	psi := d.calculatePSI(reference, current)
	
	// Interpret PSI
	var driftType string
	if psi < 0.1 {
		driftType = "no drift"
	} else if psi < 0.2 {
		driftType = "minor drift"
	} else {
		driftType = "significant drift"
	}
	
	driftDetected := psi > threshold
	
	return driftDetected, psi, driftType
}

// calculatePSI calculates Population Stability Index
func (d *DriftDetector) calculatePSI(reference, current []float64) float64 {
	// Create bins
	numBins := 10
	minVal := math.Min(d.min(reference), d.min(current))
	maxVal := math.Max(d.max(reference), d.max(current))
	binWidth := (maxVal - minVal) / float64(numBins)
	
	if binWidth == 0 {
		return 0.0
	}
	
	// Calculate distributions
	refDist := d.binData(reference, minVal, binWidth, numBins)
	currDist := d.binData(current, minVal, binWidth, numBins)
	
	// Calculate PSI
	psi := 0.0
	for i := 0; i < numBins; i++ {
		refPct := refDist[i] / float64(len(reference))
		currPct := currDist[i] / float64(len(current))
		
		// Avoid log(0)
		if refPct == 0 || currPct == 0 {
			continue
		}
		
		psi += (currPct - refPct) * math.Log(currPct/refPct)
	}
	
	return psi
}

// binData bins data into histogram
func (d *DriftDetector) binData(data []float64, minVal, binWidth float64, numBins int) []float64 {
	bins := make([]float64, numBins)
	
	for _, value := range data {
		binIdx := int((value - minVal) / binWidth)
		if binIdx >= numBins {
			binIdx = numBins - 1
		}
		if binIdx < 0 {
			binIdx = 0
		}
		bins[binIdx]++
	}
	
	return bins
}

// min returns minimum value in slice
func (d *DriftDetector) min(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	
	min := data[0]
	for _, v := range data[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// max returns maximum value in slice
func (d *DriftDetector) max(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	
	max := data[0]
	for _, v := range data[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// AnalyzeRootCause provides root cause analysis
func (d *DriftDetector) AnalyzeRootCause(driftScore float64, driftType string) string {
	if driftScore < 0.1 {
		return "No significant distribution shift detected"
	} else if driftScore < 0.2 {
		return "Minor distribution shift: data patterns slightly changed, possibly due to seasonal variations"
	} else {
		return "Significant distribution shift: major change in data patterns, likely due to market changes, new user behavior, or data quality issues"
	}
}

// RecommendAction recommends action based on drift
func (d *DriftDetector) RecommendAction(driftScore float64) string {
	if driftScore < 0.1 {
		return "Continue monitoring. No action needed."
	} else if driftScore < 0.2 {
		return "Increase monitoring frequency. Consider retraining in 7 days if drift persists."
	} else {
		return "Immediate retraining recommended. Deploy new model with A/B testing (5% traffic initially)."
	}
}

// TriggerRetraining triggers automatic model retraining
func (r *AutoRetrainer) TriggerRetraining(modelName string, driftScore float64) bool {
	// Only auto-retrain if drift is significant (>0.2)
	if driftScore < 0.2 {
		return false
	}
	
	log.Printf("Auto-retraining triggered for model '%s' (drift score: %.3f)", modelName, driftScore)
	
	// In production, this would:
	// 1. Fetch latest training data
	// 2. Trigger training job via Training Agent
	// 3. Deploy new model with canary deployment
	// 4. Monitor performance
	// 5. Rollback if new model underperforms
	
	// For now, just log
	log.Printf("Retraining job for '%s' queued", modelName)
	
	return true
}

// HTTPHandler provides HTTP endpoints
func (a *DriftDetectionAgent) HTTPHandler() http.Handler {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/v1/check-drift", a.handleCheckDrift)
	mux.HandleFunc("/health", a.handleHealth)
	
	return mux
}

func (a *DriftDetectionAgent) handleCheckDrift(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DriftCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Default threshold
	if req.Threshold == 0 {
		req.Threshold = 0.05
	}

	// Check drift
	driftDetected, driftScore, driftType := a.detector.CheckDrift(
		req.ReferenceData, req.CurrentData, req.Threshold)
	
	// Analyze root cause
	rootCause := a.detector.AnalyzeRootCause(driftScore, driftType)
	
	// Get recommendation
	recommendation := a.detector.RecommendAction(driftScore)
	
	// Trigger auto-retrain if needed
	autoRetrainTriggered := false
	if driftDetected {
		autoRetrainTriggered = a.retrainer.TriggerRetraining(req.ModelName, driftScore)
	}

	resp := DriftCheckResponse{
		DriftDetected:        driftDetected,
		DriftScore:           driftScore,
		DriftType:            driftType,
		RootCause:            rootCause,
		Recommendation:       recommendation,
		AutoRetrainTriggered: autoRetrainTriggered,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *DriftDetectionAgent) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Start starts the drift detection agent
func (a *DriftDetectionAgent) Start(ctx context.Context, addr string) error {
	server := &http.Server{
		Addr:    addr,
		Handler: a.HTTPHandler(),
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	log.Printf("Drift Detection Agent listening on %s", addr)
	return server.ListenAndServe()
}

