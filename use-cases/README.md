# TensorFusion Platform - Use Cases & Demos

This directory contains ready-to-run scripts demonstrating real-world use cases solved by the TensorFusion platform.

## 📋 Available Use Cases

| # | Use Case | Script | Duration | Prerequisites |
|---|----------|--------|----------|---------------|
| 1 | Multi-Tenant GPU Quotas | `01-multi-tenant-quotas.sh` | 3 min | None |
| 2 | Cost-Optimized Auto-Scaling | `02-cost-optimization.sh` | 5 min | GPU quota |
| 3 | Fractional GPU Sharing | `03-fractional-gpu.sh` | 4 min | GPU node |
| 4 | Intelligent LLM Routing | `04-llm-routing.sh` | 4 min | None |
| 5 | Distributed Training (A2A) | `05-distributed-training.sh` | 5 min | Redis |
| 6 | Workload Intelligence | `06-workload-intelligence.sh` | 4 min | None |
| 7 | vLLM Deployment & Inference | `07-vllm-deployment.sh` | 5 min | None |
| 8 | LoRA Training & Custom Models | `08-lora-training.sh` | 5 min | None |
| 9 | GPU Resource Monitoring | `09-gpu-monitoring.sh` | 4 min | None |
| 10 | Azure Cloud Bursting | `10-azure-bursting.sh` | 4 min | Azure creds |

## 🚀 Quick Start

### Run All Use Cases
```bash
bash run-all-demos.sh
```

### Run Individual Use Case
```bash
cd use-cases
bash 01-multi-tenant-quotas.sh
```

## 📊 What Each Script Does

### USE CASE 1: Multi-Tenant GPU Quotas
**Problem Solved**: Fair GPU resource allocation across multiple teams

**What to Expect**:
- Creates 2 namespaces (team-a, team-b) with different quotas
- Team A: 50 TFlops, 10Gi VRAM
- Team B: 30 TFlops, 5Gi VRAM
- Deploys workloads within quota → SUCCESS
- Attempts to exceed quota → REJECTION
- Shows quota usage and enforcement

**Key Metrics**:
- 100% isolation between tenants
- Predictable costs per tenant
- No "noisy neighbor" problems

---

### USE CASE 2: Cost-Optimized Auto-Scaling
**Problem Solved**: Eliminate GPU costs when idle, auto-provision on demand

**What to Expect**:
- Shows 0 GPU nodes (cost = $0/hour)
- Deploys GPU workload
- Watches autoscaler provision GPU node (2-3 minutes)
- Pod schedules to new GPU node
- Demonstrates cost savings (60-80% reduction)

**Key Metrics**:
- 60-80% cost reduction vs always-on
- 5-minute scale-up time
- Automatic cleanup of idle resources

---

### USE CASE 3: Fractional GPU Sharing
**Problem Solved**: Multiple workloads sharing a single GPU efficiently

**What to Expect**:
- Shows 1 GPU with 65 TFlops available
- Deploys 3 workloads requesting fractional GPUs
- All 3 schedule to same GPU node
- Demonstrates 3x GPU utilization improvement
- Cost savings calculation

**Key Metrics**:
- 4-10× more workloads per GPU
- 70-90% cost reduction
- Maintained performance isolation

---

### USE CASE 4: Intelligent LLM Routing
**Problem Solved**: Optimize cost & latency with smart routing across providers

**What to Expect**:
- Creates cost-based routing (short → Azure, long → self-hosted)
- Pattern-based routing (code → CodeLlama, support → tuned model)
- Automatic failover configuration
- Cost impact analysis showing 58% savings

**Key Metrics**:
- 58% cost reduction vs single provider
- Automatic failover on backend failure
- Request pattern matching

---

### USE CASE 5: Distributed Training with A2A
**Problem Solved**: Efficient multi-GPU training with gradient synchronization

**What to Expect**:
- Deploys 3 training workers with Redis pub/sub
- Shows gradient synchronization messages
- Demonstrates worker coordination
- Tests message passing latency

**Key Metrics**:
- Sub-millisecond message latency
- Scales to 100+ agents
- Distributed parameter updates

---

### USE CASE 6: Workload Intelligence & Auto-Recommendations
**Problem Solved**: Users don't know what GPU resources they need

**What to Expect**:
- Creates workload profiles (7B LLM, 70B LLM, LoRA training)
- AI-generated resource recommendations
- Cost comparison (full fine-tuning vs LoRA: 98% cheaper!)
- Right-sizing to prevent over/under-provisioning

**Key Metrics**:
- 40-60% savings through right-sizing
- LoRA 98% cheaper than full fine-tuning
- Accurate cost forecasting

---

### USE CASE 7: vLLM Deployment & High-Performance Inference
**Problem Solved**: Need cost-effective, high-throughput LLM serving

**What to Expect**:
- Explains vLLM and PagedAttention technology
- Performance comparison (6× better than naive PyTorch)
- Continuous batching for 100% GPU utilization
- LoRA adapter support (100+ customers per GPU)
- Integration with Tensor Fusion

**Key Metrics**:
- 6× better throughput vs PyTorch
- 6× cost reduction
- OpenAI-compatible API

---

### USE CASE 8: LoRA Training & Custom Model Creation
**Problem Solved**: Custom AI models are expensive & slow to train

**What to Expect**:
- Explains LoRA (Low-Rank Adaptation) technology
- Cost comparison: $100 vs $5,000 for customization
- Training workflow from data to deployment
- Multi-tenant model serving (100+ models per GPU)
- Real-world use cases (legal, healthcare, support)

