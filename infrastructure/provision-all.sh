#!/bin/bash

###############################################################################
# TensorFusion Infrastructure Provisioning Agent
# 
# This script automatically provisions the entire TensorFusion infrastructure
# in Azure, including:
# - AKS cluster with GPU nodes
# - Storage components (Qdrant, Redis, etc.)
# - Networking and security
# - TensorFusion deployment
#
# Prerequisites:
# - Azure CLI installed and logged in (az login)
# - kubectl installed
# - Helm 3.x installed
# - Sufficient Azure subscription quota for GPU VMs
###############################################################################

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration Variables
SUBSCRIPTION_ID="${AZURE_SUBSCRIPTION_ID:-}"
LOCATION="${AZURE_LOCATION:-eastus}"
RESOURCE_GROUP="${RESOURCE_GROUP:-tensor-fusion-rg}"
AKS_CLUSTER_NAME="${AKS_CLUSTER_NAME:-tensor-fusion-aks}"
STORAGE_ACCOUNT_NAME="tfstore$(openssl rand -hex 4)"
KEY_VAULT_NAME="tf-kv-$(openssl rand -hex 4)"
LOG_WORKSPACE_NAME="tensor-fusion-logs"
MANAGED_IDENTITY_NAME="tensor-fusion-identity"

# AKS Configuration
AKS_VERSION=""  # Use latest stable (empty string lets Azure pick)
SYSTEM_NODE_COUNT="${SYSTEM_NODE_COUNT:-2}"  # Start with 2 nodes for minimal quota usage
SYSTEM_NODE_SIZE="${SYSTEM_NODE_SIZE:-Standard_D4s_v3}"  # 4 vCPUs each = 8 total (fits even tight quotas)
GPU_NODE_SIZE="${GPU_NODE_SIZE:-Standard_NC24ads_A100_v4}"
GPU_NODE_MIN="${GPU_NODE_MIN:-0}"
GPU_NODE_MAX="${GPU_NODE_MAX:-5}"

# Namespaces
NAMESPACE_TF="tensor-fusion-sys"
NAMESPACE_STORAGE="storage"
NAMESPACE_OBSERVABILITY="observability"

# Feature Flags
DEPLOY_QDRANT="${DEPLOY_QDRANT:-true}"
DEPLOY_REDIS="${DEPLOY_REDIS:-true}"
DEPLOY_PORTKEY="${DEPLOY_PORTKEY:-true}"
DEPLOY_OBSERVABILITY="${DEPLOY_OBSERVABILITY:-true}"
ENABLE_GPU_POOL="${ENABLE_GPU_POOL:-true}"

###############################################################################
# Helper Functions
###############################################################################

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check Azure CLI
    if ! command -v az &> /dev/null; then
        log_error "Azure CLI not found. Install from: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
        exit 1
    fi
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl not found. Install from: https://kubernetes.io/docs/tasks/tools/"
        exit 1
    fi
    
    # Check Helm
    if ! command -v helm &> /dev/null; then
        log_error "Helm not found. Install from: https://helm.sh/docs/intro/install/"
        exit 1
    fi
    
    # Check Azure login
    if ! az account show &> /dev/null; then
        log_error "Not logged into Azure. Run: az login"
        exit 1
    fi
    
    # Remove aks-preview extension if installed (can cause API version conflicts)
    if az extension show --name aks-preview &> /dev/null; then
        log_warning "Found aks-preview extension that causes API version conflicts"
        log_error "Please manually remove it by running these commands in your terminal:"
        echo ""
        echo "  az extension remove --name aks-preview"
        echo "  exit  # Close this terminal"
        echo "  # Open a NEW terminal and re-run: ./provision-all.sh"
        echo ""
        exit 1
    fi
    
    log_success "All prerequisites met!"
}

set_subscription() {
    if [ -z "$SUBSCRIPTION_ID" ]; then
        log_info "No subscription specified, using current subscription..."
        SUBSCRIPTION_ID=$(az account show --query id -o tsv)
    else
        log_info "Setting subscription to: $SUBSCRIPTION_ID"
        az account set --subscription "$SUBSCRIPTION_ID"
    fi
    log_success "Using subscription: $SUBSCRIPTION_ID"
}

###############################################################################
# Infrastructure Provisioning Functions
###############################################################################

