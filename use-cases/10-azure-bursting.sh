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
║     USE CASE 10: Azure Cloud Bursting & Auto-Scaling          ║
║     Problem: Handle traffic spikes without over-provisioning   ║
╚════════════════════════════════════════════════════════════════╝
EOF
}

cleanup() {
  info "🧹 Cleaning up test resources..."
  kubectl delete azuregpusource demo-burst-source --ignore-not-found=true >/dev/null 2>&1 || true
  success "Cleanup complete"
}

trap cleanup EXIT

banner
echo ""

info "📖 This demo shows:"
echo "   • Hybrid on-prem + cloud architecture"
echo "   • Automatic cloud bursting during peak load"
echo "   • Cost-optimized scaling (spot instances)"
echo "   • Multi-region GPU federation"
echo "   • Azure GPU source configuration"
echo ""
sleep 2

# Step 1: The Problem
info "❌ Step 1: The Traditional Scaling Problem"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Scenario: AI platform with variable load"
echo ""
echo "  Baseline: 10 users (2 GPUs needed)"
echo "  Peak: 100 users (20 GPUs needed)"
echo "  Peak duration: 2 hours/day"
echo ""
echo "Traditional Solutions:"
echo ""
echo "  Option 1: Provision for peak (20 GPUs 24/7)"
echo "    • Cost: 20 × \$3/hour × 720 hours = \$43,200/month"
echo "    • Utilization: 10% (18 GPUs idle 22 hours/day)"
echo "    • Waste: \$38,880/month (90%)"
echo ""
echo "  Option 2: Provision for baseline (2 GPUs)"
echo "    • Cost: 2 × \$3/hour × 720 hours = \$4,320/month"
echo "    • Peak problem: 90% of requests fail"
echo "    • User experience: ❌ Poor"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 2: Cloud Bursting Solution
info "✅ Step 2: Cloud Bursting Solution"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Hybrid Architecture:"
echo ""
echo "  On-Premises / Reserved:"
echo "    • 2 GPUs (always on for baseline)"
echo "    • Cost: \$4,320/month"
echo "    • Handles: Normal load (90% of time)"
echo ""
echo "  Cloud Burst (Azure Spot):"
echo "    • 18 GPUs (only during peak)"
echo "    • Duration: 2 hours/day × 30 days = 60 hours/month"
echo "    • Spot pricing: \$1.20/hour (60% discount)"
echo "    • Cost: 18 × \$1.20 × 60 = \$1,296/month"
echo ""
echo "  Total Cost: \$4,320 + \$1,296 = \$5,616/month"
echo "  vs Always-Peak: \$43,200/month"
echo "  Savings: \$37,584/month (87%) 🎉"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 3: Azure GPU Source CRD
info "🚀 Step 3: Configuring Azure GPU Source"
echo ""
echo "═══════════════════════════════════════════════════════════════"
AZURE_SOURCE_COUNT=$(kubectl get azuregpusource --all-namespaces --no-headers 2>/dev/null | wc -l | tr -d ' ')

if [ "$AZURE_SOURCE_COUNT" -gt 0 ]; then
  success "Azure GPU sources found: $AZURE_SOURCE_COUNT"
  echo ""
  kubectl get azuregpusource --all-namespaces 2>/dev/null
else
  info "Creating example Azure GPU source configuration..."
  echo ""
fi

cat <<EOF | kubectl apply -f - >/dev/null 2>&1
apiVersion: tensor-fusion.ai/v1
kind: AzureGPUSource
metadata:
  name: demo-burst-source
  namespace: default
spec:
  # Azure Configuration
  subscriptionID: "your-subscription-id"
  resourceGroup: "tensor-fusion-burst-rg"
  location: "eastus"
  
  # VM Configuration
  vmSize: "Standard_NC6s_v3"  # Tesla V100
  priority: "Spot"  # 60-90% cheaper
  maxPrice: "1.50"  # Max spot price
  
  # Scaling Configuration
  minInstances: 0
  maxInstances: 20
  scaleUpThreshold: 0.80    # Scale up at 80% utilization
  scaleDownThreshold: 0.30  # Scale down at 30% utilization
  cooldownPeriod: "5m"
  
  # Cost Controls
  maxMonthlyCost: "2000"  # Budget limit
  enableCostAlerts: true
  
  # Network
  virtualNetwork: "aks-vnet"
  subnet: "burst-subnet"
  
  # Auto-provisioning
  autoProvision: true
  healthCheckInterval: "30s"
