#!/bin/bash

set -euo pipefail

echo "📚 TESTING: MODEL CATALOG & REGISTRY"
echo "====================================="

echo ""
echo "1. WHAT: Check Model Catalog deployment"
echo "   HOW: kubectl get deployment -n tensor-fusion-sys | grep model-catalog"
kubectl get deployment -n tensor-fusion-sys | grep model-catalog || echo "❌ Model catalog not deployed"

echo ""
echo "2. WHAT: Test Model Catalog health"
echo "   HOW: curl http://localhost:8095/health"
curl -s http://localhost:8095/health || echo "❌ Model catalog health check failed"

echo ""
echo "3. WHAT: Register a new model"
echo "   HOW: curl -X POST http://localhost:8095/models"
cat <<EOF | curl -X POST http://localhost:8095/models \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-llm-model",
    "version": "1.0.0",
    "type": "language_model",
    "framework": "transformers",
    "description": "Test language model for validation",
    "metadata": {
      "architecture": "transformer",
      "parameters": "7B",
      "training_data": "wikipedia"
    }
  }' || echo "❌ Model registration failed"
EOF

echo ""
echo "4. WHAT: Retrieve model information"
echo "   HOW: curl http://localhost:8095/models/test-llm-model"
curl -s http://localhost:8095/models/test-llm-model || echo "❌ Model retrieval failed"

echo ""
echo "5. WHAT: Update model metadata"
echo "   HOW: curl -X PUT http://localhost:8095/models/test-llm-model"
cat <<EOF | curl -X PUT http://localhost:8095/models/test-llm-model \
  -H "Content-Type: application/json" \
  -d '{
    "performance_metrics": {
      "accuracy": 0.85,
      "perplexity": 12.3,
      "latency_ms": 150
    }
  }' || echo "❌ Model update failed"
EOF

echo ""
echo "6. WHAT: Search models by criteria"
echo "   HOW: curl http://localhost:8095/models/search?type=language_model"
curl -s "http://localhost:8095/models/search?type=language_model" || echo "❌ Model search failed"

echo ""
echo "7. WHAT: Create model version"
echo "   HOW: curl -X POST http://localhost:8095/models/test-llm-model/versions"
cat <<EOF | curl -X POST http://localhost:8095/models/test-llm-model/versions \
  -H "Content-Type: application/json" \
  -d '{
    "version": "1.1.0",
    "changes": ["Improved accuracy", "Reduced latency"],
    "artifacts": {
      "model_file": "s3://models/test-llm-v1.1.pth",
      "config_file": "s3://models/config-v1.1.json"
    }
  }' || echo "❌ Version creation failed"
EOF

echo ""
echo "🎯 EXPECTED RESULTS:"
echo "• Model catalog deployment running"
echo "• Health checks return 200 OK"
echo "• Models registered successfully"
echo "• Model metadata retrieved correctly"
echo "• Model updates applied"
echo "• Model search returns filtered results"
echo "• Model versions tracked properly"

echo ""
echo "🧹 CLEANUP:"
echo "# Test models persist - cleanup if needed"

echo ""
echo "✅ MODEL CATALOG & REGISTRY TEST COMPLETE"
