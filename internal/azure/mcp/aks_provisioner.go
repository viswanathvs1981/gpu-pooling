package mcp

import (
	"context"
	"fmt"
	"time"

	tfv1 "github.com/NexusGPU/tensor-fusion/api/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
)

// AKSProvisioner handles AKS cluster and node provisioning via MCP protocol
type AKSProvisioner struct {
	client        *MCPClient
	subscriptionID string
	resourceGroup  string
	logger         klog.Logger
}

// NewAKSProvisioner creates a new AKS provisioner
func NewAKSProvisioner(subscriptionID, resourceGroup string, auth AuthConfig) (*AKSProvisioner, error) {
	client, err := NewMCPClient(auth)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client: %w", err)
	}

	return &AKSProvisioner{
		client:         client,
		subscriptionID: subscriptionID,
		resourceGroup:  resourceGroup,
		logger:         klog.NewKlogr().WithName("aks-provisioner"),
	}, nil
}

// ProvisionNodeRequest represents a request to provision a new AKS node
type ProvisionNodeRequest struct {
	ClusterName   string
	NodePoolName  string
	VMSize        string
	NodeCount     int32
	MinNodes      int32
	MaxNodes      int32
	GPUCount      int32
	Labels        map[string]string
	Taints        []tfv1.Taint
	AvailabilityZones []string
}

// ProvisionNodeResponse represents the response from provisioning
type ProvisionNodeResponse struct {
	NodePoolID    string
	NodeNames     []string
	ProvisionedAt time.Time
	Status        string
}

// ProvisionNodes provisions new GPU nodes in an AKS cluster
func (p *AKSProvisioner) ProvisionNodes(ctx context.Context, req *ProvisionNodeRequest) (*ProvisionNodeResponse, error) {
	p.logger.Info("Provisioning AKS nodes", "cluster", req.ClusterName, "vmSize", req.VMSize, "count", req.NodeCount)

	// Build MCP request
	mcpReq := &MCPRequest{
		Method: "POST",
		Path:   fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s/agentPools/%s",
			p.subscriptionID, p.resourceGroup, req.ClusterName, req.NodePoolName),
		APIVersion: "2024-01-01",
		Body: map[string]interface{}{
			"properties": map[string]interface{}{
				"count":              req.NodeCount,
				"vmSize":             req.VMSize,
				"minCount":           req.MinNodes,
				"maxCount":           req.MaxNodes,
				"enableAutoScaling":  true,
				"type":               "VirtualMachineScaleSets",
				"mode":               "User",
				"nodeLabels":         req.Labels,
				"nodeTaints":         convertTaintsToStrings(req.Taints),
				"availabilityZones":  req.AvailabilityZones,
				"enableNodePublicIP": false,
			},
		},
	}

	resp, err := p.client.Send(ctx, mcpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to provision nodes: %w", err)
	}

	// Parse response
	nodePoolID, _ := resp.Data["id"].(string)
	
	return &ProvisionNodeResponse{
		NodePoolID:    nodePoolID,
		ProvisionedAt: time.Now(),
		Status:        "Provisioning",
	}, nil
}

// ScaleNodePool scales an existing AKS node pool
func (p *AKSProvisioner) ScaleNodePool(ctx context.Context, clusterName, nodePoolName string, newCount int32) error {
	p.logger.Info("Scaling AKS node pool", "cluster", clusterName, "nodePool", nodePoolName, "newCount", newCount)

	mcpReq := &MCPRequest{
		Method: "PATCH",
		Path:   fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s/agentPools/%s",
			p.subscriptionID, p.resourceGroup, clusterName, nodePoolName),
		APIVersion: "2024-01-01",
		Body: map[string]interface{}{
			"properties": map[string]interface{}{
				"count": newCount,
			},
		},
	}

	_, err := p.client.Send(ctx, mcpReq)
	return err
}

// DeleteNodePool deletes an AKS node pool
func (p *AKSProvisioner) DeleteNodePool(ctx context.Context, clusterName, nodePoolName string) error {
	p.logger.Info("Deleting AKS node pool", "cluster", clusterName, "nodePool", nodePoolName)

	mcpReq := &MCPRequest{
		Method: "DELETE",
		Path:   fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s/agentPools/%s",
			p.subscriptionID, p.resourceGroup, clusterName, nodePoolName),
		APIVersion: "2024-01-01",
	}

	_, err := p.client.Send(ctx, mcpReq)
	return err
}

// GetNodePoolStatus gets the current status of a node pool
func (p *AKSProvisioner) GetNodePoolStatus(ctx context.Context, clusterName, nodePoolName string) (string, error) {
	mcpReq := &MCPRequest{
		Method: "GET",
		Path:   fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s/agentPools/%s",
			p.subscriptionID, p.resourceGroup, clusterName, nodePoolName),
		APIVersion: "2024-01-01",
	}

	resp, err := p.client.Send(ctx, mcpReq)
	if err != nil {
		return "", err
	}

	props, ok := resp.Data["properties"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	status, _ := props["provisioningState"].(string)
	return status, nil
}

