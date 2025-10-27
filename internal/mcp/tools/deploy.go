package tools

import (
	"context"
	"fmt"
	"time"

	tfv1 "github.com/NexusGPU/tensor-fusion/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeployTool handles model deployment operations
type DeployTool struct {
	k8sClient client.Client
	clientset *kubernetes.Clientset
}

// NewDeployTool creates a new deployment tool
func NewDeployTool(k8sClient client.Client, clientset *kubernetes.Clientset) *DeployTool {
	return &DeployTool{
		k8sClient: k8sClient,
		clientset: clientset,
	}
}

// DeployModel deploys a model with vLLM
func (d *DeployTool) DeployModel(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	modelID, ok := params["model_id"].(string)
	if !ok {
		return nil, fmt.Errorf("model_id is required")
	}

	customerID, ok := params["customer_id"].(string)
	if !ok {
		return nil, fmt.Errorf("customer_id is required")
	}

	config, ok := params["config"].(map[string]interface{})
	if !ok {
		config = make(map[string]interface{})
	}

	// Get configuration with defaults
	vgpu := 1.0
	if v, ok := config["vgpu"].(float64); ok {
		vgpu = v
	}

	replicas := int32(1)
	if r, ok := config["replicas"].(float64); ok {
		replicas = int32(r)
	}

	image := "vllm/vllm-openai:latest"
	if img, ok := config["image"].(string); ok {
		image = img
	}

	// Create deployment name
	deploymentName := fmt.Sprintf("vllm-%s", modelID)

	// Create deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: customerID,
			Labels: map[string]string{
				"app":         "vllm",
				"model":       modelID,
				"customer":    customerID,
				"managed-by":  "tensor-fusion",
			},
			Annotations: map[string]string{
				"tensor-fusion.ai/enabled":   "true",
				"tensor-fusion.ai/vgpu":      fmt.Sprintf("%.2f", vgpu),
				"tensor-fusion.ai/pool-name": "default-pool",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":   "vllm",
					"model": modelID,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":      "vllm",
						"model":    modelID,
						"customer": customerID,
					},
					Annotations: map[string]string{
						"tensor-fusion.ai/enabled":   "true",
						"tensor-fusion.ai/vgpu":      fmt.Sprintf("%.2f", vgpu),
						"tensor-fusion.ai/pool-name": "default-pool",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "vllm",
							Image: image,
							Args: []string{
								"--model", modelID,
								"--tensor-parallel-size", "1",
								"--max-model-len", "4096",
								"--gpu-memory-utilization", "0.9",
								"--enable-lora",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8000,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "MODEL_NAME",
									Value: modelID,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("2"),
									corev1.ResourceMemory: resource.MustParse("8Gi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("4"),
									corev1.ResourceMemory: resource.MustParse("16Gi"),
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt(8000),
									},
								},
								InitialDelaySeconds: 60,
								PeriodSeconds:       30,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt(8000),
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
							},
						},
					},
				},
			},
		},
	}

	// Create or update deployment
	if err := d.k8sClient.Create(ctx, deployment); err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: customerID,
			Labels: map[string]string{
				"app":   "vllm",
				"model": modelID,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":   "vllm",
				"model": modelID,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       8000,
					TargetPort: intstr.FromInt(8000),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	if err := d.k8sClient.Create(ctx, service); err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	// Calculate estimated cost (simplified)
	hourlyCost := vgpu * 2.40 // $2.40/hour per vGPU
	monthlyCost := hourlyCost * 720

	return map[string]interface{}{
		"status":         "success",
		"deployment":     deploymentName,
		"namespace":      customerID,
		"endpoint_url":   fmt.Sprintf("http://%s.%s.svc.cluster.local:8000", deploymentName, customerID),
		"vgpu_allocated": vgpu,
		"replicas":       replicas,
		"estimated_cost": map[string]interface{}{
			"hourly":  hourlyCost,
			"monthly": monthlyCost,
		},
		"created_at": time.Now().Format(time.RFC3339),
	}, nil
}

// AllocateGPU allocates GPU resources via GPUNodeClaim
func (d *DeployTool) AllocateGPU(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	vgpuSize, ok := params["vgpu_size"].(float64)
	if !ok {
		return nil, fmt.Errorf("vgpu_size is required")
	}

	poolName, ok := params["pool_name"].(string)
	if !ok {
		poolName = "default-pool"
	}

	duration, ok := params["duration"].(string)
	if !ok {
		duration = "24h"
	}

	// Create GPUNodeClaim
	claimName := fmt.Sprintf("claim-%d", time.Now().Unix())
	claim := &tfv1.GPUNodeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claimName,
			Namespace: "default",
		},
		Spec: tfv1.GPUNodeClaimSpec{
			PoolName: poolName,
			Resources: tfv1.ResourceRequirements{
				GPU: fmt.Sprintf("%.2f", vgpuSize),
			},
		},
	}

	if err := d.k8sClient.Create(ctx, claim); err != nil {
		return nil, fmt.Errorf("failed to create GPU claim: %w", err)
	}

	return map[string]interface{}{
		"status":    "success",
		"claim_id":  claimName,
		"vgpu_size": vgpuSize,
		"pool_name": poolName,
		"duration":  duration,
		"created_at": time.Now().Format(time.RFC3339),
	}, nil
}

// UpdateRouting updates LLM routing configuration
func (d *DeployTool) UpdateRouting(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	routeName, ok := params["route_name"].(string)
	if !ok {
		return nil, fmt.Errorf("route_name is required")
	}

	policy, ok := params["policy"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("policy is required")
	}

	// Create or update LLMRoute
	route := &tfv1.LLMRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeName,
			Namespace: "default",
		},
		Spec: tfv1.LLMRouteSpec{
			Routes: []tfv1.Route{
				{
					Name:     routeName,
					Priority: 100,
				},
			},
		},
	}

	if err := d.k8sClient.Create(ctx, route); err != nil {
		// If already exists, update it
		existingRoute := &tfv1.LLMRoute{}
		if err := d.k8sClient.Get(ctx, client.ObjectKey{Name: routeName, Namespace: "default"}, existingRoute); err != nil {
			return nil, fmt.Errorf("failed to get existing route: %w", err)
		}

		if err := d.k8sClient.Update(ctx, route); err != nil {
			return nil, fmt.Errorf("failed to update route: %w", err)
		}
	}

	return map[string]interface{}{
		"status":     "success",
		"route_name": routeName,
		"policy":     policy,
		"updated_at": time.Now().Format(time.RFC3339),
	}, nil
}

