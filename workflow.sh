#!/bin/bash

cat << 'EOF'
╔════════════════════════════════════════════════════════════════╗
║                NexusAI Platform - Workflow                     ║
╚════════════════════════════════════════════════════════════════╝

INITIAL SETUP (One-time only):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

1. Setup ACR (creates separate resource group for ACR):
   ./setup-acr.sh
   
   ✓ Creates: nexusai-acr-rg resource group
   ✓ Creates: ACR with unique name
   ✓ Saves config to: .acr-config

2. Build & Push Images (30-40 minutes):
   ./build-images.sh
   
   ✓ Builds all 9 images
   ✓ Pushes to ACR
   ✓ Run only when code changes

INFRASTRUCTURE DEPLOYMENT (Repeatable):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

3. Deploy Infrastructure (10-15 minutes):
   ./deploy-infra.sh
   
   ✓ Creates: tensor-fusion-rg resource group
   ✓ Deploys: AKS cluster, GPU nodes
   ✓ Deploys: Supporting services (Redis, Qdrant, PostgreSQL, MinIO, etc.)
   ✓ Can be deleted and recreated anytime

4. Deploy NexusAI Platform (Choose Your Approach):
   
   Option A: All-in-One (original, ~3-5 minutes)
   ./deploy-platform.sh
   
   Option B: Split Deployment (NEW! Recommended)
   ./deploy-platform-all.sh      # All 5 steps automated
   
   OR deploy step-by-step:
   ./deploy-1-core.sh            # Step 1: Core Controller
   ./deploy-2-node-discovery.sh  # Step 2: Node Discovery
   ./deploy-3-platform-services.sh  # Step 3: Memory/Catalog
   ./deploy-4-agent-services.sh  # Step 4: DataOps/AI Safety
   ./deploy-5-python-agents.sh   # Step 5: Python Agents
   
   ✓ Deploys: All NexusAI components
   ✓ Uses custom images from ACR
   ✓ Split deployment avoids rate limiting
   ✓ Easier debugging per component

5. Verify Deployment:
   ./scripts/verify-all.sh

CLEANUP:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Delete infrastructure only (keeps ACR & images):
   ./delete-all.sh

Delete everything including ACR & images:
   DELETE_ACR=true ./delete-all.sh

TYPICAL WORKFLOWS:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🆕 Fresh Start (first time):
   ./setup-acr.sh          # Setup ACR (one-time)
   ./build-images.sh       # Build images (~30-40 min)
   ./deploy-infra.sh       # Deploy infrastructure (~10-15 min)
   ./deploy-platform.sh    # Deploy NexusAI (~3-5 min)
   ./scripts/verify-all.sh # Verify deployment

🔄 After Code Changes:
   ./build-images.sh       # Rebuild images
   ./deploy-platform.sh    # Upgrade platform (auto-detects upgrade)

🔄 Platform Upgrade Only:
   ./deploy-platform.sh    # Will prompt to upgrade if exists

🔄 Infrastructure Reset (keeps images):
   ./delete-all.sh         # Delete infra
   ./deploy-infra.sh       # Redeploy infra
   ./deploy-platform.sh    # Redeploy platform

🗑️ Complete Reset (including images):
   DELETE_ACR=true ./delete-all.sh
   ./setup-acr.sh
   ./build-images.sh
   ./deploy-infra.sh
   ./deploy-platform.sh

RESOURCE GROUPS:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

nexusai-acr-rg    → ACR (persistent, one-time setup)
tensor-fusion-rg  → AKS + Services (ephemeral, can recreate)

SCRIPT SUMMARY:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

setup-acr.sh          - One-time ACR setup
build-images.sh       - Build and push all images to ACR
deploy-infra.sh       - Deploy AKS + supporting services
deploy-platform.sh    - Deploy NexusAI Helm chart (NEW!)
delete-all.sh         - Clean up (keeps ACR by default)
scripts/verify-all.sh - Comprehensive verification
workflow.sh           - This guide

EOF
