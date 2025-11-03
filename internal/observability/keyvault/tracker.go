package keyvault

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// Tracker tracks API key usage across LLM requests
// Now integrates with Portkey for token tracking
type Tracker struct {
	usage          map[string]*KeyUsage
	mu             sync.RWMutex
	logger         klog.Logger
	portkeyEnabled bool
	portkeyURL     string
	portkeyAPIKey  string
	httpClient     *http.Client
}

// NewTracker creates a new API key usage tracker
func NewTracker() *Tracker {
	return NewTrackerWithPortkey("", "")
}

// NewTrackerWithPortkey creates a tracker with Portkey integration
func NewTrackerWithPortkey(portkeyURL, portkeyAPIKey string) *Tracker {
	enabled := portkeyURL != "" && portkeyAPIKey != ""
	
	return &Tracker{
		usage:          make(map[string]*KeyUsage),
		logger:         klog.NewKlogr().WithName("key-tracker"),
		portkeyEnabled: enabled,
		portkeyURL:     portkeyURL,
		portkeyAPIKey:  portkeyAPIKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// KeyUsage represents usage statistics for an API key
type KeyUsage struct {
	APIKey            string
	TenantID          string
	TotalRequests     int64
	SuccessfulRequests int64
	FailedRequests    int64
	TotalTokens       int64
	PromptTokens      int64
	CompletionTokens  int64
	TotalCost         float64
	FirstUsed         time.Time
	LastUsed          time.Time
	RequestsPerMinute int64
	DailyLimits       *Limits
	MonthlyLimits     *Limits
	CurrentDayUsage   *DayUsage
	CurrentMonthUsage *MonthUsage
}

// Limits represents quota limits
type Limits struct {
	MaxRequestsPerMinute int64
	MaxTokensPerDay      int64
	MaxCostPerMonth      float64
}

// DayUsage tracks daily usage
type DayUsage struct {
	Date     string
	Requests int64
	Tokens   int64
	Cost     float64
}

// MonthUsage tracks monthly usage
type MonthUsage struct {
	Month    string
	Requests int64
	Tokens   int64
	Cost     float64
}

// TrackRequest records an API request
func (t *Tracker) TrackRequest(ctx context.Context, req *RequestInfo) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	usage, exists := t.usage[req.APIKey]
	if !exists {
		usage = &KeyUsage{
			APIKey:   req.APIKey,
			TenantID: req.TenantID,
			FirstUsed: time.Now(),
			DailyLimits: &Limits{
				MaxRequestsPerMinute: 1000,
				MaxTokensPerDay:      1000000,
				MaxCostPerMonth:      1000.0,
			},
			CurrentDayUsage: &DayUsage{
				Date: time.Now().Format("2006-01-02"),
			},
			CurrentMonthUsage: &MonthUsage{
				Month: time.Now().Format("2006-01"),
			},
		}
		t.usage[req.APIKey] = usage
	}

	// Update usage statistics
	usage.TotalRequests++
	if req.Success {
		usage.SuccessfulRequests++
	} else {
		usage.FailedRequests++
	}

	usage.TotalTokens += req.TokensUsed
	usage.PromptTokens += req.PromptTokens
	usage.CompletionTokens += req.CompletionTokens
	usage.TotalCost += req.Cost
	usage.LastUsed = time.Now()

	// Update daily usage
	currentDay := time.Now().Format("2006-01-02")
	if usage.CurrentDayUsage.Date != currentDay {
		// New day, reset counters
		usage.CurrentDayUsage = &DayUsage{
			Date: currentDay,
		}
	}
	usage.CurrentDayUsage.Requests++
	usage.CurrentDayUsage.Tokens += req.TokensUsed
	usage.CurrentDayUsage.Cost += req.Cost

	// Update monthly usage
	currentMonth := time.Now().Format("2006-01")
	if usage.CurrentMonthUsage.Month != currentMonth {
		// New month, reset counters
		usage.CurrentMonthUsage = &MonthUsage{
			Month: currentMonth,
		}
	}
	usage.CurrentMonthUsage.Requests++
	usage.CurrentMonthUsage.Tokens += req.TokensUsed
	usage.CurrentMonthUsage.Cost += req.Cost

	return nil
}

// RequestInfo contains information about an API request
type RequestInfo struct {
	APIKey           string
	TenantID         string
	Model            string
	TokensUsed       int64
	PromptTokens     int64
	CompletionTokens int64
	Cost             float64
	Success          bool
	Latency          int64 // milliseconds
	Timestamp        time.Time
}

// GetUsage retrieves usage statistics for an API key
func (t *Tracker) GetUsage(apiKey string) (*KeyUsage, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	usage, exists := t.usage[apiKey]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	usageCopy := *usage
	return &usageCopy, true
}

// CheckQuota checks if an API key has exceeded its quota
func (t *Tracker) CheckQuota(apiKey string) (bool, string) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	usage, exists := t.usage[apiKey]
	if !exists {
		return true, "" // No usage yet, allow
	}

	if usage.DailyLimits == nil {
		return true, "" // No limits set
	}

	// Check daily token limit
	if usage.DailyLimits.MaxTokensPerDay > 0 && 
		usage.CurrentDayUsage.Tokens >= usage.DailyLimits.MaxTokensPerDay {
		return false, "daily token limit exceeded"
	}

	// Check monthly cost limit
	if usage.DailyLimits.MaxCostPerMonth > 0 && 
		usage.CurrentMonthUsage.Cost >= usage.DailyLimits.MaxCostPerMonth {
		return false, "monthly cost limit exceeded"
	}

	// Check rate limit (requests per minute)
	// This would require more sophisticated time-window tracking
	// Simplified implementation here

	return true, ""
}

