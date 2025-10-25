package ml

import (
	"context"
	"fmt"

	tfv1 "github.com/NexusGPU/tensor-fusion/api/v1"
	"github.com/NexusGPU/tensor-fusion/internal/intelligence/vectordb"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
)

// WorkloadProfiler profiles workloads and extracts features for ML
type WorkloadProfiler struct {
	embedder   *vectordb.Embedder
	vectorDB   vectordb.Client
	logger     klog.Logger
}

// NewWorkloadProfiler creates a new workload profiler
func NewWorkloadProfiler(embedder *vectordb.Embedder, vectorDB vectordb.Client) *WorkloadProfiler {
	return &WorkloadProfiler{
		embedder: embedder,
		vectorDB: vectorDB,
		logger:   klog.NewKlogr().WithName("workload-profiler"),
	}
}

// WorkloadProfile represents a complete workload profile
type WorkloadProfile struct {
	ID               string
	Features         *vectordb.WorkloadFeatures
	Embedding        []float32
	ActualResources  *ResourceUsage
	Performance      *PerformanceMetrics
	GPUAssignment    string
	Cost             float64
}

// ResourceUsage represents actual resource usage
type ResourceUsage struct {
	CPUUsed       float64
	MemoryUsed    float64
	GPUUtilization float64
	VRAMUsed      float64
	Duration      int64 // seconds
}

// PerformanceMetrics represents performance metrics
type PerformanceMetrics struct {
	Latency       int32   // milliseconds
	Throughput    float64 // items/second
	ErrorRate     float64 // percentage
	SLACompliance float64 // percentage
}

// ProfileWorkload profiles a workload and stores it
func (wp *WorkloadProfiler) ProfileWorkload(ctx context.Context, pod *v1.Pod, actual *ResourceUsage, perf *PerformanceMetrics) error {
	wp.logger.Info("Profiling workload", "pod", pod.Name)
	
	// Extract features
	features := wp.embedder.ExtractFeaturesFromPod(pod)
	
	// Generate embedding
	embedding, err := wp.embedder.GenerateEmbedding(ctx, features)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}
	
	// Create profile
	profile := &WorkloadProfile{
		ID:              fmt.Sprintf("%s-%s", pod.Namespace, pod.Name),
		Features:        features,
		Embedding:       embedding,
		ActualResources: actual,
		Performance:     perf,
	}
	
	// Store in vector DB
	metadata := wp.buildMetadata(features, actual, perf)
	vector := vectordb.Vector{
		ID:       profile.ID,
		Values:   embedding,
		Metadata: metadata,
	}
	
	if err := wp.vectorDB.Upsert(ctx, []vectordb.Vector{vector}); err != nil {
		return fmt.Errorf("failed to store profile: %w", err)
	}
	
	wp.logger.Info("Stored workload profile", "id", profile.ID)
	return nil
}

// FindSimilarWorkloads finds similar workloads based on features
func (wp *WorkloadProfiler) FindSimilarWorkloads(ctx context.Context, pod *v1.Pod, topK int) ([]SimilarWorkload, error) {
	wp.logger.V(2).Info("Finding similar workloads", "pod", pod.Name, "topK", topK)
	
	// Extract features and generate embedding for query
	features := wp.embedder.ExtractFeaturesFromPod(pod)
	embedding, err := wp.embedder.GenerateEmbedding(ctx, features)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}
	
	// Search vector DB
	results, err := wp.vectorDB.Search(ctx, embedding, topK, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector DB: %w", err)
	}
	
	// Convert results
	similar := make([]SimilarWorkload, 0, len(results))
	for _, result := range results {
		similar = append(similar, SimilarWorkload{
			ID:             result.ID,
			Similarity:     result.Score,
			Framework:      getMetadataString(result.Metadata, "framework"),
			WorkloadType:   getMetadataString(result.Metadata, "workload_type"),
			GPUUsed:        getMetadataString(result.Metadata, "gpu_used"),
			ActualTFlops:   getMetadataFloat(result.Metadata, "actual_tflops"),
			ActualVRAM:     getMetadataFloat(result.Metadata, "actual_vram"),
			Cost:           getMetadataFloat(result.Metadata, "cost"),
			AvgLatency:     getMetadataFloat(result.Metadata, "avg_latency"),
		})
	}
	
	return similar, nil
}

// PredictRequirements predicts resource requirements based on similar workloads
func (wp *WorkloadProfiler) PredictRequirements(ctx context.Context, pod *v1.Pod) (*PredictedRequirements, error) {
	// Find similar workloads
	similar, err := wp.FindSimilarWorkloads(ctx, pod, 10)
	if err != nil {
		return nil, err
	}
	
	if len(similar) == 0 {
		return wp.getDefaultPrediction(), nil
	}
	
	// Aggregate predictions from similar workloads (weighted by similarity)
	var totalWeight float64
	var weightedTFlops, weightedVRAM, weightedCost, weightedLatency float64
	
	for _, sim := range similar {
		weight := sim.Similarity
		totalWeight += weight
		weightedTFlops += sim.ActualTFlops * weight
		weightedVRAM += sim.ActualVRAM * weight
		weightedCost += sim.Cost * weight
		weightedLatency += sim.AvgLatency * weight
	}
	
	if totalWeight == 0 {
		return wp.getDefaultPrediction(), nil
	}
	
	prediction := &PredictedRequirements{
		TFlops:     resource.NewQuantity(int64(weightedTFlops/totalWeight*1000), resource.DecimalSI),
		VRAM:       resource.NewQuantity(int64(weightedVRAM/totalWeight*1024*1024*1024), resource.BinarySI),
		Cost:       weightedCost / totalWeight,
		Latency:    int32(weightedLatency / totalWeight),
		Confidence: wp.calculateConfidence(similar),
		GPUModel:   wp.recommendGPUModel(similar),
		Source:     "similarity-based",
	}
	
	return prediction, nil
}