// ListAvailableVMSizes lists available VM sizes with GPUs in a region
func (p *AKSProvisioner) ListAvailableVMSizes(ctx context.Context, region string) ([]AzureVMSize, error) {
	mcpReq := &MCPRequest{
		Method: "GET",
		Path:   fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Compute/locations/%s/vmSizes",
			p.subscriptionID, region),
		APIVersion: "2024-03-01",
	}

	resp, err := p.client.Send(ctx, mcpReq)
	if err != nil {
		return nil, err
	}

	var sizes []AzureVMSize
	if value, ok := resp.Data["value"].([]interface{}); ok {
		for _, v := range value {
			if vmSize, ok := v.(map[string]interface{}); ok {
				// Filter for GPU-enabled VMs
				if gpuCount, hasGPU := vmSize["numberOfGpus"].(float64); hasGPU && gpuCount > 0 {
					sizes = append(sizes, parseVMSize(vmSize))
				}
			}
		}
	}

	return sizes, nil
}

// AzureVMSize represents an Azure VM size with GPU specifications
type AzureVMSize struct {
	Name                string
	NumberOfGPUs        int32
	GPUType             string
	MemoryGB            int32
	NumberOfCores       int32
	EstimatedTFlops     resource.Quantity
	EstimatedVRAM       resource.Quantity
	ResourceSKU         string
}

func parseVMSize(data map[string]interface{}) AzureVMSize {
	size := AzureVMSize{}
	
	if name, ok := data["name"].(string); ok {
		size.Name = name
	}
	if gpus, ok := data["numberOfGpus"].(float64); ok {
		size.NumberOfGPUs = int32(gpus)
	}
	if mem, ok := data["memoryInMB"].(float64); ok {
		size.MemoryGB = int32(mem / 1024)
	}
	if cores, ok := data["numberOfCores"].(float64); ok {
		size.NumberOfCores = int32(cores)
	}

	// Estimate GPU specs based on VM size name
	size.EstimatedTFlops = estimateTFlopsFromVMSize(size.Name, size.NumberOfGPUs)
	size.EstimatedVRAM = estimateVRAMFromVMSize(size.Name, size.NumberOfGPUs)
	size.GPUType = extractGPUTypeFromVMSize(size.Name)
	size.ResourceSKU = size.Name

	return size
}

func convertTaintsToStrings(taints []tfv1.Taint) []string {
	result := make([]string, len(taints))
	for i, t := range taints {
		result[i] = fmt.Sprintf("%s=%s:%s", t.Key, t.Value, t.Effect)
	}
	return result
}

// Helper functions to estimate GPU specs from VM size names
func estimateTFlopsFromVMSize(vmSize string, gpuCount int32) resource.Quantity {
	// Standard_NC24ads_A100_v4 -> A100 -> 19.5 TFlops per GPU
	// Standard_NC6s_v3 -> V100 -> 15.7 TFlops per GPU
	// These are approximations based on common Azure GPU VMs
	
	baseFlops := 10.0 // Default
	
	switch {
	case contains(vmSize, "A100"):
		baseFlops = 19.5
	case contains(vmSize, "V100"):
		baseFlops = 15.7
	case contains(vmSize, "T4"):
		baseFlops = 8.1
	case contains(vmSize, "K80"):
		baseFlops = 8.73
	case contains(vmSize, "H100"):
		baseFlops = 60.0
	}

	totalFlops := baseFlops * float64(gpuCount)
	return *resource.NewQuantity(int64(totalFlops*1000), resource.DecimalSI)
}

func estimateVRAMFromVMSize(vmSize string, gpuCount int32) resource.Quantity {
	baseVRAM := 16 // Default GB
	
	switch {
	case contains(vmSize, "A100"):
		baseVRAM = 40
	case contains(vmSize, "V100"):
		baseVRAM = 16
	case contains(vmSize, "T4"):
		baseVRAM = 16
	case contains(vmSize, "K80"):
		baseVRAM = 12
	case contains(vmSize, "H100"):
		baseVRAM = 80
	}

	totalVRAM := baseVRAM * int(gpuCount)
	return *resource.NewQuantity(int64(totalVRAM)*1024*1024*1024, resource.BinarySI)
}

func extractGPUTypeFromVMSize(vmSize string) string {
	switch {
	case contains(vmSize, "A100"):
		return "A100"
	case contains(vmSize, "V100"):
		return "V100"
	case contains(vmSize, "T4"):
		return "T4"
	case contains(vmSize, "K80"):
		return "K80"
	case contains(vmSize, "H100"):
		return "H100"
	default:
		return "Unknown"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && s[0:len(substr)] == substr || len(s) > len(substr) && s[len(s)-len(substr):] == substr)
}


