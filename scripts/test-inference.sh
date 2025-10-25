#!/bin/bash

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║   TensorFusion Inference Testing                               ║"
echo "╚════════════════════════════════════════════════════════════════╝"

# Configuration
NAMESPACE="${NAMESPACE:-vllm}"
MODEL_NAME="${MODEL_NAME:-llama-3-8b}"
API_ENDPOINT="${API_ENDPOINT:-http://localhost:8000}"

log_info "Setting up port forward..."
POD_NAME=$(kubectl get pods -n $NAMESPACE -l app=vllm,model=$MODEL_NAME -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -z "$POD_NAME" ]; then
    log_error "No vLLM pods found in namespace $NAMESPACE"
    exit 1
fi

kubectl port-forward -n $NAMESPACE pod/$POD_NAME 8000:8000 &
PF_PID=$!

# Wait for port forward
sleep 5

log_info "Test 1: List available models..."
MODELS=$(curl -s http://localhost:8000/v1/models)
echo $MODELS | jq .
if [ $? -eq 0 ]; then
    log_success "Models listed successfully"
else
    log_error "Failed to list models"
fi

log_info "Test 2: Simple completion request..."
RESPONSE=$(curl -s http://localhost:8000/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "'$MODEL_NAME'",
    "prompt": "What is 2+2?",
    "max_tokens": 50,
    "temperature": 0.7
  }')

echo $RESPONSE | jq .
if [ $? -eq 0 ]; then
    log_success "Completion request successful"
else
    log_error "Failed completion request"
fi

log_info "Test 3: Chat completion request..."
CHAT_RESPONSE=$(curl -s http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "'$MODEL_NAME'",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "Explain GPU virtualization in one sentence."}
    ],
    "max_tokens": 100,
    "temperature": 0.7
  }')

echo $CHAT_RESPONSE | jq .
if [ $? -eq 0 ]; then
    log_success "Chat completion successful"
    
    # Extract and display the response
    MESSAGE=$(echo $CHAT_RESPONSE | jq -r '.choices[0].message.content')
    echo ""
    log_info "AI Response:"
    echo "$MESSAGE"
else
    log_error "Failed chat completion"
fi

# Cleanup
kill $PF_PID 2>/dev/null || true

log_success "Testing complete!"



