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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LLMRouteSpec defines the desired state of LLMRoute
type LLMRouteSpec struct {
	// Portkey configuration ID (if using Portkey managed configs)
	// +optional
	PortkeyConfigID string `json:"portkeyConfigID,omitempty"`

	// Routing strategy
	// +kubebuilder:default=cost-optimized
	Strategy RoutingStrategy `json:"strategy,omitempty"`

	// Target LLM endpoints in priority order
	Targets []LLMTarget `json:"targets"`

	// Request selector to match incoming requests
	// +optional
	Selector *LLMSelector `json:"selector,omitempty"`

	// Cost budget constraints
	// +optional
	Budget *CostBudget `json:"budget,omitempty"`

	// Caching configuration
	// +optional
	Caching *CachingConfig `json:"caching,omitempty"`

	// Retry configuration
	// +optional
	Retry *RetryConfig `json:"retry,omitempty"`

	// Timeout for requests
	// +kubebuilder:default="30s"
	Timeout string `json:"timeout,omitempty"`

	// Enable fallback to next target on failure
	// +kubebuilder:default=true
	EnableFallback bool `json:"enableFallback,omitempty"`
}

// +kubebuilder:validation:Enum=cost-optimized;latency-optimized;round-robin;weighted;priority;loadtest
type RoutingStrategy string

const (
	RoutingStrategyCostOptimized    RoutingStrategy = "cost-optimized"
	RoutingStrategyLatencyOptimized RoutingStrategy = "latency-optimized"
	RoutingStrategyRoundRobin       RoutingStrategy = "round-robin"
	RoutingStrategyWeighted         RoutingStrategy = "weighted"
	RoutingStrategyPriority         RoutingStrategy = "priority"
	RoutingStrategyLoadTest         RoutingStrategy = "loadtest"
)

type LLMTarget struct {
	// Target name
	Name string `json:"name"`

	// Provider (openai, azure, anthropic, cohere, etc.)
	Provider string `json:"provider"`

	// Model name
	Model string `json:"model"`

	// TensorFusion GPU pool to use (if using TF-managed GPUs)
	// +optional
	GPUPool string `json:"gpuPool,omitempty"`

	// Azure GPU source to use (if applicable)
	// +optional
	AzureGPUSource string `json:"azureGPUSource,omitempty"`

	// Weight for weighted routing (1-100)
	// +kubebuilder:default=50
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	Weight int32 `json:"weight,omitempty"`

	// Priority (lower = higher priority)
	// +kubebuilder:default=50
	Priority int32 `json:"priority,omitempty"`

	// API endpoint override
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// API key secret reference
	// +optional
	APIKeySecret *SecretReference `json:"apiKeySecret,omitempty"`

	// Additional parameters
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`

	// Enable this target
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
}

type LLMSelector struct {
	// Namespace selector
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`

	// Label selector for matching pods
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// Model pattern to match (regex)
	// +optional
	ModelPattern string `json:"modelPattern,omitempty"`

	// Match specific user/tenant
	// +optional
	Users []string `json:"users,omitempty"`
}

type CostBudget struct {
	// Maximum cost per request (in USD)
	// +optional
	MaxCostPerRequest string `json:"maxCostPerRequest,omitempty"`

	// Maximum tokens per request
	// +optional
	MaxTokensPerRequest int32 `json:"maxTokensPerRequest,omitempty"`

	// Daily budget limit (in USD)
	// +optional
	DailyLimit string `json:"dailyLimit,omitempty"`

	// Monthly budget limit (in USD)
	// +optional
	MonthlyLimit string `json:"monthlyLimit,omitempty"`

	// Action to take when budget exceeded
	// +kubebuilder:validation:Enum=reject;queue;fallback-cheaper
	// +kubebuilder:default=reject
	OnExceeded BudgetExceededAction `json:"onExceeded,omitempty"`
}

// +kubebuilder:validation:Enum=reject;queue;fallback-cheaper
type BudgetExceededAction string

