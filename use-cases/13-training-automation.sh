#!/bin/bash

# USE CASE 13: End-to-End Training & Deployment Automation
# ================================================================
#
# WHAT THIS DEMONSTRATES:
#   - Autonomous LoRA training workflow
#   - Training job monitoring and validation
#   - Quality checks on trained adapters
#   - Automatic deployment after training completes
#   - Full MLOps automation (train → validate → deploy)
#
# WHAT TO EXPECT:
#   1. Submit training + deployment request
#   2. Training Agent creates Kubernetes Job
#   3. Job trains LoRA adapter on dataset
#   4. Adapter quality is validated
#   5. Model + adapter automatically deployed
#   6. Endpoint URL returned for inference
#
# ================================================================

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  USE CASE 13: Training & Deployment Automation               ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Configuration
NAMESPACE="ml-team"
BASE_MODEL="meta-llama/Llama-3-8b"
DATASET_PATH="/data/customer-support.jsonl"
ORCHESTRATOR_SVC="tensor-fusion-orchestrator.tensor-fusion-sys.svc.cluster.local:9000"

echo -e "${YELLOW}→ Step 1: Ensuring ML team namespace exists${NC}"
kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -
echo ""

echo -e "${YELLOW}→ Step 2: Submitting training + deployment request${NC}"
echo "  Base Model: ${BASE_MODEL}"
echo "  Dataset: ${DATASET_PATH}"
echo "  Task: Fine-tune for customer support use case"
echo "  LoRA Config: rank=32, alpha=64"
echo ""

# Port forward orchestrator
kubectl port-forward -n tensor-fusion-sys svc/tensor-fusion-orchestrator 9000:9000 &
PF_PID=$!
sleep 3

# Submit training request
REQUEST_PAYLOAD=$(cat <<EOF
{
  "request": "Train a model on my dataset and deploy it",
  "params": {
    "dataset_path": "${DATASET_PATH}",
    "base_model": "${BASE_MODEL}",
    "customer_id": "${NAMESPACE}",
    "lora_config": {
      "rank": 32,
      "alpha": 64
    },
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

echo -e "${YELLOW}→ Step 3: Monitoring training workflow${NC}"
echo "  Workflow: TrainAndDeploy"
echo "  Steps:"
echo "    1. Validate Dataset"
echo "    2. Start Training Job"
echo "    3. Monitor Training Progress"
echo "    4. Validate Trained Adapter"
echo "    5. Deploy Model with Adapter"
echo ""

# Poll for completion
MAX_ATTEMPTS=60  # Training takes longer
ATTEMPT=0
STATUS="pending"

while [ "$STATUS" != "completed" ] && [ "$STATUS" != "failed" ] && [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
  sleep 10
  ATTEMPT=$((ATTEMPT + 1))
  
  STATUS_RESPONSE=$(curl -s http://localhost:9000/api/v1/requests/${REQUEST_ID})
  STATUS=$(echo ${STATUS_RESPONSE} | jq -r '.status')
  
  echo "  [Attempt $ATTEMPT/$MAX_ATTEMPTS] Status: ${STATUS}"
  
  # Show which step we're on
  if echo ${STATUS_RESPONSE} | jq -e '.result.start_training' >/dev/null 2>&1; then
    echo "    → Training job started"
  fi
  
  if echo ${STATUS_RESPONSE} | jq -e '.result.monitor_training' >/dev/null 2>&1; then
    echo "    → Training in progress..."
  fi
  
  if echo ${STATUS_RESPONSE} | jq -e '.result.validate_adapter' >/dev/null 2>&1; then
    echo "    → Adapter validation complete"
  fi
  
  if [ "$STATUS" == "completed" ]; then
    echo -e "${GREEN}✓ Workflow completed successfully!${NC}"
    echo ""
    
    echo -e "${YELLOW}→ Step 4: Reviewing training results${NC}"
    echo ${STATUS_RESPONSE} | jq '.result'
    echo ""
    
    ADAPTER_ID=$(echo ${STATUS_RESPONSE} | jq -r '.result.monitor_training.adapter_id')
    ENDPOINT=$(echo ${STATUS_RESPONSE} | jq -r '.result.deploy_model.endpoint_url')
    
    echo -e "${GREEN}✓ Trained Adapter ID: ${ADAPTER_ID}${NC}"
    echo -e "${GREEN}✓ Model Endpoint: ${ENDPOINT}${NC}"
    break
  elif [ "$STATUS" == "failed" ]; then
    echo -e "${YELLOW}⚠ Workflow failed${NC}"
    ERROR=$(echo ${STATUS_RESPONSE} | jq -r '.error')
    echo "  Error: ${ERROR}"
    break
  fi
done

# Cleanup port forward
kill ${PF_PID} 2>/dev/null || true

echo ""
echo -e "${YELLOW}→ Step 5: Verifying deployment${NC}"
if kubectl get deployment -n ${NAMESPACE} >/dev/null 2>&1; then
  echo -e "${GREEN}✓ Deployment found in namespace ${NAMESPACE}${NC}"
  kubectl get deployment -n ${NAMESPACE}
else
  echo -e "${YELLOW}⚠ Deployment not found (may still be creating)${NC}"
fi
echo ""

echo -e "${YELLOW}→ Step 6: Checking training job logs (last 20 lines)${NC}"
TRAINING_JOBS=$(kubectl get jobs -n default -l app=lora-training --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null || echo "")
if [ -n "$TRAINING_JOBS" ]; then
  echo "  Job: ${TRAINING_JOBS}"
  echo ""
  kubectl logs -n default job/${TRAINING_JOBS} --tail=20 2>/dev/null || echo "  (Job may still be running or completed)"
else
  echo "  No training jobs found (may have been cleaned up)"
fi
echo ""

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  KEY TAKEAWAYS                                                ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "  ✅ Full MLOps automation: Train → Validate → Deploy"
echo "  ✅ No manual intervention required"
echo "  ✅ LoRA training optimized for efficiency (rank=32)"
echo "  ✅ Automatic quality validation of adapters"
echo "  ✅ Seamless deployment with trained adapter"
echo "  ✅ GPU allocation automated for training job"
echo ""

echo -e "${YELLOW}💡 BUSINESS IMPACT:${NC}"
echo "  • Reduced ML workflow time from days to hours"
echo "  • Eliminated manual handoffs between teams"
echo "  • Consistent quality validation"
echo "  • Faster iteration cycles for model improvement"
echo "  • Built-in best practices (LoRA, GPU optimization)"
echo ""

echo -e "${YELLOW}📊 TRAINING METRICS:${NC}"
echo "  • Estimated Time: 2-3 hours (for demo dataset)"
echo "  • GPU Allocation: 0.5 vGPU"
echo "  • Estimated Cost: ~\$100 per training run"
echo "  • Adapter Size: ~50MB (LoRA is parameter-efficient)"
echo ""

echo -e "${YELLOW}To clean up:${NC}"
echo "  kubectl delete namespace ${NAMESPACE}"
echo "  kubectl delete jobs -n default -l app=lora-training"
echo ""