EOF

success "Azure GPU source configured!"
echo ""
sleep 2

# Step 4: Scaling Workflow
info "⚡ Step 4: Auto-Scaling Workflow"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Time: 8:00 AM - Traffic starts increasing"
echo ""
echo "  Step 1: Monitor detects 75% GPU utilization"
echo "  Step 2: Threshold exceeded (> 80%)"
echo "  Step 3: Azure GPU Source triggers scale-up"
echo "  Step 4: Azure provisions 5× NC6s_v3 spot instances"
echo "  Step 5: VMs join cluster (~3 minutes)"
echo "  Step 6: Tensor Fusion discovers new GPUs"
echo "  Step 7: Workloads distributed across 7 total GPUs"
echo "  Step 8: Utilization drops to 60%"
echo ""
echo "Time: 10:00 AM - Traffic subsides"
echo ""
echo "  Step 1: Monitor detects 25% GPU utilization"
echo "  Step 2: Below threshold (< 30%)"
echo "  Step 3: Cooldown period (5 min) expires"
echo "  Step 4: Azure GPU Source triggers scale-down"
echo "  Step 5: Drain workloads from cloud GPUs"
echo "  Step 6: Deallocate 5 spot instances"
echo "  Step 7: Back to 2 baseline GPUs"
echo "  Step 8: Cost accumulation stops"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 5: Real-world scenarios
info "🌍 Step 5: Real-World Scenarios"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Scenario 1: E-commerce AI during Black Friday"
echo "  • Baseline: 5 GPUs year-round"
echo "  • Black Friday week: Burst to 50 GPUs"
echo "  • Duration: 7 days"
echo "  • Cost: \$10,800 baseline + \$3,024 burst = \$13,824"
echo "  • vs Always-50: \$108,000/month"
echo "  • Savings: \$94,176 (87%)"
echo "─────────────────────────────────────────────────────────────"
echo ""
echo "Scenario 2: Research team with batch training jobs"
echo "  • Baseline: 2 GPUs for experiments"
echo "  • Training runs: Burst to 32 GPUs"
echo "  • Frequency: 3× per week, 4 hours each"
echo "  • Monthly burst: 48 hours"
echo "  • Cost: \$4,320 baseline + \$1,843 burst = \$6,163"
echo "  • vs Always-32: \$69,120"
echo "  • Savings: \$62,957 (91%)"
echo "─────────────────────────────────────────────────────────────"
echo ""
echo "Scenario 3: Startup with unpredictable growth"
echo "  • Start: 1 GPU"
echo "  • Week 1-4: Gradual burst to 3 GPUs"
echo "  • Week 5-8: Burst to 8 GPUs"
echo "  • Auto-scaling handles organic growth"
echo "  • No manual provisioning needed"
echo "  • Pay only for actual usage"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 6: Multi-region federation
info "🌐 Step 6: Multi-Region Federation"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Global Architecture:"
echo ""
echo "  Primary Cluster (East US):"
echo "    • 10 GPUs baseline"
echo "    • Handles: US & Europe traffic"
echo ""
echo "  Burst Regions (configured via AzureGPUSource):"
echo "    • West US: 0-20 GPUs (low latency for US West)"
echo "    • North Europe: 0-15 GPUs (GDPR compliance)"
echo "    • Southeast Asia: 0-10 GPUs (APAC traffic)"
echo ""
echo "  Intelligent Routing:"
echo "    • Latency-based: Route to nearest GPU"
echo "    • Compliance: Keep EU data in EU"
echo "    • Cost-based: Prefer cheaper regions"
echo "    • Failover: Auto-switch if region down"
echo ""
echo "  Benefits:"
echo "    ✓ < 100ms global latency"
echo "    ✓ Data sovereignty compliance"
echo "    ✓ 99.99% availability"
echo "    ✓ Optimal cost per region"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 7: Spot instance handling
info "💰 Step 7: Spot Instance Management"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "What are Spot Instances?"
echo "  • Azure's spare capacity sold at 60-90% discount"
echo "  • Can be evicted with 30-second notice"
echo "  • Perfect for burst workloads"
echo ""
echo "Tensor Fusion Spot Handling:"
echo ""
echo "  1. Preemption Detection"
echo "     • Azure sends eviction notice (30s warning)"
echo "     • Platform immediately drains workloads"
echo "     • Migrates to other available GPUs"
echo ""
echo "  2. Graceful Degradation"
echo "     • If spot unavailable → fallback to on-demand"
echo "     • Temporarily higher cost for reliability"
echo "     • Alert sent to ops team"
echo ""
echo "  3. Checkpointing"
echo "     • Long-running jobs checkpoint every 5 min"
echo "     • Resume on new GPU after eviction"
echo "     • No work lost"
echo ""
echo "  4. Cost Optimization"
echo "     • Mix: 70% spot, 30% on-demand"
echo "     • Balances cost vs reliability"
echo "     • Average savings: 55%"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 8: Configuration example
info "⚙️  Step 8: Complete Configuration Example"
echo ""
echo "═══════════════════════════════════════════════════════════════"
cat << 'YAML'
apiVersion: tensor-fusion.ai/v1
kind: AzureGPUSource
metadata:
  name: production-burst