const (
	BudgetExceededActionReject          BudgetExceededAction = "reject"
	BudgetExceededActionQueue           BudgetExceededAction = "queue"
	BudgetExceededActionFallbackCheaper BudgetExceededAction = "fallback-cheaper"
)

type CachingConfig struct {
	// Enable semantic caching
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// TTL for cached responses
	// +kubebuilder:default="1h"
	TTL string `json:"ttl,omitempty"`

	// Similarity threshold for semantic matching (0.0-1.0)
	// +kubebuilder:default="0.95"
	SimilarityThreshold string `json:"similarityThreshold,omitempty"`

	// Cache key includes (model, temperature, etc.)
	// +optional
	KeyIncludes []string `json:"keyIncludes,omitempty"`
}

type RetryConfig struct {
	// Maximum number of retries
	// +kubebuilder:default=3
	MaxRetries int32 `json:"maxRetries,omitempty"`

	// Initial retry delay
	// +kubebuilder:default="1s"
	InitialDelay string `json:"initialDelay,omitempty"`

	// Maximum retry delay
	// +kubebuilder:default="10s"
	MaxDelay string `json:"maxDelay,omitempty"`

	// Backoff multiplier
	// +kubebuilder:default="2"
	BackoffMultiplier string `json:"backoffMultiplier,omitempty"`

	// Retry on specific error codes
	// +optional
	RetryOnErrors []string `json:"retryOnErrors,omitempty"`
}

// LLMRouteStatus defines the observed state of LLMRoute
type LLMRouteStatus struct {
	// Phase of the route
	// +kubebuilder:default=Pending
	Phase LLMRoutePhase `json:"phase"`

	// Conditions
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Currently active target
	// +optional
	ActiveTarget string `json:"activeTarget,omitempty"`

	// Request statistics
	// +optional
	Stats LLMRouteStats `json:"stats,omitempty"`

	// Cost tracking
	// +optional
	CostTracking CostTracking `json:"costTracking,omitempty"`

	// Last updated time
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
}

// +kubebuilder:validation:Enum=Pending;Active;Error;BudgetExceeded
type LLMRoutePhase string

const (
	LLMRoutePhasePending        LLMRoutePhase = "Pending"
	LLMRoutePhaseActive         LLMRoutePhase = "Active"
	LLMRoutePhaseError          LLMRoutePhase = "Error"
	LLMRoutePhaseBudgetExceeded LLMRoutePhase = "BudgetExceeded"
)

type LLMRouteStats struct {
	TotalRequests    int64   `json:"totalRequests"`
	SuccessfulRequests int64 `json:"successfulRequests"`
	FailedRequests   int64   `json:"failedRequests"`
	CachedRequests   int64   `json:"cachedRequests"`
	AverageLatency   string  `json:"averageLatency,omitempty"`
	P95Latency       string  `json:"p95Latency,omitempty"`
	P99Latency       string  `json:"p99Latency,omitempty"`
}

type CostTracking struct {
	TotalCost         string `json:"totalCost,omitempty"`
	CostToday         string `json:"costToday,omitempty"`
	CostThisMonth     string `json:"costThisMonth,omitempty"`
	TotalTokens       int64  `json:"totalTokens"`
	EstimatedMonthlyCost string `json:"estimatedMonthlyCost,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Strategy",type="string",JSONPath=".spec.strategy"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Active Target",type="string",JSONPath=".status.activeTarget"
// +kubebuilder:printcolumn:name="Total Requests",type="integer",JSONPath=".status.stats.totalRequests"
// +kubebuilder:printcolumn:name="Cost Today",type="string",JSONPath=".status.costTracking.costToday"

// LLMRoute is the Schema for the llmroutes API
type LLMRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LLMRouteSpec   `json:"spec,omitempty"`
	Status LLMRouteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LLMRouteList contains a list of LLMRoute
type LLMRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LLMRoute `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LLMRoute{}, &LLMRouteList{})
}


