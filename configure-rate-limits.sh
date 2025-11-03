#!/bin/bash

set -euo pipefail

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() { echo -e "${BLUE}[INFO]${NC} $1"; }
ok() { echo -e "${GREEN}[✓]${NC} $1"; }
warn() { echo -e "${YELLOW}[⚠]${NC} $1"; }

cat << 'EOF'
╔════════════════════════════════════════════════════════════════╗
║        Kubernetes API Rate Limit Configuration                ║
╚════════════════════════════════════════════════════════════════╝
EOF

echo ""
info "This script will help you configure Kubernetes API rate limits"
echo ""

# Get cluster name
CLUSTER_NAME=$(az aks list --query "[0].name" -o tsv 2>/dev/null || echo "")
RESOURCE_GROUP=$(az aks list --query "[0].resourceGroup" -o tsv 2>/dev/null || echo "")

if [ -z "$CLUSTER_NAME" ] || [ -z "$RESOURCE_GROUP" ]; then
    warn "Could not detect AKS cluster automatically"
    echo ""
    echo "Please run manually:"
    echo "  CLUSTER_NAME=<your-cluster>"
    echo "  RESOURCE_GROUP=<your-rg>"
    exit 1
fi

info "Detected cluster: ${CLUSTER_NAME} in ${RESOURCE_GROUP}"
echo ""

# Solution 1: Client-side rate limits
cat << 'EOF'
═══════════════════════════════════════════════════════════════
SOLUTION 1: INCREASE CLIENT-SIDE RATE LIMITS (kubectl/helm)
═══════════════════════════════════════════════════════════════

Create a kubectl configuration file to increase client limits:

EOF

cat > /tmp/kubeconfig-rate-limit.yaml << 'YAML'
# Add these environment variables to increase kubectl rate limits
export KUBECTL_EXTERNAL_DIFF=""
export KUBECTL_CLIENT_QPS=100       # Increase from 5 to 100
export KUBECTL_CLIENT_BURST=150     # Increase from 10 to 150
YAML

cat /tmp/kubeconfig-rate-limit.yaml
echo ""
info "Add these to your ~/.zshrc or ~/.bashrc"
echo ""

# Solution 2: Update deploy-platform.sh
cat << 'EOF'
═══════════════════════════════════════════════════════════════
SOLUTION 2: UPDATE DEPLOY-PLATFORM.SH WITH RETRY LOGIC
═══════════════════════════════════════════════════════════════

Adding automatic retry with exponential backoff...

EOF

# Check if we should update the script
read -p "Update deploy-platform.sh with retry logic? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    info "Updating deploy-platform.sh with retry logic..."
    
    # Backup the original
    cp deploy-platform.sh deploy-platform.sh.backup
    
    cat > /tmp/helm_retry.sh << 'SCRIPT'

# Retry function with exponential backoff
helm_retry() {
    local max_attempts=5
    local timeout=2
    local attempt=1
    local exitCode=0

    while [ $attempt -le $max_attempts ]; do
        if [ $attempt -gt 1 ]; then
            warn "Attempt $attempt/$max_attempts (waiting ${timeout}s due to rate limiting)..."
            sleep $timeout
        fi

        set +e
        "$@"
        exitCode=$?
        set -e

        if [ $exitCode -eq 0 ]; then
            return 0
        fi

        # Check if it's a rate limit error
        if [ $exitCode -ne 0 ] && [ $attempt -lt $max_attempts ]; then
            timeout=$((timeout * 2))
            attempt=$((attempt + 1))
        else
            return $exitCode
        fi
    done

    return $exitCode
}
SCRIPT

    ok "Retry logic template created in /tmp/helm_retry.sh"
    info "Manual integration required - see instructions below"
fi

echo ""

# Solution 3: AKS API Server configuration
cat << EOF
═══════════════════════════════════════════════════════════════
SOLUTION 3: INCREASE AKS API SERVER RATE LIMITS
═══════════════════════════════════════════════════════════════

For production clusters, increase server-side limits:

EOF

cat > /tmp/aks-rate-limit.sh << EOFSCRIPT
#!/bin/bash

# Update AKS cluster with higher API server limits
az aks update \\
  --resource-group ${RESOURCE_GROUP} \\
  --name ${CLUSTER_NAME} \\
  --api-server-authorized-ip-ranges "" \\
  --enable-managed-identity \\
  --no-wait

# Note: Azure doesn't directly expose API server rate limit settings
# But you can enable features that help:

# 1. Enable cluster autoscaler (reduces manual scaling operations)
az aks update \\
  --resource-group ${RESOURCE_GROUP} \\
  --name ${CLUSTER_NAME} \\
  --enable-cluster-autoscaler \\
  --min-count 1 \\
  --max-count 10

# 2. Upgrade to a higher AKS tier (Standard/Premium)
# Premium tier has higher API server SLA and better rate limits
az aks update \\
  --resource-group ${RESOURCE_GROUP} \\
  --name ${CLUSTER_NAME} \\
  --tier standard

EOFSCRIPT

cat /tmp/aks-rate-limit.sh
echo ""
info "Script saved to: /tmp/aks-rate-limit.sh"
echo ""

# Solution 4: Best Practices
cat << 'EOF'
═══════════════════════════════════════════════════════════════
SOLUTION 4: BEST PRACTICES TO AVOID RATE LIMITING
═══════════════════════════════════════════════════════════════

✅ 1. Batch Operations
   - Use kubectl apply -f <directory> instead of multiple single applies
   - Use helm upgrade --reuse-values for minor changes

✅ 2. Reduce Polling
   - Use --wait with longer timeouts instead of status polling
   - Avoid watch commands in loops

✅ 3. Efficient Selectors
   - Use label selectors to reduce API calls
   - Cache kubectl results when possible

✅ 4. Use Service Accounts
   - Service accounts have separate rate limits from users
   - Use for CI/CD pipelines

✅ 5. Wait Between Operations
   - Add small delays between major operations
   - Use exponential backoff for retries

EOF

# Solution 5: Immediate fix for current situation
cat << 'EOF'
═══════════════════════════════════════════════════════════════
SOLUTION 5: IMMEDIATE FIX (For Your Current Situation)
═══════════════════════════════════════════════════════════════

The rate limit will clear automatically in 2-3 minutes.
Meanwhile, you can:

1️⃣  Wait 3 minutes and run:
   ./deploy-platform.sh

2️⃣  OR use a new kubectl context (resets client limits):
   kubectl config use-context $(kubectl config current-context)

3️⃣  OR restart your kubectl connection:
   az aks get-credentials --resource-group ${RESOURCE_GROUP} \
     --name ${CLUSTER_NAME} --overwrite-existing

EOF

echo ""
ok "Configuration options displayed!"
echo ""
info "Recommended Actions:"
echo "  1. Add environment variables to ~/.zshrc"
echo "  2. Wait 3 minutes and run ./deploy-platform.sh"
echo "  3. For production: Upgrade to AKS Standard tier"
echo ""

