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
echo "║         NexusAI Platform - Build & Push Images to ACR          ║"
echo "╚════════════════════════════════════════════════════════════════╝"

# Load ACR config
if [ -f ".acr-config" ]; then
  source .acr-config
  log_info "Loaded ACR configuration from .acr-config"
else
  log_error "ACR configuration not found. Run ./setup-acr.sh first"
  exit 1
fi

# Verify ACR exists
if ! az acr show -n "${ACR_NAME}" -g "${ACR_RESOURCE_GROUP}" >/dev/null 2>&1; then
  log_error "ACR ${ACR_NAME} not found. Run ./setup-acr.sh first"
  exit 1
fi

log_info "Building images for ACR: ${ACR_NAME}"
log_info "This will take approximately 30-40 minutes..."
echo ""

# Run the parallel build script
if [ -f "./build-all-images.sh" ]; then
  # Update build-all-images.sh to use ACR_NAME from config
  export ACR_NAME
  ./build-all-images.sh
else
  log_error "build-all-images.sh not found"
  exit 1
fi

echo ""
log_success "═══════════════════════════════════════════════════════════════"
log_success "Image Build Complete!"
log_success "═══════════════════════════════════════════════════════════════"
echo ""
log_info "All images have been built and pushed to: ${ACR_LOGIN_SERVER}"
echo ""
log_info "Next steps:"
log_info "  1. Deploy infrastructure: ./deploy-infra.sh"
log_info "  2. Verify deployment: ./verify-all.sh"
echo ""