**Key Metrics**:
- 50× cheaper training
- 24× faster (2 hours vs 48 hours)
- 320× smaller model files (50MB vs 16GB)

---

### USE CASE 9: GPU Resource Monitoring & Observability
**Problem Solved**: Need visibility into GPU usage, costs, and performance

**What to Expect**:
- Real-time GPU utilization tracking
- Multi-tenant resource breakdown
- Per-tenant cost allocation (50% savings with vGPU)
- Performance metrics (latency, throughput)
- Proactive alerts & anomaly detection

**Key Metrics**:
- Complete observability across all layers
- Per-tenant cost tracking
- Automated capacity planning

---

### USE CASE 10: Azure Cloud Bursting & Hybrid Scaling
**Problem Solved**: Handle traffic spikes without over-provisioning

**What to Expect**:
- AzureGPUSource CRD configuration
- Auto-scaling workflow (scale up/down)
- Spot instance management (60-90% discount)
- Multi-region federation
- Cost optimization (87% savings vs always-peak)

**Key Metrics**:
- 87% cost reduction through bursting
- 3-minute scale-up time
- Graceful spot instance eviction handling

---

## 🎬 Demo Recommendations

### For Executives (5 minutes)
Show the business value:
1. `02-cost-optimization.sh` - 60-80% cost savings
2. `03-fractional-gpu.sh` - 70-90% efficiency gains
3. `04-llm-routing.sh` - 58% LLM cost reduction

### For Technical Audience (15 minutes)
Show the technical capabilities:
1. `03-fractional-gpu.sh` - GPU virtualization
2. `07-vllm-deployment.sh` - High-performance inference
3. `08-lora-training.sh` - Custom model training
4. `09-gpu-monitoring.sh` - Observability

### For Deep Dive (30 minutes)
Run all 10 use cases with `run-all-demos.sh`

### For AI/ML Teams (20 minutes)
Show the ML workflow:
1. `06-workload-intelligence.sh` - Resource planning
2. `08-lora-training.sh` - Model customization
3. `07-vllm-deployment.sh` - Serving & inference
4. `04-llm-routing.sh` - Smart routing

## 📋 Prerequisites

### Minimum Requirements
- ✅ TensorFusion platform deployed (`bash deploy-all.sh`)
- ✅ kubectl configured
- ✅ bash, jq installed

### Optional (for specific use cases)
- GPU quota for use cases 2, 3, 5
- Azure credentials for use case 10
- vLLM deployed for use case 7, 8

## 🛠️ Troubleshooting

### GPU nodes scaled to 0
```bash
# Check GPU nodes
kubectl get nodes -l nvidia.com/gpu.present=true

# Manually trigger workload to scale up
bash use-cases/02-cost-optimization.sh
```

### Script hangs waiting for pod
```bash
# Check pod status
kubectl get pods -A

# Check events
kubectl get events --sort-by='.lastTimestamp' | tail -20

# Check autoscaler logs
kubectl logs -n kube-system -l app=cluster-autoscaler
```

### Cleanup between demos
```bash
# Clean up all test resources
bash cleanup-all-demos.sh
```

## 📊 Script Features

All scripts include:
- ✅ **Self-documenting**: Clear output explaining each step
- ✅ **Color-coded**: Green (success), Blue (info), Yellow (warning)
- ✅ **Progress indicators**: Shows what's happening in real-time
- ✅ **Automatic cleanup**: Removes test resources at end
- ✅ **Error handling**: Graceful failures with helpful messages
- ✅ **Expected results**: Tells you what to look for

## 🔄 Running All Demos

```bash
# Run all use cases in sequence
bash run-all-demos.sh

# Skip specific use cases
SKIP_CASES="2,10" bash run-all-demos.sh
```

## 💡 Tips for Great Demos

1. **Pre-provision GPU node** if showing use cases 2, 3, or 5
2. **Check quota** before running use case 2
3. **Review script first** to understand expected behavior
4. **Run verify-all.sh** before demos to ensure platform health
5. **Run cleanup** between demos to reset state

## 🎯 What Each Demo Validates

| Use Case | Validates |
|----------|-----------|
| 1 | GPUResourceQuota CRD, quota enforcement |
| 2 | Cluster autoscaler, cost optimization |
| 3 | Fractional GPU sharing, vGPU scheduling |
| 4 | LLMRoute CRD, Portkey integration |
| 5 | A2A communication, Redis pub/sub |
| 6 | WorkloadIntelligence CRD, recommendations |
| 7 | vLLM integration, PagedAttention |
| 8 | LoRA training, adapter management |
| 9 | Prometheus/Grafana, metrics collection |
| 10 | AzureGPUSource CRD, cloud bursting |

## 📖 Additional Resources

- **Platform Documentation**: See NexusAI architecture HTML file
- **Deployment Guide**: `../deploy-all.sh`
- **Verification Tests**: `../scripts/verify-all.sh`
- **API Examples**: `../examples/`
- **Quick Start**: `QUICKSTART.md`

## 🚀 Success Criteria

After running a use case script, you should see:
- ✅ Green checkmarks for successful steps
- 📊 Metrics showing resource usage
- 🎯 Expected outcomes clearly displayed
- 💰 Cost savings calculations
- 🧹 Automatic cleanup confirmation

---

**Ready to demo?** Start with these based on your audience:
- **Business**: `02-cost-optimization.sh` (most impressive ROI)
- **Technical**: `07-vllm-deployment.sh` (best technology explanation)
- **ML Engineers**: `08-lora-training.sh` (most relevant workflow)

🚀 **Quick Demo Command**: `bash run-all-demos.sh`
