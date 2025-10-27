# 🎯 Updated Deployment & Validation Scripts

## Changes Made

### 1. `deploy-all.sh` Enhancements

#### **GPU Quota Checking**
- Added `check_gpu_quota()` function that validates NCASv3_T4 quota before attempting GPU node creation
- Automatically selects `Standard_NC4as_T4_v3` (most cost-effective T4 GPU) when quota is available
- Provides clear guidance when quota is insufficient

#### **Improved GPU Nodepool Creation**
- Checks for both 'gpu' and 'gpunodes' nodepool names (backward compatible)
- Better error handling with actionable guidance
- Auto-detection of GPU quota availability
- Clear messaging about autoscaling configuration (min=0, max=2)

#### **GPU Node Verification**
- Added `verify_gpu_nodes()` function in deployment summary
- Checks GPU node count
- Verifies GPU detection by Kubernetes (nvidia.com/gpu capacity)
- Reports total GPU count across all nodes
- Warns if GPU drivers are still installing

#### **Enhanced Deployment Summary**
- Shows cluster nodes with full details
- Displays GPU verification results
- Reports GPU operator status
- Clear next steps for adding GPU nodes if quota was insufficient

### 2. `scripts/verify-all.sh` Enhancements

#### **Comprehensive GPU Node Testing**
- Enhanced `test_gpu_nodes()` function:
  - Iterates through all GPU nodes
  - Checks GPU capacity on each node
  - Reports per-node GPU count
  - Verifies GPU Operator pod status
  - Provides guidance when GPUs not detected

#### **New GPU Workload Test**
- Added `test_gpu_workload()` function:
  - Deploys actual GPU test pod (`nvidia/cuda`)
  - Runs nvidia-smi to verify GPU access
  - Waits for pod completion (60s timeout)
  - Cleans up test resources
  - Validates GPU workload scheduling works end-to-end

#### **Better Error Handling**
- Graceful handling when GPU nodes scaled to 0
- Clear messaging about autoscaler behavior
- Actionable guidance for adding GPU nodes

---

## What Gets Verified

### Infrastructure
- ✅ Namespaces (7 total)
- ✅ Pods in all namespaces
- ✅ All 14 CRDs installed

### Core Components
- ✅ Redis (message bus)
- ✅ GreptimeDB (time-series)
- ✅ Qdrant (vector DB)
- ✅ Prometheus & Grafana
- ✅ Portkey Gateway
- ✅ TensorFusion Controller
- ✅ Alert Manager

### GPU Features
- ✅ GPU nodes presence
- ✅ GPU capacity detection (per node)
- ✅ GPU Operator pods running
- ✅ Total GPU count
- ✅ **NEW**: Actual GPU workload deployment
- ✅ **NEW**: nvidia-smi execution
- ✅ **NEW**: GPU scheduling validation

### CRD Functionality
- ✅ All 14 CRDs registered
- ✅ GPU Pools working
- ✅ GPU Nodes CRD
- ✅ GPU Resource Quotas
- ✅ LLM Routes
- ✅ Workload Intelligence
- ✅ TensorFusion Clusters
- ✅ Azure GPU Sources

### Workflows
- ✅ Fractional GPU allocation
- ✅ A2A communication

---

## Usage

### Deploy with GPU Quota Check
```bash
# Full deployment with automatic GPU quota validation
./deploy-all.sh

# Deploy without GPU nodes
ENABLE_GPU_POOL=false ./deploy-all.sh

# Custom GPU configuration
GPU_NODE_SIZE="Standard_NC4as_T4_v3" \
GPU_NODE_MIN=0 \
GPU_NODE_MAX=2 \
./deploy-all.sh
```

### Comprehensive Validation
```bash
# Run full verification suite (includes GPU workload test)
./scripts/verify-all.sh
```

### What to Expect

#### Successful Deployment Output:
```
╔════════════════════════════════════════════════════════════════╗
║   Deployment Complete                                          ║
╚════════════════════════════════════════════════════════════════╝

[INFO] Cluster nodes:
NAME                             STATUS   ROLES    AGE     VERSION
aks-gpu-27560491-vmss000001      Ready    <none>   10m     v1.32.7
aks-system-16932086-vmss000000   Ready    <none>   2h      v1.32.7

[INFO] Verifying GPU nodes...
[SUCCESS] GPU nodes available: 1
[SUCCESS] Node aks-gpu-27560491-vmss000001: 1 GPU(s) detected
[SUCCESS] Total GPUs available in cluster: 1
```

#### Successful Verification Output:
```
[INFO] Testing GPU Nodes
[✓] GPU nodes available: 1
[✓] Node aks-gpu-27560491-vmss000001: 1 GPU(s) detected
[✓] Total GPUs in cluster: 1
[✓] NVIDIA GPU Operator running on 1 nodes

[INFO] Testing GPU Workload Capability
[INFO] Deploying test GPU workload...
[✓] GPU workload test succeeded
[✓] nvidia-smi output detected

Tests Passed: 23/23
Platform Status: ✅ OPERATIONAL
```

---

## Key Improvements

1. **Proactive Quota Checking**: Validates GPU quota before attempting node creation
2. **Better Error Messages**: Clear guidance when quota is insufficient
3. **GPU Verification**: Actually tests GPU workloads, not just node presence
4. **End-to-End Validation**: Confirms GPU scheduling and CUDA access works
5. **Autoscaler Aware**: Handles nodes scaling to 0 gracefully
6. **Cost Optimization**: Uses most cost-effective GPU (NC4as_T4_v3)
7. **Backward Compatible**: Works with existing 'gpunodes' or new 'gpu' nodepool names

---

## Troubleshooting

### If GPU Quota is Insufficient:
```
[WARNING] NCASv3_T4 quota: 0 vCPUs (need at least 4 for 1 GPU node)
[WARNING] To request quota: https://portal.azure.com/...
[WARNING] After quota approval, add GPU nodes with: ./add-gpu-node.sh
```

### If GPU Nodes Scaled to 0:
```
[⚠] No GPU nodes found (may be scaled to 0 by autoscaler)
[INFO] To add GPU nodes: ./add-gpu-node.sh
```

### If GPU Drivers Installing:
```
[⚠] Node aks-gpu-xxx: GPUs not detected (driver may be installing)
[⚠] GPU nodes exist but GPUs not yet detected by Kubernetes
[INFO] Check GPU operator status: kubectl get pods -n gpu-operator
```

---

## Files Modified

1. `/deploy-all.sh`
   - Added `check_gpu_quota()`
   - Enhanced `create_aks()` with quota validation
   - Added `verify_gpu_nodes()`
   - Updated `print_summary()` with GPU verification

2. `/scripts/verify-all.sh`
   - Enhanced `test_gpu_nodes()` with per-node GPU detection
   - Added `test_gpu_workload()` for end-to-end GPU testing
   - Integrated GPU workload test into main flow

---

**Result**: Deployment is now more robust, provides better feedback, and validates GPU functionality end-to-end! 🚀