// SimilarWorkload represents a similar workload
type SimilarWorkload struct {
	ID           string
	Similarity   float64
	Framework    string
	WorkloadType string
	GPUUsed      string
	ActualTFlops float64
	ActualVRAM   float64
	Cost         float64
	AvgLatency   float64
}

// PredictedRequirements represents predicted resource requirements
type PredictedRequirements struct {
	TFlops     *resource.Quantity
	VRAM       *resource.Quantity
	Cost       float64
	Latency    int32
	Confidence float64
	GPUModel   string
	Source     string
}

func (wp *WorkloadProfiler) buildMetadata(features *vectordb.WorkloadFeatures, actual *ResourceUsage, perf *PerformanceMetrics) map[string]interface{} {
	metadata := make(map[string]interface{})
	
	metadata["framework"] = features.Framework
	metadata["workload_type"] = features.WorkloadType
	metadata["model_family"] = features.ModelFamily
	metadata["image"] = features.Image
	
	if actual != nil {
		metadata["cpu_used"] = actual.CPUUsed
		metadata["memory_used"] = actual.MemoryUsed
		metadata["gpu_utilization"] = actual.GPUUtilization
		metadata["vram_used"] = actual.VRAMUsed
		metadata["duration"] = actual.Duration
		
		// Approximate TFlops used (this would be measured in production)
		metadata["actual_tflops"] = actual.GPUUtilization * 10.0 // Placeholder
		metadata["actual_vram"] = actual.VRAMUsed
	}
	
	if perf != nil {
		metadata["avg_latency"] = float64(perf.Latency)
		metadata["throughput"] = perf.Throughput
		metadata["error_rate"] = perf.ErrorRate
		metadata["sla_compliance"] = perf.SLACompliance
	}
	
	return metadata
}

func (wp *WorkloadProfiler) calculateConfidence(similar []SimilarWorkload) float64 {
	if len(similar) == 0 {
		return 0.0
	}
	
	// Confidence based on:
	// 1. Number of similar workloads found
	// 2. Average similarity score
	// 3. Variance in resource usage
	
	countFactor := float64(len(similar)) / 10.0
	if countFactor > 1.0 {
		countFactor = 1.0
	}
	
	var avgSimilarity float64
	for _, sim := range similar {
		avgSimilarity += sim.Similarity
	}
	avgSimilarity /= float64(len(similar))
	
	confidence := (countFactor * 0.3) + (avgSimilarity * 0.7)
	return confidence
}

func (wp *WorkloadProfiler) recommendGPUModel(similar []SimilarWorkload) string {
	// Count GPU models used by similar workloads
	gpuCounts := make(map[string]int)
	
	for _, sim := range similar {
		if sim.GPUUsed != "" {
			gpuCounts[sim.GPUUsed]++
		}
	}
	
	// Find most common GPU
	maxCount := 0
	recommendedGPU := ""
	for gpu, count := range gpuCounts {
		if count > maxCount {
			maxCount = count
			recommendedGPU = gpu
		}
	}
	
	if recommendedGPU == "" {
		recommendedGPU = "A100" // Default fallback
	}
	
	return recommendedGPU
}

func (wp *WorkloadProfiler) getDefaultPrediction() *PredictedRequirements {
	return &PredictedRequirements{
		TFlops:     resource.NewQuantity(5000, resource.DecimalSI),
		VRAM:       resource.NewQuantity(16*1024*1024*1024, resource.BinarySI),
		Cost:       1.0,
		Latency:    100,
		Confidence: 0.3,
		GPUModel:   "A100",
		Source:     "default",
	}
}

func getMetadataString(metadata map[string]interface{}, key string) string {
	if v, ok := metadata[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getMetadataFloat(metadata map[string]interface{}, key string) float64 {
	if v, ok := metadata[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return 0.0
}

// ConvertToPredictedRecommendation converts prediction to WorkloadGPURecommendation
func ConvertToPredictedRecommendation(pred *PredictedRequirements, workloadID string) tfv1.WorkloadGPURecommendation {
	return tfv1.WorkloadGPURecommendation{
		WorkloadID:        workloadID,
		RecommendedGPU:    pred.GPUModel,
		RecommendedTFlops: *pred.TFlops,
		RecommendedVRAM:   *pred.VRAM,
		ConfidenceScore:   fmt.Sprintf("%.2f", pred.Confidence),
		EstimatedCost:     fmt.Sprintf("$%.2f/hr", pred.Cost),
		EstimatedPerformance: &tfv1.PerformanceEstimate{
			ExpectedLatencyP50: fmt.Sprintf("%dms", pred.Latency),
			ExpectedLatencyP95: fmt.Sprintf("%dms", int(float64(pred.Latency)*1.5)),
			ExpectedThroughput: "estimated",
		},
	}
}


