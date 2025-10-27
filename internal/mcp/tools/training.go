package tools

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TrainingTool handles model training operations
type TrainingTool struct {
	k8sClient client.Client
	clientset *kubernetes.Clientset
}

// NewTrainingTool creates a new training tool
func NewTrainingTool(k8sClient client.Client, clientset *kubernetes.Clientset) *TrainingTool {
	return &TrainingTool{
		k8sClient: k8sClient,
		clientset: clientset,
	}
}

// StartTraining starts a LoRA training job
func (t *TrainingTool) StartTraining(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	datasetPath, ok := params["dataset_path"].(string)
	if !ok {
		return nil, fmt.Errorf("dataset_path is required")
	}

	baseModel, ok := params["base_model"].(string)
	if !ok {
		baseModel = "meta-llama/Llama-3-8b"
	}

	loraConfig, ok := params["lora_config"].(map[string]interface{})
	if !ok {
		loraConfig = make(map[string]interface{})
	}

	// Get LoRA config with defaults
	rank := 32
	if r, ok := loraConfig["rank"].(float64); ok {
		rank = int(r)
	}

	alpha := 64
	if a, ok := loraConfig["alpha"].(float64); ok {
		alpha = int(a)
	}

	// Create job name
	jobName := fmt.Sprintf("training-%d", time.Now().Unix())

	// Training script
	trainingScript := fmt.Sprintf(`
#!/bin/bash
set -e

echo "Installing dependencies..."
pip install transformers peft datasets accelerate bitsandbytes

echo "Starting LoRA training..."
python3 <<EOF
import torch
from transformers import AutoModelForCausalLM, AutoTokenizer, Trainer, TrainingArguments
from peft import LoraConfig, get_peft_model
from datasets import load_dataset

print("Loading base model: %s")
model = AutoModelForCausalLM.from_pretrained(
    "%s",
    torch_dtype=torch.float16,
    device_map="auto"
)
tokenizer = AutoTokenizer.from_pretrained("%s")

print("Configuring LoRA...")
lora_config = LoraConfig(
    r=%d,
    lora_alpha=%d,
    target_modules=["q_proj", "v_proj", "k_proj", "o_proj"],
    lora_dropout=0.1,
    bias="none",
    task_type="CAUSAL_LM"
)

model = get_peft_model(model, lora_config)
model.print_trainable_parameters()

print("Loading dataset from: %s")
# For now, use a sample dataset
dataset = load_dataset("json", data_files="%s", split="train[:1000]")

print("Starting training...")
training_args = TrainingArguments(
    output_dir="/output/adapter",
    per_device_train_batch_size=8,
    learning_rate=3e-4,
    num_train_epochs=3,
    fp16=True,
    logging_steps=10,
    save_strategy="epoch",
)

trainer = Trainer(
    model=model,
    args=training_args,
    train_dataset=dataset,
)

trainer.train()

print("Saving adapter...")
model.save_pretrained("/output/adapter")
tokenizer.save_pretrained("/output/adapter")

print("Training complete!")
EOF

echo "Adapter saved to /output/adapter"
ls -lh /output/adapter/
`, baseModel, baseModel, baseModel, rank, alpha, datasetPath, datasetPath)

	// Create training job
	backoffLimit := int32(3)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: "default",
			Labels: map[string]string{
				"app":        "lora-training",
				"managed-by": "tensor-fusion",
			},
			Annotations: map[string]string{
				"tensor-fusion.ai/enabled":   "true",
				"tensor-fusion.ai/vgpu":      "0.5",
				"tensor-fusion.ai/pool-name": "default-pool",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "lora-training",
						"job": jobName,
					},
					Annotations: map[string]string{
						"tensor-fusion.ai/enabled":   "true",
						"tensor-fusion.ai/vgpu":      "0.5",
						"tensor-fusion.ai/pool-name": "default-pool",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    "training",
							Image:   "pytorch/pytorch:2.0.1-cuda11.8-cudnn8-runtime",
							Command: []string{"/bin/bash", "-c"},
							Args:    []string{trainingScript},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("4"),
									corev1.ResourceMemory: resource.MustParse("16Gi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("8"),
									corev1.ResourceMemory: resource.MustParse("32Gi"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "output",
									MountPath: "/output",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "output",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	// Create the job
	if err := t.k8sClient.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create training job: %w", err)
	}

	// Calculate estimated time and cost
	estimatedTime := "2-3 hours"
	estimatedCost := 100.0 // $100 for ~2.5 hours on 0.5 vGPU

	return map[string]interface{}{
		"status":         "started",
		"job_id":         jobName,
		"namespace":      "default",
		"dataset_path":   datasetPath,
		"base_model":     baseModel,
		"lora_config": map[string]interface{}{
			"rank":  rank,
			"alpha": alpha,
		},
		"estimated_time": estimatedTime,
		"estimated_cost": estimatedCost,
		"created_at":     time.Now().Format(time.RFC3339),
	}, nil
}

