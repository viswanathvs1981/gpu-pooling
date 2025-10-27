# 🧪 TensorFusion - Systematic Testing Plan

## Overview
We'll test all 17 capabilities in order of priority and complexity.

---

## 📊 Summary of Capabilities

**Total Capabilities**: 17
- 🎮 Core GPU Management: 4 capabilities
- 🤖 AI/ML Workloads: 3 capabilities  
- ☁️ Cloud Integration: 1 capability
- 🌐 Multi-Cluster: 2 capabilities
- 🎯 Scheduling: 1 capability
- 📊 Observability: 2 capabilities
- 🔧 Infrastructure: 3 capabilities
- 🔐 Security: 1 capability

---

## 🚀 Testing Order (Priority-Based)

### **Phase 1: Core GPU Features** (30-40 mins)
These are essential features that everything else depends on.

**Test 1**: GPU Pooling & Allocation ⭐⭐⭐
**Test 2**: Simple GPU Workload ⭐⭐⭐
**Test 3**: Fractional GPU Sharing ⭐⭐⭐
**Test 4**: GPU Resource Quotas ⭐⭐⭐

### **Phase 2: AI/ML Workloads** (45-60 mins)
Core AI/ML capabilities.

**Test 5**: LLM Routing & Load Balancing ⭐⭐
**Test 6**: Workload Intelligence ⭐⭐
**Test 7**: Distributed Training ⭐⭐

### **Phase 3: Observability** (20-30 mins)
Monitoring and analytics.

**Test 8**: GPU Metrics & Monitoring ⭐⭐
**Test 9**: Storage & Data Management ⭐
**Test 10**: AI Gateway (Portkey) ⭐

### **Phase 4: Advanced Features** (60-90 mins)
Complex, enterprise features.

**Test 11**: Azure GPU Auto-Provisioning ⭐
**Test 12**: Multi-Cluster Management ⭐
**Test 13**: Custom Scheduling Policies ⭐
**Test 14**: Multi-Tenant Isolation ⭐

---

## ✅ Pre-Flight Check

Before we start testing, let's verify the environment:

```bash
# 1. Check cluster status
kubectl get nodes

# 2. Verify GPU nodes
kubectl get nodes -l pool=gpu

# 3. Check GPU availability
kubectl describe node -l pool=gpu | grep nvidia.com/gpu

# 4. Verify all CRDs
kubectl get crds | grep tensor-fusion

# 5. Check TensorFusion controller
kubectl get pods -n tensor-fusion-sys

# 6. Verify existing resources
kubectl get gpupools,gpuresourcequotas,llmroutes,workloadintelligences -A
```

---

## 📋 Detailed Test Cases

### TEST 1: GPU Pooling & Allocation (5 mins) ⭐⭐⭐

**Objective**: Verify GPU pool creation and GPU discovery

**Steps**:
1. Check existing GPU pool
2. Verify GPUs are discovered
3. Check GPU allocation state
4. View GPU details

**Expected Results**:
- GPU pool exists
- GPUs are registered
- GPU status shows available/allocated
- GPU metadata visible

**Commands**:
```bash
kubectl get gpupool -A
kubectl get gpus -A
kubectl get gpunodes -A
kubectl describe gpupool -A
```

---

### TEST 2: Simple GPU Workload (10 mins) ⭐⭐⭐

**Objective**: Deploy basic GPU workload and verify execution

**Steps**:
1. Deploy CUDA test pod
2. Verify pod scheduling
3. Check GPU allocation
4. View pod logs (nvidia-smi output)
5. Monitor GPU usage

**Expected Results**:
- Pod scheduled on GPU node
- GPU allocated to pod
- nvidia-smi shows GPU info
- Pod completes successfully

**Test Pod**:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: gpu-test-simple
spec:
  containers:
  - name: cuda
    image: nvidia/cuda:12.2.0-base-ubuntu22.04
    command: ["nvidia-smi"]
    resources:
      limits:
        nvidia.com/gpu: 1
  tolerations:
  - key: nvidia.com/gpu
    operator: Exists
    effect: NoSchedule
