package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LLMEndpoint represents a discovered LLM endpoint
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
type LLMEndpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LLMEndpointSpec   `json:"spec,omitempty"`
	Status LLMEndpointStatus `json:"status,omitempty"`
}

// LLMEndpointSpec defines the desired state
type LLMEndpointSpec struct {
	// Name of the endpoint
	Name string `json:"name"`

	// URL of the endpoint
	URL string `json:"url"`

	// Type of endpoint
	// +kubebuilder:validation:Enum=vllm;openai;azure;custom
	Type string `json:"type"`

	// Provider
	Provider string `json:"provider"`

	// Authentication configuration
	Authentication AuthConfig `json:"authentication,omitempty"`

	// Priority (1-100, higher = prefer)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	Priority int `json:"priority,omitempty"`
}

// AuthConfig defines authentication settings
type AuthConfig struct {
	// Type of authentication
	// +kubebuilder:validation:Enum=none;api-key;bearer
	Type string `json:"type"`

	// Reference to Kubernetes secret
	SecretRef string `json:"secretRef,omitempty"`
}

// LLMEndpointStatus defines the observed state
type LLMEndpointStatus struct {
	// Phase of the endpoint
	// +kubebuilder:validation:Enum=Discovered;Healthy;Degraded;Unhealthy
	Phase string `json:"phase,omitempty"`

	// When endpoint was discovered
	DiscoveredAt *metav1.Time `json:"discoveredAt,omitempty"`

	// Last health check time
	LastHealthCheck *metav1.Time `json:"lastHealthCheck,omitempty"`

	// Health status
	Health HealthStatus `json:"health,omitempty"`

	// Model capabilities
	Capabilities []ModelCapability `json:"capabilities,omitempty"`

	// Performance metrics
	Metrics PerformanceMetrics `json:"metrics,omitempty"`

	// Conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// HealthStatus represents health information
type HealthStatus struct {
	Status    string  `json:"status"`
	ErrorRate float64 `json:"errorRate"`
	Latency   string  `json:"latency"`
	Capacity  int     `json:"capacity"`
	Message   string  `json:"message,omitempty"`
}

// ModelCapability represents a model's capabilities
type ModelCapability struct {
	ModelID       string   `json:"modelId"`
	ContextLength int      `json:"contextLength"`
	Features      []string `json:"features"`
	MaxTokens     int      `json:"maxTokens"`
}

// PerformanceMetrics represents performance data
type PerformanceMetrics struct {
	RequestsPerSecond float64 `json:"requestsPerSecond"`
	TokensPerSecond   float64 `json:"tokensPerSecond"`
	AverageLatency    string  `json:"averageLatency"`
	ErrorRate         float64 `json:"errorRate"`
	Uptime            string  `json:"uptime"`
}

// +kubebuilder:object:root=true

// LLMEndpointList contains a list of LLMEndpoint
type LLMEndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LLMEndpoint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LLMEndpoint{}, &LLMEndpointList{})
}

