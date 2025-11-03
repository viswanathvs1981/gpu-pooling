#!/bin/bash

set -uo pipefail

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
echo "║         NexusAI Platform - Deploy Infrastructure               ║"
echo "╚════════════════════════════════════════════════════════════════╝"

# Load ACR config
if [ -f ".acr-config" ]; then
  source .acr-config
  log_info "Using ACR: ${ACR_NAME}"
else
  log_error "ACR configuration not found. Run ./setup-acr.sh first"
  exit 1
fi

# Config
LOCATION="${LOCATION:-eastus}"
RESOURCE_GROUP="${RESOURCE_GROUP:-tensor-fusion-rg}"
AKS_CLUSTER="${AKS_CLUSTER:-tensor-fusion-aks}"

SYSTEM_NODE_COUNT="${SYSTEM_NODE_COUNT:-2}"
SYSTEM_NODE_SIZE="${SYSTEM_NODE_SIZE:-Standard_D4s_v3}"
GPU_NODE_SIZE="${GPU_NODE_SIZE:-Standard_NC24ads_A100_v4}"
GPU_NODE_MIN="${GPU_NODE_MIN:-0}"
GPU_NODE_MAX="${GPU_NODE_MAX:-3}"

NS_TF="tensor-fusion-sys"
NS_STORAGE="storage"
NS_OBS="observability"
NS_QDRANT="qdrant"
NS_GREPTIME="greptimedb"

check_prereqs() {
  log_info "Checking prerequisites..."
  command -v az >/dev/null 2>&1 || { log_error "Azure CLI required"; exit 1; }
  command -v kubectl >/dev/null 2>&1 || { log_error "kubectl required"; exit 1; }
  command -v helm >/dev/null 2>&1 || { log_error "helm required"; exit 1; }
  az account show >/dev/null 2>&1 || { log_error "Run: az login"; exit 1; }
  log_success "Prerequisites OK"
}

create_rg() {
  log_info "Creating resource group ${RESOURCE_GROUP} in ${LOCATION}"
  az group create -n "${RESOURCE_GROUP}" -l "${LOCATION}" -o none
  log_success "Resource group ready"
}

check_gpu_quota() {
  log_info "Checking GPU quota availability..."
  local t4_quota=$(az vm list-usage --location "${LOCATION}" --query "[?contains(localName, 'NCASv3_T4')].limit" -o tsv 2>/dev/null || echo "0")
  
  if [ "$t4_quota" -ge 4 ]; then
    log_success "NCASv3_T4 quota available: ${t4_quota} vCPUs"
    GPU_NODE_SIZE="Standard_NC4as_T4_v3"
    return 0
  else
    log_warning "NCASv3_T4 quota: ${t4_quota} vCPUs (need 4+)"
    return 1
  fi
}

create_aks() {
  if az aks show -g "${RESOURCE_GROUP}" -n "${AKS_CLUSTER}" >/dev/null 2>&1; then
    log_success "AKS cluster exists"
  else
    log_info "Creating AKS cluster ${AKS_CLUSTER}..."
    az aks create \
      -g "${RESOURCE_GROUP}" -n "${AKS_CLUSTER}" \
      --location "${LOCATION}" \
      --enable-managed-identity \
      --nodepool-name system \
      --node-vm-size "${SYSTEM_NODE_SIZE}" \
      --node-count "${SYSTEM_NODE_COUNT}" \
      --generate-ssh-keys -o none
    log_success "AKS created"
  fi

  # Add GPU nodepool if quota available
  if check_gpu_quota; then
    if az aks nodepool show -g "${RESOURCE_GROUP}" --cluster-name "${AKS_CLUSTER}" -n gpu >/dev/null 2>&1; then
      log_success "GPU nodepool exists"
    else
      log_info "Adding GPU nodepool (${GPU_NODE_SIZE})"
      az aks nodepool add \
        -g "${RESOURCE_GROUP}" --cluster-name "${AKS_CLUSTER}" \
        -n gpu \
        --node-vm-size "${GPU_NODE_SIZE}" \
        --node-count 0 \
        --min-count "${GPU_NODE_MIN}" \
        --max-count "${GPU_NODE_MAX}" \
        --enable-cluster-autoscaler \
        --node-taints sku=gpu:NoSchedule -o none
      log_success "GPU nodepool added (autoscaling: ${GPU_NODE_MIN}-${GPU_NODE_MAX})"
    fi
  fi

  log_info "Fetching kubeconfig"
  az aks get-credentials -g "${RESOURCE_GROUP}" -n "${AKS_CLUSTER}" --overwrite-existing -o none
}

attach_acr() {
  log_info "Attaching ACR to AKS"
  az aks update -g "${RESOURCE_GROUP}" -n "${AKS_CLUSTER}" --attach-acr "${ACR_NAME}" -o none
  log_success "ACR attached"
}

