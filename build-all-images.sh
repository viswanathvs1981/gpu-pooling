#!/bin/bash
set -e

# Try to get ACR_NAME from environment or .acr-config
if [ -z "${ACR_NAME}" ]; then
  if [ -f ".acr-config" ]; then
    source .acr-config
  else
    # Fallback: try to find ACR in nexusai-acr-rg
    ACR_NAME=$(az acr list -g nexusai-acr-rg --query "[0].name" -o tsv 2>/dev/null || echo "")
    if [ -z "${ACR_NAME}" ]; then
      echo "ERROR: ACR_NAME not found. Run ./setup-acr.sh first"
      exit 1
    fi
  fi
fi

ACR_LOGIN_SERVER="${ACR_NAME}.azurecr.io"

echo "================================================"
echo "Building all NexusAI images to ACR: $ACR_NAME"
echo "================================================"

# Build operator image
echo "[1/9] Building operator image..."
az acr build --registry $ACR_NAME \
  --image nexusai/operator:latest \
  --platform linux/amd64 \
  --file dockerfile/operator.Dockerfile \
  . &
PID1=$!

# Build node-discovery image
echo "[2/9] Building node-discovery image..."
az acr build --registry $ACR_NAME \
  --image nexusai/node-discovery:latest \
  --platform linux/amd64 \
  --file dockerfile/node-discovery.Dockerfile \
  . &
PID2=$!

# Build prompt-optimizer image
echo "[3/9] Building prompt-optimizer image..."
az acr build --registry $ACR_NAME \
  --image nexusai/prompt-optimizer:latest \
  --platform linux/amd64 \
  --file dockerfile/prompt-optimizer.Dockerfile \
  . &
PID3=$!

# Build dataops-agents image
echo "[4/9] Building dataops-agents image..."
az acr build --registry $ACR_NAME \
  --image nexusai/dataops-agents:latest \
  --platform linux/amd64 \
  --file dockerfile/dataops-agents.Dockerfile \
  . &
PID4=$!

# Build aisafety-service image
echo "[5/9] Building aisafety-service image..."
az acr build --registry $ACR_NAME \
  --image nexusai/aisafety-service:latest \
  --platform linux/amd64 \
  --file dockerfile/aisafety-service.Dockerfile \
  . &
PID5=$!

# Build memory-service image
echo "[6/9] Building memory-service image..."
az acr build --registry $ACR_NAME \
  --image nexusai/memory-service:latest \
  --platform linux/amd64 \
  --file dockerfile/memory-service.Dockerfile \
  . &
PID6=$!

# Build model-catalog image
echo "[7/9] Building model-catalog image..."
az acr build --registry $ACR_NAME \
  --image nexusai/model-catalog:latest \
  --platform linux/amd64 \
  --file dockerfile/model-catalog.Dockerfile \
  . &
PID7=$!

# Build discovery-agent image
echo "[8/9] Building discovery-agent image..."
az acr build --registry $ACR_NAME \
  --image nexusai/discovery-agent:latest \
  --platform linux/amd64 \
  --file dockerfile/discovery-agent.Dockerfile \
  . &
PID8=$!

# Build python-agents image
echo "[9/9] Building python-agents image..."
az acr build --registry $ACR_NAME \
  --image nexusai/python-agents:latest \
  --platform linux/amd64 \
  --file dockerfile/python-agents.Dockerfile \
  . &
PID9=$!

echo ""
echo "Waiting for all builds to complete..."
echo "This may take 30-40 minutes..."
echo ""

# Wait for all builds
wait $PID1 && echo "✓ operator build complete" || echo "✗ operator build failed"
wait $PID2 && echo "✓ node-discovery build complete" || echo "✗ node-discovery build failed"
wait $PID3 && echo "✓ prompt-optimizer build complete" || echo "✗ prompt-optimizer build failed"
wait $PID4 && echo "✓ dataops-agents build complete" || echo "✗ dataops-agents build failed"
wait $PID5 && echo "✓ aisafety-service build complete" || echo "✗ aisafety-service build failed"
wait $PID6 && echo "✓ memory-service build complete" || echo "✗ memory-service build failed"
wait $PID7 && echo "✓ model-catalog build complete" || echo "✗ model-catalog build failed"
wait $PID8 && echo "✓ discovery-agent build complete" || echo "✗ discovery-agent build failed"
wait $PID9 && echo "✓ python-agents build complete" || echo "✗ python-agents build failed"

echo ""
echo "================================================"
echo "All images built successfully!"
echo "================================================"
echo ""
echo "To update deployments with new images:"
echo "  kubectl rollout restart deployment -n tensor-fusion-sys"
echo ""
