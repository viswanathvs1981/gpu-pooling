#!/bin/bash

set -euo pipefail

echo "🤖 TESTING: AI AGENT FRAMEWORK (MSAF)"
echo "======================================"

echo ""
echo "1. WHAT: Check Python Agents deployment"
echo "   HOW: kubectl get deployment -n tensor-fusion-sys | grep msaf"
kubectl get deployment -n tensor-fusion-sys | grep msaf || echo "❌ MSAF agents not deployed"

echo ""
echo "2. WHAT: Check Orchestrator Agent deployment"
echo "   HOW: kubectl get deployment -n tensor-fusion-sys | grep orchestrator"
kubectl get deployment -n tensor-fusion-sys | grep orchestrator || echo "❌ Orchestrator not deployed"

echo ""
echo "3. WHAT: Check individual agent pods"
echo "   HOW: kubectl get pods -n tensor-fusion-sys | grep msaf"
kubectl get pods -n tensor-fusion-sys | grep msaf || echo "❌ No MSAF pods running"

echo ""
echo "4. WHAT: Test Orchestrator Agent health"
echo "   HOW: curl http://msaf-orchestrator.tensor-fusion-sys/health"
curl -s http://msaf-orchestrator.tensor-fusion-sys/health || echo "❌ Orchestrator health check failed"

echo ""
echo "5. WHAT: Check agent communication via Redis"
echo "   HOW: kubectl exec -n tensor-fusion-sys deployment/redis -- redis-cli ping"
kubectl exec -n tensor-fusion-sys deployment/redis -- redis-cli ping || echo "❌ Redis not accessible"

echo ""
echo "6. WHAT: Test agent workflow creation"
echo "   HOW: Send workflow request to orchestrator"
cat <<EOF | curl -X POST http://msaf-orchestrator.tensor-fusion-sys/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_type": "training_pipeline",
    "parameters": {
      "model_type": "llm",
      "dataset": "sample_data",
      "epochs": 5
    }
  }' || echo "❌ Workflow creation failed"
EOF

echo ""
echo "7. WHAT: Check workflow execution status"
echo "   HOW: curl http://msaf-orchestrator.tensor-fusion-sys/workflows"
curl -s http://msaf-orchestrator.tensor-fusion-sys/workflows | head -10 || echo "❌ Cannot get workflow status"

echo ""
echo "8. WHAT: Test Cost Agent functionality"
echo "   HOW: curl http://msaf-agents.tensor-fusion-sys/agents/cost/estimate"
curl -X POST http://msaf-agents.tensor-fusion-sys/agents/cost/estimate \
  -H "Content-Type: application/json" \
  -d '{
    "resource_type": "gpu",
    "hours": 24,
    "instance_type": "Standard_NC4as_T4_v3"
  }' || echo "❌ Cost estimation failed"

echo ""
echo "9. WHAT: Test Training Agent capabilities"
echo "   HOW: curl http://msaf-agents.tensor-fusion-sys/agents/training/status"
curl -s http://msaf-agents.tensor-fusion-sys/agents/training/status || echo "❌ Training agent not responding"

echo ""
echo "10. WHAT: Test Deployment Agent"
echo "    HOW: curl http://msaf-agents.tensor-fusion-sys/agents/deployment/deploy"
cat <<EOF | curl -X POST http://msaf-agents.tensor-fusion-sys/agents/deployment/deploy \
  -H "Content-Type: application/json" \
  -d '{
    "model_name": "test-model",
    "version": "v1.0",
    "target": "kubernetes"
  }' || echo "❌ Deployment agent failed"
EOF

echo ""
echo "🎯 EXPECTED RESULTS:"
echo "• MSAF orchestrator and agents deployments running"
echo "• Health checks return 200 OK"
echo "• Redis connectivity working"
echo "• Workflow creation succeeds"
echo "• Agent APIs respond with valid data"
echo "• Cost, training, deployment agents functional"

echo ""
echo "🧹 CLEANUP:"
echo "# No cleanup needed - agents are part of platform"

echo ""
echo "✅ AI AGENT FRAMEWORK TEST COMPLETE"
