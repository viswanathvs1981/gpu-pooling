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

package lora

import (
	"context"
	"fmt"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Adapter represents a LoRA adapter
type Adapter struct {
	Name       string                 `json:"name"`
	Path       string                 `json:"path"`
	BaseModel  string                 `json:"base_model"`
	Rank       int                    `json:"rank"`
	Alpha      int                    `json:"alpha"`
	TargetModules []string            `json:"target_modules"`
	Size       int64                  `json:"size_bytes"`
	Created    metav1.Time            `json:"created"`
	Metadata   map[string]string      `json:"metadata,omitempty"`
}

// Registry manages LoRA adapters
type Registry struct {
	Client       client.Client
	StoragePath  string
	Namespace    string
}

// NewRegistry creates a new adapter registry
func NewRegistry(client client.Client, storagePath, namespace string) *Registry {
	return &Registry{
		Client:      client,
		StoragePath: storagePath,
		Namespace:   namespace,
	}
}

// RegisterAdapter registers a new LoRA adapter
func (r *Registry) RegisterAdapter(ctx context.Context, adapter *Adapter) error {
	// Create ConfigMap to store adapter metadata
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("lora-adapter-%s", adapter.Name),
			Namespace: r.Namespace,
			Labels: map[string]string{
				"app":        "tensor-fusion",
				"component":  "lora-adapter",
				"base-model": adapter.BaseModel,
			},
		},
		Data: map[string]string{
			"name":       adapter.Name,
			"path":       adapter.Path,
			"base_model": adapter.BaseModel,
			"rank":       fmt.Sprintf("%d", adapter.Rank),
			"alpha":      fmt.Sprintf("%d", adapter.Alpha),
		},
	}

	if err := r.Client.Create(ctx, cm); err != nil {
		return fmt.Errorf("failed to register adapter: %w", err)
	}

	return nil
}

// GetAdapter retrieves an adapter by name
func (r *Registry) GetAdapter(ctx context.Context, name string) (*Adapter, error) {
	cm := &corev1.ConfigMap{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      fmt.Sprintf("lora-adapter-%s", name),
		Namespace: r.Namespace,
	}, cm); err != nil {
		return nil, fmt.Errorf("adapter not found: %w", err)
	}

	adapter := &Adapter{
		Name:      cm.Data["name"],
		Path:      cm.Data["path"],
		BaseModel: cm.Data["base_model"],
	}

	return adapter, nil
}

// ListAdapters lists all registered adapters
func (r *Registry) ListAdapters(ctx context.Context, baseModel string) ([]*Adapter, error) {
	cmList := &corev1.ConfigMapList{}
	listOpts := []client.ListOption{
		client.InNamespace(r.Namespace),
		client.MatchingLabels{"component": "lora-adapter"},
	}

	if baseModel != "" {
		listOpts = append(listOpts, client.MatchingLabels{"base-model": baseModel})
	}

	if err := r.Client.List(ctx, cmList, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list adapters: %w", err)
	}

	adapters := make([]*Adapter, 0, len(cmList.Items))
	for _, cm := range cmList.Items {
		adapter := &Adapter{
			Name:      cm.Data["name"],
			Path:      cm.Data["path"],
			BaseModel: cm.Data["base_model"],
		}
		adapters = append(adapters, adapter)
	}

	return adapters, nil
}

// DeleteAdapter removes an adapter from the registry
func (r *Registry) DeleteAdapter(ctx context.Context, name string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("lora-adapter-%s", name),
			Namespace: r.Namespace,
		},
	}

	if err := r.Client.Delete(ctx, cm); err != nil {
		return fmt.Errorf("failed to delete adapter: %w", err)
	}

	return nil
}

// GetAdapterPath returns the full path to an adapter
func (r *Registry) GetAdapterPath(adapterName string) string {
	return filepath.Join(r.StoragePath, adapterName)
}

// EnsureStorage creates PVC for LoRA adapter storage
func (r *Registry) EnsureStorage(ctx context.Context, size string) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lora-adapter-storage",
			Namespace: r.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(size),
				},
			},
		},
	}

	if err := r.Client.Create(ctx, pvc); err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	return nil
}



