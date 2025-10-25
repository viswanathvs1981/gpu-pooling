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
echo "║   Azure AI Foundry Integration Setup                          ║"
echo "╚════════════════════════════════════════════════════════════════╝"

# Configuration
RESOURCE_GROUP="${RESOURCE_GROUP:-tensor-fusion-rg}"
LOCATION="${LOCATION:-eastus}"
AI_PROJECT_NAME="${AI_PROJECT_NAME:-tensor-fusion-ai}"
NAMESPACE="${NAMESPACE:-tensor-fusion-sys}"

log_info "Creating Azure AI Foundry project..."
# Check if AI services are available
if ! az cognitiveservices account list &>/dev/null; then
    log_warning "Azure Cognitive Services CLI not available"
    log_info "Attempting to use Azure OpenAI..."
fi

# Create Azure OpenAI service
OPENAI_NAME="tf-openai-${RANDOM}"
log_info "Creating Azure OpenAI service: $OPENAI_NAME..."

az cognitiveservices account create \
    --name "$OPENAI_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --kind OpenAI \
    --sku S0 \
    --location "$LOCATION" \
    --yes 2>/dev/null || log_warning "Azure OpenAI creation may require manual approval"

log_success "Azure OpenAI service created (or pending approval)"

# Get endpoint and key
log_info "Retrieving credentials..."
ENDPOINT=$(az cognitiveservices account show \
    --name "$OPENAI_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --query "properties.endpoint" -o tsv 2>/dev/null || echo "")

KEY=$(az cognitiveservices account keys list \
    --name "$OPENAI_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --query "key1" -o tsv 2>/dev/null || echo "")

if [ -z "$ENDPOINT" ] || [ -z "$KEY" ]; then
    log_warning "Could not retrieve credentials automatically"
    log_info "Please manually retrieve from Azure Portal"
    echo "Resource Group: $RESOURCE_GROUP"
    echo "Service Name: $OPENAI_NAME"
    exit 0
fi

# Create Kubernetes secret
log_info "Creating Kubernetes secret..."
kubectl create secret generic azure-openai-credentials \
    --from-literal=endpoint="$ENDPOINT" \
    --from-literal=api-key="$KEY" \
    --namespace "$NAMESPACE" \
    --dry-run=client -o yaml | kubectl apply -f -

log_success "Credentials stored in Kubernetes!"

# Deploy GPT-4 model
log_info "Deploying GPT-4 model..."
DEPLOYMENT_NAME="gpt-4"

az cognitiveservices account deployment create \
    --name "$OPENAI_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --deployment-name "$DEPLOYMENT_NAME" \
    --model-name "gpt-4" \
    --model-version "0613" \
    --model-format OpenAI \
    --sku-capacity 10 \
    --sku-name "Standard" 2>/dev/null || log_warning "Model deployment may require approval"

log_success "GPT-4 deployment initiated"

# Create AzureGPUSource CR
log_info "Creating AzureGPUSource custom resource..."
cat <<EOF | kubectl apply -f -
apiVersion: tensor-fusion.ai/v1
kind: AzureGPUSource
metadata:
  name: azure-openai-foundry
  namespace: $NAMESPACE
spec:
  sourceType: "azure-openai"
  enabled: true
  endpoint: "$ENDPOINT"
  subscriptionID: "$(az account show --query id -o tsv)"
  resourceGroup: "$RESOURCE_GROUP"
  serviceName: "$OPENAI_NAME"
  syncInterval: "5m"
  priority: 90
  credentials:
    secretRef:
      name: azure-openai-credentials
      namespace: $NAMESPACE
EOF

log_success "AzureGPUSource created!"

# Summary
echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║   Azure AI Foundry Setup Complete!                            ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""
echo "Service Name: $OPENAI_NAME"
echo "Endpoint: $ENDPOINT"
echo "Deployment: $DEPLOYMENT_NAME"
echo ""
log_info "Test with:"
echo "  curl -X POST \"$ENDPOINT/openai/deployments/$DEPLOYMENT_NAME/chat/completions?api-version=2024-02-15-preview\" \\"
echo "    -H \"api-key: $KEY\" \\"
echo "    -H \"Content-Type: application/json\" \\"
echo "    -d '{\"messages\": [{\"role\": \"user\", \"content\": \"Hello\"}]}'"



