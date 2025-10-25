package keyvault

import (
	"context"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// Tracker tracks API key usage across LLM requests
type Tracker struct {
	usage  map[string]*KeyUsage
	mu     sync.RWMutex
	logger klog.Logger
}

// NewTracker creates a new API key usage tracker
func NewTracker() *Tracker {
	return &Tracker{
		usage:  make(map[string]*KeyUsage),
		logger: klog.NewKlogr().WithName("key-tracker"),
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
		defer dailyTicker.Stop()
		defer monthlyTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-dailyTicker.C:
				t.ResetDailyCounters()
			case <-monthlyTicker.C:
				t.ResetMonthlyCounters()
			}
		}
	}()
}


