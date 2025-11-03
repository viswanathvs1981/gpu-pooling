#!/bin/bash

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║              NexusAI Platform - ACR Setup (One-Time)           ║"
echo "╚════════════════════════════════════════════════════════════════╝"

# Config
LOCATION="${LOCATION:-eastus}"
ACR_RESOURCE_GROUP="${ACR_RESOURCE_GROUP:-nexusai-acr-rg}"
ACR_NAME="${ACR_NAME:-}"

# Check Azure login
az account show >/dev/null 2>&1 || { log_error "Run: az login"; exit 1; }

# Generate ACR name if not provided
if [ -z "${ACR_NAME}" ]; then
  ACR_NAME="nexusaiacr$(openssl rand -hex 3)"
  log_info "Generated ACR name: ${ACR_NAME}"
fi

# Create ACR resource group
log_info "Creating ACR resource group: ${ACR_RESOURCE_GROUP}"
az group create -n "${ACR_RESOURCE_GROUP}" -l "${LOCATION}" -o none
log_success "ACR resource group created"

# Check if ACR exists
if az acr show -n "${ACR_NAME}" -g "${ACR_RESOURCE_GROUP}" >/dev/null 2>&1; then
  log_success "ACR ${ACR_NAME} already exists"
else
  log_info "Creating Azure Container Registry: ${ACR_NAME}"
  az acr create \
    -g "${ACR_RESOURCE_GROUP}" \
    -n "${ACR_NAME}" \
    --sku Standard \
    --location "${LOCATION}" \
    -o none
  log_success "ACR ${ACR_NAME} created"
fi

# Get ACR login server
ACR_LOGIN_SERVER=$(az acr show -n "${ACR_NAME}" -g "${ACR_RESOURCE_GROUP}" --query loginServer -o tsv)

echo ""
log_success "═══════════════════════════════════════════════════════════════"
log_success "ACR Setup Complete!"
log_success "═══════════════════════════════════════════════════════════════"
echo ""
log_info "ACR Name: ${ACR_NAME}"
log_info "ACR Login Server: ${ACR_LOGIN_SERVER}"
log_info "Resource Group: ${ACR_RESOURCE_GROUP}"
echo ""
log_info "Save these values for future deployments:"
echo "  export ACR_NAME=${ACR_NAME}"
echo "  export ACR_RESOURCE_GROUP=${ACR_RESOURCE_GROUP}"
echo ""
log_info "Next steps:"
log_info "  1. Build images: ./build-images.sh"
log_info "  2. Deploy infrastructure: ./deploy-infra.sh"
echo ""

# Save ACR info to file
cat > .acr-config << EOF
# NexusAI ACR Configuration (Auto-generated)
export ACR_NAME=${ACR_NAME}
export ACR_RESOURCE_GROUP=${ACR_RESOURCE_GROUP}
export ACR_LOGIN_SERVER=${ACR_LOGIN_SERVER}
EOF

log_success "ACR configuration saved to .acr-config"
log_info "Source this file in your scripts: source .acr-config"
echo ""

