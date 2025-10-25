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

package training

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TrainingConfig contains configuration for a LoRA training job
type TrainingConfig struct {
	Name               string
	Namespace          string
	BaseModel          string
	DatasetPath        string
	OutputPath         string
	Rank               int
	Alpha              int
	LearningRate       float64
	BatchSize          int
	Epochs             int
	VGPUResources      string
	TargetModules      []string
	TrainingImage      string
	TrainingImageTag   string
}

// JobManager manages training jobs
type JobManager struct {
	Client client.Client
}

// NewJobManager creates a new job manager
func NewJobManager(client client.Client) *JobManager {
	return &JobManager{
		Client: client,
	}
}

// CreateTrainingJob creates a Kubernetes Job for LoRA training
func (jm *JobManager) CreateTrainingJob(ctx context.Context, config *TrainingConfig) error {
	job := jm.buildTrainingJob(config)

	if err := jm.Client.Create(ctx, job); err != nil {
		return fmt.Errorf("failed to create training job: %w", err)
	}

	return nil
}

// GetJobStatus retrieves the status of a training job
func (jm *JobManager) GetJobStatus(ctx context.Context, name, namespace string) (*batchv1.JobStatus, error) {
	job := &batchv1.Job{}
	if err := jm.Client.Get(ctx, client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, job); err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &job.Status, nil
}

// DeleteJob deletes a training job
func (jm *JobManager) DeleteJob(ctx context.Context, name, namespace string) error {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	propagationPolicy := metav1.DeletePropagationBackground
	if err := jm.Client.Delete(ctx, job, &client.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	}); err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	return nil
}

// buildTrainingJob constructs a Kubernetes Job for LoRA training
func (jm *JobManager) buildTrainingJob(config *TrainingConfig) *batchv1.Job {
	labels := map[string]string{
		"app":        "tensor-fusion",
		"component":  "training",
		"base-model": config.BaseModel,
		"job-name":   config.Name,
	}

	backoffLimit := int32(3)
	completions := int32(1)
	parallelism := int32(1)

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Completions:  &completions,
			Parallelism:  &parallelism,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:  "trainer",
							Image: fmt.Sprintf("%s:%s", config.TrainingImage, config.TrainingImageTag),
							Command: []string{
								"python",
								"-m",
								"train_lora",
							},
							Args: []string{
								"--base-model", config.BaseModel,
								"--data-path", config.DatasetPath,
								"--output-dir", config.OutputPath,
								"--lora-r", fmt.Sprintf("%d", config.Rank),
								"--lora-alpha", fmt.Sprintf("%d", config.Alpha),
								"--learning-rate", fmt.Sprintf("%f", config.LearningRate),
								"--batch-size", fmt.Sprintf("%d", config.BatchSize),
								"--num-epochs", fmt.Sprintf("%d", config.Epochs),
								"--target-modules", joinTargetModules(config.TargetModules),
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"tensor-fusion.ai/vgpu": resource.MustParse(config.VGPUResources),
									"memory":                 resource.MustParse("32Gi"),
									"cpu":                    resource.MustParse("8"),
								},
								Limits: corev1.ResourceList{
									"tensor-fusion.ai/vgpu": resource.MustParse(config.VGPUResources),
									"memory":                 resource.MustParse("64Gi"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "dataset",
									MountPath: "/data",
								},
								{
									Name:      "output",
									MountPath: "/output",
								},
								{
									Name:      "cache",
									MountPath: "/root/.cache/huggingface",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "WANDB_DISABLED",
									Value: "true",
								},
								{
									Name:  "TRANSFORMERS_OFFLINE",
									Value: "0",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "dataset",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "training-datasets",
								},
							},
						},
						{
							Name: "output",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "lora-adapter-storage",
								},
							},
						},
						{
							Name: "cache",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: resource.NewQuantity(100*1024*1024*1024, resource.BinarySI), // 100Gi
								},
							},
						},
					},
				},
			},
		},
	}
}

func joinTargetModules(modules []string) string {
	if len(modules) == 0 {
		return "q_proj,v_proj"
	}
	result := modules[0]
	for i := 1; i < len(modules); i++ {
		result += "," + modules[i]
	}
	return result
}



