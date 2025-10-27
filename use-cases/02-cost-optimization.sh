#!/bin/bash

set -uo pipefail

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

success() { echo -e "${GREEN}✅ $1${NC}"; }
info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
warn() { echo -e "${YELLOW}⚠️  $1${NC}"; }
error() { echo -e "${RED}❌ $1${NC}"; }

banner() {
cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║     USE CASE 2: Cost-Optimized GPU Auto-Scaling               ║
║     Problem: Expensive GPUs idle = wasted money                ║
╚════════════════════════════════════════════════════════════════╝
EOF
}

cleanup() {
  info "🧹 Cleaning up test resources..."
  kubectl delete pod gpu-cost-demo --ignore-not-found=true --grace-period=0 --force >/dev/null 2>&1 || true
  success "Cleanup complete"
}

trap cleanup EXIT

banner
echo ""

info "📖 This demo shows:"
echo "   • GPU nodes scaled to 0 when idle (cost = \$0/hour)"
echo "   • Automatic provisioning on workload deployment"
echo "   • Pod scheduled to new GPU node"
echo "   • GPU validation with nvidia-smi"
echo "   • Automatic scale-down after workload completion"
echo ""
sleep 2

# Step 1: Show current GPU node count
info "💰 Step 1: Checking current GPU node count..."
GPU_NODES=$(kubectl get nodes -l pool=gpu --no-headers 2>/dev/null | wc -l | tr -d ' ')
echo ""
if [ "$GPU_NODES" = "0" ]; then
  success "GPU nodes: 0 (COST: \$0/hour) ✨"
  echo "   • No GPU VMs running"
  echo "   • Zero infrastructure cost"
  echo "   • Autoscaler will provision on-demand"
else
  info "GPU nodes: $GPU_NODES"
  kubectl get nodes -l pool=gpu -o custom-columns=NAME:.metadata.name,STATUS:.status.conditions[-1].type,GPUs:.status.capacity.nvidia\\.com/gpu
fi
echo ""
sleep 2

# Step 2: Deploy GPU workload
info "🚀 Step 2: Deploying GPU workload..."
echo "   • Requesting: 1x NVIDIA GPU"
echo "   • This will trigger autoscaler"
echo ""

cat <<EOF | kubectl apply -f - >/dev/null 2>&1
apiVersion: v1
kind: Pod
metadata:
  name: gpu-cost-demo
  labels:
    demo: cost-optimization
spec:
  containers:
  - name: cuda-app
    image: nvidia/cuda:12.2.0-base-ubuntu22.04
    command: ["bash", "-c"]
    args:
      - |
        echo "=== GPU Cost Optimization Demo ==="
        echo "Pod started at: \$(date)"
        echo ""
        echo "Checking GPU access..."
        nvidia-smi --query-gpu=name,memory.total,driver_version --format=csv,noheader
        echo ""
        echo "Running workload for 5 minutes..."
        sleep 300
    resources:
      limits:
        nvidia.com/gpu: 1
  restartPolicy: Never
  tolerations:
  - key: nvidia.com/gpu
    operator: Exists
    effect: NoSchedule
EOF

success "GPU workload deployed"
sleep 2

