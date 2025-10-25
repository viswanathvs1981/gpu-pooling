/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkloadIntelligenceSpec defines the desired state of WorkloadIntelligence
type WorkloadIntelligenceSpec struct {
	// Enable ML-based predictions
	// +kubebuilder:default=true
	EnablePrediction bool `json:"enablePrediction,omitempty"`

	// Enable vectorization and similarity search
	// +kubebuilder:default=true
	EnableVectorization bool `json:"enableVectorization,omitempty"`

	// Vector database configuration
	// +optional
	VectorDBConfig *VectorDBConfig `json:"vectorDBConfig,omitempty"`

	// ML model training configuration
	// +optional
	ModelTraining *ModelTrainingConfig `json:"modelTraining,omitempty"`

	// Feature extractors to use
	// +optional
	FeatureExtractors []string `json:"featureExtractors,omitempty"`

	// Prediction confidence threshold (0.0-1.0)
	// +kubebuilder:default="0.7"
	ConfidenceThreshold string `json:"confidenceThreshold,omitempty"`

	// Target namespaces for intelligence (empty = all)
	// +optional
	TargetNamespaces []string `json:"targetNamespaces,omitempty"`
}

type VectorDBConfig struct {
	// Vector DB type
	// +kubebuilder:validation:Enum=qdrant;weaviate;pinecone;milvus
	// +kubebuilder:default=qdrant
	Type VectorDBType `json:"type,omitempty"`

	// Endpoint URL
	Endpoint string `json:"endpoint"`

	// Collection/Index name
	// +kubebuilder:default="tensor-fusion-workloads"
	CollectionName string `json:"collectionName,omitempty"`

	// Vector dimension
	// +kubebuilder:default=768
	VectorDimension int32 `json:"vectorDimension,omitempty"`

	// Authentication
	// +optional
	Auth *VectorDBAuth `json:"auth,omitempty"`

	// Sync interval for updating vectors
	// +kubebuilder:default="5m"
	SyncInterval string `json:"syncInterval,omitempty"`
}

// +kubebuilder:validation:Enum=qdrant;weaviate;pinecone;milvus
type VectorDBType string

const (
	VectorDBTypeQdrant   VectorDBType = "qdrant"
	VectorDBTypeWeaviate VectorDBType = "weaviate"
	VectorDBTypePinecone VectorDBType = "pinecone"
	VectorDBTypeMilvus   VectorDBType = "milvus"
)

type VectorDBAuth struct {
	// API key secret reference
	// +optional
	APIKeySecret *SecretReference `json:"apiKeySecret,omitempty"`

	// Token secret reference
	// +optional
	TokenSecret *SecretReference `json:"tokenSecret,omitempty"`
}

type ModelTrainingConfig struct {
	// Training schedule (cron expression)
	// +kubebuilder:default="0 2 * * 0"
	Schedule string `json:"schedule,omitempty"`

	// Training data retention period
	// +kubebuilder:default="90d"
	DataRetention string `json:"dataRetention,omitempty"`

	// Minimum samples required for training
	// +kubebuilder:default=100
	MinSamples int32 `json:"minSamples,omitempty"`

	// Models to train
	// +optional
	Models []MLModelConfig `json:"models,omitempty"`

	// GPU resources for training
	// +optional
	TrainingResources *Resources `json:"trainingResources,omitempty"`
}

type MLModelConfig struct {
	// Model name
	// +kubebuilder:validation:Enum=lstm-forecaster;transformer-classifier;gradient-boosting-regressor;rl-scheduler
	Name MLModelType `json:"name"`

	// Enable this model
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Hyperparameters
	// +optional
	Hyperparameters map[string]string `json:"hyperparameters,omitempty"`

	// Training epochs
	// +kubebuilder:default=50
	Epochs int32 `json:"epochs,omitempty"`

	// Batch size
	// +kubebuilder:default=32
	BatchSize int32 `json:"batchSize,omitempty"`
}

// +kubebuilder:validation:Enum=lstm-forecaster;transformer-classifier;gradient-boosting-regressor;rl-scheduler
type MLModelType string

const (
	MLModelTypeLSTMForecaster         MLModelType = "lstm-forecaster"
	MLModelTypeTransformerClassifier  MLModelType = "transformer-classifier"
	MLModelTypeGradientBoostingRegressor MLModelType = "gradient-boosting-regressor"
	MLModelTypeRLScheduler            MLModelType = "rl-scheduler"
)

