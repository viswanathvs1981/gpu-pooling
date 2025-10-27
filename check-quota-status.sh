#!/bin/bash

# Quick script to check if quota increase is approved

echo "🔍 Checking quota status..."
echo ""

CURRENT=$(az vm list-usage --location eastus --query "[?localName=='Total Regional vCPUs'].limit" -o tsv 2>/dev/null)

if [ "$CURRENT" = "14" ]; then
  echo "✅ APPROVED! Quota is now 14 vCPUs"
  echo ""
  echo "Next step: Add GPU node"
  echo "  ./add-gpu-node.sh"
elif [ "$CURRENT" = "10" ]; then
  echo "⏳ Still pending... Current limit: 10 vCPUs"
  echo ""
  echo "Check status:"
  echo "  - Portal: https://portal.azure.com/#view/Microsoft_Azure_Capacity/QuotaMenuBlade/~/myQuotas"
  echo "  - Wait time: Usually 1-2 hours for small increases"
else
  echo "📊 Current limit: $CURRENT vCPUs"
  echo ""
  if [ "$CURRENT" -ge 14 ]; then
    echo "✅ You have enough quota! Run:"
    echo "  ./add-gpu-node.sh"
  else
    echo "⏳ Still waiting for approval..."
  fi
fi

echo ""
echo "Full quota status:"
az vm list-usage --location eastus --query "[?localName=='Total Regional vCPUs']" -o table

