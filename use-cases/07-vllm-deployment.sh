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
║     USE CASE 7: vLLM Deployment & Inference                    ║
║     Problem: Need high-performance LLM serving                 ║
╚════════════════════════════════════════════════════════════════╝
EOF
}

banner
echo ""

info "📖 This demo shows:"
echo "   • What vLLM is and why it's needed"
echo "   • Key features: PagedAttention, continuous batching"
echo "   • Performance comparison vs alternatives"
echo "   • Integration with Tensor Fusion GPU sharing"
echo "   • LoRA adapter support"
echo ""
sleep 2

# Step 1: Explain vLLM
info "🎯 Step 1: Understanding vLLM"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "What is vLLM?"
echo "  vLLM (Very Large Language Model serving) is a high-performance"
echo "  inference engine optimized for serving LLMs efficiently."
echo ""
echo "Why Not Just Use PyTorch/HuggingFace?"
echo "  ❌ Naive PyTorch:"
echo "     • Processes 1 request at a time"
echo "     • Wastes 70% of GPU memory"
echo "     • 5-10 requests/second"
echo "     • Poor memory management"
echo ""
echo "  ✅ vLLM Optimizations:"
echo "     • Processes 20+ requests simultaneously"
echo "     • Uses 90% of GPU memory efficiently"
echo "     • 40-50 requests/second"
echo "     • PagedAttention for memory"
echo "     • Continuous batching"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 2: PagedAttention explained
info "🧠 Step 2: PagedAttention Technology"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "The Memory Problem:"
echo "  Traditional systems allocate memory for worst case"
echo "  • Max sequence length: 4096 tokens"
echo "  • Each token needs ~500KB (KV cache)"
echo "  • Allocation: 4096 × 500KB = 2GB per request"
echo "  • If only 100 tokens used → 97% wasted!"
echo ""
echo "vLLM's Solution: PagedAttention"
echo "  • Allocate memory in small 'pages' (like OS virtual memory)"
echo "  • Need 100 tokens? Get 100 tokens worth - no waste"
echo "  • Dynamic allocation as conversation grows"
echo "  • Pages can be shared across requests (prefix caching)"
echo ""
echo "  Result: 3-5× more requests fit in same GPU! 🚀"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 3: Continuous batching
info "⚡ Step 3: Continuous Batching"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Traditional Batching:"
echo "  • Batch 8 requests together"
echo "  • All requests finish at same time"
echo "  • Short requests wait for long ones"
echo "  • GPU idle between batches"
echo ""
echo "vLLM Continuous Batching:"
echo "  • Requests join/leave batch dynamically"
echo "  • Short request done? Add new one immediately"
echo "  • No waiting, no idle time"
echo "  • GPU always working at 100%"
echo ""
echo "  Example:"
echo "    Time 0: Batch [A, B, C, D, E, F, G, H]"
echo "    Time 1: A done → replace with I: [I, B, C, D, E, F, G, H]"
echo "    Time 2: C done → replace with J: [I, B, J, D, E, F, G, H]"
echo "    ...continuous processing..."
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 4: Performance comparison
info "📊 Step 4: Performance Comparison"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Benchmark: Llama-3-8B on NVIDIA A100 (80GB)"
echo ""
echo "Metric              │ PyTorch │ HF Transformers │ vLLM   │ Improvement"
echo "────────────────────┼─────────┼─────────────────┼────────┼────────────"
echo "Throughput (req/s)  │    8    │       12        │   48   │   6×"
echo "Latency (p99)       │  2.1s   │      1.5s       │ 0.35s  │   6×"
echo "GPU Memory Used     │  45GB   │      52GB       │  72GB  │  1.6×"
echo "Max Concurrent      │    4    │        8        │   32   │   8×"
echo "Cost per 1M tokens  │  \$0.50 │      \$0.35     │ \$0.08 │  6× cheaper"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 5: Check if vLLM is deployed
info "🔍 Step 5: Checking vLLM Deployment Status"
echo ""

if [ -f "../scripts/deploy-vllm.sh" ]; then
  success "vLLM deployment script found: scripts/deploy-vllm.sh"
  info "  To deploy vLLM, run: bash scripts/deploy-vllm.sh"
else
  info "vLLM deployment script location: scripts/deploy-vllm.sh"
fi
echo ""

