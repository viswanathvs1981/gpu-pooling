# TensorFusion Use Cases - Quick Start Guide

## 🚀 Run Your First Demo (30 seconds)

```bash
cd use-cases
bash 02-cost-optimization.sh
```

This will show you how TensorFusion saves 67% on GPU costs with auto-scaling!

---

## 📋 All Available Demos

| Demo | Time | Command |
|------|------|---------|
| **Multi-Tenant Quotas** | 2 min | `bash 01-multi-tenant-quotas.sh` |
| **Cost Optimization** ⭐ | 5 min | `bash 02-cost-optimization.sh` |
| **Fractional GPU** | 3 min | `bash 03-fractional-gpu.sh` |
| **Distributed Training** | 3 min | `bash 05-distributed-training.sh` |

⭐ = Recommended first demo

---

## 🎬 Run All Demos at Once

```bash
# Run all demos with 3-second delays between them
bash run-all-demos.sh

# Or with custom delay
bash run-all-demos.sh --delay 10

# Skip specific demos
bash run-all-demos.sh --skip 2
```

---

## 📊 What Each Demo Shows

### 1. Multi-Tenant Quotas
**Shows**: Fair GPU allocation, quota enforcement, multi-tenancy

**You'll see**:
- Team A gets 50 TFlops quota
- Team B gets 30 TFlops quota
- Deployments within quota ✅ succeed
- Deployments exceeding quota ❌ rejected

### 2. Cost Optimization ⭐
**Shows**: Auto-scaling, zero-cost idle state, on-demand provisioning

**You'll see**:
- 0 GPU nodes = $0/hour
- Deploy workload → node auto-provisions in 2-3 min
- Pod runs nvidia-smi on Tesla T4
- 67% cost savings calculation

### 3. Fractional GPU
**Shows**: GPU sharing, 3x utilization improvement, cost efficiency

**You'll see**:
- 3 workloads requesting 20 TFlops each
- All 3 share 1 GPU (65 TFlops total)
- 92% utilization vs 33% typical
- 66% cost reduction

### 5. Distributed Training
**Shows**: Multi-worker training, A2A communication, Redis pub/sub

**You'll see**:
- 3 training workers (rank 0, 1, 2)
- Gradient synchronization messages
- <5ms communication latency
- Worker coordination

---

## 🧹 Cleanup

```bash
# Clean up all demo resources
bash cleanup-all-demos.sh
```

This removes all test pods, namespaces, and resources created by demos.

---

## 💡 Tips for Great Demos

### For Quick Demos (5 min)
Run just the cost optimization demo:
```bash
bash 02-cost-optimization.sh
```

### For Technical Audiences (15 min)
Run these in sequence:
```bash
bash 02-cost-optimization.sh
bash 03-fractional-gpu.sh
bash 05-distributed-training.sh
```

### For Executive Presentations
Focus on business value:
1. Cost savings: `02-cost-optimization.sh`
2. Efficiency: `03-fractional-gpu.sh`
3. Governance: `01-multi-tenant-quotas.sh`

---

## 🔧 Prerequisites

### Minimum (for demos 1, 5)
- ✅ TensorFusion deployed
- ✅ kubectl configured

### For GPU Demos (2, 3)
- ✅ GPU quota in Azure (NCASv3_T4)
- ✅ `add-gpu-node.sh` script available

To add GPU quota:
```bash
# Request in Azure Portal
# OR run deployment which checks quota automatically
cd ..
bash deploy-all.sh
```

---

## ❓ Troubleshooting

### "No GPU nodes found"
GPU nodes scaled to 0 (normal for cost savings).

**Solution**: Deploy GPU workload and wait 2-3 minutes:
```bash
cd ..
bash add-gpu-node.sh
```

### "Demo pods pending"
Waiting for GPU node to provision.

**Check status**:
```bash
kubectl get nodes -l pool=gpu -w
```

### "Script permission denied"
Make scripts executable:
```bash
chmod +x *.sh
```

---

## 📖 Learn More

- **Full README**: See `README.md` in this directory
- **Platform docs**: See `../README.md`
- **Deployment**: See `../deploy-all.sh`
- **Verification**: See `../scripts/verify-all.sh`

---

## 🎯 Expected Results

Every demo script shows:
- ✅ **Green checkmarks** for successful steps
- 📊 **Metrics** showing resource usage
- 💰 **Cost analysis** where applicable
- 🎯 **Key takeaways** summarizing benefits

---

## 🚀 Ready to Demo!

Start with the cost optimization demo - it's the most impressive:

```bash
bash 02-cost-optimization.sh
```

Watch as GPU nodes auto-provision on-demand, showing real cost savings in action! 💰✨

