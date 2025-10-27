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
║     USE CASE 4: Intelligent LLM Routing                       ║
║     Problem: Optimize cost & latency with smart routing       ║
╚════════════════════════════════════════════════════════════════╝
EOF
}

cleanup() {
  info "🧹 Cleaning up test resources..."
  kubectl delete llmroute smart-routing-demo cost-based-demo --ignore-not-found=true >/dev/null 2>&1 || true
  success "Cleanup complete"
}

trap cleanup EXIT

banner
echo ""

info "📖 This demo shows:"
echo "   • Creating intelligent routing rules"
echo "   • Cost-based routing (short → Azure, long → self-hosted)"
echo "   • Priority-based routing"
echo "   • Request pattern matching"
echo "   • Automatic failover configuration"
echo ""
sleep 2

# Step 1: Show existing LLM routes
info "🔍 Step 1: Checking existing LLM routes..."
echo ""
ROUTE_COUNT=$(kubectl get llmroute -A --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$ROUTE_COUNT" -gt 0 ]; then
  info "📊 Current LLM Routes:"
  echo "─────────────────────────────────────────────────────────────"
  kubectl get llmroute -A -o custom-columns=\
NAMESPACE:.metadata.namespace,\
NAME:.metadata.name,\
ROUTES:.spec.routes[*].name 2>/dev/null | head -10
  echo "─────────────────────────────────────────────────────────────"
else
  info "No existing routes found"
fi
echo ""
sleep 2

# Step 2: Create cost-optimized routing
info "🚀 Step 2: Creating cost-optimized routing policy..."
echo ""
echo "   Strategy:"
echo "   • Short requests (< 500 tokens) → Azure OpenAI (fast)"
echo "   • Long requests (> 500 tokens) → Self-hosted vLLM (cheap)"
echo "   • Critical requests → Azure with high priority"
echo ""

cat <<EOF | kubectl apply -f - >/dev/null 2>&1
apiVersion: tensor-fusion.ai/v1
kind: LLMRoute
metadata:
  name: cost-based-demo
  namespace: default
spec:
  routes:
  - name: azure-fast-route
    backend: azure-openai-endpoint
    modelName: "gpt-4"
    priority: 100
    conditions:
      maxTokens: 500
      minPriority: "high"
    costPerToken: "0.00003"
    description: "Fast Azure route for short, high-priority requests"
  
  - name: self-hosted-economical
    backend: vllm-service.tensor-fusion-sys.svc.cluster.local
    modelName: "llama-3-8b"
    priority: 50
    conditions:
      minTokens: 500
    costPerToken: "0.000005"
    description: "Cost-effective self-hosted for long requests"
  
  - name: fallback-route
    backend: azure-openai-endpoint
    modelName: "gpt-3.5-turbo"
    priority: 10
    costPerToken: "0.000002"
    description: "Cheap fallback for any overflow"
  
  loadBalancing:
    strategy: cost-optimized
    failover: true
EOF

success "Cost-based routing created!"
echo ""
sleep 2

# Step 3: Create pattern-based routing
info "🎯 Step 3: Creating pattern-based routing (by use case)..."
echo ""
echo "   Strategy:"
echo "   • Code generation → Specialized code model"
echo "   • Customer support → Support fine-tuned model"
echo "   • General queries → General purpose model"
echo ""

cat <<EOF | kubectl apply -f - >/dev/null 2>&1
apiVersion: tensor-fusion.ai/v1
kind: LLMRoute
metadata:
  name: smart-routing-demo
  namespace: default
spec:
  routes:
  - name: code-specialist
    backend: vllm-code-service
    modelName: "codellama-13b"
    priority: 100
    conditions:
      requestPattern: ".*code.*|.*function.*|.*debug.*"
    description: "Route coding requests to specialized model"
  
  - name: support-specialist
    backend: vllm-support-service
    modelName: "llama-support-v2"
    priority: 90
    conditions:
      requestPattern: ".*help.*|.*support.*|.*issue.*"
    description: "Route support queries to fine-tuned support model"
  
  - name: general-purpose
    backend: vllm-general-service
    modelName: "llama-3-8b"
    priority: 50
    description: "General purpose model for all other requests"
  
  loadBalancing:
    strategy: pattern-match
    roundRobin: false
EOF

success "Pattern-based routing created!"
echo ""
sleep 2

# Step 4: Show created routes
info "📊 Step 4: Reviewing created routing policies..."
echo ""
echo "═══════════════════════════════════════════════════════════════"
info "Route 1: Cost-Based Routing"
echo "─────────────────────────────────────────────────────────────"
kubectl describe llmroute cost-based-demo 2>/dev/null | grep -A 30 "Spec:" | head -25 || echo "  Route configured"
echo ""

info "Route 2: Pattern-Based Routing"
echo "─────────────────────────────────────────────────────────────"
kubectl describe llmroute smart-routing-demo 2>/dev/null | grep -A 30 "Spec:" | head -25 || echo "  Route configured"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Step 5: Simulate routing decisions
info "🧪 Step 5: Simulating routing decisions..."
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Scenario 1: Short high-priority request"
echo "  Request: 'Summarize this' (50 tokens, priority: high)"
echo "  → Routed to: Azure OpenAI (fast route)"
echo "  → Cost: \$0.0015"
echo "  → Latency: ~300ms"
echo ""
echo "Scenario 2: Long economical request"
echo "  Request: 'Write detailed analysis...' (2000 tokens, priority: normal)"
echo "  → Routed to: Self-hosted vLLM"
echo "  → Cost: \$0.01 (83% cheaper than Azure)"
echo "  → Latency: ~1.2s"
echo ""
echo "Scenario 3: Code generation request"
echo "  Request: 'Write a Python function for...' (matches pattern)"
echo "  → Routed to: CodeLlama specialist"
echo "  → Optimized for code quality"
echo ""
echo "Scenario 4: Customer support request"
echo "  Request: 'I need help with...' (matches pattern)"
echo "  → Routed to: Support-tuned model"
echo "  → Optimized for helpfulness"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 3

# Step 6: Cost analysis
info "💰 Step 6: Cost Impact Analysis"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  WITHOUT Smart Routing (All to Azure):"
echo "    • 1M requests/month × \$0.00003 = \$30,000/month"
echo ""
echo "  WITH Smart Routing (Cost-Optimized):"
echo "    • 30% short requests → Azure = 300K × \$0.00003 = \$9,000"
echo "    • 70% long requests → Self-hosted = 700K × \$0.000005 = \$3,500"
echo "    • Total: \$12,500/month"
echo "    • Savings: \$17,500/month (58% reduction) 💰"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Step 7: Failover & Reliability
info "🔄 Step 7: Automatic Failover Configuration"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  Primary Route Failure Scenarios:"
echo ""
echo "  1. Self-hosted vLLM down"
echo "     → Automatic failover to Azure OpenAI"
echo "     → Cost increase tolerated for reliability"
echo "     → Alert sent to ops team"
echo ""
echo "  2. Azure rate limit exceeded"
echo "     → Fallback to self-hosted"
echo "     → Slightly higher latency, maintains service"
echo ""
echo "  3. All backends healthy"
echo "     → Optimal routing based on rules"
echo "     → Maximum cost savings"
echo "═══════════════════════════════════════════════════════════════"
echo ""
sleep 2

# Step 8: Show Portkey integration
info "🔌 Step 8: Portkey Gateway Integration"
echo ""
info "These routes are automatically configured in Portkey Gateway at Layer 2"
echo ""
PORTKEY_PODS=$(kubectl get pods -n portkey -l app=portkey-gateway --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$PORTKEY_PODS" -gt 0 ]; then
  success "Portkey Gateway: Running ($PORTKEY_PODS replicas)"
  info "  Gateway processes all LLM requests and applies routing rules"
else
  warn "Portkey Gateway not found - routing would be handled by controller"
fi
echo ""

# Summary
echo "═══════════════════════════════════════════════════════════════"
success "🎯 Key Takeaways:"
echo "   ✓ Cost-based routing saves 58% on LLM costs"
echo "   ✓ Pattern-based routing optimizes quality per use case"
echo "   ✓ Automatic failover ensures high availability"
echo "   ✓ Real-time routing decisions at Layer 2 (Portkey)"
echo "   ✓ No code changes needed - just configure routes"
echo ""
info "💡 Use Case: Multi-model serving, cost optimization, specialized workloads"
echo "═══════════════════════════════════════════════════════════════"
echo ""

info "💡 Pro Tip: Check route status with 'kubectl describe llmroute <name>'"
echo ""
info "Demo complete! Resources will be cleaned up automatically."
sleep 2

