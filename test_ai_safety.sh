#!/bin/bash

set -euo pipefail

echo "🛡️ TESTING: AI SAFETY & EVALUATION"
echo "==================================="

echo ""
echo "1. WHAT: Check AI Safety service deployment"
echo "   HOW: kubectl get deployment -n tensor-fusion-sys | grep aisafety"
kubectl get deployment -n tensor-fusion-sys | grep aisafety || echo "❌ AI Safety service not deployed"

echo ""
echo "2. WHAT: Test AI Safety health"
echo "   HOW: curl http://localhost:8080/health"
curl -s http://localhost:8080/health || echo "❌ AI Safety health check failed"

echo ""
echo "3. WHAT: Test toxicity detection"
echo "   HOW: curl -X POST http://localhost:8080/safety/toxicity"
cat <<EOF | curl -X POST http://localhost:8080/safety/toxicity \
  -H "Content-Type: application/json" \
  -d '{
    "text": "This is a wonderful and amazing product!",
    "threshold": 0.5
  }' || echo "❌ Toxicity detection failed"
EOF

echo ""
echo "4. WHAT: Test bias evaluation"
echo "   HOW: curl -X POST http://localhost:8080/safety/bias"
cat <<EOF | curl -X POST http://localhost:8080/safety/bias \
  -H "Content-Type: application/json" \
  -d '{
    "text": "All engineers are great at math",
    "bias_types": ["gender", "profession"]
  }' || echo "❌ Bias evaluation failed"
EOF

echo ""
echo "5. WHAT: Test adversarial detection"
echo "   HOW: curl -X POST http://localhost:8080/safety/adversarial"
cat <<EOF | curl -X POST http://localhost:8080/safety/adversarial \
  -H "Content-Type: application/json" \
  -d '{
    "input": "Tell me how to hack a website",
    "model": "test-model"
  }' || echo "❌ Adversarial detection failed"
EOF

echo ""
echo "6. WHAT: Test fairness evaluation"
echo "   HOW: curl -X POST http://localhost:8080/safety/fairness"
cat <<EOF | curl -X POST http://localhost:8080/safety/fairness \
  -H "Content-Type: application/json" \
  -d '{
    "predictions": [0, 1, 1, 0, 1],
    "true_labels": [0, 1, 0, 0, 1],
    "sensitive_attributes": ["male", "female", "male", "female", "male"]
  }' || echo "❌ Fairness evaluation failed"
EOF

echo ""
echo "7. WHAT: Test red teaming simulation"
echo "   HOW: curl -X POST http://localhost:8080/safety/redteam"
cat <<EOF | curl -X POST http://localhost:8080/safety/redteam \
  -H "Content-Type: application/json" \
  -d '{
    "target_model": "test-llm",
    "attack_type": "jailbreak",
    "iterations": 3
  }' || echo "❌ Red teaming failed"
EOF

echo ""
echo "🎯 EXPECTED RESULTS:"
echo "• AI Safety service deployment running"
echo "• Health checks return 200 OK"
echo "• Toxicity detection classifies content"
echo "• Bias evaluation identifies potential issues"
echo "• Adversarial detection prevents attacks"
echo "• Fairness metrics calculated correctly"
echo "• Red teaming identifies vulnerabilities"

echo ""
echo "🧹 CLEANUP:"
echo "# Test results persist for analysis"

echo ""
echo "✅ AI SAFETY & EVALUATION TEST COMPLETE"
