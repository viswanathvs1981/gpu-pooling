#!/bin/bash

set -euo pipefail

echo "💬 TESTING: PROMPT OPTIMIZATION"
echo "================================"

echo ""
echo "1. WHAT: Check Prompt Optimizer deployment"
echo "   HOW: kubectl get deployment -n tensor-fusion-sys | grep prompt"
kubectl get deployment -n tensor-fusion-sys | grep prompt || echo "❌ Prompt optimizer not deployed"

echo ""
echo "2. WHAT: Test Prompt Optimizer health"
echo "   HOW: curl http://localhost:8082/health"
curl -s http://localhost:8082/health || echo "❌ Prompt optimizer health check failed"

echo ""
echo "3. WHAT: Test chain-of-thought optimization"
echo "   HOW: curl -X POST http://localhost:8082/optimize/cot"
cat <<EOF | curl -X POST http://localhost:8082/optimize/cot \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Solve 2x + 3 = 7",
    "domain": "mathematics",
    "complexity": "basic"
  }' || echo "❌ CoT optimization failed"
EOF

echo ""
echo "4. WHAT: Test few-shot learning optimization"
echo "   HOW: curl -X POST http://localhost:8082/optimize/fewshot"
cat <<EOF | curl -X POST http://localhost:8082/optimize/fewshot \
  -H "Content-Type: application/json" \
  -d '{
    "task": "sentiment_analysis",
    "examples": [
      {"text": "I love this product", "label": "positive"},
      {"text": "This is terrible", "label": "negative"}
    ],
    "target_text": "This is amazing!"
  }' || echo "❌ Few-shot optimization failed"
EOF

echo ""
echo "5. WHAT: Test context enrichment"
echo "   HOW: curl -X POST http://localhost:8082/optimize/context"
cat <<EOF | curl -X POST http://localhost:8082/optimize/context \
  -H "Content-Type: application/json" \
  -d '{
    "base_prompt": "Explain quantum computing",
    "domain": "physics",
    "audience_level": "beginner",
    "additional_context": ["quantum_mechanics", "computer_science"]
  }' || echo "❌ Context enrichment failed"
EOF

echo ""
echo "6. WHAT: Test prompt clarity enhancement"
echo "   HOW: curl -X POST http://localhost:8082/optimize/clarity"
cat <<EOF | curl -X POST http://localhost:8082/optimize/clarity \
  -H "Content-Type: application/json" \
  -d '{
    "ambiguous_prompt": "Make it better and stuff",
    "context": "software_development",
    "target_clarity_score": 0.8
  }' || echo "❌ Clarity enhancement failed"
EOF

echo ""
echo "7. WHAT: Test bias mitigation"
echo "   HOW: curl -X POST http://localhost:8082/optimize/bias"
cat <<EOF | curl -X POST http://localhost:8082/optimize/bias \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Write about successful CEOs",
    "bias_types": ["gender", "ethnicity"],
    "neutrality_threshold": 0.9
  }' || echo "❌ Bias mitigation failed"
EOF

echo ""
echo "🎯 EXPECTED RESULTS:"
echo "• Prompt optimizer deployment running"
echo "• Health checks return 200 OK"
echo "• CoT optimization adds reasoning steps"
echo "• Few-shot optimization selects relevant examples"
echo "• Context enrichment adds domain knowledge"
echo "• Clarity enhancement improves specificity"
echo "• Bias mitigation creates neutral prompts"

echo ""
echo "🧹 CLEANUP:"
echo "# Test prompts persist for analysis"

echo ""
echo "✅ PROMPT OPTIMIZATION TEST COMPLETE"