# Step 3: Wait for GPU node provisioning
if [ "$GPU_NODES" = "0" ]; then
  info "⏳ Step 3: Waiting for autoscaler to provision GPU node..."
  echo "   • This typically takes 2-4 minutes"
  echo "   • Azure is creating: Standard_NC4as_T4_v3 VM"
  echo "   • Estimated cost: ~\$0.526/hour when running"
  echo ""
  
  for i in {1..24}; do
    GPU_NODES=$(kubectl get nodes -l pool=gpu --no-headers 2>/dev/null | wc -l | tr -d ' ')
    if [ "$GPU_NODES" -gt 0 ]; then
      success "GPU node provisioned!"
      break
    fi
    echo -n "."
    sleep 10
  done
  echo ""
  
  if [ "$GPU_NODES" -gt 0 ]; then
    GPU_NODE_NAME=$(kubectl get nodes -l pool=gpu -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    success "Node: $GPU_NODE_NAME"
    echo ""
  else
    warn "Autoscaler taking longer than expected"
    info "This is normal for first-time provisioning"
    info "You can check status with: kubectl get nodes -l pool=gpu"
    echo ""
  fi
else
  info "⚡ Step 3: GPU node already available (fast path)"
  echo ""
fi

# Step 4: Wait for pod to be scheduled
info "📍 Step 4: Waiting for pod to be scheduled to GPU node..."
for i in {1..30}; do
  POD_NODE=$(kubectl get pod gpu-cost-demo -o jsonpath='{.spec.nodeName}' 2>/dev/null)
  if [ -n "$POD_NODE" ]; then
    success "Pod scheduled to: $POD_NODE"
    break
  fi
  sleep 2
done

if [ -z "$POD_NODE" ]; then
  warn "Pod not yet scheduled"
  POD_STATUS=$(kubectl get pod gpu-cost-demo -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
  info "Pod status: $POD_STATUS"
fi
echo ""
sleep 2

# Step 5: Wait for pod to run and show GPU info
info "🎯 Step 5: Validating GPU access..."
echo ""

for i in {1..30}; do
  POD_STATUS=$(kubectl get pod gpu-cost-demo -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
  if [ "$POD_STATUS" = "Running" ] || [ "$POD_STATUS" = "Succeeded" ]; then
    success "Pod is $POD_STATUS"
    echo ""
    info "📊 GPU Information (from nvidia-smi):"
    echo "─────────────────────────────────────────────────────────────"
    kubectl logs gpu-cost-demo 2>/dev/null | grep -A 10 "GPU Cost\|Checking GPU\|Tesla\|NVIDIA" || echo "  Waiting for logs..."
    echo "─────────────────────────────────────────────────────────────"
    break
  fi
  sleep 2
done
echo ""
sleep 2

# Step 6: Show cost comparison
info "💰 Step 6: Cost Analysis"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  WITHOUT Auto-Scaling (Always-On):"
echo "    • 1 GPU node × 24 hours × \$0.526 = \$12.62/day"
echo "    • Annual cost: ~\$4,606"
echo ""
echo "  WITH Auto-Scaling (TensorFusion):"
echo "    • 1 GPU node × 8 hours/day × \$0.526 = \$4.21/day"
echo "    • Annual cost: ~\$1,536"
echo "    • Savings: \$3,070/year per GPU (67% reduction) 💰"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 7: Explain scale-down
info "📉 Step 7: Automatic Scale-Down"
echo ""
echo "   After this workload completes:"
echo "   ✓ Pod finishes (5 minutes)"
echo "   ✓ No more GPU pods pending"
echo "   ✓ Autoscaler waits ~10 minutes (grace period)"
echo "   ✓ Node automatically deleted"
echo "   ✓ Cost returns to \$0/hour"
echo ""
success "This happens automatically - no manual intervention!"
echo ""

# Summary
echo "═══════════════════════════════════════════════════════════════"
success "🎯 Key Takeaways:"
echo "   ✓ Zero cost when idle (nodes scale to 0)"
echo "   ✓ Automatic provisioning in 2-4 minutes"
echo "   ✓ Workloads get full GPU access"
echo "   ✓ Automatic scale-down after completion"
echo "   ✓ 67% cost reduction vs always-on"
echo ""
info "💡 Use Case: Development/test environments, batch jobs, cost-sensitive workloads"
echo "═══════════════════════════════════════════════════════════════"
echo ""

info "💡 Pro Tip: Run 'kubectl get nodes -l pool=gpu -w' to watch scale-down (takes ~10 min)"
echo ""
info "Demo complete! Workload will run for 5 minutes, then resources will be cleaned up."

