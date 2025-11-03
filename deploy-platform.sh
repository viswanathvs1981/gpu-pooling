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

# Retry function with exponential backoff for rate limiting
retry_with_backoff() {
    local max_attempts=5
    local timeout=5
    local attempt=1
    local exitCode=0

    while [ $attempt -le $max_attempts ]; do
        if [ $attempt -gt 1 ]; then
            warn "Attempt $attempt/$max_attempts (waiting ${timeout}s due to rate limiting)..."
            sleep $timeout
        fi

        set +e
        "$@"
        exitCode=$?
        set -e

        if [ $exitCode -eq 0 ]; then
            return 0
        fi

        # Check if it's a rate limit error
        if [ $exitCode -ne 0 ] && [ $attempt -lt $max_attempts ]; then
            timeout=$((timeout * 2))
            attempt=$((attempt + 1))
        else
            return $exitCode
        fi
    done

    return $exitCode
}

banner() {
cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║         NexusAI Platform - Helm Chart Deployment              ║
╚════════════════════════════════════════════════════════════════╝
EOF
}

# Configuration
NAMESPACE="${NAMESPACE:-tensor-fusion-sys}"
RELEASE_NAME="${RELEASE_NAME:-tensor-fusion}"
CHART_PATH="./charts/tensor-fusion"
TIMEOUT="${TIMEOUT:-10m}"

banner

# Check prerequisites
info "Checking prerequisites..."
if ! command -v helm &> /dev/null; then
    fail "helm not found. Please install Helm 3.x"
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    fail "kubectl not found. Please install kubectl"
    exit 1
fi
ok "Prerequisites OK"

# Load ACR configuration
if [ ! -f ".acr-config" ]; then
    fail "ACR configuration not found (.acr-config)"
    echo ""
    echo "Please run: ./setup-acr.sh first"
    exit 1
fi

source .acr-config
ok "Loaded ACR configuration: ${ACR_NAME}"

# Check if release already exists
if helm list -n "${NAMESPACE}" | grep -q "^${RELEASE_NAME}"; then
    # Check if release is in failed state
    RELEASE_STATUS=$(helm list -n "${NAMESPACE}" -o json | grep -A 10 "\"name\":\"${RELEASE_NAME}\"" | grep "\"status\"" | cut -d'"' -f4)
    
    if [ "${RELEASE_STATUS}" = "failed" ]; then
        warn "Found failed Helm release '${RELEASE_NAME}', removing it..."
        helm uninstall ${RELEASE_NAME} -n ${NAMESPACE} --wait 2>&1 || true
        kubectl delete secret tf-cloud-vendor-credentials -n ${NAMESPACE} --ignore-not-found 2>&1 || true
        sleep 3
        ok "Failed release removed"
        HELM_ACTION="install"
    else
        warn "Helm release '${RELEASE_NAME}' already exists in namespace '${NAMESPACE}'"
        echo ""
        read -p "Do you want to upgrade it? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            info "Deployment cancelled"
            exit 0
        fi
        HELM_ACTION="upgrade"
    fi
else
    HELM_ACTION="install"
fi

# Create namespace if it doesn't exist
if ! kubectl get namespace "${NAMESPACE}" &> /dev/null; then
    info "Creating namespace: ${NAMESPACE}"
    kubectl create namespace "${NAMESPACE}"
    ok "Namespace created"
else
    info "Namespace ${NAMESPACE} already exists"
fi

# Label existing infrastructure namespaces so Helm can work with them
info "Ensuring infrastructure namespaces are Helm-compatible..."
for ns in greptimedb qdrant storage observability portkey; do
    if kubectl get namespace "${ns}" &> /dev/null; then
        kubectl annotate namespace "${ns}" \
            meta.helm.sh/release-name=${RELEASE_NAME} \
            meta.helm.sh/release-namespace=${NAMESPACE} \
            --overwrite &> /dev/null || true
        kubectl label namespace "${ns}" \
            app.kubernetes.io/managed-by=Helm \
            --overwrite &> /dev/null || true
    fi
done
ok "Infrastructure namespaces labeled"

# Clean up non-Helm resources in tensor-fusion-sys if no Helm release exists
if [ "${HELM_ACTION}" = "install" ]; then
    info "Checking for non-Helm resources in ${NAMESPACE}..."
    if kubectl get deployment portkey-gateway -n ${NAMESPACE} &> /dev/null; then
        warn "Found existing non-Helm resources, cleaning up..."
        kubectl delete deployment portkey-gateway -n ${NAMESPACE} --ignore-not-found &> /dev/null || true
        kubectl delete service portkey-gateway -n ${NAMESPACE} --ignore-not-found &> /dev/null || true
        sleep 3
        ok "Cleanup completed"
    fi
