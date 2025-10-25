# TensorFusion - NexusAI Agentic Platform

**Complete GPU Virtualization & LLM Orchestration Platform with Agent-to-Agent Communication**

## 🚀 Quick Start

### Prerequisites

- Azure CLI (`az`)
- kubectl
- helm
- jq
- docker (optional, for building custom images)

### One-Command Deployment

Deploy the entire platform (CPU-only, recommended for testing):

```bash
chmod +x ./deploy-all.sh
ENABLE_GPU_POOL=false BUILD_IMAGES=false ./deploy-all.sh
```

With GPU nodes and custom image builds (requires GPU quota):

```bash
ENABLE_GPU_POOL=true BUILD_IMAGES=true ./deploy-all.sh
```

### What Gets Deployed

The `deploy-all.sh` script automatically provisions:

1. **Azure Infrastructure**
   - Resource Group
   - AKS Cluster (with optional GPU nodepool)
   - Azure Container Registry (ACR)

2. **Supporting Services**
   - **GPU Operator** - NVIDIA GPU management
   - **GreptimeDB** - Time-series metrics storage
   - **Qdrant** - Vector database for workload intelligence
   - **Redis** - Message bus for A2A communication
   - **Prometheus** - Metrics collection
   - **Grafana** - Observability dashboards
   - **Portkey Gateway** - LLM routing and cost optimization

3. **TensorFusion Platform**
   - Custom Resource Definitions (14 CRDs)
   - Controllers for GPU management, LLM routing, workload intelligence
   - Autonomous agents with A2A communication
   - Sample custom resources for testing

### Comprehensive Verification

After deployment, verify everything is working:

```bash
bash scripts/verify-all.sh
```

This validates:
- ✅ All namespaces created
- ✅ All pods running
- ✅ All 14 CRDs installed
- ✅ Redis message bus operational
- ✅ GreptimeDB, Qdrant, Portkey APIs responding
- ✅ Prometheus & Grafana accessible
- ✅ TensorFusion controller running
- ✅ GPU Pool, GPU Quota, LLM Route, Workload Intelligence CRDs functional
- ✅ Fractional GPU allocation working
- ✅ A2A communication operational

---

## 🎯 Core Capabilities

### 1. GPU Virtualization & Management

**Features:**
- GPU Pooling across multiple nodes
- Fractional GPU (vGPU) allocation
- VRAM expansion to host memory/disk
- Dynamic GPU scheduling
- GPU resource quotas per namespace
- QoS and priority-based preemption

**Example:**
```bash
# Create a GPU pool
kubectl apply -f examples/01-gpu-pool.yaml

# Set namespace quotas
kubectl apply -f examples/03-gpu-quota.yaml

# Deploy workload with fractional GPU
kubectl apply -f examples/fractional-gpu-sharing.yaml

# Verify
kubectl get gpupool -A
kubectl get gpuresourcequota -A
kubectl describe pod vgpu-workload-1
```

### 2. Intelligent LLM Routing

**Features:**
- Cost-based routing across multiple LLM providers
- Latency-optimized routing
- Fallback and retry strategies
- Budget tracking and alerting
- Rate limiting and caching
- Portkey AI Gateway integration

**Example:**
```bash
# Create LLM routes
kubectl apply -f examples/03-llm-route.yaml

# Verify routes
kubectl get llmroute -A

# Port-forward Portkey Gateway
kubectl port-forward -n tensor-fusion-sys svc/portkey-gateway 8787:8787

# Test routing
curl http://localhost:8787/routes
```

### 3. Workload Intelligence & ML-based Optimization

**Features:**
- Workload profiling and classification
- Performance prediction
- Load forecasting
- Optimal GPU placement using RL
- Feedback loop for continuous improvement
- Vector database for historical patterns

**Example:**
```bash
# Create workload intelligence profile
kubectl apply -f examples/05-workload-intelligence.yaml

# Verify
kubectl get workloadintelligence -A
kubectl describe workloadintelligence ml-training-profile
```

### 4. Agent-to-Agent (A2A) Communication

**Features:**
- Redis Pub/Sub based message bus
- Request/response patterns
- Multi-agent workflows
- Orchestrator agent for complex tasks
- Autonomous decision-making

**Agents:**
- **Orchestrator Agent** - Coordinates multi-agent workflows
- **Deployment Agent** - Manages model deployments
- **Cost Agent** - Optimizes costs across providers
- **Router Agent** - Optimizes LLM routing
- **Resource Agent** - Manages GPU resources
- **Training Agent** - Handles model training
- **Monitoring Agent** - Tracks system health

**Example:**
```bash
# Test A2A communication
bash test/a2a-communication-test.sh

# View agent logs
kubectl logs -n tensor-fusion-sys -l app.kubernetes.io/component=agents --tail=50
```

### 5. Azure AI Foundry Integration

