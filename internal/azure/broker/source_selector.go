package broker

import (
	"context"
	"fmt"
	"sort"

	tfv1 "github.com/NexusGPU/tensor-fusion/api/v1"
	"github.com/NexusGPU/tensor-fusion/internal/azure/foundry"
	"github.com/NexusGPU/tensor-fusion/internal/azure/mcp"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
)

// SourceSelector selects the best GPU source (AKS or Foundry) for a workload
type SourceSelector struct {
	aksProvisioner     *mcp.AKSProvisioner
	foundryScanner     *foundry.SubscriptionScanner
	logger             klog.Logger
}

// NewSourceSelector creates a new source selector
func NewSourceSelector(aksProvisioner *mcp.AKSProvisioner, foundryScanner *foundry.SubscriptionScanner) *SourceSelector {
	return &SourceSelector{
		aksProvisioner: aksProvisioner,
		foundryScanner: foundryScanner,
		logger:         klog.NewKlogr().WithName("source-selector"),
	}
}

// SelectionRequest represents a request to select a GPU source
type SelectionRequest struct {
	WorkloadName      string
	RequiredTFlops    resource.Quantity
	RequiredVRAM      resource.Quantity
	GPUModel          string
	QoS               tfv1.QoSLevel
	MaxCostPerHour    float64
	MaxLatencyMs      int32
	PreferredRegion   string
	AllowBurst        bool
}

// SelectionResult represents the selected GPU source
type SelectionResult struct {
	SourceType        SourceType
	SourceName        string
	SubscriptionID    string
	Region            string
	GPUModel          string
	EstimatedCost     float64
	EstimatedLatency  int32
	Confidence        float64
	Alternatives      []AlternativeSource
	Reason            string
}

// SourceType indicates the type of GPU source
type SourceType string

const (
	SourceTypeAKS     SourceType = "aks"
	SourceTypeFoundry SourceType = "foundry"
)

// AlternativeSource represents an alternative GPU source option
type AlternativeSource struct {
	SourceType       SourceType
	SourceName       string
	EstimatedCost    float64
	EstimatedLatency int32
	Reason           string
}

// SelectBestSource selects the best GPU source for a workload
func (s *SourceSelector) SelectBestSource(ctx context.Context, req *SelectionRequest) (*SelectionResult, error) {
	s.logger.Info("Selecting GPU source", 
		"workload", req.WorkloadName,
		"tflops", req.RequiredTFlops.String(),
		"vram", req.RequiredVRAM.String())

	// Evaluate all available sources
	aksSources, err := s.evaluateAKSSources(ctx, req)
	if err != nil {
		s.logger.V(2).Info("Error evaluating AKS sources", "error", err)
	}

	foundrySources, err := s.evaluateFoundrySources(ctx, req)
	if err != nil {
		s.logger.V(2).Info("Error evaluating Foundry sources", "error", err)
	}

	// Combine and rank all sources
	allSources := append(aksSources, foundrySources...)
	if len(allSources) == 0 {
		return nil, fmt.Errorf("no suitable GPU sources found for workload %s", req.WorkloadName)
	}

	// Sort by score (highest first)
	sort.Slice(allSources, func(i, j int) bool {
		return allSources[i].Confidence > allSources[j].Confidence
	})

	// Select best source
	best := allSources[0]
	
	// Populate alternatives
	alternatives := make([]AlternativeSource, 0, len(allSources)-1)
	for i := 1; i < len(allSources) && i < 4; i++ {
		alternatives = append(alternatives, AlternativeSource{
			SourceType:       allSources[i].SourceType,
			SourceName:       allSources[i].SourceName,
			EstimatedCost:    allSources[i].EstimatedCost,
			EstimatedLatency: allSources[i].EstimatedLatency,
			Reason:           allSources[i].Reason,
		})
	}

	best.Alternatives = alternatives

	s.logger.Info("Selected GPU source",
		"sourceType", best.SourceType,
		"sourceName", best.SourceName,
		"cost", best.EstimatedCost,
		"latency", best.EstimatedLatency,
		"confidence", best.Confidence)

	return &best, nil
}

// evaluateAKSSources evaluates available AKS GPU sources
func (s *SourceSelector) evaluateAKSSources(ctx context.Context, req *SelectionRequest) ([]SelectionResult, error) {
	if s.aksProvisioner == nil {
		return nil, fmt.Errorf("AKS provisioner not configured")
	}

	// Get available VM sizes in the region
	region := req.PreferredRegion
	if region == "" {
		region = "eastus" // Default region
	}

	vmSizes, err := s.aksProvisioner.ListAvailableVMSizes(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to list AKS VM sizes: %w", err)
	}

	results := make([]SelectionResult, 0)
	
	for _, vmSize := range vmSizes {
		// Check if VM size meets requirements
		if !s.meetsRequirements(vmSize.EstimatedTFlops, vmSize.EstimatedVRAM, req) {
			continue
		}

		// Estimate cost (hypothetical pricing)
		costPerHour := s.estimateAKSCost(vmSize.Name, vmSize.NumberOfGPUs)
		
		// Check cost constraint
		if req.MaxCostPerHour > 0 && costPerHour > req.MaxCostPerHour {
			continue
		}

		// Calculate confidence score
		confidence := s.calculateAKSConfidence(vmSize, req, costPerHour)

		result := SelectionResult{
			SourceType:       SourceTypeAKS,
			SourceName:       vmSize.Name,
			Region:           region,
			GPUModel:         vmSize.GPUType,
			EstimatedCost:    costPerHour,
			EstimatedLatency: 50, // AKS local latency typically ~50ms
			Confidence:       confidence,
			Reason:           fmt.Sprintf("AKS %s with %d x %s GPUs", vmSize.Name, vmSize.NumberOfGPUs, vmSize.GPUType),
		}

		results = append(results, result)
	}

	return results, nil
}

