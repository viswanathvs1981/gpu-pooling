package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AgentMemory is the Schema for the agentmemories API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
type AgentMemory struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentMemorySpec   `json:"spec,omitempty"`
	Status AgentMemoryStatus `json:"status,omitempty"`
}

// AgentMemorySpec defines the desired state of AgentMemory
type AgentMemorySpec struct {
	// AgentID is the unique identifier for the agent
	AgentID string `json:"agentId"`

	// MemoryTypes specifies which memory types to provision
	// +kubebuilder:validation:Enum=semantic;episodic;longterm
	MemoryTypes []string `json:"memoryTypes"`

	// Retention specifies how long to keep memory data
	// +kubebuilder:default="30d"
	Retention string `json:"retention,omitempty"`

	// MaxSize specifies the maximum storage size
	// +kubebuilder:default="10Gi"
	MaxSize string `json:"maxSize,omitempty"`
}

// AgentMemoryStatus defines the observed state of AgentMemory
type AgentMemoryStatus struct {
	// Phase represents the current phase of memory provisioning
	// +kubebuilder:validation:Enum=Provisioning;Active;Cleanup;Failed
	Phase string `json:"phase,omitempty"`

	// SemanticURL is the URL for semantic memory access
	SemanticURL string `json:"semanticUrl,omitempty"`

	// EpisodicURL is the URL for episodic memory access
	EpisodicURL string `json:"episodicUrl,omitempty"`

	// LongtermURL is the URL for long-term memory access
	LongtermURL string `json:"longtermUrl,omitempty"`

	// UsageBytes tracks current memory usage
	UsageBytes int64 `json:"usageBytes,omitempty"`

	// LastAccessTime records the last time memory was accessed
	LastAccessTime *metav1.Time `json:"lastAccessTime,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

// AgentMemoryList contains a list of AgentMemory
type AgentMemoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentMemory `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AgentMemory{}, &AgentMemoryList{})
}

