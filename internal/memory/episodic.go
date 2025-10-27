package memory

import (
	"context"
	"fmt"
	"time"
)

// EpisodicMemory provides time-series based episodic memory storage
type EpisodicMemory struct {
	greptimeURL string
}

// EpisodicMemoryEntry represents an episodic memory entry (event)
type EpisodicMemoryEntry struct {
	ID        string                 `json:"id"`
	Event     string                 `json:"event"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// NewEpisodicMemory creates a new episodic memory backend
func NewEpisodicMemory(greptimeURL string) *EpisodicMemory {
	return &EpisodicMemory{
		greptimeURL: greptimeURL,
	}
}

// Add adds an episodic memory event
func (em *EpisodicMemory) Add(ctx context.Context, agentID string, entry *EpisodicMemoryEntry) error {
	// In production, this would:
	// 1. Store event in GreptimeDB time-series table
	// 2. Table structure: agent_id, event, timestamp, metadata

	entry.ID = fmt.Sprintf("epi-%d", time.Now().UnixNano())
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Placeholder: In real implementation, insert into GreptimeDB
	return nil
}

// Query queries episodic memory within a time range
func (em *EpisodicMemory) Query(ctx context.Context, agentID string, startTime, endTime time.Time) ([]*EpisodicMemoryEntry, error) {
	// In production, this would:
	// 1. Query GreptimeDB for events in time range
	// 2. Return chronologically ordered events

	// For now, return empty results
	return []*EpisodicMemoryEntry{}, nil
}

// GetTimeline retrieves a chronological timeline of events
func (em *EpisodicMemory) GetTimeline(ctx context.Context, agentID string, limit int) ([]*EpisodicMemoryEntry, error) {
	// In production, query last N events ordered by timestamp DESC
	return []*EpisodicMemoryEntry{}, nil
}

