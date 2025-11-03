package foundry

import (
	"context"
	"fmt"
	"sync"

	tfv1 "github.com/NexusGPU/tensor-fusion/api/v1"
	"k8s.io/klog/v2"
)

// SubscriptionScanner scans multiple Azure subscriptions for available Foundry resources
type SubscriptionScanner struct {
	clients map[string]*FoundryClient
	logger  klog.Logger
	mu      sync.RWMutex
}

// NewSubscriptionScanner creates a new subscription scanner
func NewSubscriptionScanner() *SubscriptionScanner {
	return &SubscriptionScanner{
		clients: make(map[string]*FoundryClient),
		logger:  klog.NewKlogr().WithName("subscription-scanner"),
	}
}

// AddSubscription adds a subscription to scan
func (s *SubscriptionScanner) AddSubscription(subscriptionID, endpoint, apiKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[subscriptionID] = NewFoundryClient(endpoint, apiKey, subscriptionID)
	s.logger.Info("Added subscription to scanner", "subscriptionID", subscriptionID)
}

// RemoveSubscription removes a subscription from scanning
func (s *SubscriptionScanner) RemoveSubscription(subscriptionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, subscriptionID)
	s.logger.Info("Removed subscription from scanner", "subscriptionID", subscriptionID)
}

// ScanAllSubscriptions scans all registered subscriptions for available models
func (s *SubscriptionScanner) ScanAllSubscriptions(ctx context.Context) (map[string][]FoundryModel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make(map[string][]FoundryModel)
	errors := make([]error, 0)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for subID, client := range s.clients {
		wg.Add(1)
		go func(subscriptionID string, c *FoundryClient) {
			defer wg.Done()

			s.logger.V(2).Info("Scanning subscription", "subscriptionID", subscriptionID)
			
			models, err := c.ListAvailableModels(ctx)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("subscription %s: %w", subscriptionID, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			results[subscriptionID] = models
			mu.Unlock()

			s.logger.Info("Scanned subscription successfully", 
				"subscriptionID", subscriptionID, 
				"modelsFound", len(models))
		}(subID, client)
	}

	wg.Wait()

	if len(errors) > 0 {
		return results, fmt.Errorf("scan completed with %d errors: %v", len(errors), errors)
	}

	return results, nil
}

// GetAggregatedInventory returns aggregated GPU inventory across all subscriptions
func (s *SubscriptionScanner) GetAggregatedInventory(ctx context.Context) (*AggregatedInventory, error) {
	modelsBySubscription, err := s.ScanAllSubscriptions(ctx)
	if err != nil {
		s.logger.Error(err, "Failed to scan subscriptions")
		// Continue with partial results
	}

	inventory := &AggregatedInventory{
		TotalSubscriptions: len(s.clients),
		ModelsBySubscription: modelsBySubscription,
		UniqueModels: make(map[string]ModelAvailability),
	}

	// Aggregate unique models across subscriptions
	for subID, models := range modelsBySubscription {
		for _, model := range models {
			if existing, found := inventory.UniqueModels[model.Name]; found {
				// Model exists in multiple subscriptions
				existing.Subscriptions = append(existing.Subscriptions, subID)
				existing.TotalInstances++
				inventory.UniqueModels[model.Name] = existing
			} else {
				// First occurrence of this model
				inventory.UniqueModels[model.Name] = ModelAvailability{
					Model:          model,
					Subscriptions:  []string{subID},
					TotalInstances: 1,
				}
			}
		}
	}

	inventory.TotalUniqueModels = len(inventory.UniqueModels)
	inventory.TotalModelInstances = s.countTotalInstances(modelsBySubscription)

	return inventory, nil
}

// GetModelAvailability checks availability of a specific model across subscriptions
func (s *SubscriptionScanner) GetModelAvailability(ctx context.Context, modelName string) ([]ModelLocation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var locations []ModelLocation
	var wg sync.WaitGroup
	var mu sync.Mutex

	for subID, client := range s.clients {
		wg.Add(1)
		go func(subscriptionID string, c *FoundryClient) {
			defer wg.Done()

			available, err := c.CheckModelAvailability(ctx, modelName)
			if err != nil {
				s.logger.V(2).Info("Error checking model availability", 
					"model", modelName, 
					"subscription", subscriptionID, 
					"error", err)
				return
			}

			if available {
				metrics, _ := c.GetModelMetrics(ctx, modelName)
				
				mu.Lock()
				locations = append(locations, ModelLocation{
					SubscriptionID:  subscriptionID,
					ModelName:       modelName,
					Available:       true,
					Metrics:         metrics,
				})
				mu.Unlock()
			}
		}(subID, client)
	}

	wg.Wait()

	return locations, nil
}

// FindBestModelLocation finds the best location for a model based on criteria
func (s *SubscriptionScanner) FindBestModelLocation(ctx context.Context, modelName string, criteria SelectionCriteria) (*ModelLocation, error) {
	locations, err := s.GetModelAvailability(ctx, modelName)
	if err != nil {
		return nil, err
	}

	if len(locations) == 0 {
		return nil, fmt.Errorf("model %s not available in any subscription", modelName)
	}

	// Score and select best location
	bestLocation := &locations[0]
	bestScore := s.scoreLocation(&locations[0], criteria)

	for i := 1; i < len(locations); i++ {
		score := s.scoreLocation(&locations[i], criteria)
		if score > bestScore {
			bestScore = score
			bestLocation = &locations[i]
		}
	}

	return bestLocation, nil
}

// AggregatedInventory represents GPU inventory across all subscriptions
type AggregatedInventory struct {
	TotalSubscriptions    int
	TotalUniqueModels     int
	TotalModelInstances   int
	ModelsBySubscription  map[string][]FoundryModel
	UniqueModels          map[string]ModelAvailability
}

// ModelAvailability tracks where a model is available
type ModelAvailability struct {
	Model          FoundryModel
	Subscriptions  []string
	TotalInstances int
}

// ModelLocation represents a specific model instance in a subscription
type ModelLocation struct {
	SubscriptionID string
	ModelName      string
	Available      bool
	Metrics        *ModelMetrics
}

// SelectionCriteria defines criteria for selecting the best model location
type SelectionCriteria struct {
	PreferLowLatency      bool
	PreferLowUtilization  bool
	PreferSpecificRegion  string
	MaxAcceptableLatency  int32
}

// scoreLocation scores a model location based on criteria
func (s *SubscriptionScanner) scoreLocation(location *ModelLocation, criteria SelectionCriteria) float64 {
	if !location.Available || location.Metrics == nil {
		return 0
	}

	score := 100.0

	// Latency scoring
	if criteria.PreferLowLatency {
		latencyPenalty := float64(location.Metrics.AverageLatencyMs) / 10.0
		score -= latencyPenalty
	}

	// Utilization scoring
	if criteria.PreferLowUtilization {
		// Lower queue depth is better
		queuePenalty := float64(location.Metrics.QueueDepth) * 2.0
		score -= queuePenalty
	}

	// Error rate penalty
	errorPenalty := location.Metrics.ErrorRate * 50.0
	score -= errorPenalty

	// Max latency threshold
	if criteria.MaxAcceptableLatency > 0 && location.Metrics.AverageLatencyMs > float64(criteria.MaxAcceptableLatency) {
		score -= 50.0
	}

	return score
}

func (s *SubscriptionScanner) countTotalInstances(modelsBySubscription map[string][]FoundryModel) int {
	total := 0
	for _, models := range modelsBySubscription {
		total += len(models)
	}
	return total
}

// ConvertInventoryToAzureGPUModels converts aggregated inventory to AzureGPUModels
func ConvertInventoryToAzureGPUModels(inventory *AggregatedInventory) []tfv1.AzureGPUModel {
	models := make([]tfv1.AzureGPUModel, 0, len(inventory.UniqueModels))
	
	for _, availability := range inventory.UniqueModels {
		model := ConvertToAzureGPUModel(availability.Model)
		models = append(models, model)
	}
	
	return models
}


