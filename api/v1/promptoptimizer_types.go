package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PromptOptimizerSpec defines the desired state of PromptOptimizer
type PromptOptimizerSpec struct {
	// Model configuration
	BaseModel    string `json:"baseModel"`              // e.g., "llama-3.1-7b"
	LoRAAdapter  string `json:"loraAdapter,omitempty"`  // Path to LoRA adapter
	Quantization string `json:"quantization,omitempty"` // e.g., "int8", "int4"

	// Resource allocation
	GPUFraction string `json:"gpuFraction,omitempty"` // e.g., "0.1"
	Replicas    int32  `json:"replicas,omitempty"`

	// Optimization strategies
	EnabledTechniques []string `json:"enabledTechniques,omitempty"` // e.g., ["chain-of-thought", "few-shot", "safety"]

	// Integration points
	PortkeyIntegration bool `json:"portkeyIntegration,omitempty"` // Enable Portkey gateway integration
	A2AChannel         bool `json:"a2aChannel,omitempty"`         // Enable A2A communication
}

// PromptOptimizerStatus defines the observed state of PromptOptimizer
type PromptOptimizerStatus struct {
	Phase                 string             `json:"phase"` // e.g., "Initializing", "Ready", "Failed"
	ModelEndpoint         string             `json:"modelEndpoint,omitempty"`
	OptimizationsPerformed int64              `json:"optimizationsPerformed,omitempty"`
	SuccessRate           float64            `json:"successRate,omitempty"`
	AverageLatencyMs      float64            `json:"averageLatencyMs,omitempty"`
	Conditions            []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Success Rate",type="string",JSONPath=".status.successRate"
// +kubebuilder:printcolumn:name="Optimizations",type="integer",JSONPath=".status.optimizationsPerformed"

// PromptOptimizer is the Schema for the promptoptimizers API
type PromptOptimizer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PromptOptimizerSpec   `json:"spec,omitempty"`
	Status PromptOptimizerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PromptOptimizerList contains a list of PromptOptimizer
type PromptOptimizerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PromptOptimizer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PromptOptimizer{}, &PromptOptimizerList{})
}

