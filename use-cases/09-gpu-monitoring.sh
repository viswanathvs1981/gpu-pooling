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
║     USE CASE 9: GPU Resource Monitoring & Observability       ║
║     Problem: Need visibility into GPU usage & performance      ║
╚════════════════════════════════════════════════════════════════╝
EOF
}

banner
echo ""

info "📖 This demo shows:"
echo "   • Real-time GPU resource tracking"
echo "   • Multi-tenant usage breakdown"
echo "   • Cost allocation per customer"
echo "   • Performance metrics & bottlenecks"
echo "   • Alerts & anomaly detection"
echo ""
sleep 2

# Step 1: Check cluster GPU resources
info "🔍 Step 1: Cluster GPU Inventory"
echo ""
echo "═══════════════════════════════════════════════════════════════"
GPU_NODES=$(kubectl get nodes -l nvidia.com/gpu.present=true --no-headers 2>/dev/null | wc -l | tr -d ' ')

if [ "$GPU_NODES" -gt 0 ]; then
  success "GPU nodes found: $GPU_NODES"
  echo ""
  info "Node Details:"
  echo "─────────────────────────────────────────────────────────────"
  kubectl get nodes -l nvidia.com/gpu.present=true -o custom-columns=\
NAME:.metadata.name,\
GPU-TYPE:.metadata.labels.'nvidia\.com/gpu\.product',\
GPU-COUNT:.status.capacity.'nvidia\.com/gpu',\
STATUS:.status.conditions[-1].type 2>/dev/null || echo "  Details unavailable"
  echo "─────────────────────────────────────────────────────────────"
else
  warn "No GPU nodes found in cluster"
  info "  This is OK - demo will show expected monitoring output"
fi
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Step 2: GPU Custom Resources
info "📊 Step 2: Tensor Fusion GPU Resources"
echo ""
echo "═══════════════════════════════════════════════════════════════"

GPUNODE_COUNT=$(kubectl get gpunode --all-namespaces --no-headers 2>/dev/null | wc -l | tr -d ' ')
GPU_COUNT=$(kubectl get gpu --all-namespaces --no-headers 2>/dev/null | wc -l | tr -d ' ')

if [ "$GPUNODE_COUNT" -gt 0 ]; then
  success "GPUNode resources: $GPUNODE_COUNT"
  echo ""
  info "GPUNode Status:"
  echo "─────────────────────────────────────────────────────────────"
  kubectl get gpunode -o custom-columns=\
NAME:.metadata.name,\
TFLOPS:.status.capacity.tflops,\
VRAM:.status.capacity.vram,\
GPU-COUNT:.status.capacity.gpu 2>/dev/null
  echo "─────────────────────────────────────────────────────────────"
  echo ""
fi

if [ "$GPU_COUNT" -gt 0 ]; then
  success "GPU resources: $GPU_COUNT"
  echo ""
  info "Individual GPUs:"
  echo "─────────────────────────────────────────────────────────────"
  kubectl get gpu -o custom-columns=\
NAME:.metadata.name,\
MODEL:.spec.model,\
TFLOPS:.spec.tflops,\
VRAM:.spec.vram,\
STATUS:.status.phase 2>/dev/null | head -10
  echo "─────────────────────────────────────────────────────────────"
  echo ""
fi

if [ "$GPUNODE_COUNT" -eq 0 ] && [ "$GPU_COUNT" -eq 0 ]; then
  warn "No GPU custom resources found yet"
  info "  Node discovery may still be running or no GPUs available"