spec:
  # Azure Connection
  subscriptionID: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  resourceGroup: "tf-production-rg"
  location: "eastus"
  
  # GPU Configuration
  vmSize: "Standard_NC24ads_A100_v4"  # NVIDIA A100
  priority: "Spot"
  maxPrice: "5.00"
  evictionPolicy: "Deallocate"
  
  # Auto-Scaling
  minInstances: 0
  maxInstances: 50
  targetUtilization: 0.75
  scaleUpThreshold: 0.80
  scaleDownThreshold: 0.30
  cooldownPeriod: "10m"
  
  # Cost Management
  maxMonthlyCost: "10000"
  costAlertThreshold: 0.80
  enableCostAlerts: true
  
  # Scheduling
  schedule:
    # Scale up before business hours
    scaleUp:
    - cron: "0 8 * * 1-5"  # 8 AM Mon-Fri
      instances: 10
    # Scale down after hours
    scaleDown:
    - cron: "0 18 * * 1-5"  # 6 PM Mon-Fri
      instances: 2
  
  # Networking
  virtualNetwork: "aks-production-vnet"
  subnet: "gpu-burst-subnet"
  networkSecurityGroup: "gpu-nsg"
  
  # Health & Monitoring
  healthCheckInterval: "30s"
  healthCheckTimeout: "10s"
  unhealthyThreshold: 3
  
  # Integration
  targetGPUPool: "default-pool"
  tags:
    environment: "production"
    cost-center: "ml-platform"
    auto-shutdown: "true"
YAML
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Step 9: Check existing sources
info "📊 Step 9: Current Azure GPU Sources"
echo ""
AZURE_SOURCES=$(kubectl get azuregpusource --all-namespaces -o custom-columns=\
NAMESPACE:.metadata.namespace,\
NAME:.metadata.name,\
LOCATION:.spec.location,\
VM-SIZE:.spec.vmSize,\
MIN:.spec.minInstances,\
MAX:.spec.maxInstances 2>/dev/null)

if [ -n "$AZURE_SOURCES" ]; then
  echo "$AZURE_SOURCES"
else
  info "No Azure GPU sources configured yet"
  info "  Use the configuration above to create one"
fi
echo ""
sleep 2

# Summary
echo "═══════════════════════════════════════════════════════════════"
success "🎯 Key Takeaways:"
echo "   ✓ Cloud bursting reduces costs by 87% vs always-peak"
echo "   ✓ Spot instances provide 60-90% discount on GPU costs"
echo "   ✓ Auto-scaling handles traffic spikes automatically"
echo "   ✓ Multi-region federation for global low latency"
echo "   ✓ Graceful handling of spot instance evictions"
echo "   ✓ Budget controls prevent cost overruns"
echo "   ✓ Scheduled scaling for predictable patterns"
echo ""
info "💡 Use Case: Variable workloads, cost optimization, global platforms"
echo "═══════════════════════════════════════════════════════════════"
echo ""

info "💡 Next Steps:"
echo "  1. Configure Azure credentials"
echo "  2. Create AzureGPUSource with your settings"
echo "  3. Test scaling: kubectl apply -f burst-config.yaml"
echo "  4. Monitor costs: kubectl describe azuregpusource <name>"
echo ""
info "Demo complete! Resources will be cleaned up automatically."
sleep 2

