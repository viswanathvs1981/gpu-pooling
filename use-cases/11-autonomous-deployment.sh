#!/bin/bash

# USE CASE 11: Autonomous Model Deployment via Orchestrator
# ================================================================
#
# WHAT THIS DEMONSTRATES:
#   - Submit natural language requests to Orchestrator Agent
#   - Automatic workflow execution (validate → allocate GPU → deploy → verify)
#   - End-to-end model deployment automation
#   - Multi-agent collaboration (Orchestrator → Deployment Agent → MCP Server)
#
# WHAT TO EXPECT:
#   1. Orchestrator receives deployment request
#   2. Workflow engine coordinates multi-step deployment
#   3. Deployment Agent creates vLLM deployment
#   4. Model is deployed and endpoint URL is returned
#   5. Health checks confirm successful deployment
#
# ================================================================

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  USE CASE 11: Autonomous Model Deployment                    ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Configuration
NAMESPACE="acme-corp"
MODEL_ID="meta-llama/Llama-3-8b"
ORCHESTRATOR_SVC="tensor-fusion-orchestrator.tensor-fusion-sys.svc.cluster.local:9000"

echo -e "${YELLOW}→ Step 1: Ensuring customer namespace exists${NC}"
kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -
echo ""

echo -e "${YELLOW}→ Step 2: Submitting deployment request to Orchestrator${NC}"
echo "  Model: ${MODEL_ID}"
echo "  Customer: ${NAMESPACE}"
echo "  Request: 'Deploy model for customer acme-corp'"
echo ""

# Port forward orchestrator temporarily
kubectl port-forward -n tensor-fusion-sys svc/tensor-fusion-orchestrator 9000:9000 &
PF_PID=$!
sleep 3

# Submit request via REST API
REQUEST_PAYLOAD=$(cat <<EOF
{
  "request": "Deploy model ${MODEL_ID} for customer ${NAMESPACE}",
  "params": {
    "model_id": "${MODEL_ID}",
    "customer_id": "${NAMESPACE}",
    "config": {
      "vgpu": 0.5,
      "replicas": 1
    }
  }
}
EOF
)

echo "  Sending request to Orchestrator API..."
RESPONSE=$(curl -s -X POST http://localhost:9000/api/v1/requests \
  -H "Content-Type: application/json" \
  -d "${REQUEST_PAYLOAD}")

REQUEST_ID=$(echo ${RESPONSE} | jq -r '.id')
echo -e "${GREEN}✓ Request submitted: ${REQUEST_ID}${NC}"
echo ""

echo -e "${YELLOW}→ Step 3: Monitoring workflow execution${NC}"
echo "  Workflow: DeployModel"
echo "  Steps: Validate Customer → Allocate GPU → Deploy Model → Validate Deployment"
echo ""

# Poll for completion
MAX_ATTEMPTS=30
ATTEMPT=0
STATUS="pending"

while [ "$STATUS" != "completed" ] && [ "$STATUS" != "failed" ] && [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
  sleep 5
  ATTEMPT=$((ATTEMPT + 1))
  
  STATUS_RESPONSE=$(curl -s http://localhost:9000/api/v1/requests/${REQUEST_ID})
  STATUS=$(echo ${STATUS_RESPONSE} | jq -r '.status')
  
  echo "  [Attempt $ATTEMPT/$MAX_ATTEMPTS] Status: ${STATUS}"
  
  if [ "$STATUS" == "completed" ]; then
    echo -e "${GREEN}✓ Workflow completed successfully!${NC}"
    echo ""
    
    echo -e "${YELLOW}→ Step 4: Retrieving deployment details${NC}"
    echo ${STATUS_RESPONSE} | jq '.result'
    echo ""
    
    ENDPOINT=$(echo ${STATUS_RESPONSE} | jq -r '.result.deploy_model.endpoint_url')
    echo -e "${GREEN}✓ Model endpoint: ${ENDPOINT}${NC}"
    break
  elif [ "$STATUS" == "failed" ]; then
    echo -e "${YELLOW}⚠ Workflow failed${NC}"
    ERROR=$(echo ${STATUS_RESPONSE} | jq -r '.error')
    echo "  Error: ${ERROR}"
    break
  fi
done

# Cleanup
kill ${PF_PID} 2>/dev/null || true

echo ""
echo -e "${YELLOW}→ Step 5: Verifying deployment in Kubernetes${NC}"
if kubectl get deployment -n ${NAMESPACE} vllm-${MODEL_ID##*/} >/dev/null 2>&1; then
  echo -e "${GREEN}✓ Deployment found in namespace ${NAMESPACE}${NC}"
  kubectl get deployment -n ${NAMESPACE} vllm-${MODEL_ID##*/}
else
  echo -e "${YELLOW}⚠ Deployment not found (may still be creating)${NC}"
fi
echo ""

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  KEY TAKEAWAYS                                                ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "  ✅ Natural language request → automatic execution"
echo "  ✅ Multi-step workflow coordinated by Orchestrator"
echo "  ✅ GPU allocation automated"
echo "  ✅ Model deployed without manual intervention"
echo "  ✅ Health checks and validation built-in"
echo ""

echo -e "${YELLOW}💡 BUSINESS IMPACT:${NC}"
echo "  • Reduced deployment time from hours to minutes"
echo "  • No manual Kubernetes expertise required"
echo "  • Consistent deployment patterns across teams"
echo "  • Built-in best practices and validation"
echo ""

echo -e "${YELLOW}To clean up:${NC}"
echo "  kubectl delete namespace ${NAMESPACE}"
echo ""

