#!/bin/bash

set -euo pipefail

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info() { echo -e "${BLUE}[INFO]${NC} $1"; }
ok() { echo -e "${GREEN}[✓]${NC} $1"; }
warn() { echo -e "${YELLOW}[⚠]${NC} $1"; }
fail() { echo -e "${RED}[✗]${NC} $1"; }

cat << 'EOF'
╔════════════════════════════════════════════════════════════════╗
║         Step 1/5: Deploy Core Controller (Operator)           ║
╚════════════════════════════════════════════════════════════════╝
EOF

NAMESPACE="${NAMESPACE:-tensor-fusion-sys}"
RELEASE_NAME="${RELEASE_NAME:-tensor-fusion}"

# Load ACR configuration
if [ ! -f ".acr-config" ]; then
    fail "ACR configuration not found (.acr-config)"
    echo "Please run: ./setup-acr.sh first"
    exit 1
fi

source .acr-config
ok "Loaded ACR: ${ACR_NAME}"

# Create namespace if it doesn't exist
if ! kubectl get namespace "${NAMESPACE}" &> /dev/null; then
    info "Creating namespace: ${NAMESPACE}"
    kubectl create namespace "${NAMESPACE}"
    ok "Namespace created"
fi

# Remove existing release if present
if helm list -n ${NAMESPACE} -a | grep -q "^${RELEASE_NAME}\s"; then
    warn "Existing release '${RELEASE_NAME}' detected. Removing for a clean start..."
    helm uninstall ${RELEASE_NAME} -n ${NAMESPACE} 2>/dev/null || true
    sleep 5
    ok "Previous release removed"
fi

# Label infrastructure namespaces so Helm can adopt them
info "Deploying NexusAI Operator (Core Controller)..."
echo ""

TMP_VALUES=$(mktemp)
trap 'rm -f "${TMP_VALUES}"' EXIT

cat > "${TMP_VALUES}" <<EOF
namespace: ${NAMESPACE}
controller:
  autoScale:
    enabled: false
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/operator
    tag: latest
    pullPolicy: Always
greptime:
  installStandalone: false
nodeDiscovery:
  enabled: false
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/node-discovery
    tag: latest
    pullPolicy: Always
promptOptimizer:
  enabled: false
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/prompt-optimizer
    tag: latest
    pullPolicy: Always
dataopsAgents:
  enabled: false
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/dataops-agents
    tag: latest
    pullPolicy: Always
aiSafety:
  enabled: false
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/aisafety-service
    tag: latest
    pullPolicy: Always
memoryService:
  enabled: false
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/memory-service
    tag: latest
    pullPolicy: Always
modelCatalog:
  enabled: false
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/model-catalog
    tag: latest
    pullPolicy: Always
discoveryAgent:
  enabled: false
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/discovery-agent
    tag: latest
    pullPolicy: Always
mcpServer:
  enabled: false
orchestrator:
  enabled: false
agents:
  enabled: false
msafOrchestrator:
  enabled: false
msafAgents:
  enabled: false
portkey:
  enabled: false
alert:
  enabled: true
EOF

helm upgrade --install ${RELEASE_NAME} ./charts/tensor-fusion \
  --namespace ${NAMESPACE} \
  --create-namespace \
  --values "${TMP_VALUES}" \
  --wait --timeout 5m

if [ $? -eq 0 ]; then
    echo ""
    ok "✅ Step 1/5 Complete: Operator deployed successfully!"
    echo ""
    info "Operator Status:"
    kubectl get deployment -n ${NAMESPACE} -l app.kubernetes.io/component=operator
    echo ""
    info "Next Step: Run ./deploy-2-node-discovery.sh"
else
    fail "Deployment failed!"
    exit 1
fi

