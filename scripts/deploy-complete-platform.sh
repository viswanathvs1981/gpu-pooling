#!/bin/bash

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║   TensorFusion Complete Platform Deployment                    ║"
echo "╚════════════════════════════════════════════════════════════════╝"

# Step 1: Check if infrastructure exists
log_info "Checking infrastructure..."
if ! kubectl get namespace greptimedb &>/dev/null; then
    log_warning "GreptimeDB not found, running infrastructure provisioning..."
    cd infrastructure && ./provision-all.sh
    cd ..
else
    log_success "Infrastructure already deployed"
fi

# Step 2: Build and push Docker images
log_info "Building and pushing TensorFusion images..."
RESOURCE_GROUP="tensor-fusion-rg"
ACR_NAME=$(az acr list --resource-group $RESOURCE_GROUP --query "[0].name" -o tsv 2>/dev/null || echo "")

if [ -z "$ACR_NAME" ]; then
    log_error "No ACR found. Please run infrastructure provisioning first."
    exit 1
fi

log_info "Building operator image..."
az acr build \
    --registry "$ACR_NAME" \
    --image tensor-fusion/operator:latest \
    --file dockerfile/operator.Dockerfile \
    .

log_info "Building node-discovery image..."
az acr build \
    --registry "$ACR_NAME" \
    --image tensor-fusion/node-discovery:latest \
    --file dockerfile/node-discovery.Dockerfile \
    .

log_success "Images built and pushed!"

# Step 3: Deploy TensorFusion controllers
log_info "Deploying TensorFusion controllers..."
ACR_LOGIN_SERVER="${ACR_NAME}.azurecr.io"

cat > /tmp/tensor-fusion-values.yaml <<EOF
image:
  repository: ${ACR_LOGIN_SERVER}/tensor-fusion/operator
  tag: latest
  pullPolicy: Always

nodeDiscovery:
  image:
    repository: ${ACR_LOGIN_SERVER}/tensor-fusion/node-discovery
    tag: latest
    pullPolicy: Always

greptime:
  host: greptimedb-standalone.greptimedb.svc.cluster.local
  port: 4000

redis:
  host: redis-master.storage.svc.cluster.local
  port: 6379

qdrant:
  host: qdrant.qdrant.svc.cluster.local
  port: 6333

portkey:
  host: portkey-gateway.portkey.svc.cluster.local
  port: 8787

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi
EOF

helm upgrade --install tensor-fusion ./charts/tensor-fusion \
    --namespace tensor-fusion-sys \
    --create-namespace \
    --values /tmp/tensor-fusion-values.yaml \
    --wait \
    --timeout 10m

log_success "TensorFusion controllers deployed!"

# Step 4: Apply enhanced RBAC
log_info "Applying enhanced RBAC..."
if [ -f "rbac-enhanced.yaml" ]; then
    kubectl apply -f rbac-enhanced.yaml
    log_success "Enhanced RBAC applied!"
else
    log_warning "rbac-enhanced.yaml not found, skipping..."
fi

# Step 5: Deploy vLLM (optional)
read -p "Deploy vLLM inference engine? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    log_info "Deploying vLLM..."
    bash scripts/deploy-vllm.sh
fi

# Step 6: Create sample resources
log_info "Creating sample resources..."

# Create GPUPool
cat <<EOF | kubectl apply -f -
apiVersion: tensor-fusion.ai/v1
kind: GPUPool
metadata:
  name: default-pool
  namespace: tensor-fusion-sys
spec:
  gpuType: "NVIDIA-A100-SXM4-80GB"
  capacityConfig:
    maxGPUCount: 10
    reservedGPUCount: 2
  nodeManagerConfig:
    provisioningEnabled: true
    autoScalingEnabled: true
  qosConfig:
    enableQoS: true
    priorityLevels:
      - name: "high"
        priority: 100
      - name: "medium"
        priority: 50
      - name: "low"
        priority: 10
EOF

log_success "Sample GPUPool created!"

# Summary
echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║   Deployment Complete!                                         ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""
log_info "Platform Status:"
kubectl get pods -n tensor-fusion-sys
echo ""
log_info "Next steps:"
echo "  1. Test inference: bash scripts/test-inference.sh"
echo "  2. Create workloads: kubectl apply -f examples/"
echo "  3. Monitor: kubectl logs -n tensor-fusion-sys -l app=tensor-fusion"
echo ""
log_success "Platform is ready!"



