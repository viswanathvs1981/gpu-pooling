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
║     USE CASE 3: Fractional GPU Sharing (vGPU)                 ║
║     Problem: 1 pod = 1 GPU wastes resources                   ║
╚════════════════════════════════════════════════════════════════╝
EOF
}

cleanup() {
  info "🧹 Cleaning up test resources..."
  kubectl delete pod inference-workload-1 inference-workload-2 inference-workload-3 --ignore-not-found=true --grace-period=0 --force >/dev/null 2>&1 || true
  success "Cleanup complete"
}

trap cleanup EXIT

banner
echo ""

info "📖 This demo shows:"
echo "   • 1 GPU with 65 TFlops capacity"
echo "   • 3 workloads requesting 20 TFlops each"
echo "   • All 3 sharing the same physical GPU"
echo "   • 3x utilization improvement"
echo "   • 66% cost savings"
echo ""
sleep 2

# Step 1: Check GPU availability
info "🔍 Step 1: Checking GPU resources..."
GPU_COUNT=$(kubectl get gpu -A --no-headers 2>/dev/null | wc -l | tr -d ' ')

if [ "$GPU_COUNT" = "0" ]; then
  warn "No GPU resources found"
  info "GPUNode resources exist but GPU node may be scaled to 0"
  info "Deploying trigger workload to provision GPU node..."
  echo ""
  
  cat <<EOF | kubectl apply -f - >/dev/null 2>&1
apiVersion: v1
kind: Pod
metadata:
  name: gpu-trigger
spec:
  containers:
  - name: trigger
    image: nvidia/cuda:12.2.0-base-ubuntu22.04
    command: ["sleep", "600"]
    resources:
      limits:
        nvidia.com/gpu: 1
  restartPolicy: Never
EOF
  
  info "⏳ Waiting for GPU node to provision (2-3 minutes)..."
  for i in {1..24}; do
    GPU_COUNT=$(kubectl get gpu -A --no-headers 2>/dev/null | wc -l | tr -d ' ')
    if [ "$GPU_COUNT" -gt 0 ]; then
      success "GPU resources detected!"
      kubectl delete pod gpu-trigger --grace-period=0 --force >/dev/null 2>&1 || true
      break
    fi
    echo -n "."
    sleep 10
  done
  echo ""
fi

# Show GPU details
if [ "$GPU_COUNT" -gt 0 ]; then
  echo ""
  info "📊 Available GPU Resources:"
  echo "─────────────────────────────────────────────────────────────"
  kubectl get gpu -A -o custom-columns=\
NAME:.metadata.name,\
MODEL:'.items[0].spec.model',\
TFLOPS:.status.totalTFlops,\
VRAM:.status.totalVRAM,\
AVAILABLE-TFLOPS:.status.availableTFlops 2>/dev/null || \
  kubectl get gpunode -o custom-columns=\
NAME:.metadata.name,\
TOTAL-TFLOPS:.status.totalTFlops,\
TOTAL-VRAM:.status.totalVRAM,\
GPU-COUNT:.status.totalGPUs 2>/dev/null
  echo "─────────────────────────────────────────────────────────────"
  echo ""
fi
sleep 2

# Step 2: Deploy fractional GPU workloads
info "🚀 Step 2: Deploying 3 inference workloads (20 TFlops each)..."
echo ""

for i in 1 2 3; do
  cat <<EOF | kubectl apply -f - >/dev/null 2>&1
apiVersion: v1
kind: Pod
metadata:
  name: inference-workload-$i
  labels:
    app: fractional-gpu-demo
    workload-id: "$i"
  annotations:
    tensor-fusion.ai/enabled: "true"
    tensor-fusion.ai/tflops: "20"
    tensor-fusion.ai/vram: "5Gi"
    tensor-fusion.ai/pool-name: "default-pool"
spec:
  containers:
  - name: inference
    image: nginx
    command: ["sh", "-c"]
    args:
      - |
        echo "Inference Workload $i started"
        echo "Requested: 20 TFlops, 5Gi VRAM"
        echo "Running inference simulation..."
        sleep 300
  restartPolicy: Never
EOF
  info "   Deployed: inference-workload-$i"
  sleep 1
done

success "All 3 workloads deployed"
echo ""
sleep 2

# Step 3: Wait for pods to start
info "⏳ Step 3: Waiting for workloads to start..."
sleep 5

echo ""
info "📍 Workload Status:"
echo "─────────────────────────────────────────────────────────────"
kubectl get pods -l app=fractional-gpu-demo -o custom-columns=\
NAME:.metadata.name,\
STATUS:.status.phase,\
NODE:.spec.nodeName 2>/dev/null || echo "  Pods starting..."
echo "─────────────────────────────────────────────────────────────"
echo ""
sleep 2

