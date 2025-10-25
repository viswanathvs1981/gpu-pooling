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
echo "║   TensorFusion Platform Verification                          ║"
echo "╚════════════════════════════════════════════════════════════════╝"

FAILED=0

# Check namespaces
log_info "Checking namespaces..."
NAMESPACES=("tensor-fusion-sys" "greptimedb" "storage" "observability" "qdrant" "portkey")
for ns in "${NAMESPACES[@]}"; do
    if kubectl get namespace $ns &>/dev/null; then
        log_success "Namespace $ns exists"
    else
        log_error "Namespace $ns missing"
        FAILED=1
    fi
done

# Check TensorFusion controllers
log_info "Checking TensorFusion controllers..."
CONTROLLERS=$(kubectl get pods -n tensor-fusion-sys -l app=tensor-fusion -o jsonpath='{.items[*].status.phase}' 2>/dev/null || echo "")
if [[ "$CONTROLLERS" == *"Running"* ]]; then
    log_success "TensorFusion controllers running"
else
    log_warning "TensorFusion controllers not ready"
fi

# Check CRDs
log_info "Checking Custom Resource Definitions..."
CRDS=(
    "gpupools.tensor-fusion.ai"
    "gpunodes.tensor-fusion.ai"
    "tensorfusionworkloads.tensor-fusion.ai"
    "llmroutes.tensor-fusion.ai"
    "azuregpusources.tensor-fusion.ai"
    "workloadintelligences.tensor-fusion.ai"
)

for crd in "${CRDS[@]}"; do
    if kubectl get crd $crd &>/dev/null; then
        log_success "CRD $crd registered"
    else
        log_warning "CRD $crd not found"
    fi
done

# Check infrastructure services
log_info "Checking infrastructure services..."

# GreptimeDB
if kubectl get pods -n greptimedb -l app=greptimedb | grep -q Running; then
    log_success "GreptimeDB running"
else
    log_warning "GreptimeDB not running"
fi

# Redis
if kubectl get pods -n storage -l app.kubernetes.io/name=redis | grep -q Running; then
    log_success "Redis running"
else
    log_warning "Redis not running"
fi

# Qdrant
if kubectl get pods -n qdrant -l app=qdrant | grep -q Running; then
    log_success "Qdrant running"
else
    log_warning "Qdrant not running"
fi

# Portkey
if kubectl get pods -n portkey -l app=portkey-gateway | grep -q Running; then
    log_success "Portkey running"
else
    log_warning "Portkey not running"
fi

# Prometheus
if kubectl get pods -n observability -l app.kubernetes.io/name=prometheus | grep -q Running; then
    log_success "Prometheus running"
else
    log_warning "Prometheus not running"
fi

# Check vLLM deployments
log_info "Checking vLLM deployments..."
VLLM_PODS=$(kubectl get pods --all-namespaces -l app=vllm 2>/dev/null | grep -c Running || echo "0")
if [ "$VLLM_PODS" -gt 0 ]; then
    log_success "$VLLM_PODS vLLM pod(s) running"
else
    log_info "No vLLM deployments found (optional)"
fi

# Summary
echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║   Verification Summary                                         ║"
echo "╚════════════════════════════════════════════════════════════════╝"

if [ $FAILED -eq 0 ]; then
    log_success "All critical components verified!"
    echo ""
    log_info "Platform is ready for use"
    echo ""
    echo "Next steps:"
    echo "  1. Deploy vLLM: bash scripts/deploy-vllm.sh"
    echo "  2. Setup Azure Foundry: bash scripts/setup-azure-foundry.sh"
    echo "  3. Test inference: bash scripts/test-inference.sh"
else
    log_error "Some components are missing"
    echo ""
    log_info "Run deployment script:"
    echo "  bash scripts/deploy-complete-platform.sh"
fi

exit $FAILED