fi

# Deploy/Upgrade NexusAI Platform
info "Deploying NexusAI Platform..."
echo ""
info "Configuration:"
echo "  Release Name: ${RELEASE_NAME}"
echo "  Namespace:    ${NAMESPACE}"
echo "  Chart:        ${CHART_PATH}"
echo "  ACR:          ${ACR_LOGIN_SERVER}"
echo "  Timeout:      ${TIMEOUT}"
echo ""

retry_with_backoff helm ${HELM_ACTION} ${RELEASE_NAME} ${CHART_PATH} \
  --namespace ${NAMESPACE} \
  --create-namespace \
  --set image.repository=${ACR_LOGIN_SERVER}/nexusai \
  --set image.tag=latest \
  --set image.pullPolicy=Always \
  --set operator.image.repository=${ACR_LOGIN_SERVER}/nexusai/operator \
  --set operator.image.tag=latest \
  --set nodeDiscovery.image.repository=${ACR_LOGIN_SERVER}/nexusai/node-discovery \
  --set nodeDiscovery.image.tag=latest \
  --set promptOptimizer.image.repository=${ACR_LOGIN_SERVER}/nexusai/prompt-optimizer \
  --set promptOptimizer.image.tag=latest \
  --set dataopsAgents.image.repository=${ACR_LOGIN_SERVER}/nexusai/dataops-agents \
  --set dataopsAgents.image.tag=latest \
  --set aisafetyService.image.repository=${ACR_LOGIN_SERVER}/nexusai/aisafety-service \
  --set aisafetyService.image.tag=latest \
  --set memoryService.image.repository=${ACR_LOGIN_SERVER}/nexusai/memory-service \
  --set memoryService.image.tag=latest \
  --set modelCatalog.image.repository=${ACR_LOGIN_SERVER}/nexusai/model-catalog \
  --set modelCatalog.image.tag=latest \
  --set discoveryAgent.image.repository=${ACR_LOGIN_SERVER}/nexusai/discovery-agent \
  --set discoveryAgent.image.tag=latest \
  --set pythonAgents.image.repository=${ACR_LOGIN_SERVER}/nexusai/python-agents \
  --set pythonAgents.image.tag=latest \
  --set msafOrchestrator.enabled=false \
  --set msafTrainingAgent.enabled=false \
  --set msafDeploymentAgent.enabled=false \
  --set msafCostAgent.enabled=false \
  --set msafSmallmodelAgent.enabled=false \
  --set msafPipelineAgent.enabled=false \
  --set msafDriftAgent.enabled=false \
  --set msafSecurityAgent.enabled=false \
  --set mcp.enabled=false \
  --set orchestrator.enabled=false \
  --set trainingAgent.enabled=false \
  --set deploymentAgent.enabled=false \
  --set costAgent.enabled=false \
  --wait --timeout ${TIMEOUT}

if [ $? -eq 0 ]; then
    echo ""
    ok "NexusAI Platform deployed successfully!"
    echo ""
    
    # Show deployment status
    info "Deployment Status:"
    echo "═══════════════════════════════════════════════════════════════"
    helm list -n ${NAMESPACE}
    echo ""
    
    info "Pods Status:"
    echo "═══════════════════════════════════════════════════════════════"
    kubectl get pods -n ${NAMESPACE}
    echo ""
    
    info "Services:"
    echo "═══════════════════════════════════════════════════════════════"
    kubectl get svc -n ${NAMESPACE}
    echo ""
    
    ok "Platform is ready!"
    echo ""
    echo "Next Steps:"
    echo "  1. Verify deployment: ./scripts/verify-all.sh"
    echo "  2. Add GPU nodes:     ./add-gpu-node.sh"
    echo "  3. View logs:         kubectl logs -n ${NAMESPACE} -l app.kubernetes.io/name=tensor-fusion"
    echo "  4. Access Grafana:    kubectl port-forward -n observability svc/grafana 3000:80"
    echo ""
else
    fail "Deployment failed!"
    echo ""
    echo "Troubleshooting:"
    echo "  - Check Helm status: helm status ${RELEASE_NAME} -n ${NAMESPACE}"
    echo "  - Check pod logs:    kubectl logs -n ${NAMESPACE} <pod-name>"
    echo "  - Check events:      kubectl get events -n ${NAMESPACE} --sort-by='.lastTimestamp'"
    echo ""
    exit 1
fi

