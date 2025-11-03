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
║         Step 5/5: Deploy Python Agents (Final Step)           ║
╚════════════════════════════════════════════════════════════════╝
EOF

NAMESPACE="${NAMESPACE:-tensor-fusion-sys}"
RELEASE_NAME="${RELEASE_NAME:-tensor-fusion}"

# Load ACR configuration
if [ ! -f ".acr-config" ]; then
    fail "ACR configuration not found (.acr-config)"
    exit 1
fi

source .acr-config
ok "Loaded ACR: ${ACR_NAME}"

if ! helm list -n ${NAMESPACE} | grep -q "^${RELEASE_NAME}\s"; then
    fail "Release '${RELEASE_NAME}' not found. Please run previous steps first."
    exit 1
fi

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
  enabled: true
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/node-discovery
    tag: latest
    pullPolicy: Always
memoryService:
  enabled: true
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/memory-service
    tag: latest
    pullPolicy: Always
modelCatalog:
  enabled: true
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/model-catalog
    tag: latest
    pullPolicy: Always
discoveryAgent:
  enabled: true
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/discovery-agent
    tag: latest
    pullPolicy: Always
promptOptimizer:
  enabled: true
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/prompt-optimizer
    tag: latest
    pullPolicy: Always
dataopsAgents:
  enabled: true
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/dataops-agents
    tag: latest
    pullPolicy: Always
aiSafety:
  enabled: true
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/aisafety-service
    tag: latest
    pullPolicy: Always
mcpServer:
  enabled: false
orchestrator:
  enabled: false
agents:
  enabled: false
msafOrchestrator:
  enabled: true
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/python-agents
    tag: latest
    pullPolicy: Always
msafAgents:
  enabled: true
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/python-agents
    tag: latest
    pullPolicy: Always
portkey:
  enabled: false
alert:
  enabled: true
EOF

info "Deploying Python Agents (Microsoft Agent Framework)..."
echo ""

helm upgrade --install ${RELEASE_NAME} ./charts/tensor-fusion \
  --namespace ${NAMESPACE} \
  --values "${TMP_VALUES}" \
  --wait --timeout 5m

if [ $? -eq 0 ]; then
    echo ""
    ok "✅ Step 5/5 Complete: Python Agents deployed successfully!"
    echo ""
    ok "🎉 ALL NEXUSAI PLATFORM COMPONENTS DEPLOYED!"
    echo ""
    
    info "Complete Deployment Status:"
    echo "═══════════════════════════════════════════════════════════════"
    kubectl get deployment -n ${NAMESPACE}
    echo ""
    kubectl get daemonset -n ${NAMESPACE}
    echo ""
    
    info "Helm Releases:"
    echo "═══════════════════════════════════════════════════════════════"
    helm list -n ${NAMESPACE}
    echo ""
    
    ok "NexusAI Platform Deployment Complete!"
    echo ""
    echo "Next Steps:"
    echo "  1. Verify deployment: ./scripts/verify-all.sh"
    echo "  2. Check pod logs:    kubectl logs -n ${NAMESPACE} <pod-name>"
    echo "  3. Access services:   kubectl get svc -n ${NAMESPACE}"
    echo ""
else
    fail "Deployment failed!"
    exit 1
fi