// evaluateFoundrySources evaluates available Azure Foundry sources
func (s *SourceSelector) evaluateFoundrySources(ctx context.Context, req *SelectionRequest) ([]SelectionResult, error) {
	if s.foundryScanner == nil {
		return nil, fmt.Errorf("Foundry scanner not configured")
	}

	inventory, err := s.foundryScanner.GetAggregatedInventory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Foundry inventory: %w", err)
	}

	results := make([]SelectionResult, 0)

	for modelName, availability := range inventory.UniqueModels {
		model := availability.Model

		// Check if model meets requirements
		if !s.meetsRequirements(model.EstimatedTFlops, model.EstimatedVRAM, req) {
			continue
		}

		// Estimate cost per hour
		costPerHour := (model.InputCostPer1K + model.OutputCostPer1K) * 500 // Assume 500K tokens/hr

		// Check cost constraint
		if req.MaxCostPerHour > 0 && costPerHour > req.MaxCostPerHour {
			continue
		}

		// Calculate confidence score
		confidence := s.calculateFoundryConfidence(model, req, costPerHour)

		// Get best location for this model
		locations, _ := s.foundryScanner.GetModelAvailability(ctx, modelName)
		subscriptionID := ""
		if len(locations) > 0 {
			subscriptionID = locations[0].SubscriptionID
		}

		result := SelectionResult{
			SourceType:       SourceTypeFoundry,
			SourceName:       modelName,
			SubscriptionID:   subscriptionID,
			Region:           model.Region,
			GPUModel:         model.ModelFamily,
			EstimatedCost:    costPerHour,
			EstimatedLatency: model.AverageLatencyMs,
			Confidence:       confidence,
			Reason:           fmt.Sprintf("Foundry %s model", modelName),
		}

		results = append(results, result)
	}

	return results, nil
}

// meetsRequirements checks if GPU specs meet workload requirements
func (s *SourceSelector) meetsRequirements(tflops, vram resource.Quantity, req *SelectionRequest) bool {
	// Check TFlops
	if tflops.Cmp(req.RequiredTFlops) < 0 {
		return false
	}

	// Check VRAM
	if vram.Cmp(req.RequiredVRAM) < 0 {
		return false
	}

	return true
}

// calculateAKSConfidence calculates confidence score for AKS source
func (s *SourceSelector) calculateAKSConfidence(vmSize mcp.AzureVMSize, req *SelectionRequest, cost float64) float64 {
	score := 100.0

	// Cost efficiency (lower cost = higher score)
	if req.MaxCostPerHour > 0 {
		costRatio := cost / req.MaxCostPerHour
		score -= (costRatio - 1.0) * 20.0
	}

	// Latency bonus (AKS is typically lower latency)
	score += 15.0

	// QoS bonuses
	if req.QoS == tfv1.QoSHigh || req.QoS == tfv1.QoSCritical {
		score += 10.0 // AKS preferred for high QoS
	}

	// Over-provisioning penalty (if GPU is much more powerful than needed)
	tflopsRatio := float64(vmSize.EstimatedTFlops.Value()) / float64(req.RequiredTFlops.Value())
	if tflopsRatio > 2.0 {
		score -= (tflopsRatio - 2.0) * 5.0
	}

	// Normalize score to 0-1 range
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score / 100.0
}

// calculateFoundryConfidence calculates confidence score for Foundry source
func (s *SourceSelector) calculateFoundryConfidence(model foundry.FoundryModel, req *SelectionRequest, cost float64) float64 {
	score := 100.0

	// Cost efficiency
	if req.MaxCostPerHour > 0 {
		costRatio := cost / req.MaxCostPerHour
		score -= (costRatio - 1.0) * 20.0
	}

	// Latency penalty (Foundry typically has higher latency)
	if model.AverageLatencyMs > 100 {
		latencyPenalty := float64(model.AverageLatencyMs-100) / 10.0
		score -= latencyPenalty
	}

	// Serverless bonus (no setup time)
	score += 10.0

	// Burst workload bonus
	if req.AllowBurst {
		score += 15.0 // Foundry is great for burst workloads
	}

	// Low QoS bonus (Foundry is cost-effective for low QoS)
	if req.QoS == tfv1.QoSLow || req.QoS == tfv1.QoSMedium {
		score += 10.0
	}

	// Normalize score to 0-1 range
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score / 100.0
}

// estimateAKSCost estimates hourly cost for AKS VM size
func (s *SourceSelector) estimateAKSCost(vmSize string, gpuCount int32) float64 {
	// Rough estimates based on Azure pricing (as of 2024)
	baseCost := 0.5

	switch {
	case contains(vmSize, "A100"):
		baseCost = 3.0 * float64(gpuCount)
	case contains(vmSize, "V100"):
		baseCost = 2.5 * float64(gpuCount)
	case contains(vmSize, "T4"):
		baseCost = 0.9 * float64(gpuCount)
	case contains(vmSize, "H100"):
		baseCost = 5.0 * float64(gpuCount)
	default:
		baseCost = 1.5 * float64(gpuCount)
	}

	return baseCost
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}


