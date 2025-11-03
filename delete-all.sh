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
echo "║           NexusAI Platform - Delete Infrastructure             ║"
echo "╚════════════════════════════════════════════════════════════════╝"

# Config
RESOURCE_GROUP="${RESOURCE_GROUP:-tensor-fusion-rg}"
ACR_RESOURCE_GROUP="${ACR_RESOURCE_GROUP:-nexusai-acr-rg}"
DELETE_ACR="${DELETE_ACR:-false}"

# Check Azure login
az account show >/dev/null 2>&1 || { log_error "Run: az login"; exit 1; }

echo ""
log_warning "This will delete the following resource groups:"
log_warning "  - ${RESOURCE_GROUP} (AKS, networking, etc.)"
if [ "${DELETE_ACR}" = "true" ]; then
  log_warning "  - ${ACR_RESOURCE_GROUP} (ACR with all images) ⚠️  IMAGES WILL BE DELETED"
else
  log_info "  - ${ACR_RESOURCE_GROUP} will be PRESERVED (use DELETE_ACR=true to delete)"
fi
echo ""

read -p "Are you sure you want to delete these resources? (yes/no): " -r
echo
if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    log_info "Deletion cancelled"
    exit 0
fi

# Delete main resource group (AKS, networking, etc.)
log_info "Deleting resource group: ${RESOURCE_GROUP}..."
if az group show -n "${RESOURCE_GROUP}" >/dev/null 2>&1; then
  az group delete -n "${RESOURCE_GROUP}" --yes --no-wait
  log_success "Deletion initiated for ${RESOURCE_GROUP}"
else
  log_info "Resource group ${RESOURCE_GROUP} does not exist"
fi

# Optionally delete ACR resource group
if [ "${DELETE_ACR}" = "true" ]; then
  log_info "Deleting ACR resource group: ${ACR_RESOURCE_GROUP}..."
  if az group show -n "${ACR_RESOURCE_GROUP}" >/dev/null 2>&1; then
    az group delete -n "${ACR_RESOURCE_GROUP}" --yes --no-wait
    log_success "Deletion initiated for ${ACR_RESOURCE_GROUP}"
  else
    log_info "Resource group ${ACR_RESOURCE_GROUP} does not exist"
  fi
fi

echo ""
log_success "Deletion initiated for all requested resources"
log_info "Deletion is running in the background. Check status with:"
log_info "  az group list -o table"
echo ""