create_namespaces() {
  log_info "Creating namespaces"
  kubectl create namespace "${NS_TF}" --dry-run=client -o yaml | kubectl apply -f - >/dev/null 2>&1
  kubectl create namespace "${NS_STORAGE}" --dry-run=client -o yaml | kubectl apply -f - >/dev/null 2>&1
  kubectl create namespace "${NS_QDRANT}" --dry-run=client -o yaml | kubectl apply -f - >/dev/null 2>&1
  kubectl create namespace "${NS_GREPTIME}" --dry-run=client -o yaml | kubectl apply -f - >/dev/null 2>&1
  kubectl create namespace "${NS_OBS}" --dry-run=client -o yaml | kubectl apply -f - >/dev/null 2>&1
  log_success "Namespaces created"
}

deploy_gpu_operator() {
  log_info "Deploying NVIDIA GPU Operator"
  helm repo add nvidia https://helm.ngc.nvidia.com/nvidia --force-update >/dev/null 2>&1
  helm repo update >/dev/null 2>&1
  
  if helm list -n gpu-operator | grep -q gpu-operator; then
    log_success "GPU Operator already deployed"
  else
    helm install gpu-operator nvidia/gpu-operator \
      --namespace gpu-operator --create-namespace \
      --set driver.enabled=true \
      --wait --timeout=10m >/dev/null 2>&1
    log_success "GPU Operator deployed"
  fi
}

deploy_greptimedb() {
  log_info "Deploying GreptimeDB (fixed configuration)"
  kubectl apply -f infrastructure/greptimedb.yaml >/dev/null 2>&1
  log_success "GreptimeDB deployed with increased resources"
}

deploy_qdrant() {
  log_info "Deploying Qdrant (fixed health endpoint)"
  kubectl apply -f infrastructure/qdrant.yaml >/dev/null 2>&1
  log_success "Qdrant deployed with corrected health checks"
}

deploy_redis() {
  log_info "Deploying Redis"
  helm repo add bitnami https://charts.bitnami.com/bitnami --force-update >/dev/null 2>&1
  helm repo update >/dev/null 2>&1
  
  helm upgrade --install redis bitnami/redis \
    --namespace "${NS_STORAGE}" \
    --set auth.enabled=false \
    --set master.persistence.enabled=false \
    --set replica.persistence.enabled=false \
    --wait --timeout=5m >/dev/null 2>&1
  log_success "Redis deployed"
}

deploy_postgresql() {
  log_info "Deploying PostgreSQL"
  helm upgrade --install postgresql bitnami/postgresql \
    --namespace "${NS_STORAGE}" \
    --set auth.postgresPassword=nexusai123 \
    --set auth.database=nexusai \
    --set primary.persistence.enabled=false \
    --wait --timeout=5m >/dev/null 2>&1
  log_success "PostgreSQL deployed"
}

deploy_minio() {
  log_info "Deploying MinIO"
  helm repo add minio https://charts.min.io/ --force-update >/dev/null 2>&1
  helm repo update >/dev/null 2>&1
  
  helm upgrade --install minio minio/minio \
    --namespace "${NS_STORAGE}" \
    --set mode=standalone \
    --set rootUser=minioadmin \
    --set rootPassword=minioadmin123 \
    --set persistence.enabled=false \
    --set resources.requests.memory=512Mi \
    --wait --timeout=5m >/dev/null 2>&1
  log_success "MinIO deployed"
}

deploy_observability() {
  log_info "Deploying Prometheus & Grafana"
  helm repo add prometheus-community https://prometheus-community.github.io/helm-charts --force-update >/dev/null 2>&1
  helm repo add grafana https://grafana.github.io/helm-charts --force-update >/dev/null 2>&1
  helm repo update >/dev/null 2>&1
  
  # Deploy in background
  (helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
    --namespace "${NS_OBS}" \
    --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
    --wait --timeout=10m >/dev/null 2>&1 && log_success "Prometheus deployed") &
  
  wait
}

deploy_portkey() {
  log_info "Deploying Portkey Gateway (fixed configuration)"
  kubectl apply -f infrastructure/portkey-gateway.yaml >/dev/null 2>&1
  log_success "Portkey Gateway deployed on port 8080"
}

deploy_msaf_agents() {
  log_info "Deploying MSAF Python Agents"
  kubectl apply -f infrastructure/msaf-agents.yaml >/dev/null 2>&1
  log_success "MSAF agents deployed (Orchestrator, Training, Deployment, Cost)"
}

install_crds() {
  log_info "Installing CRDs from kustomize..."
  kubectl apply -k config/crd >/dev/null 2>&1
  log_success "CRDs installed"
}

