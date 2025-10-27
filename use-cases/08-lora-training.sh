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
║     USE CASE 8: LoRA Training & Custom Model Creation         ║
║     Problem: Custom AI models are expensive & slow to train    ║
╚════════════════════════════════════════════════════════════════╝
EOF
}

banner
echo ""

info "📖 This demo shows:"
echo "   • What LoRA is and why it matters"
echo "   • Cost comparison: LoRA vs full fine-tuning"
echo "   • Training workflow from data to deployment"
echo "   • Multi-tenant model serving"
echo "   • Real-world use cases"
echo ""
sleep 2

# Step 1: The Problem
info "❌ Step 1: The Traditional Fine-Tuning Problem"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Scenario: You need a legal contract analysis AI"
echo ""
echo "Traditional Full Fine-Tuning:"
echo "  • Train ALL 8 billion parameters"
echo "  • Hardware: 64× A100 GPUs"
echo "  • Time: 48+ hours"
echo "  • Cost: ~\$5,000"
echo "  • Storage: 16GB model file per customer"
echo "  • Memory: Can't serve multiple on same GPU"
echo ""
echo "For 100 Customers:"
echo "  • Training cost: 100 × \$5,000 = \$500,000 💸"
echo "  • Storage: 100 × 16GB = 1.6TB"
echo "  • GPUs needed: 100 (one per customer)"
echo ""
echo "This doesn't scale! 😱"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 2: LoRA Solution
info "✅ Step 2: The LoRA Solution"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "LoRA: Low-Rank Adaptation of Large Language Models"
echo ""
echo "The Key Insight:"
echo "  • Base model (Llama-3-8B) already knows language"
echo "  • Don't retrain everything, just add small 'adapters'"
echo "  • Adapters are 0.1% the size of the full model"
echo "  • Base model + adapter = customized model"
echo ""
echo "How It Works:"
echo "  1. Freeze base model (8B params) ❄️"
echo "  2. Add small adapter layers (8M params) ➕"
echo "  3. Train ONLY the adapters"
echo "  4. At inference: base + adapter"
echo ""
echo "Think of it like:"
echo "  • Base model = Universal translator"
echo "  • Adapter = Accent/dialect module"
echo "  • Same core, different specializations"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 3: Cost Comparison
info "💰 Step 3: Cost & Time Comparison"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Metric                  │ Full Fine-Tuning │ LoRA           │ Savings"
echo "────────────────────────┼──────────────────┼────────────────┼────────"
echo "Parameters Trained      │ 8,000,000,000    │ 8,000,000      │ 1000×"
echo "GPUs Needed             │ 64               │ 1              │ 64×"
echo "Training Time           │ 48 hours         │ 2 hours        │ 24×"
echo "Training Cost           │ \$5,000          │ \$100          │ 50×"
echo "Model File Size         │ 16GB             │ 50MB           │ 320×"
echo "Memory Per Model        │ 16GB             │ 50MB (shared)  │ 320×"
echo "Deploy Time             │ 5 minutes        │ 50ms           │ 6000×"
echo "Models Per GPU          │ 1                │ 100+           │ 100×"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 4: LoRA Architecture
info "🏗️  Step 4: LoRA Architecture Explained"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Original Transformer Layer:"
echo "  Input → [Weight Matrix W (8B×8B)] → Output"
echo ""
echo "LoRA-Modified Layer:"
echo "  Input → [W (frozen)] + [A (8B×32) × B (32×8B)] → Output"
echo "           ↑                ↑          ↑"
echo "        Original        Low-rank    Low-rank"
echo "      (not trained)    (trainable) (trainable)"
echo ""
echo "Key Parameters:"
echo "  • Rank (r): Size of low-rank decomposition"
echo "    - r=8: Fast, lower quality (good for simple tasks)"
echo "    - r=32: Balanced (most common)"
echo "    - r=64: High quality (complex domains)"
echo ""
echo "  • LoRA Alpha (α): Scaling factor"
echo "    - Typically α = 2×r"
echo "    - Controls how much adapter affects output"
echo ""
echo "Example Configuration:"
echo "  • Base: Llama-3-8B"
echo "  • Rank: 32"
echo "  • Alpha: 64"
echo "  • Target modules: q_proj, v_proj (attention)"
echo "  • Trainable params: ~8M (0.1% of base)"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 5: Training Workflow
info "🎓 Step 5: Complete Training Workflow"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Step 1: Prepare Training Data"
echo "  • Collect domain-specific data (legal contracts, medical notes, etc.)"
echo "  • Format: Instruction-response pairs"
echo "  • Size: 1K-50K examples (much smaller than full fine-tuning!)"
echo "  • Quality > Quantity for LoRA"
echo ""
echo "Step 2: Configure Training"
cat << 'PYTHON'
from peft import LoraConfig, get_peft_model