**Features:**
- Seamless integration with Azure AI Foundry
- Model deployment to Azure
- Hybrid cloud LLM routing
- Azure-managed GPU discovery

**Example:**
```bash
# Configure Azure Foundry credentials
kubectl create secret generic foundry-keys -n tensor-fusion-sys \
  --from-literal=api-key='<YOUR_API_KEY>' \
  --from-literal=endpoint='<YOUR_ENDPOINT>'

# Deploy Azure GPU source
kubectl apply -f examples/06-azure-gpu-source.yaml
```

### 6. Observability & Monitoring

**Features:**
- Prometheus metrics collection
- Grafana dashboards
- GreptimeDB for time-series data
- Vector log aggregation
- Custom metrics for GPU utilization, cost, latency

**Access Grafana:**
```bash
kubectl port-forward -n observability svc/grafana 3000:80
# Open http://localhost:3000
# Login: admin/admin
```

**Access Prometheus:**
```bash
kubectl port-forward -n observability svc/prometheus-server 9090:80
# Open http://localhost:9090
```

---

## 📋 Custom Resource Definitions (CRDs)

TensorFusion provides 14 CRDs for declarative platform management:

1. **GPUPool** - Define GPU resource pools
2. **GPUResourceQuota** - Set GPU quotas per namespace
3. **LLMRoute** - Configure LLM routing strategies
4. **WorkloadIntelligence** - Define ML-based optimization profiles
5. **AzureGPUSource** - Discover Azure GPU resources
6. **LoRAAdapter** - Manage LoRA fine-tuning adapters
7. **VLLMDeployment** - Deploy vLLM inference engines
8. **GPUAllocation** - Track GPU allocations
9. **GPUNode** - Represent GPU-enabled nodes
10. **SchedulingConfig** - Define scheduling strategies
11. **CostReport** - Track and report costs
12. **ModelDeployment** - Manage model deployments
13. **TrainingJob** - Define training workloads
14. **InferenceEndpoint** - Expose inference APIs

View all CRDs:
```bash
kubectl get crds | grep tensor-fusion.ai
```

---

## 🧪 Testing & Validation

### Component Tests

```bash
# Test Redis
kubectl exec -n storage redis-master-0 -- redis-cli PING

# Test GreptimeDB
kubectl port-forward -n greptimedb svc/greptimedb-standalone 4000:4000
curl http://localhost:4000/health

# Test Qdrant
kubectl port-forward -n qdrant svc/qdrant 6333:6333
curl http://localhost:6333/ | jq .version

# Test Portkey
kubectl get pods -n tensor-fusion-sys -l app=portkey-gateway
```

### Workflow Tests

**1. GPU Pool Creation and Management**
```bash
kubectl apply -f examples/01-gpu-pool.yaml
kubectl get gpupool default-pool -o yaml
```

**2. Fractional GPU Allocation**
```bash
kubectl apply -f examples/fractional-gpu-sharing.yaml
kubectl get pods -l app=vgpu-test
kubectl describe pod vgpu-workload-1
```

**3. LLM Routing with Cost Optimization**
```bash
kubectl apply -f examples/03-llm-route.yaml
kubectl get llmroute gpt4-cost-optimized -o yaml
```

**4. Workload Intelligence**
```bash
kubectl apply -f examples/05-workload-intelligence.yaml
kubectl get workloadintelligence ml-training-profile -o yaml
```

**5. A2A Communication**
```bash
bash test/a2a-communication-test.sh
```

---

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LOCATION` | `eastus` | Azure region |
| `RESOURCE_GROUP` | `tensor-fusion-rg` | Resource group name |
| `AKS_CLUSTER` | `tensor-fusion-aks` | AKS cluster name |
| `ACR_NAME` | (auto-generated) | ACR name |
| `ENABLE_GPU_POOL` | `true` | Create GPU nodepool |
| `BUILD_IMAGES` | `true` | Build custom Docker images |
| `SYSTEM_NODE_COUNT` | `2` | Number of system nodes |
| `GPU_NODE_MIN` | `0` | Min GPU nodes (autoscaling) |
| `GPU_NODE_MAX` | `3` | Max GPU nodes (autoscaling) |

### Update Secrets

After deployment, update with your actual credentials:

```bash
# Portkey API Key
kubectl create secret generic portkey-credentials -n portkey \
  --from-literal=api-key='<YOUR_PORTKEY_API_KEY>' \
  --dry-run=client -o yaml | kubectl apply -f -

# Azure Foundry
kubectl create secret generic foundry-keys -n tensor-fusion-sys \
  --from-literal=api-key='<YOUR_FOUNDRY_API_KEY>' \
  --from-literal=endpoint='<YOUR_FOUNDRY_ENDPOINT>' \
  --dry-run=client -o yaml | kubectl apply -f -
