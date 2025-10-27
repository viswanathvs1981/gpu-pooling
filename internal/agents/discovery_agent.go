package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	tfv1 "github.com/NexusGPU/tensor-fusion/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DiscoveryAgent discovers and monitors LLM endpoints
type DiscoveryAgent struct {
	k8sClient   client.Client
	clientset   *kubernetes.Clientset
	httpClient  *http.Client
	endpoints   map[string]*DiscoveredEndpoint
	healthCheck *HealthChecker
}

// DiscoveredEndpoint represents a discovered LLM endpoint
type DiscoveredEndpoint struct {
	Name         string
	URL          string
	Type         string
	LastSeen     time.Time
	Health       string
	LatencyP99   time.Duration
	ErrorRate    float64
}

// HealthChecker performs health checks on endpoints
type HealthChecker struct {
	client *http.Client
}

// NewDiscoveryAgent creates a new LLM discovery agent
func NewDiscoveryAgent(k8sClient client.Client, clientset *kubernetes.Clientset) *DiscoveryAgent {
	return &DiscoveryAgent{
		k8sClient: k8sClient,
		clientset: clientset,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		endpoints: make(map[string]*DiscoveredEndpoint),
		healthCheck: &HealthChecker{
			client: &http.Client{Timeout: 5 * time.Second},
		},
	}
}

// Start starts the discovery agent
func (da *DiscoveryAgent) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting LLM Discovery Agent")

	// Start service watcher
	go da.watchServices(ctx)

	// Start periodic health checker
	go da.periodicHealthCheck(ctx)

	<-ctx.Done()
	logger.Info("LLM Discovery Agent shutting down")
	return nil
}

// watchServices watches for Kubernetes services with LLM labels
func (da *DiscoveryAgent) watchServices(ctx context.Context) {
	logger := log.Log.WithName("discovery-watch")

	// Watch services across all namespaces
	watcher, err := da.clientset.CoreV1().Services(corev1.NamespaceAll).Watch(ctx, metav1.ListOptions{
		LabelSelector: "llm-provider=true",
	})
	if err != nil {
		logger.Error(err, "Failed to start service watcher")
		return
	}
	defer watcher.Stop()

	logger.Info("Watching for LLM services with label: llm-provider=true")

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-watcher.ResultChan():
			if event.Type == watch.Added || event.Type == watch.Modified {
				service, ok := event.Object.(*corev1.Service)
				if !ok {
					continue
				}

				logger.Info("Discovered LLM service", "name", service.Name, "namespace", service.Namespace)
				da.handleDiscoveredService(ctx, service)
			} else if event.Type == watch.Deleted {
				service, ok := event.Object.(*corev1.Service)
				if ok {
					da.handleRemovedService(ctx, service)
				}
			}
		}
	}
}

// handleDiscoveredService processes a newly discovered service
func (da *DiscoveryAgent) handleDiscoveredService(ctx context.Context, service *corev1.Service) {
	logger := log.Log.WithName("discovery-agent")

	// Build service URL
	url := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d",
		service.Name,
		service.Namespace,
		service.Spec.Ports[0].Port,
	)

	// Determine type from labels
	llmType := service.Labels["llm-type"]
	if llmType == "" {
		llmType = "custom"
	}

	// Create or update endpoint
	endpoint := &DiscoveredEndpoint{
		Name:     fmt.Sprintf("%s-%s", service.Namespace, service.Name),
		URL:      url,
		Type:     llmType,
		LastSeen: time.Now(),
		Health:   "unknown",
	}

	da.endpoints[endpoint.Name] = endpoint

	// Perform initial health check
	go da.checkEndpointHealth(ctx, endpoint)

	// Create LLMEndpoint CR
	llmEndpoint := &tfv1.LLMEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name: endpoint.Name,
		},
		Spec: tfv1.LLMEndpointSpec{
			Name:     endpoint.Name,
			URL:      url,
			Type:     llmType,
			Provider: "self-hosted",
			Priority: 50,
		},
	}

	if err := da.k8sClient.Create(ctx, llmEndpoint); err != nil {
		logger.Error(err, "Failed to create LLMEndpoint CR", "name", endpoint.Name)
	} else {
		logger.Info("Created LLMEndpoint CR", "name", endpoint.Name, "url", url)
	}
}

// handleRemovedService processes a removed service
func (da *DiscoveryAgent) handleRemovedService(ctx context.Context, service *corev1.Service) {
	logger := log.Log.WithName("discovery-agent")
	name := fmt.Sprintf("%s-%s", service.Namespace, service.Name)

	delete(da.endpoints, name)

	// Delete LLMEndpoint CR
	llmEndpoint := &tfv1.LLMEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	if err := da.k8sClient.Delete(ctx, llmEndpoint); err != nil {
		logger.Error(err, "Failed to delete LLMEndpoint CR", "name", name)
	} else {
		logger.Info("Deleted LLMEndpoint CR", "name", name)
	}
}

// periodicHealthCheck runs health checks periodically
func (da *DiscoveryAgent) periodicHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, endpoint := range da.endpoints {
				go da.checkEndpointHealth(ctx, endpoint)
			}
		}
	}
}

// checkEndpointHealth performs a health check on an endpoint
func (da *DiscoveryAgent) checkEndpointHealth(ctx context.Context, endpoint *DiscoveredEndpoint) {
	logger := log.Log.WithName("health-check")

	start := time.Now()
	
	// Try /health endpoint first (common pattern)
	healthURL := endpoint.URL + "/health"
	resp, err := da.httpClient.Get(healthURL)
	
	if err != nil {
		// Try /v1/models (OpenAI-compatible)
		modelsURL := endpoint.URL + "/v1/models"
		resp, err = da.httpClient.Get(modelsURL)
	}

	latency := time.Since(start)

	if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
		endpoint.Health = "unhealthy"
		endpoint.ErrorRate = 1.0
		logger.Info("Endpoint unhealthy", "name", endpoint.Name, "error", err)
	} else {
		endpoint.Health = "healthy"
		endpoint.ErrorRate = 0.0
		endpoint.LatencyP99 = latency
		logger.Info("Endpoint healthy", "name", endpoint.Name, "latency", latency)
	}

	if resp != nil {
		resp.Body.Close()
	}

	// Update LLMEndpoint CR status
	da.updateEndpointStatus(ctx, endpoint)
}

// updateEndpointStatus updates the LLMEndpoint CR status
func (da *DiscoveryAgent) updateEndpointStatus(ctx context.Context, endpoint *DiscoveredEndpoint) {
	llmEndpoint := &tfv1.LLMEndpoint{}
	if err := da.k8sClient.Get(ctx, client.ObjectKey{Name: endpoint.Name}, llmEndpoint); err != nil {
		return
	}

	now := metav1.Now()
	llmEndpoint.Status.Phase = endpoint.Health
	llmEndpoint.Status.LastHealthCheck = &now
	llmEndpoint.Status.Health = tfv1.HealthStatus{
		Status:    endpoint.Health,
		ErrorRate: endpoint.ErrorRate,
		Latency:   endpoint.LatencyP99.String(),
		Capacity:  100,
	}

	if err := da.k8sClient.Status().Update(ctx, llmEndpoint); err != nil {
		log.Log.Error(err, "Failed to update LLMEndpoint status")
	}
}