lora_config = LoraConfig(
    r=32,                    # Rank
    lora_alpha=64,           # Scaling
    target_modules=["q_proj", "v_proj"],  # Which layers
    lora_dropout=0.1,
    bias="none",
    task_type="CAUSAL_LM"
)
PYTHON
echo ""
echo "Step 3: Training Loop"
echo "  • Batch size: 8 (fits in 1 GPU)"
echo "  • Learning rate: 3e-4"
echo "  • Epochs: 3"
echo "  • Gradient accumulation: 4 steps"
echo "  • Mixed precision: fp16 (faster)"
echo "  • Time: ~2 hours on A100"
echo ""
echo "Step 4: Save Adapter"
echo "  • Only adapter weights saved (50MB)"
echo "  • Upload to model registry"
echo "  • Ready for deployment"
echo ""
echo "Step 5: Deploy to vLLM"
echo "  • vLLM loads base model once"
echo "  • Adapter loaded per request (50ms)"
echo "  • Instant model switching"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 6: Multi-Tenant Serving
info "🏢 Step 6: Multi-Tenant Serving Example"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Scenario: AI Platform with 100 customers"
echo ""
echo "Without LoRA:"
echo "  • 100 separate models (16GB each)"
echo "  • Need 100 GPUs (1 per customer)"
echo "  • Cost: 100 × \$3/hour = \$300/hour = \$219,000/month"
echo ""
echo "With LoRA + vLLM:"
echo "  • 1 base model (16GB)"
echo "  • 100 adapters (50MB each = 5GB total)"
echo "  • Total: 21GB (fits on 1 GPU!)"
echo "  • Cost: \$3/hour = \$2,190/month"
echo "  • Savings: \$216,810/month (99% reduction!) 🎉"
echo ""
echo "Request Flow:"
echo "  Customer A request → Load adapter A → Generate → Response"
echo "  Customer B request → Load adapter B → Generate → Response"
echo "  (Base model stays loaded, adapters swap)"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 7: Real-world Use Cases
info "🌍 Step 7: Real-World Use Cases"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Use Case 1: Legal Tech Company"
echo "  • Base: Llama-3-70B"
echo "  • Training data: 10K legal contracts"
echo "  • Adapters: Contract analysis, clause extraction, risk scoring"
echo "  • Cost: \$300 training, \$1,500/month serving"
echo "  • vs GPT-4: \$0.03/page × 1M pages = \$30,000/month"
echo "  • ROI: Break even in 2 weeks"
echo "─────────────────────────────────────────────────────────────"
echo ""
echo "Use Case 2: Healthcare AI Platform"
echo "  • Base: Med-PaLM-2"
echo "  • Training data: 50K medical notes (HIPAA-compliant)"
echo "  • Adapters: Diagnosis, treatment plans, coding"
echo "  • Benefit: Data never leaves private cloud"
echo "  • Training: \$500 one-time"
echo "  • Serving: \$2,000/month"
echo "─────────────────────────────────────────────────────────────"
echo ""
echo "Use Case 3: Customer Support SaaS"
echo "  • Base: Mistral-7B"
echo "  • Per-customer adapters: Company knowledge, tone, policies"
echo "  • 500 customers × \$50 training = \$25,000 one-time"
echo "  • vs 500 full models: \$2.5M + ongoing costs"
echo "  • Each customer gets personalized AI"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 8: Training Script Example
info "📝 Step 8: Example Training Script"
echo ""
echo "═══════════════════════════════════════════════════════════════"
cat << 'PYTHON'
# train_lora.py
from transformers import AutoModelForCausalLM, AutoTokenizer, Trainer
from peft import LoraConfig, get_peft_model
from datasets import load_dataset

# 1. Load base model
model = AutoModelForCausalLM.from_pretrained(
    "meta-llama/Llama-3-8b",
    torch_dtype=torch.float16
)
tokenizer = AutoTokenizer.from_pretrained("meta-llama/Llama-3-8b")

# 2. Add LoRA adapters
lora_config = LoraConfig(
    r=32,
    lora_alpha=64,
    target_modules=["q_proj", "v_proj", "k_proj", "o_proj"],
    lora_dropout=0.1
)
model = get_peft_model(model, lora_config)

# 3. Load training data
dataset = load_dataset("your-custom-data")

# 4. Train
trainer = Trainer(
    model=model,
    train_dataset=dataset,
    args=TrainingArguments(
        per_device_train_batch_size=8,
        learning_rate=3e-4,
        num_train_epochs=3,
        fp16=True,
        output_dir="./lora-adapter"
    )
)
trainer.train()

# 5. Save adapter (only 50MB!)
model.save_pretrained("./lora-adapter")
PYTHON
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Summary
echo "═══════════════════════════════════════════════════════════════"
success "🎯 Key Takeaways:"
echo "   ✓ LoRA enables 50× cheaper customization (\$100 vs \$5,000)"
echo "   ✓ 24× faster training (2 hours vs 48 hours)"
echo "   ✓ 320× smaller model files (50MB vs 16GB)"
echo "   ✓ 100+ customers can share 1 GPU"
echo "   ✓ Perfect for multi-tenant AI platforms"
echo "   ✓ Data stays private (train on your own infrastructure)"
echo "   ✓ Rapid experimentation (try 10 variants for \$1,000)"
echo ""
info "💡 Use Case: SaaS platforms, enterprise AI, custom domain models"
echo "═══════════════════════════════════════════════════════════════"
echo ""

info "💡 Next Steps:"
echo "  1. Prepare training data in instruction format"
echo "  2. Train adapter: python train_lora.py"
echo "  3. Deploy to vLLM: See demo 07-vllm-deployment.sh"
echo "  4. Test: curl http://vllm-service:8000/v1/completions -d '{\"lora_id\": \"my-adapter\"}'"
echo ""
info "Demo complete!"
sleep 2