// WorkloadIntelligenceStatus defines the observed state of WorkloadIntelligence
type WorkloadIntelligenceStatus struct {
	// Phase
	// +kubebuilder:default=Pending
	Phase WorkloadIntelligencePhase `json:"phase"`

	// Conditions
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Model statistics
	// +optional
	Models []MLModelStatus `json:"models,omitempty"`

	// Vector DB statistics
	// +optional
	VectorDB *VectorDBStatus `json:"vectorDB,omitempty"`

	// Prediction statistics
	// +optional
	PredictionStats *PredictionStats `json:"predictionStats,omitempty"`

	// Recent recommendations
	// +optional
	RecentRecommendations []WorkloadGPURecommendation `json:"recentRecommendations,omitempty"`

	// Last updated time
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
}

// +kubebuilder:validation:Enum=Pending;Initializing;Ready;Training;Error
type WorkloadIntelligencePhase string

const (
	WorkloadIntelligencePhasePending      WorkloadIntelligencePhase = "Pending"
	WorkloadIntelligencePhaseInitializing WorkloadIntelligencePhase = "Initializing"
	WorkloadIntelligencePhaseReady        WorkloadIntelligencePhase = "Ready"
	WorkloadIntelligencePhaseTraining     WorkloadIntelligencePhase = "Training"
	WorkloadIntelligencePhaseError        WorkloadIntelligencePhase = "Error"
)

type MLModelStatus struct {
	Name             MLModelType  `json:"name"`
	Accuracy         string       `json:"accuracy,omitempty"`
	LastTrained      *metav1.Time `json:"lastTrained,omitempty"`
	TrainingSamples  int32        `json:"trainingSamples"`
	Version          string       `json:"version,omitempty"`
	Status           string       `json:"status,omitempty"`
}

type VectorDBStatus struct {
	Connected       bool         `json:"connected"`
	TotalVectors    int64        `json:"totalVectors"`
	LastSync        *metav1.Time `json:"lastSync,omitempty"`
	CollectionSize  string       `json:"collectionSize,omitempty"`
}

type PredictionStats struct {
	TotalPredictions     int64  `json:"totalPredictions"`
	SuccessfulPredictions int64 `json:"successfulPredictions"`
	AverageConfidence    string `json:"averageConfidence,omitempty"`
	AccuracyScore        string `json:"accuracyScore,omitempty"`
}

type WorkloadGPURecommendation struct {
	// Workload identifier
	WorkloadID string `json:"workloadId"`

	// Workload pattern description
	WorkloadPattern string `json:"workloadPattern,omitempty"`

	// Recommended GPU type
	RecommendedGPU string `json:"recommendedGpu"`

	// Recommended TFlops
	RecommendedTFlops resource.Quantity `json:"recommendedTflops,omitempty"`

	// Recommended VRAM
	RecommendedVRAM resource.Quantity `json:"recommendedVram,omitempty"`

	// Confidence score (0.0-1.0)
	ConfidenceScore string `json:"confidenceScore"`

	// Estimated hourly cost
	EstimatedCost string `json:"estimatedCost,omitempty"`

	// Estimated performance metrics
	EstimatedPerformance *PerformanceEstimate `json:"estimatedPerformance,omitempty"`

	// Alternative recommendations
	// +optional
	Alternatives []AlternativeRecommendation `json:"alternatives,omitempty"`

	// Timestamp
	Timestamp metav1.Time `json:"timestamp"`
}

type PerformanceEstimate struct {
	ExpectedLatencyP50 string `json:"expectedLatencyP50,omitempty"`
	ExpectedLatencyP95 string `json:"expectedLatencyP95,omitempty"`
	ExpectedThroughput string `json:"expectedThroughput,omitempty"`
}

type AlternativeRecommendation struct {
	GPUType          string  `json:"gpuType"`
	ConfidenceScore  string  `json:"confidenceScore"`
	EstimatedCost    string  `json:"estimatedCost,omitempty"`
	CostSavings      string  `json:"costSavings,omitempty"`
	PerformanceImpact string `json:"performanceImpact,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Predictions",type="integer",JSONPath=".status.predictionStats.totalPredictions"
// +kubebuilder:printcolumn:name="Accuracy",type="string",JSONPath=".status.predictionStats.accuracyScore"
// +kubebuilder:printcolumn:name="Vectors",type="integer",JSONPath=".status.vectorDB.totalVectors"

// WorkloadIntelligence is the Schema for the workloadintelligences API
type WorkloadIntelligence struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadIntelligenceSpec   `json:"spec,omitempty"`
	Status WorkloadIntelligenceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WorkloadIntelligenceList contains a list of WorkloadIntelligence
type WorkloadIntelligenceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkloadIntelligence `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorkloadIntelligence{}, &WorkloadIntelligenceList{})
}