create_resource_group() {
    log_info "Creating resource group: $RESOURCE_GROUP in $LOCATION..."
    
    if az group exists --name "$RESOURCE_GROUP" --output tsv | grep -q "true"; then
        log_warning "Resource group already exists, skipping..."
    else
        az group create \
            --name "$RESOURCE_GROUP" \
            --location "$LOCATION" \
            --output none
        log_success "Resource group created!"
    fi
}

create_storage_account() {
    log_info "Creating storage account: $STORAGE_ACCOUNT_NAME..."
    
    az storage account create \
        --name "$STORAGE_ACCOUNT_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --location "$LOCATION" \
        --sku Standard_LRS \
        --kind StorageV2 \
        --output none
    
    log_success "Storage account created!"
}

create_key_vault() {
    log_info "Creating Key Vault: $KEY_VAULT_NAME..."
    
    az keyvault create \
        --name "$KEY_VAULT_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --location "$LOCATION" \
        --enable-rbac-authorization true \
        --output none
    
    log_success "Key Vault created!"
}

create_log_analytics() {
    log_info "Creating Log Analytics workspace: $LOG_WORKSPACE_NAME..."
    
    az monitor log-analytics workspace create \
        --resource-group "$RESOURCE_GROUP" \
        --workspace-name "$LOG_WORKSPACE_NAME" \
        --location "$LOCATION" \
        --output none
    
    WORKSPACE_ID=$(az monitor log-analytics workspace show \
        --resource-group "$RESOURCE_GROUP" \
        --workspace-name "$LOG_WORKSPACE_NAME" \
        --query id -o tsv)
    
    log_success "Log Analytics workspace created!"
}

