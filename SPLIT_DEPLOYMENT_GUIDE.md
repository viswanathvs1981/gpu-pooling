# NexusAI Platform - Split Deployment Guide

## Overview

The NexusAI platform deployment has been split into **5 sequential steps** to provide better control, easier debugging, and avoid API rate limiting issues.

## 🎯 Why Split Deployment?

### Benefits
✅ **Better Control**: Deploy one component at a time
✅ **Easier Debugging**: Isolate issues to specific components
✅ **Rate Limit Prevention**: Reduces API load per operation
✅ **Flexible Updates**: Update individual components without full redeployment
✅ **Clear Progress**: See exactly what's deployed at each stage
✅ **Selective Deployment**: Deploy only what you need

---

## 📦 Deployment Steps

### Step 1: Core Controller (Operator)
**Script**: `deploy-1-core.sh`
**Time**: ~1-2 minutes

Deploys the main NexusAI operator that manages all custom resources (CRDs).

```bash
./deploy-1-core.sh
```

**What it deploys:**
- NexusAI Operator (Controller Manager)
- Manages: GPUNode, GPUPool, LLMRoute, etc.

**Verify:**
```bash
kubectl get deployment -n tensor-fusion-sys -l app.kubernetes.io/component=operator
```

---

### Step 2: Node Discovery
**Script**: `deploy-2-node-discovery.sh`
**Time**: ~1-2 minutes

Deploys the DaemonSet that discovers GPU nodes and creates GPUNode CRs.

```bash
./deploy-2-node-discovery.sh
```

**What it deploys:**
- Node Discovery DaemonSet (runs on every node)
- Detects NVIDIA GPUs
- Creates GPUNode and GPU custom resources

**Verify:**
```bash
kubectl get daemonset -n tensor-fusion-sys -l app.kubernetes.io/component=node-discovery
kubectl get gpunode
```

---

### Step 3: Platform Services
**Script**: `deploy-3-platform-services.sh`
**Time**: ~2-3 minutes

Deploys core platform services for memory, model management, and discovery.

```bash
./deploy-3-platform-services.sh
```

**What it deploys:**
- **Memory Service**: Agent memory management (semantic, episodic, long-term)
- **Model Catalog**: Model registry and metadata
- **Discovery Agent**: LLM endpoint discovery

**Verify:**
```bash
kubectl get deployment -n tensor-fusion-sys | grep -E "memory|catalog|discovery"
```

---

### Step 4: Agent Services
**Script**: `deploy-4-agent-services.sh`
**Time**: ~2-3 minutes

Deploys DataOps, AI Safety, and Prompt Optimization services.

```bash
./deploy-4-agent-services.sh
```

**What it deploys:**
- **DataOps Agents**: Data pipeline, feature engineering, drift detection
- **AI Safety Service**: Toxicity detection, bias evaluation, red teaming
- **Prompt Optimizer**: Prompt rewriting and optimization

**Verify:**
```bash
kubectl get deployment -n tensor-fusion-sys | grep -E "dataops|aisafety|prompt"
```

---

### Step 5: Python Agents
**Script**: `deploy-5-python-agents.sh`
**Time**: ~2-3 minutes

Deploys Microsoft Agent Framework-based Python agents.

```bash
./deploy-5-python-agents.sh
```

**What it deploys:**
- **Python Agents**: Advanced agents using Microsoft Agent Framework
- Graph-based workflows
- Checkpointing and state management

**Verify:**
```bash
kubectl get deployment -n tensor-fusion-sys | grep python
```

---

## 🚀 Deployment Options

### Option 1: Deploy All (Automated)
Run all 5 steps automatically with delays:

```bash
./deploy-platform-all.sh
```

**Features:**
- Runs all steps in sequence
- 10-second wait between steps (configurable)
- Stops on first failure
- Shows complete status at the end

**Configure wait time:**
```bash
WAIT_BETWEEN_STEPS=15 ./deploy-platform-all.sh
```

---

### Option 2: Deploy Manually (Step by Step)
Run each script manually with full control:

```bash
# Step 1
./deploy-1-core.sh

# Check status, then proceed
kubectl get pods -n tensor-fusion-sys

# Step 2
./deploy-2-node-discovery.sh

# Continue for steps 3-5...
```

**Benefits:**
- Full control over timing
- Inspect each component before proceeding
- Easy to debug issues
- Good for first-time deployment

---

### Option 3: Selective Deployment
Deploy only specific components:

```bash
# Only update Agent Services
./deploy-4-agent-services.sh

# Only update Python Agents
./deploy-5-python-agents.sh
```

**Use Cases:**
- Update a specific service after code changes
- Rollback a specific component
- Test new versions of individual services

---

