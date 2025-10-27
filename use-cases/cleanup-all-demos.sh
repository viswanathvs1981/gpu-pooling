#!/bin/bash

set -uo pipefail

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

success() { echo -e "${GREEN}✅ $1${NC}"; }
info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
warn() { echo -e "${YELLOW}⚠️  $1${NC}"; }

banner() {
cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║     Cleanup All Demo Resources                                 ║
╚════════════════════════════════════════════════════════════════╝
EOF
}

banner
echo ""

info "🧹 Cleaning up all demo resources..."
echo ""

# Use Case 1: Multi-Tenant Quotas
info "Cleaning up USE CASE 1: Multi-Tenant Quotas"
kubectl delete namespace team-a team-b --ignore-not-found=true >/dev/null 2>&1 &
success "  Team namespaces deleted"

# Use Case 2: Cost Optimization
info "Cleaning up USE CASE 2: Cost Optimization"
kubectl delete pod gpu-cost-demo --ignore-not-found=true --grace-period=0 --force >/dev/null 2>&1 &
kubectl delete pod gpu-trigger --ignore-not-found=true --grace-period=0 --force >/dev/null 2>&1 &
success "  Cost demo pods deleted"

# Use Case 3: Fractional GPU
info "Cleaning up USE CASE 3: Fractional GPU"
kubectl delete pod inference-workload-1 inference-workload-2 inference-workload-3 --ignore-not-found=true --grace-period=0 --force >/dev/null 2>&1 &
success "  Inference workloads deleted"

# Use Case 5: Distributed Training
info "Cleaning up USE CASE 5: Distributed Training"
kubectl delete pod trainer-rank-0 trainer-rank-1 trainer-rank-2 --ignore-not-found=true --grace-period=0 --force >/dev/null 2>&1 &
success "  Training workers deleted"

# General cleanup
info "Cleaning up miscellaneous test pods"
kubectl delete pods -l demo=cost-optimization --ignore-not-found=true --grace-period=0 --force >/dev/null 2>&1 &
kubectl delete pods -l app=fractional-gpu-demo --ignore-not-found=true --grace-period=0 --force >/dev/null 2>&1 &
kubectl delete pods -l app=distributed-training --ignore-not-found=true --grace-period=0 --force >/dev/null 2>&1 &
kubectl delete pods -l app=vgpu-test --ignore-not-found=true --grace-period=0 --force >/dev/null 2>&1 &
success "  Labeled demo pods deleted"

echo ""
info "⏳ Waiting for deletions to complete..."
wait
sleep 2

echo ""
info "📊 Remaining demo resources:"
echo ""

TEAM_NS=$(kubectl get namespace team-a team-b 2>/dev/null | grep -c team || echo "0")
DEMO_PODS=$(kubectl get pods -A -l "demo in (cost-optimization,fractional-gpu,distributed-training)" --no-headers 2>/dev/null | wc -l | tr -d ' ')

if [ "$TEAM_NS" = "0" ] && [ "$DEMO_PODS" = "0" ]; then
  success "✨ All demo resources cleaned up successfully!"
else
  if [ "$TEAM_NS" != "0" ]; then
    warn "  Team namespaces still terminating: $TEAM_NS"
  fi
  if [ "$DEMO_PODS" != "0" ]; then
    warn "  Demo pods still terminating: $DEMO_PODS"
  fi
  info "  (This is normal - Kubernetes is cleaning up in the background)"
fi

echo ""
info "💡 Note: GPU nodes may remain if autoscaler hasn't scaled down yet"
echo "   Check with: kubectl get nodes -l pool=gpu"
echo "   Nodes will auto-scale to 0 after ~10 minutes of idle time"
echo ""

success "Cleanup complete! 🎉"