```

---

### TEST 3: Fractional GPU Sharing (15 mins) ⭐⭐⭐

**Objective**: Multiple pods sharing single GPU

**Steps**:
1. Deploy example fractional GPU sharing
2. Verify multiple pods on same GPU
3. Check memory allocation
4. Monitor GPU utilization

**Expected Results**:
- Multiple pods scheduled
- All pods on same GPU node
- GPU memory partitioned
- No interference between pods

**Example**: `examples/02-fractional-gpu-pod.yaml`

---

### TEST 4: GPU Resource Quotas (10 mins) ⭐⭐⭐

**Objective**: Enforce GPU limits per namespace

**Steps**:
1. Create test namespace
2. Apply GPU quota (e.g., max 1 GPU)
3. Deploy workload within quota (should succeed)
4. Deploy workload exceeding quota (should fail)
5. Verify quota enforcement

**Expected Results**:
- Quota created successfully
- Workloads within quota allowed
- Workloads exceeding quota blocked
- Clear error message

---

### TEST 5: LLM Routing (20 mins) ⭐⭐

**Objective**: Route LLM requests with load balancing

**Steps**:
1. Check existing LLM routes
2. View route configuration
3. Deploy test LLM service (mock)
4. Send test requests
5. Verify load balancing

**Expected Results**:
- LLM routes created
- Routes target correct services
- Requests distributed
- Failover works

**Example**: `examples/03-llm-route.yaml`

---

### TEST 6: Workload Intelligence (15 mins) ⭐⭐

**Objective**: Auto-optimization of workload resources

**Steps**:
1. Check workload intelligence config
2. Deploy workload with auto-optimization
3. Monitor resource recommendations
4. Verify automatic adjustments
5. Check performance metrics

**Expected Results**:
- Intelligence profile active
- Recommendations generated
- Resources adjusted
- Metrics collected

**Example**: `examples/05-workload-intelligence.yaml`

---

### TEST 7: Distributed Training (25 mins) ⭐⭐

**Objective**: Multi-GPU distributed training

**Steps**:
1. Deploy distributed training job
2. Verify all worker pods
3. Check A2A communication
4. Monitor training progress
5. Verify gradient sync

**Expected Results**:
- All workers scheduled
- A2A communication working
- Training progresses
- Logs show synchronization

**Example**: `examples/06-distributed-training.yaml`

---

### TEST 8: GPU Metrics & Monitoring (15 mins) ⭐⭐

**Objective**: View GPU utilization and metrics

**Steps**:
1. Port-forward to Grafana
2. Access dashboards
3. View GPU metrics
4. Check Prometheus targets
5. Query GPU utilization

**Expected Results**:
- Grafana accessible
- GPU dashboards visible
- Real-time metrics showing
- Historical data available

**Commands**:
```bash
kubectl port-forward -n observability svc/grafana 3000:80
# Open: http://localhost:3000
```

---

### TEST 9: Storage & Data (10 mins) ⭐

**Objective**: Verify Qdrant, GreptimeDB, Redis

**Steps**:
1. Test Qdrant vector storage
2. Query GreptimeDB metrics
3. Test Redis pub/sub
4. Verify data persistence

**Expected Results**:
- All storage systems responding
- Data can be stored/retrieved
- Metrics queryable
- Pub/sub working

---

### TEST 10: AI Gateway (10 mins) ⭐

**Objective**: Test Portkey gateway

**Steps**:
1. Port-forward to Portkey
2. Send test API request
3. Verify routing
4. Check logs

**Expected Results**:
- Gateway accessible
- Requests routed correctly
- Logs captured

---

### TEST 11-14: Advanced Features (As Time Permits)

These require more setup and configuration. We'll test if time allows.

---

## 🎯 Success Criteria

For each test:
- ✅ Resources created successfully
- ✅ Expected behavior observed
- ✅ No errors in logs
- ✅ Metrics/monitoring working
- ✅ Documentation matches reality

---

## 📝 Test Results Template

For each test, we'll document:
- **Status**: Pass/Fail/Partial
- **Duration**: Actual time taken
- **Issues Found**: Any problems
- **Screenshots**: Key observations
- **Next Steps**: Follow-up actions

---

## 🚀 Let's Start!

We'll begin with **Phase 1: Core GPU Features** and work through systematically.

Ready to start with **TEST 1: GPU Pooling & Allocation**?