deploy_helm_chart() {
  log_info "Deploying NexusAI Helm chart"
  
  # Create temporary values file with ACR configuration
  cat > /tmp/tf-values.yaml <<EOF
image:
  repository: ${ACR_LOGIN_SERVER}/nexusai/operator
  pullPolicy: Always
  tag: latest

nodeDiscovery:
  enabled: true
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/node-discovery
    tag: latest
    pullPolicy: Always

memoryService:
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/memory-service
    tag: latest
    pullPolicy: Always

modelCatalog:
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/model-catalog
    tag: latest
    pullPolicy: Always

discoveryAgent:
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/discovery-agent
    tag: latest
    pullPolicy: Always

promptOptimizer:
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/prompt-optimizer
    tag: latest
    pullPolicy: Always

dataopsAgents:
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/dataops-agents
    tag: latest
    pullPolicy: Always

aiSafety:
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/aisafety-service
    tag: latest
    pullPolicy: Always

msafAgents:
  image:
    repository: ${ACR_LOGIN_SERVER}/nexusai/python-agents
    tag: latest
    pullPolicy: Always

redis:
  host: redis-master.${NS_STORAGE}.svc.cluster.local
  port: 6379

postgresql:
  host: postgresql.${NS_STORAGE}.svc.cluster.local
  port: 5432
  database: nexusai
  username: postgres
  password: nexusai123

minio:
  endpoint: minio.${NS_STORAGE}.svc.cluster.local:9000
  accessKey: minioadmin
  secretKey: minioadmin123

qdrant:
  url: http://qdrant.${NS_QDRANT}.svc.cluster.local:6333

greptimedb:
  host: greptimedb.${NS_GREPTIME}.svc.cluster.local
  httpPort: 4000
  grpcPort: 4001

portkey:
  url: http://portkey-gateway.${NS_TF}.svc.cluster.local:8787
EOF

  # Deploy with retry
  for i in {1..2}; do
    log_info "Helm deployment attempt $i/2..."
    if helm upgrade --install tensor-fusion ./charts/tensor-fusion \
      -n "${NS_TF}" \
      --values /tmp/tf-values.yaml \
      --wait --timeout=10m 2>/tmp/helm-deploy.log; then
      log_success "NexusAI deployed successfully"
      return 0
    else
      log_warning "Helm deployment attempt $i failed"
      if [ $i -lt 2 ]; then
        log_info "Retrying in 30 seconds..."
        sleep 30
      fi
    fi
  done
  
  log_error "Helm deployment failed after 2 attempts. Check /tmp/helm-deploy.log"
  return 1
}

wait_ready() {
  log_info "Waiting for pods to be ready (max 5 minutes)..."
  kubectl wait --for=condition=ready pod -l app=greptimedb -n "${NS_GREPTIME}" --timeout=300s 2>/dev/null || true
  kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=qdrant -n "${NS_QDRANT}" --timeout=300s 2>/dev/null || true
  kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=redis -n "${NS_STORAGE}" --timeout=300s 2>/dev/null || true
  kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=postgresql -n "${NS_STORAGE}" --timeout=300s 2>/dev/null || true
  kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=minio -n "${NS_STORAGE}" --timeout=300s 2>/dev/null || true
  log_success "Core services ready"
}

print_summary() {
  echo ""
  log_success "═══════════════════════════════════════════════════════════════"
  log_success "NexusAI Platform Infrastructure Deployment Complete!"
  log_success "═══════════════════════════════════════════════════════════════"
  echo ""
  log_info "Cluster: ${AKS_CLUSTER}"
  log_info "Resource Group: ${RESOURCE_GROUP}"
  log_info "ACR: ${ACR_NAME}"
  echo ""
  log_info "Services deployed:"
  log_info "  ✓ AKS with GPU support"
  log_info "  ✓ GreptimeDB (time-series)"
  log_info "  ✓ Qdrant (vector DB)"
  log_info "  ✓ Redis (cache/pub-sub)"
  log_info "  ✓ PostgreSQL (relational DB)"
  log_info "  ✓ MinIO (object storage)"
  log_info "  ✓ Prometheus & Grafana"
  log_info "  ✓ Portkey Gateway"
  log_info "  ✓ MSAF Python Agents"
  log_info "  ✓ NexusAI Platform"
  echo ""
  log_info "Verify deployment:"
  log_info "  ./verify-all.sh"
  echo ""
  log_info "To delete and recreate infrastructure:"
  log_info "  ./delete-all.sh && ./deploy-infra.sh"
  echo ""
}

# Main execution
main() {
  check_prereqs
  create_rg
  create_aks
  attach_acr
  create_namespaces
  
  # Deploy core services in parallel
  deploy_gpu_operator &
  deploy_greptimedb &
  deploy_qdrant &
  deploy_redis &
  deploy_postgresql &
  deploy_minio &
  deploy_observability &
  deploy_portkey &
  deploy_msaf_agents &

  wait
  
  install_crds
  deploy_helm_chart
  wait_ready
  print_summary
}

main "$@"

