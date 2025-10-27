#!/bin/bash

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║   Request Azure vCPU Quota Increase                           ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Configuration
LOCATION="${LOCATION:-eastus}"
REQUESTED_VCPUS="${REQUESTED_VCPUS:-14}"

echo -e "${BLUE}Configuration:${NC}"
echo "  Location: $LOCATION"
echo "  Current vCPU Limit: 10"
echo "  Requested vCPU Limit: $REQUESTED_VCPUS"
echo ""

# Get subscription info
SUBSCRIPTION_ID=$(az account show --query id -o tsv)
SUBSCRIPTION_NAME=$(az account show --query name -o tsv)

echo -e "${BLUE}Subscription:${NC}"
echo "  ID: $SUBSCRIPTION_ID"
echo "  Name: $SUBSCRIPTION_NAME"
echo ""

echo -e "${YELLOW}Note: Azure CLI quota requests can be complex.${NC}"
echo -e "${YELLOW}The easiest method is through the Azure Portal.${NC}"
echo ""
echo -e "${BLUE}Portal Method (Recommended):${NC}"
echo "  1. Open: https://portal.azure.com/#view/Microsoft_Azure_Capacity/QuotaMenuBlade/~/myQuotas"
echo "  2. Filter by: Compute → $LOCATION"
echo "  3. Find: 'Total Regional vCPUs'"
echo "  4. Click: 'Request increase'"
echo "  5. Enter: $REQUESTED_VCPUS"
echo "  6. Justification: 'Need GPU nodes for ML platform validation'"
echo ""
echo -e "${BLUE}Alternative: Create Support Ticket via CLI${NC}"
echo ""

# Try the modern quota API
echo -e "${BLUE}Attempting to create quota request...${NC}"

# This uses the Azure quota API (preview)
az rest --method put \
  --url "https://management.azure.com/subscriptions/${SUBSCRIPTION_ID}/providers/Microsoft.Compute/locations/${LOCATION}/providers/Microsoft.Quota/quotas/standardDSv3Family?api-version=2023-02-01" \
  --body "{
    \"properties\": {
      \"limit\": {
        \"value\": ${REQUESTED_VCPUS}
      },
      \"name\": {
        \"value\": \"standardDSv3Family\"
      }
    }
  }" 2>&1 || {
    echo ""
    echo -e "${YELLOW}Direct quota API request failed (this is common).${NC}"
    echo ""
    echo -e "${BLUE}Creating support ticket instead...${NC}"
    echo ""
    
    # Fallback: Create support ticket
    echo -e "${BLUE}Please complete the quota request manually:${NC}"
    echo ""
    echo "Run this command to open the portal:"
    echo ""
    echo "  open 'https://portal.azure.com/#view/Microsoft_Azure_Capacity/QuotaMenuBlade/~/myQuotas'"
    echo ""
    echo "Or use this direct link for $LOCATION quotas:"
    echo ""
    echo "  open 'https://portal.azure.com/#view/Microsoft_Azure_Capacity/QuotaMenuBlade/~/overview'"
    echo ""
    echo -e "${GREEN}Then:${NC}"
    echo "  1. Select 'Compute' service"
    echo "  2. Filter location: $LOCATION"
    echo "  3. Find: 'Total Regional vCPUs' (currently 10)"
    echo "  4. Click 'Request increase' (pencil icon)"
    echo "  5. New limit: $REQUESTED_VCPUS"
    echo "  6. Submit (approval usually takes 1-2 hours)"
    echo ""
    exit 0
  }

echo ""
echo -e "${GREEN}✓ Quota request submitted!${NC}"
echo ""
echo "Check status with:"
echo "  az quota show --scope \"subscriptions/${SUBSCRIPTION_ID}/providers/Microsoft.Compute/locations/${LOCATION}\" --resource-name \"standardDSv3Family\""
echo ""
echo "Or check in portal:"
echo "  https://portal.azure.com/#view/Microsoft_Azure_Capacity/QuotaMenuBlade/~/myQuotas"

