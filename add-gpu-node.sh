#!/bin/bash

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║   TensorFusion - Add GPU Node Pool                            ║"
echo "╚════════════════════════════════════════════════════════════════╝"

# Configuration
RESOURCE_GROUP="${RESOURCE_GROUP:-tensor-fusion-rg}"
CLUSTER_NAME="${CLUSTER_NAME:-tensor-fusion-aks}"
GPU_NODE_SIZE="${GPU_NODE_SIZE:-Standard_NC4as_T4_v3}"  # Smallest modern GPU
NODE_COUNT="${NODE_COUNT:-1}"
MIN_COUNT="${MIN_COUNT:-0}"
MAX_COUNT="${MAX_COUNT:-2}"
USE_SPOT="${USE_SPOT:-false}"

echo -e "${BLUE}Configuration:${NC}"
echo "  Resource Group: $RESOURCE_GROUP"
echo "  Cluster: $CLUSTER_NAME"
echo "  GPU VM Size: $GPU_NODE_SIZE"
echo "  Node Count: $NODE_COUNT"
echo "  Autoscaler: Min=$MIN_COUNT, Max=$MAX_COUNT"
echo "  Use Spot: $USE_SPOT"
echo ""

# Check current quota
echo -e "${BLUE}Checking current vCPU usage...${NC}"
CURRENT_VCPUS=$(az vm list-usage --location eastus --query "[?localName=='Total Regional vCPUs'].currentValue" -o tsv)
VCPU_LIMIT=$(az vm list-usage --location eastus --query "[?localName=='Total Regional vCPUs'].limit" -o tsv)
AVAILABLE_VCPUS=$((VCPU_LIMIT - CURRENT_VCPUS))

echo -e "  Current: ${CURRENT_VCPUS}/${VCPU_LIMIT} vCPUs"
echo -e "  Available: ${AVAILABLE_VCPUS} vCPUs"

# Get vCPU requirement for chosen VM size
case $GPU_NODE_SIZE in
  Standard_NC4as_T4_v3)
    REQUIRED_VCPUS=$((4 * NODE_COUNT))
    GPU_TYPE="NVIDIA T4"
    ;;
  Standard_NC8as_T4_v3)
    REQUIRED_VCPUS=$((8 * NODE_COUNT))
    GPU_TYPE="NVIDIA T4"
    ;;
  Standard_NV4as_v4)
    REQUIRED_VCPUS=$((4 * NODE_COUNT))
    GPU_TYPE="AMD Radeon MI25"
    ;;
  *)
    echo -e "${YELLOW}Warning: Unknown VM size, proceeding anyway${NC}"
    REQUIRED_VCPUS=4
    GPU_TYPE="Unknown"
    ;;
esac

echo -e "  Required for $NODE_COUNT node(s): ${REQUIRED_VCPUS} vCPUs"
echo -e "  GPU Type: ${GPU_TYPE}"
echo ""

if [ $AVAILABLE_VCPUS -lt $REQUIRED_VCPUS ]; then
  echo -e "${RED}ERROR: Insufficient vCPU quota!${NC}"
  echo -e "  Need: $REQUIRED_VCPUS vCPUs"
  echo -e "  Available: $AVAILABLE_VCPUS vCPUs"
  echo -e "  Shortfall: $((REQUIRED_VCPUS - AVAILABLE_VCPUS)) vCPUs"
  echo ""
  echo -e "${YELLOW}Options:${NC}"
  echo "  1. Request quota increase: https://portal.azure.com → Quotas"
  echo "  2. Deploy to westus2 (18 vCPUs available)"
  echo "  3. Scale down system nodes: az aks nodepool scale -g $RESOURCE_GROUP --cluster-name $CLUSTER_NAME --name system --node-count 1"
  echo ""
  exit 1
fi

echo -e "${GREEN}✓ Sufficient vCPU quota available${NC}"
echo ""

# Build command
CMD="az aks nodepool add \
  -g $RESOURCE_GROUP \
  --cluster-name $CLUSTER_NAME \
  --name gpu \
  --node-vm-size $GPU_NODE_SIZE \
  --node-count $NODE_COUNT \
  --enable-cluster-autoscaler \
  --min-count $MIN_COUNT \
  --max-count $MAX_COUNT \
  --node-taints nvidia.com/gpu=present:NoSchedule \
  --labels pool=gpu"

if [ "$USE_SPOT" = "true" ]; then
  echo -e "${YELLOW}Using spot instances (can be evicted, 90% cheaper)${NC}"
  CMD="$CMD --priority Spot --eviction-policy Delete --spot-max-price -1"
fi

echo -e "${BLUE}Adding GPU node pool...${NC}"
echo "$CMD"
echo ""

eval $CMD

echo ""
echo -e "${GREEN}✓ GPU node pool added successfully!${NC}"
echo ""
echo "Verify with:"
echo "  kubectl get nodes -l pool=gpu"
echo ""
echo "Deploy a test workload:"
echo "  kubectl apply -f examples/02-fractional-gpu-pod.yaml"
echo ""
echo -e "${BLUE}TensorFusion will now manage both CPU and GPU nodes!${NC}"

