package vectordb

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"math"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// Embedder generates vector embeddings from workload specifications
type Embedder struct {
	dimension int
	logger    klog.Logger
}

// NewEmbedder creates a new embedder
func NewEmbedder(dimension int) *Embedder {
	return &Embedder{
		dimension: dimension,
		logger:    klog.NewKlogr().WithName("embedder"),
	}
}

// WorkloadFeatures represents extracted features from a workload
type WorkloadFeatures struct {
	// Container features
	Image            string
	ImageRegistry    string
	Framework        string
	FrameworkVersion string
	
	// Resource features
	CPURequest       float64
	MemoryRequest    float64
	GPURequest       float64
	
	// Workload characteristics
	WorkloadType     string
	ModelFamily      string
	EstimatedTokens  int64
	BatchSize        int32
	
	// Environment features
	EnvironmentVars  map[string]string
	Commands         []string
	Args             []string
	
	// Labels and annotations
	Labels           map[string]string
	Annotations      map[string]string
}

// GenerateEmbedding generates a vector embedding from workload features
func (e *Embedder) GenerateEmbedding(ctx context.Context, features *WorkloadFeatures) ([]float32, error) {
	e.logger.V(2).Info("Generating embedding", "framework", features.Framework)
	
	// This is a simplified embedding generation
	// In production, you would use a pre-trained model or sentence transformer
	
	embedding := make([]float32, e.dimension)
	
	// Feature engineering: convert features to vector
	// This is a placeholder implementation using feature hashing
	
	// Position 0-99: Framework embedding
	e.hashFeature(features.Framework, embedding, 0, 100)
	
	// Position 100-199: Image embedding
	e.hashFeature(features.Image, embedding, 100, 100)
	
	// Position 200-299: Workload type embedding
	e.hashFeature(features.WorkloadType, embedding, 200, 100)
	
	// Position 300-399: Model family embedding
	e.hashFeature(features.ModelFamily, embedding, 300, 100)
	
	// Position 400-499: Resource embedding
	e.embedResources(features, embedding, 400)
	
	// Position 500-699: Environment and command embedding
	e.embedEnvironment(features, embedding, 500)
	
	// Position 700-767: Metadata embedding
	e.embedMetadata(features, embedding, 700)
	
	// Normalize the embedding
	e.normalize(embedding)
	
	return embedding, nil
}

// ExtractFeaturesFromPod extracts features from a Pod spec
func (e *Embedder) ExtractFeaturesFromPod(pod *v1.Pod) *WorkloadFeatures {
	features := &WorkloadFeatures{
		Labels:      pod.Labels,
		Annotations: pod.Annotations,
		EnvironmentVars: make(map[string]string),
	}
	
	if len(pod.Spec.Containers) > 0 {
		container := pod.Spec.Containers[0]
		
		features.Image = container.Image
		features.ImageRegistry = e.extractRegistry(container.Image)
		features.Framework = e.detectFramework(container.Image, container.Env)
		
		// Extract resource requests
		if cpu := container.Resources.Requests.Cpu(); cpu != nil {
			features.CPURequest = float64(cpu.MilliValue()) / 1000.0
		}
		if memory := container.Resources.Requests.Memory(); memory != nil {
			features.MemoryRequest = float64(memory.Value()) / (1024 * 1024 * 1024) // GB
		}
		
		// Extract environment variables
		for _, env := range container.Env {
			features.EnvironmentVars[env.Name] = env.Value
		}
		
		features.Commands = container.Command
		features.Args = container.Args
	}
	
	// Detect workload type from labels/annotations
	features.WorkloadType = e.detectWorkloadType(pod)
	features.ModelFamily = e.detectModelFamily(pod)
	
	return features
}

// hashFeature hashes a string feature into embedding positions
func (e *Embedder) hashFeature(feature string, embedding []float32, offset, length int) {
	if feature == "" {
		return
	}
	
	hash := sha256.Sum256([]byte(feature))
	
	for i := 0; i < length && i < len(hash); i++ {
		if offset+i < len(embedding) {
			embedding[offset+i] = float32(hash[i]) / 255.0
		}
	}
}