```

---

## 📊 Architecture

### High-Level Components

```
┌─────────────────────────────────────────────────────────────┐
│                    TensorFusion Platform                     │
├─────────────────────────────────────────────────────────────┤
│  Controllers                                                 │
│  ├─ GPU Pool Controller                                      │
│  ├─ GPU Resource Quota Controller                            │
│  ├─ LLM Route Controller                                     │
│  ├─ Workload Intelligence Controller                         │
│  └─ Azure GPU Source Controller                              │
├─────────────────────────────────────────────────────────────┤
│  Autonomous Agents (A2A Communication via Redis)             │
│  ├─ Orchestrator Agent                                       │
│  ├─ Deployment Agent                                         │
│  ├─ Cost Agent                                               │
│  ├─ Router Agent                                             │
│  ├─ Resource Agent                                           │
│  ├─ Training Agent                                           │
│  └─ Monitoring Agent                                         │
├─────────────────────────────────────────────────────────────┤
│  Supporting Services                                         │
│  ├─ Portkey Gateway (LLM Routing)                            │
│  ├─ Redis (Message Bus)                                      │
│  ├─ GreptimeDB (Metrics)                                     │
│  ├─ Qdrant (Vector DB)                                       │
│  ├─ Prometheus (Monitoring)                                  │
│  └─ Grafana (Dashboards)                                     │
└─────────────────────────────────────────────────────────────┘
```

### Agent Communication Flow

```
User Request
    ↓
Orchestrator Agent
    ↓
├─→ Deployment Agent ──→ Resource Agent ──→ GPU Allocation
├─→ Cost Agent ────────→ Router Agent ───→ LLM Selection
└─→ Training Agent ────→ Monitoring Agent → Feedback Loop
    ↓
Redis Pub/Sub (A2A Message Bus)
    ↓
Response to User
```

---

## 🛠️ Troubleshooting

### Check Pod Status
```bash
kubectl get pods -A | grep -v Running
```

### View Logs
```bash
# Controller logs
kubectl logs -n tensor-fusion-sys -l app.kubernetes.io/name=tensor-fusion -c controller

# Agent logs
kubectl logs -n tensor-fusion-sys -l app.kubernetes.io/component=agents

# Portkey logs
kubectl logs -n tensor-fusion-sys -l app=portkey-gateway
```

### Check Events
```bash
kubectl get events -n tensor-fusion-sys --sort-by='.lastTimestamp'
```

### Verify CRDs
```bash
kubectl get crds | grep tensor-fusion.ai
kubectl describe crd gpupools.tensor-fusion.ai
```

### Re-deploy
```bash
# Clean up
kubectl delete namespace tensor-fusion-sys storage observability qdrant greptimedb portkey

# Re-run deployment
bash deploy-all.sh
```

---

## 📚 Use Cases

### 1. Multi-Tenant GPU Sharing
- **Problem**: Multiple teams need GPU access with fair resource allocation
- **Solution**: GPU Pools + Resource Quotas + Fractional GPU
- **Test**: Apply `examples/01-gpu-pool.yaml` and `examples/03-gpu-quota.yaml`

### 2. Cost-Optimized LLM Inference
- **Problem**: High costs from using single LLM provider
- **Solution**: Intelligent routing across Azure, OpenAI, self-hosted models
- **Test**: Apply `examples/03-llm-route.yaml` and monitor cost metrics

### 3. Dynamic Workload Optimization
- **Problem**: Inefficient GPU allocation leads to waste
- **Solution**: ML-based workload profiling and optimal placement
- **Test**: Apply `examples/05-workload-intelligence.yaml`

### 4. Autonomous Multi-Agent Orchestration
- **Problem**: Complex workflows require manual coordination
- **Solution**: A2A communication with autonomous agents
- **Test**: Run `bash test/a2a-communication-test.sh`

### 5. Hybrid Cloud LLM Deployment
- **Problem**: Need to leverage both on-prem and Azure resources
- **Solution**: Azure Foundry integration + Portkey routing
- **Test**: Apply `examples/06-azure-gpu-source.yaml`

---

## 🤝 Contributing

This is a comprehensive platform with many moving parts. Key areas:

- **Controllers**: `internal/controller/`
- **Agents**: `pkg/agents/`
- **MCP Tools**: `internal/mcp/tools/`
- **CRDs**: `config/crd/bases/`
- **Deployment**: `deploy-all.sh`
- **Verification**: `scripts/verify-all.sh`

---

## 📄 License

[Add your license here]

---

## 🎉 What's Next?

After successful deployment and verification:

1. **Explore Grafana Dashboards**: `kubectl port-forward -n observability svc/grafana 3000:80`
2. **Deploy Your Workloads**: Use the examples as templates
3. **Monitor A2A Communication**: Check agent logs for autonomous decisions
4. **Optimize Costs**: Review LLM routing metrics in Portkey
5. **Scale GPU Resources**: Adjust GPU pool configurations dynamically

**Platform Status**: FULLY OPERATIONAL ✨

For issues or questions, check the troubleshooting section or review logs from specific components.