// GetAllUsage returns usage for all tracked keys
func (t *Tracker) GetAllUsage() map[string]*KeyUsage {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]*KeyUsage, len(t.usage))
	for k, v := range t.usage {
		usageCopy := *v
		result[k] = &usageCopy
	}

	return result
}

// ResetDailyCounters resets daily usage counters (called at midnight)
func (t *Tracker) ResetDailyCounters() {
	t.mu.Lock()
	defer t.mu.Unlock()

	currentDay := time.Now().Format("2006-01-02")
	for _, usage := range t.usage {
		if usage.CurrentDayUsage.Date != currentDay {
			usage.CurrentDayUsage = &DayUsage{
				Date: currentDay,
			}
		}
	}
}

// ResetMonthlyCounters resets monthly usage counters (called at month start)
func (t *Tracker) ResetMonthlyCounters() {
	t.mu.Lock()
	defer t.mu.Unlock()

	currentMonth := time.Now().Format("2006-01")
	for _, usage := range t.usage {
		if usage.CurrentMonthUsage.Month != currentMonth {
			usage.CurrentMonthUsage = &MonthUsage{
				Month: currentMonth,
			}
		}
	}
}

// SetLimits sets quota limits for an API key
func (t *Tracker) SetLimits(apiKey string, limits *Limits) {
	t.mu.Lock()
	defer t.mu.Unlock()

	usage, exists := t.usage[apiKey]
	if !exists {
		usage = &KeyUsage{
			APIKey:    apiKey,
			FirstUsed: time.Now(),
		}
		t.usage[apiKey] = usage
	}

	usage.DailyLimits = limits
}

// StartPeriodicCleanup starts a goroutine to reset counters periodically
func (t *Tracker) StartPeriodicCleanup(ctx context.Context) {
	go func() {
		dailyTicker := time.NewTicker(1 * time.Hour)
		monthlyTicker := time.NewTicker(24 * time.Hour)
		syncTicker := time.NewTicker(5 * time.Minute) // Sync with Portkey every 5 minutes
		defer dailyTicker.Stop()
		defer monthlyTicker.Stop()
		defer syncTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-dailyTicker.C:
				t.ResetDailyCounters()
			case <-monthlyTicker.C:
				t.ResetMonthlyCounters()
			case <-syncTicker.C:
				if t.portkeyEnabled {
					t.SyncFromPortkey(ctx)
				}
			}
		}
	}()
}

// PortkeyAnalytics represents Portkey analytics response
type PortkeyAnalytics struct {
	APIKey           string  `json:"api_key"`
	TotalRequests    int64   `json:"total_requests"`
	TotalTokens      int64   `json:"total_tokens"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalCost        float64 `json:"total_cost"`
	SuccessRate      float64 `json:"success_rate"`
}

// SyncFromPortkey syncs token usage data from Portkey
func (t *Tracker) SyncFromPortkey(ctx context.Context) error {
	if !t.portkeyEnabled {
		return fmt.Errorf("portkey integration not enabled")
	}

	// Query Portkey analytics API
	url := fmt.Sprintf("%s/v1/analytics", t.portkeyURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		t.logger.Error(err, "Failed to create Portkey request")
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.portkeyAPIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		t.logger.Error(err, "Failed to query Portkey analytics")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.logger.Error(fmt.Errorf("portkey returned %d", resp.StatusCode), "Portkey API error")
		return fmt.Errorf("portkey API returned status %d", resp.StatusCode)
	}

	var analytics []PortkeyAnalytics
	if err := json.NewDecoder(resp.Body).Decode(&analytics); err != nil {
		t.logger.Error(err, "Failed to decode Portkey response")
		return err
	}

	// Update local cache with Portkey data
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, a := range analytics {
		usage, exists := t.usage[a.APIKey]
		if !exists {
			usage = &KeyUsage{
				APIKey:    a.APIKey,
				FirstUsed: time.Now(),
				DailyLimits: &Limits{
					MaxRequestsPerMinute: 1000,
					MaxTokensPerDay:      1000000,
					MaxCostPerMonth:      1000.0,
				},
				CurrentDayUsage: &DayUsage{
					Date: time.Now().Format("2006-01-02"),
				},
				CurrentMonthUsage: &MonthUsage{
					Month: time.Now().Format("2006-01"),
				},
			}
			t.usage[a.APIKey] = usage
		}

		// Update with Portkey data (Portkey is source of truth)
		usage.TotalRequests = a.TotalRequests
		usage.TotalTokens = a.TotalTokens
		usage.PromptTokens = a.PromptTokens
		usage.CompletionTokens = a.CompletionTokens
		usage.TotalCost = a.TotalCost
		usage.SuccessfulRequests = int64(float64(a.TotalRequests) * a.SuccessRate)
		usage.FailedRequests = a.TotalRequests - usage.SuccessfulRequests
		usage.LastUsed = time.Now()
	}

	t.logger.Info("Synced token usage from Portkey", "keys", len(analytics))
	return nil
}
