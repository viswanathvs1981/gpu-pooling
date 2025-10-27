package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TrainedModel represents a trained model instance
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
type TrainedModel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrainedModelSpec   `json:"spec,omitempty"`
	Status TrainedModelStatus `json:"status,omitempty"`
}

// TrainedModelSpec defines the desired state
type TrainedModelSpec struct {
	// Base model used for training
	BaseModel string `json:"baseModel"`

	// Dataset path
	DatasetPath string `json:"datasetPath"`

	// Training configuration used
	TrainingConfig TrainingConfig `json:"trainingConfig"`

	// Owner of the trained model
	Owner string `json:"owner"`
}

// TrainedModelStatus defines the observed state
type TrainedModelStatus struct {
	// Phase of the training
	// +kubebuilder:validation:Enum=Training;Validating;Ready;Failed
	Phase string `json:"phase,omitempty"`

	// Storage location of the trained model
	ModelURL string `json:"modelUrl,omitempty"`

	// Model size in bytes
	ModelSize string `json:"modelSize,omitempty"`

	// Accuracy metrics
	Accuracy float64 `json:"accuracy,omitempty"`

	// Training duration
	TrainingTime string `json:"trainingTime,omitempty"`

	// Whether model is ready for deployment
	DeploymentReady bool `json:"deploymentReady,omitempty"`

	// Conditions represent the latest observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

// TrainedModelList contains a list of TrainedModel
type TrainedModelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TrainedModel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TrainedModel{}, &TrainedModelList{})
}

