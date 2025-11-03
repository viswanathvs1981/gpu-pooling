#!/bin/bash

set -euo pipefail

echo "🔍 TESTING: LLM DISCOVERY & MANAGEMENT"
echo "======================================="

echo ""
echo "1. WHAT: Check Discovery Agent deployment"
echo "   HOW: kubectl get deployment -n tensor-fusion-sys | grep discovery"
kubectl get deployment -n tensor-fusion-sys | grep discovery || echo "❌ Discovery agent not deployed"

echo ""
echo "2. WHAT: Test Discovery Agent health"
echo "   HOW: curl http://localhost:8081/health"
curl -s http://localhost:8081/health || echo "❌ Discovery agent health check failed"

echo ""
echo "3. WHAT: Register an LLM endpoint"
echo "   HOW: curl -X POST http://localhost:8081/endpoints"
cat <<EOF | curl -X POST http://localhost:8081/endpoints \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-llm-endpoint",
    "url": "http://test-llm-service:8000",
    "model_type": "gpt-3.5-turbo",
    "capabilities": ["chat", "completion"],
    "health_check_url": "http://test-llm-service:8000/health"
  }' || echo "❌ Endpoint registration failed"
EOF

echo ""
echo "4. WHAT: Check endpoint health"
echo "   HOW: curl http://localhost:8081/endpoints/test-llm-endpoint/health"
curl -s http://localhost:8081/endpoints/test-llm-endpoint/health || echo "❌ Health check failed"

echo ""
echo "5. WHAT: Get endpoint capabilities"
echo "   HOW: curl http://localhost:8081/endpoints/test-llm-endpoint/capabilities"
curl -s http://localhost:8081/endpoints/test-llm-endpoint/capabilities || echo "❌ Capabilities retrieval failed"

echo ""
echo "6. WHAT: Test load balancing"
echo "   HOW: curl http://localhost:8081/endpoints/balance"
cat <<EOF | curl -X POST http://localhost:8081/endpoints/balance \
  -H "Content-Type: application/json" \
  -d '{
    "request_type": "completion",
    "priority": "low",
    "max_latency_ms": 1000
  }' || echo "❌ Load balancing failed"
EOF

echo ""
echo "7. WHAT: Test fallback configuration"
echo "   HOW: curl -X POST http://localhost:8081/endpoints/fallback"
cat <<EOF | curl -X POST http://localhost:8081/endpoints/fallback \
  -H "Content-Type: application/json" \
  -d '{
    "primary_endpoint": "test-llm-endpoint",
    "fallback_endpoints": ["backup-llm-1", "backup-llm-2"],
    "fallback_criteria": ["timeout", "error_rate"]
  }' || echo "❌ Fallback configuration failed"
EOF

echo ""
echo "🎯 EXPECTED RESULTS:"
echo "• Discovery agent deployment running"
echo "• Health checks return 200 OK"
echo "• LLM endpoints registered successfully"
echo "• Endpoint health monitoring works"
echo "• Capabilities correctly identified"
echo "• Load balancing selects optimal endpoint"
echo "• Fallback mechanisms configured"

echo ""
echo "🧹 CLEANUP:"
echo "# Test endpoints persist for analysis"

echo ""
echo "✅ LLM DISCOVERY & MANAGEMENT TEST COMPLETE"
