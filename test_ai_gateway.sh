#!/bin/bash

set -euo pipefail

echo "🌐 TESTING: AI GATEWAY & TOKEN MANAGEMENT"
echo "=========================================="

echo ""
echo "1. WHAT: Check Portkey Gateway deployment"
echo "   HOW: kubectl get deployment -n tensor-fusion-sys | grep portkey"
kubectl get deployment -n tensor-fusion-sys | grep portkey || echo "❌ Portkey gateway not deployed"

echo ""
echo "2. WHAT: Test Portkey Gateway health"
echo "   HOW: curl http://portkey-gateway.tensor-fusion-sys/health"
curl -s http://portkey-gateway.tensor-fusion-sys/health || echo "❌ Portkey health check failed"

echo ""
echo "3. WHAT: Test token usage tracking"
echo "   HOW: curl http://portkey-gateway.tensor-fusion-sys/tokens/usage"
cat <<EOF | curl -X POST http://portkey-gateway.tensor-fusion-sys/tokens/usage \
  -H "Content-Type: application/json" \
  -d '{
    "api_key": "test-key-123",
    "model": "gpt-4",
    "tokens_used": 150,
    "request_type": "completion"
  }' || echo "❌ Token tracking failed"
EOF

echo ""
echo "4. WHAT: Check token budget"
echo "   HOW: curl http://portkey-gateway.tensor-fusion-sys/tokens/budget/test-key-123"
curl -s http://portkey-gateway.tensor-fusion-sys/tokens/budget/test-key-123 || echo "❌ Budget check failed"

echo ""
echo "5. WHAT: Test request routing"
echo "   HOW: curl -X POST http://portkey-gateway.tensor-fusion-sys/v1/chat/completions"
cat <<EOF | curl -X POST http://portkey-gateway.tensor-fusion-sys/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-key-123" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 50
  }' || echo "❌ Request routing failed"
EOF

echo ""
echo "6. WHAT: Test load balancing across providers"
echo "   HOW: curl http://portkey-gateway.tensor-fusion-sys/providers/status"
curl -s http://portkey-gateway.tensor-fusion-sys/providers/status || echo "❌ Provider status check failed"

echo ""
echo "7. WHAT: Test intelligent fallback"
echo "   HOW: curl -X POST http://portkey-gateway.tensor-fusion-sys/fallback/test"
cat <<EOF | curl -X POST http://portkey-gateway.tensor-fusion-sys/fallback/test \
  -H "Content-Type: application/json" \
  -d '{
    "primary_provider": "openai",
    "fallback_providers": ["anthropic", "google"],
    "request_type": "embedding"
  }' || echo "❌ Fallback test failed"
EOF

echo ""
echo "🎯 EXPECTED RESULTS:"
echo "• Portkey gateway deployment running"
echo "• Health checks return 200 OK"
echo "• Token usage tracked accurately"
echo "• Budget limits enforced"
echo "• Requests routed to appropriate providers"
echo "• Provider status monitored"
echo "• Fallback mechanisms work correctly"

echo ""
echo "🧹 CLEANUP:"
echo "# Token data persists for analysis"

echo ""
echo "✅ AI GATEWAY & TOKEN MANAGEMENT TEST COMPLETE"