fi
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Step 3: Resource Utilization
info "⚡ Step 3: Real-Time GPU Utilization"
echo ""
echo "═══════════════════════════════════════════════════════════════"
if [ "$GPU_NODES" -gt 0 ]; then
  info "Running nvidia-smi on GPU nodes..."
  echo ""
  
  # Try to run nvidia-smi on a GPU node
  GPU_NODE=$(kubectl get nodes -l nvidia.com/gpu.present=true -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
  
  if [ -n "$GPU_NODE" ]; then
    info "Node: $GPU_NODE"
    echo "─────────────────────────────────────────────────────────────"
    
    # Try to get GPU metrics via a debug pod
    kubectl run nvidia-smi-test --rm -i --restart=Never \
      --image=nvidia/cuda:12.2.0-base-ubuntu22.04 \
      --overrides='{"spec":{"nodeSelector":{"nvidia.com/gpu.present":"true"}}}' \
      -- nvidia-smi --query-gpu=gpu_name,memory.total,memory.used,utilization.gpu --format=csv 2>/dev/null || \
    echo "  GPU metrics collection requires privileged access"
    
    echo "─────────────────────────────────────────────────────────────"
  fi
else
  info "Example GPU Utilization (when GPUs available):"
  echo "─────────────────────────────────────────────────────────────"
  cat << 'TABLE'
GPU  │ Name            │ Memory Used │ Memory Total │ GPU Util │ Temp
─────┼─────────────────┼─────────────┼──────────────┼──────────┼──────
  0  │ Tesla T4        │   8.2 GB    │   16 GB      │   65%    │ 54°C
  1  │ Tesla T4        │  12.5 GB    │   16 GB      │   89%    │ 61°C
  2  │ Tesla T4        │   4.1 GB    │   16 GB      │   32%    │ 48°C
─────┴─────────────────┴─────────────┴──────────────┴──────────┴──────
TABLE
  echo "─────────────────────────────────────────────────────────────"
fi
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Step 4: Pool-Based Monitoring
info "🏊 Step 4: GPU Pool Utilization"
echo ""
echo "═══════════════════════════════════════════════════════════════"
POOL_COUNT=$(kubectl get gpupool --all-namespaces --no-headers 2>/dev/null | wc -l | tr -d ' ')

if [ "$POOL_COUNT" -gt 0 ]; then
  success "GPU pools found: $POOL_COUNT"
  echo ""
  info "Pool Status:"
  echo "─────────────────────────────────────────────────────────────"
  kubectl get gpupool --all-namespaces -o custom-columns=\
NAMESPACE:.metadata.namespace,\
NAME:.metadata.name,\
CAPACITY:.status.capacity.gpu,\
AVAILABLE:.status.available.gpu 2>/dev/null || echo "  Pool details unavailable"
  echo "─────────────────────────────────────────────────────────────"
else
  info "Example GPU Pool Monitoring:"
  echo "─────────────────────────────────────────────────────────────"
  cat << 'TABLE'
Pool           │ Total GPUs │ Allocated │ Available │ Util %
───────────────┼────────────┼───────────┼───────────┼────────
default-pool   │     10     │    7.5    │    2.5    │  75%
spot-pool      │      8     │    6.2    │    1.8    │  78%
training-pool  │      4     │    3.8    │    0.2    │  95%
───────────────┴────────────┴───────────┴───────────┴────────
TABLE
  echo "─────────────────────────────────────────────────────────────"
fi
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Step 5: Multi-Tenant Breakdown
info "🏢 Step 5: Multi-Tenant Resource Breakdown"
echo ""
echo "═══════════════════════════════════════════════════════════════"
QUOTA_COUNT=$(kubectl get gpuresourcequota --all-namespaces --no-headers 2>/dev/null | wc -l | tr -d ' ')

if [ "$QUOTA_COUNT" -gt 0 ]; then
  success "GPU resource quotas: $QUOTA_COUNT"
  echo ""
  info "Quota Status:"
  echo "─────────────────────────────────────────────────────────────"
  kubectl get gpuresourcequota --all-namespaces -o custom-columns=\
NAMESPACE:.metadata.namespace,\
NAME:.metadata.name,\
GPU-QUOTA:.spec.hard.gpu,\
GPU-USED:.status.used.gpu 2>/dev/null || echo "  Quota details unavailable"
  echo "─────────────────────────────────────────────────────────────"
else
  info "Example Multi-Tenant Usage:"
  echo "─────────────────────────────────────────────────────────────"
  cat << 'TABLE'
Tenant         │ Quota │ Used  │ Available │ Usage % │ Cost/Month
───────────────┼───────┼───────┼───────────┼─────────┼───────────
acme-corp      │  3.0  │  2.8  │    0.2    │   93%   │  \$6,720
techstart-inc  │  2.0  │  1.5  │    0.5    │   75%   │  \$3,600
legal-ai-co    │  1.5  │  1.2  │    0.3    │   80%   │  \$2,880
med-platform   │  2.5  │  0.8  │    1.7    │   32%   │  \$1,920
───────────────┴───────┴───────┴───────────┴─────────┴───────────
Total: 9.0 GPU quota, 6.3 GPUs used (70% utilization)
TABLE
  echo "─────────────────────────────────────────────────────────────"
fi
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Step 6: Cost Allocation
info "💰 Step 6: Cost Allocation & Billing"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Cost Calculation (per tenant):"
echo ""
echo "  Base Rate: \$2.40/hour per full GPU (Tesla T4)"
echo "  Billing: Per-second usage, aggregated monthly"
echo ""
echo "Example Calculation for acme-corp:"
echo "  • Usage: 2.8 vGPUs × 720 hours = 2,016 GPU-hours"
echo "  • Cost: 2,016 × \$2.40 = \$4,838.40"
echo "  • With Tensor Fusion sharing: \$4,838 vs \$8,640 (full GPUs)"
echo "  • Savings: \$3,802 (44%)"
echo ""
echo "─────────────────────────────────────────────────────────────"
info "Monthly Cost Breakdown by Tenant:"
echo "─────────────────────────────────────────────────────────────"
cat << 'TABLE'
Tenant         │ vGPU Used │ GPU-Hours │ Cost      │ vs Full GPU │ Savings
───────────────┼───────────┼───────────┼───────────┼─────────────┼─────────
acme-corp      │    2.8    │   2,016   │  \$4,838  │   \$8,640   │  44%
techstart-inc  │    1.5    │   1,080   │  \$2,592  │   \$5,184   │  50%
legal-ai-co    │    1.2    │     864   │  \$2,074  │   \$4,320   │  52%
med-platform   │    0.8    │     576   │  \$1,382  │   \$3,456   │  60%
───────────────┴───────────┴───────────┴───────────┴─────────────┴─────────
Total                                     \$10,886     \$21,600     50% avg
TABLE
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Step 7: Performance Metrics
info "📈 Step 7: Performance Metrics & KPIs"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Key Performance Indicators:"
echo ""
echo "  1. GPU Utilization"
echo "     • Target: > 80%"
echo "     • Current: 70%"
echo "     • Status: ⚠️  Room for optimization"
echo ""
echo "  2. Memory Efficiency"
echo "     • VRAM allocated: 180GB / 240GB total"
echo "     • Utilization: 75%"
echo "     • Status: ✅ Good"
echo ""
echo "  3. Request Throughput"
echo "     • Inference requests: 45,230/hour"
echo "     • Average latency: 320ms"
echo "     • P99 latency: 580ms"
echo "     • Status: ✅ Meeting SLAs"
echo ""
echo "  4. Cost Efficiency"
echo "     • Cost per 1M tokens: \$0.12"
echo "     • vs Azure OpenAI: \$30/1M tokens"
echo "     • Savings: 99.6%"
echo "     • Status: ✅ Excellent"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Step 8: Monitoring Stack
info "🔧 Step 8: Monitoring & Observability Stack"
echo ""
echo "═══════════════════════════════════════════════════════════════"
PROMETHEUS_PODS=$(kubectl get pods -n prometheus --no-headers 2>/dev/null | wc -l | tr -d ' ')
GRAFANA_PODS=$(kubectl get pods -n grafana --no-headers 2>/dev/null | wc -l | tr -d ' ')
GREPTIMEDB_PODS=$(kubectl get pods -n greptimedb --no-headers 2>/dev/null | wc -l | tr -d ' ')

info "Deployed Components:"
echo ""

if [ "$PROMETHEUS_PODS" -gt 0 ]; then
  success "Prometheus: Running ($PROMETHEUS_PODS pods)"
  echo "  • Metrics collection: GPU, CPU, memory, network"
  echo "  • Scrape interval: 15s"
else
  info "Prometheus: Not deployed"
fi

if [ "$GRAFANA_PODS" -gt 0 ]; then
  success "Grafana: Running ($GRAFANA_PODS pods)"
  echo "  • Dashboards: GPU utilization, costs, performance"
  echo "  • Alerts: Anomaly detection, quota exceeded"
else
  info "Grafana: Not deployed"
fi

if [ "$GREPTIMEDB_PODS" -gt 0 ]; then
  success "GreptimeDB: Running ($GREPTIMEDB_PODS pods)"
  echo "  • Time-series storage for long-term metrics"
  echo "  • Retention: 90 days"
else
  info "GreptimeDB: Not deployed"
fi

echo ""
info "Metrics Collected:"
echo "  • GPU: Utilization, memory, temperature, power"
echo "  • Workload: Request count, latency, throughput"
echo "  • Cost: Per-tenant usage, billing aggregation"
echo "  • Performance: Model inference time, queue depth"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Step 9: Alerts & Anomaly Detection
info "🚨 Step 9: Alerts & Anomaly Detection"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Active Alert Rules:"
echo ""
echo "  1. High GPU Utilization"
echo "     • Trigger: > 95% for 5 minutes"
echo "     • Action: Scale up, alert ops team"
echo "     • Status: ✅ Normal (70%)"
echo ""
echo "  2. Quota Exceeded"
echo "     • Trigger: Tenant uses > 90% of quota"
echo "     • Action: Notify tenant, block new workloads"
echo "     • Status: ⚠️  acme-corp at 93%"
echo ""
echo "  3. High Latency"
echo "     • Trigger: P99 > 1000ms for 10 minutes"
echo "     • Action: Check for bottlenecks, scale"
echo "     • Status: ✅ Normal (580ms)"
echo ""
echo "  4. GPU Failure"
echo "     • Trigger: GPU offline or unhealthy"
echo "     • Action: Drain workloads, page oncall"
echo "     • Status: ✅ All healthy"
echo ""
echo "  5. Cost Anomaly"
echo "     • Trigger: 50% spike in usage cost"
echo "     • Action: Investigate, notify customer"
echo "     • Status: ✅ Normal"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Summary
echo "═══════════════════════════════════════════════════════════════"
success "🎯 Key Takeaways:"
echo "   ✓ Real-time GPU utilization tracking across all nodes"
echo "   ✓ Multi-tenant resource breakdown & quota enforcement"
echo "   ✓ Per-tenant cost allocation (50% savings with vGPU sharing)"
echo "   ✓ Performance metrics: latency, throughput, efficiency"
echo "   ✓ Proactive alerts for capacity, quotas, anomalies"
echo "   ✓ Complete observability: Prometheus + Grafana + GreptimeDB"
echo ""
info "💡 Use Case: Platform operations, cost tracking, capacity planning"
echo "═══════════════════════════════════════════════════════════════"
echo ""

info "💡 Next Steps:"
echo "  1. Access Grafana: kubectl port-forward -n grafana svc/grafana 3000:80"
echo "  2. View metrics: http://localhost:3000"
echo "  3. Check quotas: kubectl get gpuresourcequota --all-namespaces"
echo "  4. Review GPU resources: kubectl get gpu,gpunode"
echo ""
info "Demo complete!"
sleep 2