VLLM_PODS=$(kubectl get pods -n tensor-fusion-sys -l app=vllm --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$VLLM_PODS" -gt 0 ]; then
  success "vLLM pods found in cluster: $VLLM_PODS"
  echo ""
  kubectl get pods -n tensor-fusion-sys -l app=vllm -o wide 2>/dev/null
else
  warn "vLLM not currently deployed"
  info "  Deploy with: bash scripts/deploy-vllm.sh"
fi
echo ""
sleep 2

# Step 6: LoRA support
info "🎨 Step 6: LoRA Adapter Support"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "vLLM + LoRA = Multi-Tenant Serving"
echo ""
echo "Traditional Approach (1 model per customer):"
echo "  • Customer A: 16GB model file"
echo "  • Customer B: 16GB model file"
echo "  • Customer C: 16GB model file"
echo "  • Total: 48GB, 3 GPUs needed"
echo ""
echo "vLLM + LoRA Approach:"
echo "  • Base model: 16GB (loaded once)"
echo "  • Customer A adapter: 50MB"
echo "  • Customer B adapter: 50MB"
echo "  • Customer C adapter: 50MB"
echo "  • Total: 16.15GB, 1 GPU serves all!"
echo ""
echo "LoRA Switching:"
echo "  • Load base model once (5 seconds)"
echo "  • Switch adapters per request (50ms)"
echo "  • 100+ customers on same GPU"
echo "  • Each gets customized model"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 7: Integration with Tensor Fusion
info "🎮 Step 7: Integration with Tensor Fusion"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "vLLM + Tensor Fusion = Maximum Efficiency"
echo ""
echo "Layer 3 (vLLM): Efficient inference engine"
echo "     ↓"
echo "Layer 4 (Tensor Fusion): GPU virtualization"
echo "     ↓"
echo "Layer 5/6 (CUDA/Hardware): Physical GPUs"
echo ""
echo "Example Deployment:"
echo "  • 1 Physical A100 (80GB)"
echo "  • Tensor Fusion creates 3 vGPUs:"
echo "     - vGPU-1 (30GB): vLLM serving Llama-3-8B"
echo "     - vGPU-2 (30GB): vLLM serving CodeLlama-13B"
echo "     - vGPU-3 (20GB): vLLM serving Mistral-7B"
echo "  • All isolated, all efficient"
echo "  • Cost: 1 GPU serves 3 workloads"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 8: Deployment configuration
info "⚙️  Step 8: Typical vLLM Deployment Configuration"
echo ""
echo "═══════════════════════════════════════════════════════════════"
cat << 'YAML'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vllm-llama3-8b
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: vllm
        image: vllm/vllm-openai:latest
        env:
        - name: MODEL_NAME
          value: "meta-llama/Llama-3-8b"
        - name: TENSOR_PARALLEL_SIZE
          value: "1"
        - name: MAX_MODEL_LEN
          value: "4096"
        - name: GPU_MEMORY_UTILIZATION
          value: "0.9"  # Use 90% of GPU
        resources:
          limits:
            nvidia.com/gpu: 1
        command:
        - python3
        - -m
        - vllm.entrypoints.openai.api_server
        - --model
        - $(MODEL_NAME)
        - --tensor-parallel-size
        - $(TENSOR_PARALLEL_SIZE)
        - --max-model-len
        - $(MAX_MODEL_LEN)
        - --gpu-memory-utilization
        - $(GPU_MEMORY_UTILIZATION)
        - --enable-lora  # LoRA support!
YAML
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Step 9: API usage example
info "📡 Step 9: Using vLLM API (OpenAI-Compatible)"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "vLLM exposes OpenAI-compatible API:"
echo ""
cat << 'CURL'
curl http://vllm-service:8000/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "meta-llama/Llama-3-8b",
    "prompt": "Explain quantum computing",
    "max_tokens": 100,
    "temperature": 0.7
  }'

# With LoRA adapter:
curl http://vllm-service:8000/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "meta-llama/Llama-3-8b",
    "prompt": "Analyze this legal contract",
    "lora_id": "legal-contract-v2",  # Custom adapter!
    "max_tokens": 500
  }'
CURL
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Summary
echo "═══════════════════════════════════════════════════════════════"
success "🎯 Key Takeaways:"
echo "   ✓ vLLM provides 6× better throughput than naive PyTorch"
echo "   ✓ PagedAttention optimizes memory (3-5× more requests per GPU)"
echo "   ✓ Continuous batching keeps GPU at 100% utilization"
echo "   ✓ LoRA support enables multi-tenant serving (100+ customers/GPU)"
echo "   ✓ OpenAI-compatible API for easy migration"
echo "   ✓ Integrates with Tensor Fusion for maximum efficiency"
echo "   ✓ 6× cost reduction vs traditional serving"
echo ""
info "💡 Use Case: High-throughput LLM serving, multi-tenant platforms, cost optimization"
echo "═══════════════════════════════════════════════════════════════"
echo ""

info "💡 Next Steps:"
echo "  1. Deploy vLLM: bash scripts/deploy-vllm.sh"
echo "  2. Test inference: curl http://vllm-service:8000/v1/models"
echo "  3. Add LoRA adapters: See demo 08-lora-training.sh"
echo ""
info "Demo complete!"
sleep 2

