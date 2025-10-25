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

// AzureGPUSourceSpec defines the desired state of AzureGPUSource
type AzureGPUSourceSpec struct {
	// SourceType indicates whether this is an AKS cluster or Azure Foundry endpoint
	// +kubebuilder:validation:Enum=aks;foundry
	SourceType AzureGPUSourceType `json:"sourceType"`

	// For Foundry: model endpoint URL
	// +optional
	FoundryEndpoint string `json:"foundryEndpoint,omitempty"`

	// For AKS: cluster name
	// +optional
	AKSClusterName string `json:"aksClusterName,omitempty"`

	// Azure subscription ID
	SubscriptionID string `json:"subscriptionID"`

	// Azure region
	Region string `json:"region"`

	// Available GPU models/SKUs
	// +optional
	AvailableModels []AzureGPUModel `json:"availableModels,omitempty"`

	// Priority when multiple sources available (higher = preferred)
	// +kubebuilder:default=50
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Priority int32 `json:"priority,omitempty"`

	// Cost per hour for this source
	// +optional
	CostPerHour string `json:"costPerHour,omitempty"`

	// Authentication configuration
	// +optional
	Auth AzureAuthConfig `json:"auth,omitempty"`

	// Enable this source
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`

	// Sync interval for checking availability
	// +kubebuilder:default="30s"
	SyncInterval string `json:"syncInterval,omitempty"`
}

// +kubebuilder:validation:Enum=aks;foundry
type AzureGPUSourceType string

const (
	AzureGPUSourceTypeAKS     AzureGPUSourceType = "aks"
	AzureGPUSourceTypeFoundry AzureGPUSourceType = "foundry"
)

type AzureGPUModel struct {
	// Model name (e.g., "gpt-4", "llama-3-70b" for Foundry, or "Standard_NC24ads_A100_v4" for AKS)
	ModelName string `json:"modelName"`

	// SKU identifier
	SKU string `json:"sku"`

	// TFlops capacity
	TFlops resource.Quantity `json:"tflops"`

	// VRAM capacity
	VRAM resource.Quantity `json:"vram"`

	// Max throughput (tokens/sec for LLM endpoints, or compute units for AKS)
	// +optional
	MaxThroughput int32 `json:"maxThroughput,omitempty"`

	// Whether this model is currently available
	// +kubebuilder:default=true
	Available bool `json:"available,omitempty"`

	// Cost per hour for this specific model
	// +optional
	CostPerHour string `json:"costPerHour,omitempty"`
}

type AzureAuthConfig struct {
	// Authentication method
	// +kubebuilder:validation:Enum=servicePrincipal;managedIdentity;accessKey
	// +kubebuilder:default=managedIdentity
	Method AzureAuthMethod `json:"method,omitempty"`

	// Service principal credentials (secret reference)
	// +optional
	ServicePrincipal *ServicePrincipalAuth `json:"servicePrincipal,omitempty"`

	// Managed identity client ID
	// +optional
	ManagedIdentityClientID string `json:"managedIdentityClientID,omitempty"`

	// Access key secret reference
	// +optional
	AccessKeySecret *SecretReference `json:"accessKeySecret,omitempty"`
}

// +kubebuilder:validation:Enum=servicePrincipal;managedIdentity;accessKey
type AzureAuthMethod string

const (
	AzureAuthMethodServicePrincipal AzureAuthMethod = "servicePrincipal"
	AzureAuthMethodManagedIdentity  AzureAuthMethod = "managedIdentity"
	AzureAuthMethodAccessKey        AzureAuthMethod = "accessKey"
)

type ServicePrincipalAuth struct {
	TenantID     string           `json:"tenantID"`
	ClientID     string           `json:"clientID"`
	ClientSecret SecretReference  `json:"clientSecret"`
}

type SecretReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
}

// AzureGPUSourceStatus defines the observed state of AzureGPUSource
type AzureGPUSourceStatus struct {
	// Phase of the GPU source
	// +kubebuilder:default=Pending
	Phase AzureGPUSourcePhase `json:"phase"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Total available TFlops across all models
	TotalAvailableTFlops resource.Quantity `json:"totalAvailableTFlops,omitempty"`

	// Total available VRAM across all models
	TotalAvailableVRAM resource.Quantity `json:"totalAvailableVRAM,omitempty"`

	// Number of available models
	AvailableModelCount int32 `json:"availableModelCount,omitempty"`

	// Last sync time
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Current model availability details
	// +optional
	ModelStatus []AzureGPUModelStatus `json:"modelStatus,omitempty"`

	// Connection status
	// +optional
	Connected bool `json:"connected,omitempty"`

	// Error message if any
	// +optional
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// +kubebuilder:validation:Enum=Pending;Ready;Error;Syncing
type AzureGPUSourcePhase string

const (
	AzureGPUSourcePhasePending AzureGPUSourcePhase = "Pending"
	AzureGPUSourcePhaseReady   AzureGPUSourcePhase = "Ready"
	AzureGPUSourcePhaseError   AzureGPUSourcePhase = "Error"
	AzureGPUSourcePhaseSyncing AzureGPUSourcePhase = "Syncing"
)

type AzureGPUModelStatus struct {
	ModelName         string            `json:"modelName"`
	Available         bool              `json:"available"`
	CurrentUtilization string           `json:"currentUtilization,omitempty"`
	EstimatedLatency  string            `json:"estimatedLatency,omitempty"`
	LastChecked       *metav1.Time      `json:"lastChecked,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.sourceType"
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.region"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Available Models",type="integer",JSONPath=".status.availableModelCount"
// +kubebuilder:printcolumn:name="Total TFlops",type="string",JSONPath=".status.totalAvailableTFlops"
// +kubebuilder:printcolumn:name="Connected",type="boolean",JSONPath=".status.connected"

// AzureGPUSource is the Schema for the azuregpusources API
type AzureGPUSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureGPUSourceSpec   `json:"spec,omitempty"`
	Status AzureGPUSourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AzureGPUSourceList contains a list of AzureGPUSource
type AzureGPUSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureGPUSource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AzureGPUSource{}, &AzureGPUSourceList{})
}


