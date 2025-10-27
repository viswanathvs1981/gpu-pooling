#!/bin/bash
set -euo pipefail

BLUE='\033[0;34m'; GREEN='\033[0;32m'; RED='\033[0;31m'; NC='\033[0m'
info(){ echo -e "${BLUE}[INFO]${NC} $1"; }
ok(){ echo -e "${GREEN}[✓]${NC} $1"; }
fail(){ echo -e "${RED}[✗]${NC} $1"; }

info "Checking T4 GPU quota status..."
echo ""

T4_QUOTA=$(az vm list-usage --location eastus --query "[?contains(localName, 'NCASv3_T4')].limit" -o tsv)

if [ "$T4_QUOTA" -ge 8 ]; then
  ok "NCASv3_T4 Family quota approved: ${T4_QUOTA} vCPUs"
  ok "You can add GPU nodes now! Run: ./add-gpu-node.sh"
else
  fail "NCASv3_T4 Family quota still at ${T4_QUOTA}"
  echo ""
  echo "📝 Request the correct family in Azure Portal:"
  echo "   1. Go to: https://portal.azure.com/#view/Microsoft_Azure_Capacity/QuotaMenuBlade"
  echo "   2. Filter: Compute → East US"
  echo "   3. Search: 'NCASv3' (NOT NVSv4)"
  echo "   4. Select: 'Standard NCASv3_T4 Family vCPUs'"
  echo "   5. Request: 8"
  echo ""
  echo "❌ You currently have 8 vCPUs for 'NVSv4' (AMD) which doesn't work with AKS."
  echo "✅ You need 'NCASv3_T4' (NVIDIA T4) for TensorFusion."
fi