## 📊 Deployment Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Step 1: Core Controller (Operator)                         │
│ ───────────────────────────────────────────────────────────│
│ • Manages all CRDs                                          │
│ • Controller logic                                          │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 2: Node Discovery (DaemonSet)                         │
│ ───────────────────────────────────────────────────────────│
│ • GPU detection                                             │
│ • GPUNode CR creation                                       │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 3: Platform Services                                  │
│ ───────────────────────────────────────────────────────────│
│ • Memory Service     • Model Catalog     • Discovery       │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 4: Agent Services                                     │
│ ───────────────────────────────────────────────────────────│
│ • DataOps Agents   • AI Safety   • Prompt Optimizer        │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 5: Python Agents (Microsoft Agent Framework)          │
│ ───────────────────────────────────────────────────────────│
│ • Advanced workflows     • Checkpointing     • MSAF         │
└─────────────────────────────────────────────────────────────┘
```

---

## 🔧 Troubleshooting

### Issue: Rate Limiting
**Symptom:** `rate: Wait(n=1) would exceed context deadline`

**Solution 1:** Use split deployment (this approach)
```bash
./deploy-platform-all.sh  # Includes delays between steps
```

**Solution 2:** Increase wait time
```bash
WAIT_BETWEEN_STEPS=20 ./deploy-platform-all.sh
```

**Solution 3:** Increase client limits
```bash
source .kubectl-rate-limit-config
```

---

### Issue: Step Failed
**Symptom:** Deployment script exits with error

**Solution:**
1. Check the error message
2. Inspect pod logs:
   ```bash
   kubectl logs -n tensor-fusion-sys <pod-name>
   ```
3. Check pod status:
   ```bash
   kubectl get pods -n tensor-fusion-sys
   ```
4. Fix the issue
5. Re-run the specific step:
   ```bash
   ./deploy-X-<step>.sh
   ```

---

### Issue: Need to Rollback
**Symptom:** Want to undo a specific deployment step

**Solution:**
```bash
# Remove a specific Helm release
helm uninstall tensor-fusion-<step> -n tensor-fusion-sys

# Examples:
helm uninstall tensor-fusion-core -n tensor-fusion-sys
helm uninstall tensor-fusion-agents -n tensor-fusion-sys
```

---

## 📝 Complete Workflow

### Fresh Deployment

```bash
# 1. One-time ACR setup
./setup-acr.sh

# 2. Build images
./build-images.sh

# 3. Deploy infrastructure (AKS, services)
./deploy-infra.sh

# 4. Deploy NexusAI platform (5 steps)
./deploy-platform-all.sh

# 5. Verify
./scripts/verify-all.sh
```

---

### Update After Code Changes

```bash
# 1. Rebuild images
./build-images.sh

# 2. Update specific component
./deploy-4-agent-services.sh   # If you changed agent code

# OR update all
./deploy-platform-all.sh
```

---

### Selective Update

```bash
# Only update one component
./deploy-3-platform-services.sh

# Verify it's running
kubectl get pods -n tensor-fusion-sys -l app=memory-service
```

---

## 🎛️ Environment Variables

### NAMESPACE
Change the deployment namespace:
```bash
NAMESPACE=my-custom-namespace ./deploy-1-core.sh
```

### RELEASE_NAME
Change the Helm release name prefix:
```bash
RELEASE_NAME=my-release ./deploy-platform-all.sh
```

### WAIT_BETWEEN_STEPS
Configure delay between automated steps:
```bash
WAIT_BETWEEN_STEPS=20 ./deploy-platform-all.sh
```

---

## 📊 Comparison: Single vs Split Deployment

| Feature | Single Script | Split Deployment |
|---------|--------------|------------------|
| Control | Limited | Full per-component |
| Debug | Harder | Easier |
| Rate Limiting | Prone | Avoids |
| Flexibility | Low | High |
| Partial Updates | No | Yes |
| Rollback | All or nothing | Component-level |
| Learning Curve | Easy | Moderate |
| Production Ready | Yes | Yes (Recommended) |

---

## ✅ Verification Commands

After each step:

```bash
# Check deployments
kubectl get deployment -n tensor-fusion-sys

# Check DaemonSets
kubectl get daemonset -n tensor-fusion-sys

# Check all pods
kubectl get pods -n tensor-fusion-sys

# Check Helm releases
helm list -n tensor-fusion-sys

# Check specific component
kubectl logs -n tensor-fusion-sys deployment/memory-service
```

---

## 📚 Related Scripts

| Script | Purpose |
|--------|---------|
| `deploy-1-core.sh` | Step 1: Core Controller |
| `deploy-2-node-discovery.sh` | Step 2: Node Discovery |
| `deploy-3-platform-services.sh` | Step 3: Platform Services |
| `deploy-4-agent-services.sh` | Step 4: Agent Services |
| `deploy-5-python-agents.sh` | Step 5: Python Agents |
| `deploy-platform-all.sh` | Run all 5 steps automatically |
| `deploy-platform.sh` | Old single-script approach (still works) |
| `deploy-infra.sh` | Deploy AKS infrastructure |
| `scripts/verify-all.sh` | Verify deployment |

---

## 🎯 Recommendations

### For Development
✅ Use **manual step-by-step** deployment
✅ Inspect each component before proceeding
✅ Good for understanding the architecture

### For Testing
✅ Use **deploy-platform-all.sh** with default settings
✅ Fast and automated
✅ Includes verification

### For Production
✅ Use **manual step-by-step** deployment first
✅ Then automate with CI/CD using individual scripts
✅ Implement health checks between steps

### For CI/CD
✅ Use individual scripts in pipeline stages
✅ Add verification after each stage
✅ Implement rollback logic per stage

---

## 🚀 Quick Start

**Fastest way to deploy everything:**

```bash
./deploy-platform-all.sh
```

**Most controlled way:**

```bash
./deploy-1-core.sh && \
sleep 10 && \
./deploy-2-node-discovery.sh && \
sleep 10 && \
./deploy-3-platform-services.sh && \
sleep 10 && \
./deploy-4-agent-services.sh && \
sleep 10 && \
./deploy-5-python-agents.sh
```

**Verify it all worked:**

```bash
./scripts/verify-all.sh
```

---

**All deployment scripts are ready! Choose your preferred approach and deploy! 🎉**

