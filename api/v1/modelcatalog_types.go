package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ModelCatalog defines a small model available for training
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
type ModelCatalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ModelCatalogSpec `json:"spec,omitempty"`
}

// ModelCatalogSpec defines model specifications
type ModelCatalogSpec struct {
	// Name of the model
	Name string `json:"name"`

	// Parameters size (e.g., "1.1B", "2.7B", "7B")
	Parameters string `json:"parameters"`

	// Context length
	ContextLength int `json:"contextLength"`

	// Architecture type
	// +kubebuilder:validation:Enum=llama;phi;mistral;gemma;stablelm
	Architecture string `json:"architecture"`

	// GPU requirement in vGPU units
	GPURequirement float64 `json:"gpuRequirement"`

	// Estimated training time
	TrainingTime string `json:"trainingTime"`

	// Best use cases
	BestFor []string `json:"bestFor"`

	// Base model URL (HuggingFace)
	BaseModelURL string `json:"baseModelUrl"`

	// Training configuration
	TrainingConfig TrainingConfig `json:"trainingConfig"`
}

// LoRAConfig defines LoRA (Low-Rank Adaptation) configuration
type LoRAConfig struct {
	// Rank is the LoRA rank (typically 4-64)
	Rank int `json:"rank"`

	// Alpha is the LoRA alpha parameter
	Alpha float64 `json:"alpha"`

	// TargetModules are the modules to apply LoRA to
	TargetModules []string `json:"targetModules"`

	// DropoutRate is the LoRA dropout rate
	DropoutRate float64 `json:"dropoutRate,omitempty"`
}

// TrainingConfig defines training parameters
type TrainingConfig struct {
	BatchSize        int        `json:"batchSize"`
	LearningRate     float64    `json:"learningRate"`
	Epochs           int        `json:"epochs"`
	LoRAConfig       LoRAConfig `json:"loraConfig"`
	QuantizationBits int        `json:"quantizationBits"` // 4, 8, or 16
	GradientAccum    int        `json:"gradientAccum"`
}

// +kubebuilder:object:root=true

// ModelCatalogList contains a list of ModelCatalog
type ModelCatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ModelCatalog `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ModelCatalog{}, &ModelCatalogList{})
}