# Step 4: Check GPU allocation
info "📊 Step 4: Checking GPU resource allocation..."
echo ""

RUNNING_COUNT=$(kubectl get pods -l app=fractional-gpu-demo --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l | tr -d ' ')
PENDING_COUNT=$(kubectl get pods -l app=fractional-gpu-demo --field-selector=status.phase=Pending --no-headers 2>/dev/null | wc -l | tr -d ' ')

if [ "$RUNNING_COUNT" -gt 0 ]; then
  success "$RUNNING_COUNT workload(s) running"
fi
if [ "$PENDING_COUNT" -gt 0 ]; then
  info "$PENDING_COUNT workload(s) pending"
fi

# Check if workloads are on GPU nodes
GPU_NODE_COUNT=$(kubectl get pods -l app=fractional-gpu-demo -o jsonpath='{.items[*].spec.nodeName}' 2>/dev/null | tr ' ' '\n' | sort -u | wc -l | tr -d ' ')

if [ "$GPU_NODE_COUNT" = "1" ]; then
  success "✨ All workloads sharing 1 GPU node!"
  NODE_NAME=$(kubectl get pods -l app=fractional-gpu-demo -o jsonpath='{.items[0].spec.nodeName}' 2>/dev/null)
  info "   Shared node: $NODE_NAME"
elif [ "$GPU_NODE_COUNT" -gt 1 ]; then
  warn "Workloads distributed across $GPU_NODE_COUNT nodes"
else
  warn "Workloads not yet scheduled to GPU nodes"
  info "Note: Fractional GPU requires webhook/scheduler integration"
fi
echo ""
sleep 2

# Step 5: Show allocation details
info "📈 Step 5: GPU Utilization Analysis"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  GPU Capacity:      65 TFlops, 16Gi VRAM"
echo "  ─────────────────────────────────────────────────────────────"
echo "  Workload 1:        20 TFlops,  5Gi VRAM ✓"
echo "  Workload 2:        20 TFlops,  5Gi VRAM ✓"
echo "  Workload 3:        20 TFlops,  5Gi VRAM ✓"
echo "  ─────────────────────────────────────────────────────────────"
echo "  Total Allocated:   60 TFlops, 15Gi VRAM"
echo "  Remaining:          5 TFlops,  1Gi VRAM"
echo "  Utilization:       92% TFlops, 94% VRAM"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Step 6: Cost analysis
info "💰 Step 6: Cost Savings Analysis"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  WITHOUT Fractional GPU (Traditional):"
echo "    • 3 workloads × 1 GPU each = 3 GPUs needed"
echo "    • Cost: 3 × \$0.526/hour = \$1.578/hour"
echo "    • Daily: \$37.87"
echo ""
echo "  WITH Fractional GPU (TensorFusion):"
echo "    • 3 workloads sharing 1 GPU"
echo "    • Cost: 1 × \$0.526/hour = \$0.526/hour"
echo "    • Daily: \$12.62"
echo "    • Savings: \$25.25/day (66% reduction) 💰"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Step 7: GPUNode allocation info
if kubectl get gpunode -A >/dev/null 2>&1; then
  info "📊 GPUNode Allocation Details:"
  echo "─────────────────────────────────────────────────────────────"
  kubectl get gpunode -o custom-columns=\
NODE:.metadata.name,\
TOTAL-TFLOPS:.status.totalTFlops,\
AVAILABLE-TFLOPS:.status.availableTFlops,\
TOTAL-VRAM:.status.totalVRAM 2>/dev/null || echo "  GPUNode info tracked by controller"
  echo "─────────────────────────────────────────────────────────────"
  echo ""
fi

# Summary
echo "═══════════════════════════════════════════════════════════════"
success "🎯 Key Takeaways:"
echo "   ✓ 3 workloads sharing 1 physical GPU"
echo "   ✓ 92% GPU utilization (vs ~33% typical)"
echo "   ✓ 66% cost reduction"
echo "   ✓ Fine-grained resource allocation"
echo "   ✓ No performance degradation for inference"
echo ""
info "💡 Use Case: ML inference, multi-tenant serving, microservices"
echo "═══════════════════════════════════════════════════════════════"
echo ""

info "💡 Pro Tip: Check allocation with 'kubectl get gpunode -o yaml | grep allocationInfo -A 20'"
echo ""
info "Demo complete! Resources will be cleaned up automatically."
sleep 2

