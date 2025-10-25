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

package vllm

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeploymentConfig contains configuration for vLLM deployment
type DeploymentConfig struct {
	Name          string
	Namespace     string
	BaseModel     string
	GPUCount      int
	VGPUResources string
	Replicas      int32
	ImageRepo     string
	ImageTag      string
	LoraBasePath  string
}

// DeploymentManager handles vLLM Kubernetes deployments
type DeploymentManager struct {
	Client client.Client
}

// NewDeploymentManager creates a new deployment manager
func NewDeploymentManager(client client.Client) *DeploymentManager {
	return &DeploymentManager{
		Client: client,
	}
}

// CreateDeployment creates a vLLM deployment in Kubernetes
func (dm *DeploymentManager) CreateDeployment(ctx context.Context, config *DeploymentConfig) error {
	deployment := dm.buildDeployment(config)

	if err := dm.Client.Create(ctx, deployment); err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create service
	service := dm.buildService(config)
	if err := dm.Client.Create(ctx, service); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	return nil
}

// DeleteDeployment deletes a vLLM deployment
func (dm *DeploymentManager) DeleteDeployment(ctx context.Context, name, namespace string) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := dm.Client.Delete(ctx, deployment); err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := dm.Client.Delete(ctx, service); err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	return nil
}

// buildDeployment constructs a vLLM deployment spec
func (dm *DeploymentManager) buildDeployment(config *DeploymentConfig) *appsv1.Deployment {
	labels := map[string]string{
		"app":       "vllm",
		"model":     config.Name,
		"component": "inference",
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &config.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "vllm",
							Image: fmt.Sprintf("%s:%s", config.ImageRepo, config.ImageTag),
							Args: []string{
								"--model", config.BaseModel,
								"--host", "0.0.0.0",
								"--port", "8000",
								"--tensor-parallel-size", fmt.Sprintf("%d", config.GPUCount),
								"--enable-lora",
								"--lora-modules", config.LoraBasePath,
								"--max-lora-rank", "64",
								"--served-model-name", config.Name,
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8000,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"tensor-fusion.ai/vgpu": resource.MustParse(config.VGPUResources),
									"memory":                 resource.MustParse("16Gi"),
									"cpu":                    resource.MustParse("4"),
								},
								Limits: corev1.ResourceList{
									"tensor-fusion.ai/vgpu": resource.MustParse(config.VGPUResources),
									"memory":                 resource.MustParse("32Gi"),
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt(8000),
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
								TimeoutSeconds:      5,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/v1/models",
										Port: intstr.FromInt(8000),
									},
								},
								InitialDelaySeconds: 15,
								PeriodSeconds:       5,
								TimeoutSeconds:      3,
							},
							Env: []corev1.EnvVar{
								{
									Name:  "VLLM_ENGINE_ITERATION_TIMEOUT_S",
									Value: "60",
								},
								{
									Name:  "VLLM_RPC_TIMEOUT",
									Value: "10000",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "lora-adapters",
									MountPath: config.LoraBasePath,
								},
								{
									Name:      "cache",
									MountPath: "/root/.cache/huggingface",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "lora-adapters",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("%s-lora-storage", config.Name),
								},
							},
						},
						{
							Name: "cache",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: resource.NewQuantity(50*1024*1024*1024, resource.BinarySI), // 50Gi
								},
							},
						},
					},
				},
			},
		},
	}
}

// buildService constructs a Kubernetes service for vLLM
func (dm *DeploymentManager) buildService(config *DeploymentConfig) *corev1.Service {
	labels := map[string]string{
		"app":   "vllm",
		"model": config.Name,
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       8000,
					TargetPort: intstr.FromInt(8000),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}



