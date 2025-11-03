#!/bin/bash

set -euo pipefail

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info() { echo -e "${BLUE}[INFO]${NC} $1"; }
ok() { echo -e "${GREEN}[✓]${NC} $1"; }
warn() { echo -e "${YELLOW}[⚠]${NC} $1"; }
fail() { echo -e "${RED}[✗]${NC} $1"; }

cat << 'EOF'
╔════════════════════════════════════════════════════════════════╗
║         NexusAI Platform - Complete Deployment (5 Steps)      ║
╚════════════════════════════════════════════════════════════════╝

This script deploys all NexusAI components in 5 sequential steps:
  
  Step 1: Core Controller (Operator)
  Step 2: Node Discovery (DaemonSet)
  Step 3: Platform Services (Memory, Catalog, Discovery)
  Step 4: Agent Services (DataOps, AI Safety, Prompt Optimizer)
  Step 5: Python Agents (Microsoft Agent Framework)

═══════════════════════════════════════════════════════════════

EOF

WAIT_BETWEEN_STEPS="${WAIT_BETWEEN_STEPS:-10}"

info "Wait time between steps: ${WAIT_BETWEEN_STEPS}s (set WAIT_BETWEEN_STEPS to change)"
echo ""
read -p "Start deployment? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    info "Deployment cancelled"
    exit 0
fi

echo ""
echo "═══════════════════════════════════════════════════════════════"
info "Starting Step 1/5: Core Controller..."
echo "═══════════════════════════════════════════════════════════════"
./deploy-1-core.sh
if [ $? -ne 0 ]; then
    fail "Step 1 failed! Stopping deployment."
    exit 1
fi
echo ""
info "Waiting ${WAIT_BETWEEN_STEPS}s before next step..."
sleep ${WAIT_BETWEEN_STEPS}

echo ""
echo "═══════════════════════════════════════════════════════════════"
info "Starting Step 2/5: Node Discovery..."
echo "═══════════════════════════════════════════════════════════════"
./deploy-2-node-discovery.sh
if [ $? -ne 0 ]; then
    fail "Step 2 failed! Stopping deployment."
    exit 1
fi
echo ""
info "Waiting ${WAIT_BETWEEN_STEPS}s before next step..."
sleep ${WAIT_BETWEEN_STEPS}

echo ""
echo "═══════════════════════════════════════════════════════════════"
info "Starting Step 3/5: Platform Services..."
echo "═══════════════════════════════════════════════════════════════"
./deploy-3-platform-services.sh
if [ $? -ne 0 ]; then
    fail "Step 3 failed! Stopping deployment."
    exit 1
fi
echo ""
info "Waiting ${WAIT_BETWEEN_STEPS}s before next step..."
sleep ${WAIT_BETWEEN_STEPS}

echo ""
echo "═══════════════════════════════════════════════════════════════"
info "Starting Step 4/5: Agent Services..."
echo "═══════════════════════════════════════════════════════════════"
./deploy-4-agent-services.sh
if [ $? -ne 0 ]; then
    fail "Step 4 failed! Stopping deployment."
    exit 1
fi
echo ""
info "Waiting ${WAIT_BETWEEN_STEPS}s before next step..."
sleep ${WAIT_BETWEEN_STEPS}

echo ""
echo "═══════════════════════════════════════════════════════════════"
info "Starting Step 5/5: Python Agents..."
echo "═══════════════════════════════════════════════════════════════"
./deploy-5-python-agents.sh
if [ $? -ne 0 ]; then
    fail "Step 5 failed!"
    exit 1
fi

echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║           🎉 COMPLETE DEPLOYMENT SUCCESSFUL! 🎉                ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""
ok "All 5 deployment steps completed successfully!"
echo ""
info "Deployment Summary:"
echo "  ✅ Step 1: Core Controller"
echo "  ✅ Step 2: Node Discovery"
echo "  ✅ Step 3: Platform Services"
echo "  ✅ Step 4: Agent Services"
echo "  ✅ Step 5: Python Agents"
echo ""
info "Next Steps:"
echo "  • Verify: ./scripts/verify-all.sh"
echo "  • Check:  kubectl get pods -n tensor-fusion-sys"
echo ""