// embedResources embeds resource requirements
func (e *Embedder) embedResources(features *WorkloadFeatures, embedding []float32, offset int) {
	if offset >= len(embedding) {
		return
	}
	
	// Normalize resources to 0-1 range
	embedding[offset] = float32(math.Min(features.CPURequest/64.0, 1.0))
	if offset+1 < len(embedding) {
		embedding[offset+1] = float32(math.Min(features.MemoryRequest/512.0, 1.0))
	}
	if offset+2 < len(embedding) {
		embedding[offset+2] = float32(math.Min(features.GPURequest/8.0, 1.0))
	}
}

// embedEnvironment embeds environment variables and commands
func (e *Embedder) embedEnvironment(features *WorkloadFeatures, embedding []float32, offset int) {
	// Combine env vars and commands into a single string
	envString := ""
	for k, v := range features.EnvironmentVars {
		envString += k + "=" + v + " "
	}
	for _, cmd := range features.Commands {
		envString += cmd + " "
	}
	for _, arg := range features.Args {
		envString += arg + " "
	}
	
	e.hashFeature(envString, embedding, offset, 200)
}

// embedMetadata embeds labels and annotations
func (e *Embedder) embedMetadata(features *WorkloadFeatures, embedding []float32, offset int) {
	metaString := ""
	for k, v := range features.Labels {
		metaString += k + "=" + v + " "
	}
	for k, v := range features.Annotations {
		metaString += k + "=" + v + " "
	}
	
	e.hashFeature(metaString, embedding, offset, 68)
}

// normalize normalizes the embedding vector to unit length
func (e *Embedder) normalize(embedding []float32) {
	var sum float64
	for _, v := range embedding {
		sum += float64(v * v)
	}
	
	norm := math.Sqrt(sum)
	if norm > 0 {
		for i := range embedding {
			embedding[i] /= float32(norm)
		}
	}
}

// detectFramework detects the ML framework from image and environment
func (e *Embedder) detectFramework(image string, envVars []v1.EnvVar) string {
	imageLower := strings.ToLower(image)
	
	frameworks := []string{"pytorch", "tensorflow", "jax", "vllm", "tensorrt", "triton", "llama"}
	for _, framework := range frameworks {
		if strings.Contains(imageLower, framework) {
			return framework
		}
	}
	
	// Check environment variables
	for _, env := range envVars {
		envLower := strings.ToLower(env.Name + env.Value)
		for _, framework := range frameworks {
			if strings.Contains(envLower, framework) {
				return framework
			}
		}
	}
	
	return "unknown"
}

// detectWorkloadType detects if this is training, inference, or other
func (e *Embedder) detectWorkloadType(pod *v1.Pod) string {
	// Check labels/annotations
	if wType, ok := pod.Labels["workload-type"]; ok {
		return wType
	}
	if wType, ok := pod.Annotations["tensor-fusion.ai/workload-type"]; ok {
		return wType
	}
	
	// Infer from pod name/labels
	nameAnnot := strings.ToLower(pod.Name + " " + jsonString(pod.Labels))
	
	if strings.Contains(nameAnnot, "train") || strings.Contains(nameAnnot, "training") {
		return "training"
	}
	if strings.Contains(nameAnnot, "infer") || strings.Contains(nameAnnot, "serving") {
		return "inference"
	}
	if strings.Contains(nameAnnot, "notebook") || strings.Contains(nameAnnot, "jupyter") {
		return "interactive"
	}
	
	return "unknown"
}

// detectModelFamily detects the model family (GPT, LLama, Mistral, etc.)
func (e *Embedder) detectModelFamily(pod *v1.Pod) string {
	searchStr := strings.ToLower(pod.Name + " " + jsonString(pod.Labels) + " " + jsonString(pod.Annotations))
	
	models := []string{"gpt", "llama", "mistral", "claude", "bert", "t5", "whisper", "stable-diffusion"}
	for _, model := range models {
		if strings.Contains(searchStr, model) {
			return model
		}
	}
	
	return "unknown"
}

// extractRegistry extracts the registry from an image name
func (e *Embedder) extractRegistry(image string) string {
	parts := strings.Split(image, "/")
	if len(parts) > 1 && strings.Contains(parts[0], ".") {
		return parts[0]
	}
	return "docker.io"
}

func jsonString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}