create_managed_identity() {
    log_info "Creating managed identity: $MANAGED_IDENTITY_NAME..."
    
    az identity create \
        --name "$MANAGED_IDENTITY_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --location "$LOCATION" \
        --output none
    
    IDENTITY_ID=$(az identity show \
        --name "$MANAGED_IDENTITY_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --query id -o tsv)
    
    IDENTITY_CLIENT_ID=$(az identity show \
        --name "$MANAGED_IDENTITY_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --query clientId -o tsv)
    
    log_success "Managed identity created!"
}

create_aks_cluster() {
    log_info "Creating AKS cluster: $AKS_CLUSTER_NAME (this may take 10-15 minutes)..."
    
    # Build AKS create command
    local aks_cmd="az aks create \
        --resource-group $RESOURCE_GROUP \
        --name $AKS_CLUSTER_NAME \
        --location $LOCATION \
        --node-count $SYSTEM_NODE_COUNT \
        --node-vm-size $SYSTEM_NODE_SIZE \
        --nodepool-name system \
        --nodepool-labels pool=system \
        --network-plugin azure \
        --enable-managed-identity \
        --assign-identity $IDENTITY_ID \
        --enable-addons monitoring \
        --workspace-resource-id $WORKSPACE_ID \
        --generate-ssh-keys \
        --output none"
    
    # Add kubernetes version if specified
    if [ -n "$AKS_VERSION" ]; then
        aks_cmd="$aks_cmd --kubernetes-version $AKS_VERSION"
    fi
    
    # Execute the command
    eval $aks_cmd
    
    log_success "AKS cluster created!"
}

add_gpu_nodepool() {
    log_info "Adding GPU node pool to AKS cluster..."
    
    # Check if GPU nodepool already exists
    if az aks nodepool show \
        --resource-group "$RESOURCE_GROUP" \
        --cluster-name "$AKS_CLUSTER_NAME" \
        --name gpunodes &> /dev/null; then
        log_warning "GPU node pool already exists, skipping..."
        return
    fi
    
    az aks nodepool add \
        --resource-group "$RESOURCE_GROUP" \
        --cluster-name "$AKS_CLUSTER_NAME" \
        --name gpunodes \
        --node-count "$GPU_NODE_MIN" \
        --node-vm-size "$GPU_NODE_SIZE" \
        --enable-cluster-autoscaler \
        --min-count "$GPU_NODE_MIN" \
        --max-count "$GPU_NODE_MAX" \
        --node-taints nvidia.com/gpu=present:NoSchedule \
        --labels pool=gpu gpu=a100 \
        --output none
    
    log_success "GPU node pool added!"
}

configure_kubectl() {
    log_info "Configuring kubectl access..."
    
    az aks get-credentials \
        --resource-group "$RESOURCE_GROUP" \
        --name "$AKS_CLUSTER_NAME" \
        --overwrite-existing \
        --output none
    
    log_success "kubectl configured!"
}

###############################################################################
# Kubernetes Setup Functions
###############################################################################

create_namespaces() {
    log_info "Creating Kubernetes namespaces..."
    
    kubectl create namespace "$NAMESPACE_TF" --dry-run=client -o yaml | kubectl apply -f -
    kubectl create namespace "$NAMESPACE_STORAGE" --dry-run=client -o yaml | kubectl apply -f -
    kubectl create namespace "$NAMESPACE_OBSERVABILITY" --dry-run=client -o yaml | kubectl apply -f -
    
    log_success "Namespaces created!"
}

install_gpu_operator() {
    log_info "Installing NVIDIA GPU Operator..."
    
    helm repo add nvidia https://helm.ngc.nvidia.com/nvidia --force-update
    helm repo update
    
    helm upgrade --install gpu-operator nvidia/gpu-operator \
        --namespace gpu-operator \
        --create-namespace \
        --set driver.enabled=true \
        --wait \
        --timeout 10m
    
    log_success "GPU Operator installed!"
}

deploy_qdrant() {
    if [ "$DEPLOY_QDRANT" != "true" ]; then
        log_info "Skipping Qdrant deployment (DEPLOY_QDRANT=false)"
        return
    fi
    
    log_info "Deploying Qdrant vector database..."
    
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: qdrant-storage
  namespace: $NAMESPACE_STORAGE
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 50Gi
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: qdrant
  namespace: $NAMESPACE_STORAGE
spec:
  serviceName: qdrant
  replicas: 1
  selector:
    matchLabels:
      app: qdrant
  template:
    metadata:
      labels:
        app: qdrant
    spec:
      containers:
      - name: qdrant
        image: qdrant/qdrant:v1.7.4
        ports:
        - containerPort: 6333
          name: http
        - containerPort: 6334
          name: grpc
        volumeMounts:
        - name: storage
          mountPath: /qdrant/storage
        resources:
          requests:
            cpu: "2"
            memory: "4Gi"
          limits:
            cpu: "4"
            memory: "8Gi"
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: qdrant-storage
---
apiVersion: v1
kind: Service
metadata:
  name: qdrant-service
  namespace: $NAMESPACE_STORAGE
spec:
  selector:
    app: qdrant
  ports:
  - port: 6333
    targetPort: 6333
    name: http
  - port: 6334
    targetPort: 6334
    name: grpc
  type: ClusterIP
EOF
    
    log_success "Qdrant deployed!"
}

deploy_redis() {
    if [ "$DEPLOY_REDIS" != "true" ]; then
        log_info "Skipping Redis deployment (DEPLOY_REDIS=false)"
        return
    fi
    
    log_info "Deploying Redis..."
    
    helm repo add bitnami https://charts.bitnami.com/bitnami --force-update
    helm repo update
    
    helm upgrade --install redis bitnami/redis \
        --namespace "$NAMESPACE_STORAGE" \
        --set auth.enabled=false \
        --set master.persistence.size=10Gi \
        --set replica.replicaCount=1 \
        --wait
    
    log_success "Redis deployed!"
}

deploy_portkey() {
    if [ "$DEPLOY_PORTKEY" != "true" ]; then
        log_info "Skipping Portkey deployment (DEPLOY_PORTKEY=false)"
        return
    fi
    
    log_info "Deploying Portkey AI Gateway..."
    
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: portkey-gateway
  namespace: $NAMESPACE_TF
spec:
  replicas: 3
  selector:
    matchLabels:
      app: portkey-gateway
  template:
    metadata:
      labels:
        app: portkey-gateway
    spec:
      containers:
      - name: portkey
        image: ghcr.io/portkey-ai/gateway:latest
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: REDIS_URL
          value: "redis://redis-master.$NAMESPACE_STORAGE:6379"
        resources:
          requests:
            cpu: "500m"
            memory: "512Mi"
          limits:
            cpu: "2"
            memory: "2Gi"
---
apiVersion: v1
kind: Service
metadata:
  name: portkey-gateway
  namespace: $NAMESPACE_TF
spec:
  selector:
    app: portkey-gateway
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
EOF
    
    log_success "Portkey deployed!"
}

deploy_observability() {
    if [ "$DEPLOY_OBSERVABILITY" != "true" ]; then
        log_info "Skipping observability stack (DEPLOY_OBSERVABILITY=false)"
        return
    fi
    
    log_info "Deploying observability stack (Prometheus + Grafana)..."
    
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts --force-update
    helm repo add grafana https://grafana.github.io/helm-charts --force-update
    helm repo update
    
    # Install Prometheus
    helm upgrade --install prometheus prometheus-community/prometheus \
        --namespace "$NAMESPACE_OBSERVABILITY" \
        --set server.persistentVolume.size=100Gi \
        --wait
    
    # Install Grafana
    helm upgrade --install grafana grafana/grafana \
        --namespace "$NAMESPACE_OBSERVABILITY" \
        --set persistence.enabled=true \
        --set persistence.size=10Gi \
        --set adminPassword=admin \
        --wait
    
    log_success "Observability stack deployed!"
}

create_secrets() {
    log_info "Creating Kubernetes secrets..."
    
    # Azure credentials
    if [ -n "${AZURE_CLIENT_ID:-}" ] && [ -n "${AZURE_CLIENT_SECRET:-}" ] && [ -n "${AZURE_TENANT_ID:-}" ]; then
        kubectl create secret generic azure-credentials \
            --from-literal=client-id="$AZURE_CLIENT_ID" \
            --from-literal=client-secret="$AZURE_CLIENT_SECRET" \
            --from-literal=tenant-id="$AZURE_TENANT_ID" \
            --from-literal=subscription-id="$SUBSCRIPTION_ID" \
            --namespace "$NAMESPACE_TF" \
            --dry-run=client -o yaml | kubectl apply -f -
        log_success "Azure credentials secret created!"
    else
        log_warning "Azure service principal not provided. Create manually later."
    fi
    
    # Placeholder secrets (user should update)
    kubectl create secret generic portkey-credentials \
        --from-literal=api-key="REPLACE_WITH_PORTKEY_API_KEY" \
        --namespace "$NAMESPACE_TF" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    kubectl create secret generic foundry-keys \
        --from-literal=api-key="REPLACE_WITH_FOUNDRY_API_KEY" \
        --namespace "$NAMESPACE_TF" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    log_success "Placeholder secrets created!"
}

###############################################################################
# Main Execution
###############################################################################

print_banner() {
    cat << "EOF"
╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║   TensorFusion Infrastructure Provisioning Agent              ║
║   Automated Azure Infrastructure Setup                         ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝
EOF
}

print_summary() {
    log_success "Infrastructure provisioning complete!"
    echo ""
    echo "═══════════════════════════════════════════════════════════════"
    echo "                  DEPLOYMENT SUMMARY"
    echo "═══════════════════════════════════════════════════════════════"
    echo "Subscription:        $SUBSCRIPTION_ID"
    echo "Resource Group:      $RESOURCE_GROUP"
    echo "Location:            $LOCATION"
    echo "AKS Cluster:         $AKS_CLUSTER_NAME"
    echo "Storage Account:     $STORAGE_ACCOUNT_NAME"
    echo "Key Vault:           $KEY_VAULT_NAME"
    echo "Managed Identity:    $MANAGED_IDENTITY_NAME"
    echo "═══════════════════════════════════════════════════════════════"
    echo ""
    echo "Next steps:"
    echo "1. Scale GPU node pool: az aks nodepool scale --node-count 1 \\"
    echo "       --resource-group $RESOURCE_GROUP \\"
    echo "       --cluster-name $AKS_CLUSTER_NAME \\"
    echo "       --name gpunodes"
    echo ""
    echo "2. Install TensorFusion:"
    echo "   cd ../charts/tensor-fusion"
    echo "   helm install tensor-fusion . \\"
    echo "       --namespace $NAMESPACE_TF \\"
    echo "       --values values-enhanced.yaml"
    echo ""
    echo "3. Access Grafana:"
    echo "   kubectl port-forward -n $NAMESPACE_OBSERVABILITY svc/grafana 3000:80"
    echo "   Username: admin"
    echo "   Password: admin"
    echo ""
    echo "4. Follow the validation guide: infrastructure/VALIDATION_GUIDE.md"
    echo "═══════════════════════════════════════════════════════════════"
}

main() {
    print_banner
    
    log_info "Starting infrastructure provisioning..."
    log_info "This will take approximately 20-30 minutes..."
    echo ""
    
    # Prerequisites
    check_prerequisites
    set_subscription
    
    # Azure Infrastructure
    create_resource_group
    create_storage_account
    create_key_vault
    create_log_analytics
    create_managed_identity
    create_aks_cluster
    add_gpu_nodepool
    configure_kubectl
    
    # Kubernetes Setup
    create_namespaces
    install_gpu_operator
    deploy_qdrant
    deploy_redis
    deploy_portkey
    deploy_observability
    create_secrets
    
    # Summary
    print_summary
}

# Run main function
main "$@"

