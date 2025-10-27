#!/bin/bash

set -uo pipefail  # Removed -e to handle errors gracefully

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
echo "║   TensorFusion - One-click Azure Deployment                    ║"
echo "╚════════════════════════════════════════════════════════════════╝"

# Config (can be overridden via env)
AZ_SUBSCRIPTION_ID="${AZ_SUBSCRIPTION_ID:-${AZURE_SUBSCRIPTION_ID:-}}"
LOCATION="${LOCATION:-eastus}"
RESOURCE_GROUP="${RESOURCE_GROUP:-tensor-fusion-rg}"
AKS_CLUSTER="${AKS_CLUSTER:-tensor-fusion-aks}"
ACR_NAME="${ACR_NAME:-}"

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
NS_PORTKEY="portkey"
ENABLE_GPU_POOL="${ENABLE_GPU_POOL:-true}"
BUILD_IMAGES="${BUILD_IMAGES:-true}"

check_prereqs() {
  log_info "Checking prerequisites..."
  command -v az >/dev/null 2>&1 || { log_error "Azure CLI is required"; exit 1; }
  command -v kubectl >/dev/null 2>&1 || { log_error "kubectl is required"; exit 1; }
  command -v helm >/dev/null 2>&1 || { log_error "helm is required"; exit 1; }
  az account show >/dev/null 2>&1 || { log_error "Run: az login"; exit 1; }
  if az extension show --name aks-preview >/dev/null 2>&1; then
    log_warning "Found aks-preview extension; it may cause API conflicts. Consider: az extension remove --name aks-preview"
  fi
  log_success "Prerequisites OK"
}

set_subscription() {
  if [[ -n "${AZ_SUBSCRIPTION_ID}" ]]; then
    az account set --subscription "${AZ_SUBSCRIPTION_ID}"
  fi
  AZ_SUBSCRIPTION_ID=$(az account show --query id -o tsv)
  log_success "Using subscription: ${AZ_SUBSCRIPTION_ID}"
}

create_rg() {
  log_info "Ensuring resource group ${RESOURCE_GROUP} in ${LOCATION}"
  az group create -n "${RESOURCE_GROUP}" -l "${LOCATION}" -o none
}

check_gpu_quota() {
  log_info "Checking GPU quota availability..."
  
  # Check NCASv3_T4 (most cost-effective)
  local t4_quota=$(az vm list-usage --location "${LOCATION}" --query "[?contains(localName, 'NCASv3_T4')].limit" -o tsv 2>/dev/null || echo "0")
  
  if [ "$t4_quota" -ge 4 ]; then
    log_success "NCASv3_T4 quota available: ${t4_quota} vCPUs (enough for T4 GPUs)"
    GPU_NODE_SIZE="Standard_NC4as_T4_v3"
    return 0
  else
    log_warning "NCASv3_T4 quota: ${t4_quota} vCPUs (need at least 4 for 1 GPU node)"
    return 1
  fi
}

create_aks() {
  if az aks show -g "${RESOURCE_GROUP}" -n "${AKS_CLUSTER}" >/dev/null 2>&1; then
    log_success "AKS cluster exists"
  else
    log_info "Creating AKS cluster ${AKS_CLUSTER} (system pool)..."
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

  # Check for either 'gpu' or 'gpunodes' nodepool
  if az aks nodepool show -g "${RESOURCE_GROUP}" --cluster-name "${AKS_CLUSTER}" --name gpu >/dev/null 2>&1; then
    log_success "GPU nodepool 'gpu' exists"
  elif az aks nodepool show -g "${RESOURCE_GROUP}" --cluster-name "${AKS_CLUSTER}" --name gpunodes >/dev/null 2>&1; then
    log_success "GPU nodepool 'gpunodes' exists"
  else
    if [[ "${ENABLE_GPU_POOL}" == "true" ]]; then
      if check_gpu_quota; then
        log_info "Adding GPU nodepool gpu (${GPU_NODE_SIZE})"
        set +e
        az aks nodepool add \
          -g "${RESOURCE_GROUP}" --cluster-name "${AKS_CLUSTER}" \
          --name gpu \
          --node-vm-size "${GPU_NODE_SIZE}" \
          --enable-cluster-autoscaler \
          --min-count "${GPU_NODE_MIN}" --max-count "${GPU_NODE_MAX}" \
          --node-taints nvidia.com/gpu=present:NoSchedule \
          --labels pool=gpu -o none
        rc=$?
        set -e
        if [[ $rc -ne 0 ]]; then
          log_warning "GPU nodepool creation failed. Continuing without GPU nodes."
          log_warning "To add GPU nodes later, run: ./add-gpu-node.sh"
          ENABLE_GPU_POOL=false
        else
          log_success "GPU nodepool added (autoscaling: ${GPU_NODE_MIN}-${GPU_NODE_MAX})"
        fi
      else
        log_warning "Insufficient GPU quota. Deploying without GPU nodes."
        log_warning "To request quota: https://portal.azure.com/#view/Microsoft_Azure_Capacity/QuotaMenuBlade"
        log_warning "After quota approval, add GPU nodes with: ./add-gpu-node.sh"
        ENABLE_GPU_POOL=false
      fi
    else
      log_warning "Skipping GPU nodepool creation (ENABLE_GPU_POOL=false)"
    fi
  fi

  log_info "Fetching kubeconfig"
  az aks get-credentials -g "${RESOURCE_GROUP}" -n "${AKS_CLUSTER}" --overwrite-existing -o none
}

ensure_acr() {
  if [[ -z "${ACR_NAME}" ]]; then
    ACR_NAME=$(az acr list -g "${RESOURCE_GROUP}" --query "[0].name" -o tsv 2>/dev/null || echo "")
  fi
  if [[ -z "${ACR_NAME}" ]]; then
    ACR_NAME="tensorfusionacr$(date +%s | tail -c 6)"
    log_info "Creating ACR ${ACR_NAME}"
    az acr create -g "${RESOURCE_GROUP}" -n "${ACR_NAME}" --sku Basic --location "${LOCATION}" --admin-enabled true -o none
  fi
  log_success "Using ACR: ${ACR_NAME}"
  log_info "Attaching ACR to AKS"
  az aks update -g "${RESOURCE_GROUP}" -n "${AKS_CLUSTER}" --attach-acr "${ACR_NAME}" -o none || true
}

create_namespaces() {
  log_info "Creating namespaces"
  kubectl create ns "${NS_TF}" --dry-run=client -o yaml | kubectl apply -f -
  kubectl create ns "${NS_STORAGE}" --dry-run=client -o yaml | kubectl apply -f -
  kubectl create ns "${NS_OBS}" --dry-run=client -o yaml | kubectl apply -f -
  kubectl create ns "${NS_QDRANT}" --dry-run=client -o yaml | kubectl apply -f -
  kubectl create ns "${NS_GREPTIME}" --dry-run=client -o yaml | kubectl apply -f -
  kubectl create ns "${NS_PORTKEY}" --dry-run=client -o yaml | kubectl apply -f -
}

deploy_gpu_operator() {
  log_info "Installing NVIDIA GPU Operator"
  helm repo add nvidia https://helm.ngc.nvidia.com/nvidia --force-update
  helm repo update
  helm upgrade --install gpu-operator nvidia/gpu-operator \
    -n gpu-operator --create-namespace \
    --set driver.enabled=true \
    --wait --timeout 15m
  log_success "GPU Operator deployed"
}

deploy_greptime() {
  log_info "Deploying GreptimeDB"
  cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: greptimedb-data
  namespace: ${NS_GREPTIME}
spec:
  accessModes: [ "ReadWriteOnce" ]
  resources:
    requests:
      storage: 20Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: greptimedb-standalone
  namespace: ${NS_GREPTIME}
spec:
  replicas: 1
  selector: { matchLabels: { app: greptimedb } }
  template:
    metadata: { labels: { app: greptimedb } }
    spec:
      containers:
      - name: greptimedb
        image: greptime/greptimedb:latest
        args: [ "standalone", "start", "--http-addr", "0.0.0.0:4000", "--rpc-addr", "0.0.0.0:4001" ]
        ports:
        - { containerPort: 4000, name: http }
        - { containerPort: 4001, name: grpc }
        volumeMounts:
        - { name: data, mountPath: /tmp/greptimedb }
      volumes:
      - name: data
        persistentVolumeClaim: { claimName: greptimedb-data }
---
apiVersion: v1
kind: Service
metadata:
  name: greptimedb-standalone
  namespace: ${NS_GREPTIME}
spec:
  selector: { app: greptimedb }
  type: ClusterIP
  ports:
  - { port: 4000, targetPort: 4000, name: http }
  - { port: 4001, targetPort: 4001, name: grpc }
EOF
}

deploy_qdrant() {
  log_info "Deploying Qdrant"
  helm repo add qdrant https://qdrant.github.io/qdrant-helm --force-update
  helm repo update
  helm upgrade --install qdrant qdrant/qdrant -n "${NS_QDRANT}" \
    --set persistence.enabled=true --wait
}

deploy_redis() {
  log_info "Deploying Redis"
  helm repo add bitnami https://charts.bitnami.com/bitnami --force-update
  helm repo update
  helm upgrade --install redis bitnami/redis -n "${NS_STORAGE}" \
    --set architecture=replication \
    --set auth.enabled=false \
    --wait
}

deploy_observability() {
  log_info "Deploying Prometheus + Grafana"
  # Check if we can access Helm repos (may fail due to cert issues)
  if helm repo add prometheus-community https://prometheus-community.github.io/helm-charts --force-update 2>/dev/null && \
     helm repo add grafana https://grafana.github.io/helm-charts --force-update 2>/dev/null && \
     helm repo update 2>/dev/null; then
    # Try Helm install
    if ! helm upgrade --install prometheus prometheus-community/prometheus -n "${NS_OBS}" --wait --timeout 5m 2>/dev/null; then
      log_warning "Helm install failed, deploying Prometheus manually"
      deploy_prometheus_manual
    fi
    if ! helm upgrade --install grafana grafana/grafana -n "${NS_OBS}" --set adminPassword=admin --wait --timeout 5m 2>/dev/null; then
      log_warning "Helm install failed, deploying Grafana manually"
      deploy_grafana_manual
    fi
  else
    log_warning "Helm repo access failed, deploying observability manually"
    deploy_prometheus_manual
    deploy_grafana_manual
  fi
}

deploy_prometheus_manual() {
  log_info "Deploying Prometheus manually"
  cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: ${NS_OBS}
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
    scrape_configs:
    - job_name: 'kubernetes-pods'
      kubernetes_sd_configs:
      - role: pod
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus-server
  namespace: ${NS_OBS}
spec:
  replicas: 1
  selector: { matchLabels: { app: prometheus } }
  template:
    metadata: { labels: { app: prometheus } }
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus:latest
        ports: [ { containerPort: 9090 } ]
        args:
        - '--config.file=/etc/prometheus/prometheus.yml'
        - '--storage.tsdb.path=/prometheus'
        volumeMounts:
        - name: config
          mountPath: /etc/prometheus
        - name: storage
          mountPath: /prometheus
      volumes:
      - name: config
        configMap: { name: prometheus-config }
      - name: storage
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: prometheus-server
  namespace: ${NS_OBS}
spec:
  selector: { app: prometheus }
  ports: [ { port: 80, targetPort: 9090 } ]
EOF
}

deploy_grafana_manual() {
  log_info "Deploying Grafana manually"
  cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: ${NS_OBS}
spec:
  replicas: 1
  selector: { matchLabels: { app: grafana } }
  template:
    metadata: { labels: { app: grafana } }
    spec:
      containers:
      - name: grafana
        image: grafana/grafana:latest
        ports: [ { containerPort: 3000 } ]
        env:
        - { name: GF_SECURITY_ADMIN_PASSWORD, value: admin }
---
apiVersion: v1
kind: Service
metadata:
  name: grafana
  namespace: ${NS_OBS}
spec:
  selector: { app: grafana }
  ports: [ { port: 80, targetPort: 3000 } ]
EOF
}

deploy_portkey() {
  log_info "Deploying Portkey Gateway"
  cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: portkey-gateway
  namespace: ${NS_PORTKEY}
spec:
  replicas: 1
  selector: { matchLabels: { app: portkey-gateway } }
  template:
    metadata: { labels: { app: portkey-gateway } }
    spec:
      containers:
      - name: portkey
        image: portkeyai/gateway:latest
        ports: [ { containerPort: 8787, name: http } ]
        env:
        - { name: LOG_LEVEL, value: "info" }
        - name: PORTKEY_API_KEY
          valueFrom:
            secretKeyRef:
              name: portkey-credentials
              key: api-key
---
apiVersion: v1
kind: Service
metadata:
  name: portkey-gateway
  namespace: ${NS_PORTKEY}
spec:
  selector: { app: portkey-gateway }
  type: ClusterIP
  ports: [ { port: 8787, targetPort: 8787, name: http } ]
EOF
}

wait_ready() {
  log_info "Waiting for core services to be ready"
  kubectl wait --for=condition=ready pod -l app=greptimedb -n "${NS_GREPTIME}" --timeout=300s || true
  kubectl wait --for=condition=ready pod -l app=qdrant -n "${NS_QDRANT}" --timeout=300s || true
  kubectl wait --for=condition=ready pod -l app=portkey-gateway -n "${NS_PORTKEY}" --timeout=300s || true
}

create_secrets() {
  log_info "Creating placeholder secrets"
  kubectl create secret generic portkey-credentials -n "${NS_PORTKEY}" \
    --from-literal=api-key="REPLACE_WITH_PORTKEY_API_KEY" \
    --dry-run=client -o yaml | kubectl apply -f -
  kubectl create secret generic foundry-keys -n "${NS_TF}" \
    --from-literal=api-key="REPLACE_WITH_FOUNDRY_API_KEY" \
    --from-literal=endpoint="REPLACE_WITH_FOUNDRY_ENDPOINT" \
    --dry-run=client -o yaml | kubectl apply -f -
}

validate_go_modules() {
  log_info "Validating Go modules before build..."
  if ! go mod verify >/dev/null 2>&1; then
    log_warning "Go modules verification failed, attempting to tidy..."
    go mod tidy
  fi
  
  # Check if all imports can be resolved
  if ! go list ./... >/dev/null 2>&1; then
    log_error "Go module dependencies cannot be resolved. Run 'go mod tidy' to fix."
    return 1
  fi
  log_success "Go modules validated"
  return 0
}

build_and_push_images() {
  if [[ "${BUILD_IMAGES}" != "true" ]]; then
    log_warning "Skipping image build (BUILD_IMAGES=false)"
    return 0
  fi
  
  # Validate Go modules before attempting to build
  if ! validate_go_modules; then
    log_error "Go module validation failed. Image builds will be skipped."
    return 1
  fi
  
  log_info "Building images in ACR"
  set +e
  
  log_info "Building operator image..."
  az acr build --registry "${ACR_NAME}" \
    --image tensor-fusion/operator:latest \
    --file dockerfile/operator.Dockerfile \
    --platform linux/amd64 \
    . 2>&1 | tee /tmp/operator-build.log
  rc1=$?
  
  if [[ $rc1 -ne 0 ]]; then
    log_error "Operator image build failed. Check /tmp/operator-build.log for details."
  else
    log_success "Operator image built successfully"
  fi
  
  log_info "Building node-discovery image (AMD64 for AKS GPU nodes)..."
  az acr build --registry "${ACR_NAME}" \
    --image tensor-fusion/node-discovery:latest \
    --file dockerfile/node-discovery.Dockerfile \
    --platform linux/amd64 \
    . 2>&1 | tee /tmp/node-discovery-build.log
  rc2=$?
  
  if [[ $rc2 -ne 0 ]]; then
    log_error "Node-discovery image build failed. Check /tmp/node-discovery-build.log for details."
  else
    log_success "Node-discovery image built successfully"
  fi
  
  log_info "Building mcp-server image..."
  az acr build --registry "${ACR_NAME}" \
    --image mcp-server:1.0.0 \
    --file dockerfile/mcp-server.Dockerfile \
    --platform linux/amd64 \
    . 2>&1 | tee /tmp/mcp-server-build.log
  rc3=$?
  
  if [[ $rc3 -ne 0 ]]; then
    log_error "MCP server image build failed. Check /tmp/mcp-server-build.log for details."
  else
    log_success "MCP server image built successfully"
  fi
  
  log_info "Building orchestrator image..."
  az acr build --registry "${ACR_NAME}" \
    --image orchestrator:1.0.0 \
    --file dockerfile/orchestrator.Dockerfile \
    --platform linux/amd64 \
    . 2>&1 | tee /tmp/orchestrator-build.log
  rc4=$?
  
  if [[ $rc4 -ne 0 ]]; then
    log_error "Orchestrator image build failed. Check /tmp/orchestrator-build.log for details."
  else
    log_success "Orchestrator image built successfully"
  fi
  
  log_info "Building deployment-agent image..."
  az acr build --registry "${ACR_NAME}" \
    --image deployment-agent:1.0.0 \
    --file dockerfile/deployment-agent.Dockerfile \
    --platform linux/amd64 \
    . 2>&1 | tee /tmp/deployment-agent-build.log
  rc5=$?
  
  if [[ $rc5 -ne 0 ]]; then
    log_error "Deployment agent image build failed. Check /tmp/deployment-agent-build.log for details."
  else
    log_success "Deployment agent image built successfully"
  fi
  
  log_info "Building training-agent image..."
  az acr build --registry "${ACR_NAME}" \
    --image training-agent:1.0.0 \
    --file dockerfile/training-agent.Dockerfile \
    --platform linux/amd64 \
    . 2>&1 | tee /tmp/training-agent-build.log
  rc6=$?
  
  if [[ $rc6 -ne 0 ]]; then
    log_error "Training agent image build failed. Check /tmp/training-agent-build.log for details."
  else
    log_success "Training agent image built successfully"
  fi
  
  log_info "Building cost-agent image..."
  az acr build --registry "${ACR_NAME}" \
    --image cost-agent:1.0.0 \
    --file dockerfile/cost-agent.Dockerfile \
    --platform linux/amd64 \
    . 2>&1 | tee /tmp/cost-agent-build.log
  rc7=$?
  
  if [[ $rc7 -ne 0 ]]; then
    log_error "Cost agent image build failed. Check /tmp/cost-agent-build.log for details."
  else
    log_success "Cost agent image built successfully"
  fi
  
  log_info "Building memory-service image..."
  az acr build --registry "${ACR_NAME}" \
    --image memory-service:1.0.0 \
    --file dockerfile/memory-service.Dockerfile \
    --platform linux/amd64 \
    . 2>&1 | tee /tmp/memory-service-build.log
  rc8=$?
  
  if [[ $rc8 -ne 0 ]]; then
    log_error "Memory service image build failed. Check /tmp/memory-service-build.log for details."
  else
    log_success "Memory service image built successfully"
  fi
  
  log_info "Building model-catalog image..."
  az acr build --registry "${ACR_NAME}" \
    --image model-catalog:1.0.0 \
    --file dockerfile/model-catalog.Dockerfile \
    --platform linux/amd64 \
    . 2>&1 | tee /tmp/model-catalog-build.log
  rc9=$?
  
  if [[ $rc9 -ne 0 ]]; then
    log_error "Model catalog image build failed. Check /tmp/model-catalog-build.log for details."
  else
    log_success "Model catalog image built successfully"
  fi
  
  log_info "Building discovery-agent image..."
  az acr build --registry "${ACR_NAME}" \
    --image discovery-agent:1.0.0 \
    --file dockerfile/discovery-agent.Dockerfile \
    --platform linux/amd64 \
    . 2>&1 | tee /tmp/discovery-agent-build.log
  rc10=$?
  
  if [[ $rc10 -ne 0 ]]; then
    log_error "Discovery agent image build failed. Check /tmp/discovery-agent-build.log for details."
  else
    log_success "Discovery agent image built successfully"
  fi
  
  set -e
  if [[ $rc1 -ne 0 || $rc2 -ne 0 || $rc3 -ne 0 || $rc4 -ne 0 || $rc5 -ne 0 || $rc6 -ne 0 || $rc7 -ne 0 || $rc8 -ne 0 || $rc9 -ne 0 || $rc10 -ne 0 ]]; then
    log_warning "One or more image builds failed; will use chart default images."
    return 1
  fi
  
  log_success "All 10 images built and pushed successfully"
  return 0
}

verify_crds_installed() {
  log_info "Verifying CRDs are installed..."
  local crds_missing=false
  
  for crd in gpupools gpunodes gpus tensorfusionclusters workloadintelligences; do
    if ! kubectl get crd ${crd}.tensor-fusion.ai >/dev/null 2>&1; then
      log_warning "CRD ${crd}.tensor-fusion.ai not found"
      crds_missing=true
    fi
  done
  
  if [[ "${crds_missing}" == "true" ]]; then
    log_info "Installing CRDs from kustomize..."
    kubectl apply -k config/crd || {
      log_error "Failed to install CRDs"
      return 1
    }
    sleep 5
    log_success "CRDs installed"
  else
    log_success "All required CRDs are present"
  fi
  return 0
}

deploy_tensorfusion() {
  log_info "Deploying TensorFusion via Helm"
  
  # Verify CRDs are installed first
  verify_crds_installed || {
    log_error "CRD verification failed. Cannot proceed with Helm deployment."
    return 1
  }
  
  local ACR_LOGIN_SERVER="${ACR_NAME}.azurecr.io"
  # Always configure service endpoints; optionally override images when built
  cat > /tmp/tf-values.yaml <<EOF
nodeDiscovery:
  enabled: true

greptime:
  installStandalone: false
  host: greptimedb-standalone.${NS_GREPTIME}.svc.cluster.local
  port: 4000

redis:
  host: redis-master.${NS_STORAGE}.svc.cluster.local
  port: 6379

qdrant:
  host: qdrant.${NS_QDRANT}.svc.cluster.local
  port: 6333

portkey:
  host: portkey-gateway.${NS_PORTKEY}.svc.cluster.local
  port: 8787
EOF

  # Only override images if they were successfully built
  if [[ -f /tmp/.images-built-successfully ]]; then
    log_info "Using custom-built images from ACR"
    cat >> /tmp/tf-values.yaml <<EOF
image:
  repository: ${ACR_LOGIN_SERVER}/tensor-fusion/operator
  tag: latest
  pullPolicy: Always

nodeDiscovery:
  image:
    repository: ${ACR_LOGIN_SERVER}/tensor-fusion/node-discovery
    tag: latest
    pullPolicy: Always

mcpServer:
  image:
    repository: ${ACR_LOGIN_SERVER}/mcp-server
    tag: 1.0.0
    pullPolicy: Always

orchestrator:
  image:
    repository: ${ACR_LOGIN_SERVER}/orchestrator
    tag: 1.0.0
    pullPolicy: Always

agents:
  deploymentAgent:
    image:
      repository: ${ACR_LOGIN_SERVER}/deployment-agent
      tag: 1.0.0
      pullPolicy: Always
  trainingAgent:
    image:
      repository: ${ACR_LOGIN_SERVER}/training-agent
      tag: 1.0.0
      pullPolicy: Always
  costAgent:
    image:
      repository: ${ACR_LOGIN_SERVER}/cost-agent
      tag: 1.0.0
      pullPolicy: Always

memoryService:
  image:
    repository: ${ACR_LOGIN_SERVER}/memory-service
    tag: 1.0.0
    pullPolicy: Always

modelCatalog:
  image:
    repository: ${ACR_LOGIN_SERVER}/model-catalog
    tag: 1.0.0
    pullPolicy: Always

discoveryAgent:
  image:
    repository: ${ACR_LOGIN_SERVER}/discovery-agent
    tag: 1.0.0
    pullPolicy: Always
EOF
  else
    log_warning "Using default images from Helm chart (custom images not built)"
  fi

  log_info "Helm values file:"
  cat /tmp/tf-values.yaml
  
  # Deploy with Helm, with retry logic
  local max_attempts=2
  local attempt=1
  
  while [[ $attempt -le $max_attempts ]]; do
    log_info "Helm deployment attempt ${attempt}/${max_attempts}..."
    
    if helm upgrade --install tensor-fusion ./charts/tensor-fusion \
      -n "${NS_TF}" --create-namespace \
      --values /tmp/tf-values.yaml \
      --wait --timeout 15m 2>&1 | tee /tmp/helm-deploy.log; then
      log_success "TensorFusion deployed successfully"
      
      # Wait a bit and verify key components
      sleep 10
      log_info "Verifying deployment..."
      kubectl wait --for=condition=ready pod \
        -l app.kubernetes.io/name=tensor-fusion \
        -n "${NS_TF}" --timeout=300s || {
        log_warning "Some pods may not be ready yet, but continuing..."
      }
      
      return 0
    else
      log_warning "Helm deployment attempt ${attempt} failed"
      if [[ $attempt -lt $max_attempts ]]; then
        log_info "Retrying in 30 seconds..."
        sleep 30
      fi
      ((attempt++))
    fi
  done
  
  log_error "Helm deployment failed after ${max_attempts} attempts. Check /tmp/helm-deploy.log"
  log_info "Checking pod status for debugging:"
  kubectl get pods -n "${NS_TF}" || true
  return 1
}

deploy_sample_crds() {
  log_info "Deploying sample CRDs for testing"
  
  # Deploy GPU Pool
  kubectl apply -f examples/01-gpu-pool.yaml || log_warning "GPU Pool creation failed"
  
  # Deploy GPU Quota
  kubectl apply -f examples/03-gpu-quota.yaml || log_warning "GPU Quota creation failed"
  
  # Deploy LLM Routes
  kubectl apply -f examples/03-llm-route.yaml || log_warning "LLM Route creation failed"
  
  # Deploy Workload Intelligence
  kubectl apply -f examples/05-workload-intelligence.yaml || log_warning "Workload Intelligence creation failed"
  
  sleep 3
  log_success "Sample CRDs deployed"
}

verify_gpu_nodes() {
  log_info "Verifying GPU nodes..."
  
  local gpu_nodes=$(kubectl get nodes -l pool=gpu --no-headers 2>/dev/null | wc -l | tr -d ' ')
  
  if [ "$gpu_nodes" -gt 0 ]; then
    log_success "GPU nodes available: $gpu_nodes"
    
    # Check if GPUs are detected
    local gpu_count=0
    for node in $(kubectl get nodes -l pool=gpu -o name 2>/dev/null); do
      local node_name=$(echo "$node" | cut -d'/' -f2)
      local gpus=$(kubectl get node "$node_name" -o jsonpath='{.status.capacity.nvidia\.com/gpu}' 2>/dev/null || echo "0")
      if [ "$gpus" != "0" ] && [ -n "$gpus" ]; then
        log_success "Node $node_name: $gpus GPU(s) detected"
        ((gpu_count+=gpus))
      else
        log_warning "Node $node_name: GPUs not yet detected (driver may still be installing)"
      fi
    done
    
    if [ "$gpu_count" -gt 0 ]; then
      log_success "Total GPUs available in cluster: $gpu_count"
      # Bootstrap GPUNode resources for node-discovery
      bootstrap_gpu_nodes
    else
      log_warning "GPU nodes exist but GPUs not yet detected. GPU operator may still be installing drivers."
      log_info "Check GPU operator status: kubectl get pods -n gpu-operator"
    fi
  else
    log_warning "No GPU nodes found (autoscaler may have scaled to 0, or quota was insufficient)"
    log_info "If you have GPU quota, you can add nodes with: ./add-gpu-node.sh"
  fi
}

bootstrap_gpu_nodes() {
  log_info "Bootstrapping GPUNode resources for node-discovery..."
  
  # Get default-pool UID
  local pool_uid=$(kubectl get gpupool default-pool -o jsonpath='{.metadata.uid}' 2>/dev/null)
  if [ -z "$pool_uid" ]; then
    log_warning "GPUPool default-pool not found, skipping GPUNode bootstrap"
    return
  fi
  
  # Create GPUNode for each GPU node
  for node in $(kubectl get nodes -l pool=gpu -o name 2>/dev/null); do
    local node_name=$(echo "$node" | cut -d'/' -f2)
    
    # Check if GPUNode already exists
    if kubectl get gpunode "$node_name" >/dev/null 2>&1; then
      log_info "GPUNode $node_name already exists"
      continue
    fi
    
    log_info "Creating GPUNode resource for $node_name..."
    cat <<EOF | kubectl apply -f - >/dev/null 2>&1
apiVersion: tensor-fusion.ai/v1
kind: GPUNode
metadata:
  name: ${node_name}
  labels:
    tensor-fusion.ai/pool: default-pool
  ownerReferences:
  - apiVersion: tensor-fusion.ai/v1
    kind: GPUPool
    name: default-pool
    uid: ${pool_uid}
    controller: false
    blockOwnerDeletion: false
status:
  phase: Pending
  totalTFlops: "0"
  totalVRAM: "0"
  totalGPUs: 0
  managedGPUs: 0
EOF
    
    if [ $? -eq 0 ]; then
      log_success "GPUNode $node_name created"
      
      # Restart node-discovery pod to trigger GPU detection
      kubectl delete pod -n tensor-fusion-sys -l app=node-discovery --grace-period=0 --force >/dev/null 2>&1 || true
      log_info "Node-discovery pod restarted to detect GPUs"
    else
      log_warning "Failed to create GPUNode $node_name"
    fi
  done
}

print_summary() {
  echo ""
  echo "╔════════════════════════════════════════════════════════════════╗"
  echo "║   Deployment Complete                                          ║"
  echo "╚════════════════════════════════════════════════════════════════╝"
  echo "Subscription: ${AZ_SUBSCRIPTION_ID}"
  echo "Resource Group: ${RESOURCE_GROUP}"
  echo "AKS: ${AKS_CLUSTER}  ACR: ${ACR_NAME}"
  echo "Namespaces: ${NS_TF}, ${NS_STORAGE}, ${NS_OBS}, ${NS_QDRANT}, ${NS_GREPTIME}, ${NS_PORTKEY}"
  echo ""
  log_info "Cluster nodes:"
  kubectl get nodes -o wide || true
  echo ""
  verify_gpu_nodes
  echo ""
  log_info "TensorFusion pods:"; kubectl get pods -n "${NS_TF}" || true
  log_info "Observability pods:"; kubectl get pods -n "${NS_OBS}" || true
  log_info "Greptime/Qdrant/Portkey:"; \
    kubectl get pods -n "${NS_GREPTIME}" || true; \
    kubectl get pods -n "${NS_QDRANT}" || true; \
    kubectl get pods -n "${NS_PORTKEY}" || true
  echo ""
  log_info "Custom Resources:"
  kubectl get gpupool -A || true
  kubectl get gpuresourcequota -A || true
  kubectl get llmroute -A || true
  kubectl get workloadintelligence -A || true
  echo ""
  log_success "Run 'bash scripts/verify-all.sh' to verify the deployment"
}

main() {
  # Clean up any previous run artifacts
  rm -f /tmp/.images-built-successfully /tmp/operator-build.log /tmp/node-discovery-build.log /tmp/helm-deploy.log
  
  check_prereqs
  set_subscription
  create_rg
  create_aks
  ensure_acr
  create_namespaces
  create_secrets
  deploy_gpu_operator
  deploy_greptime
  deploy_qdrant
  deploy_redis
  deploy_observability
  deploy_portkey
  wait_ready
  
  # Build and push images
  if build_and_push_images; then
    touch /tmp/.images-built-successfully
    log_success "Custom images will be used in deployment"
  else
    log_warning "Custom image build failed or skipped - using default images"
    BUILD_IMAGES=false
  fi
  
  # Deploy TensorFusion
  if ! deploy_tensorfusion; then
    log_error "TensorFusion deployment failed!"
    log_info "You can try to fix issues and re-run: helm upgrade --install tensor-fusion ./charts/tensor-fusion -n ${NS_TF} --values /tmp/tf-values.yaml"
    log_info "Or check the logs with: kubectl logs -n ${NS_TF} -l app.kubernetes.io/name=tensor-fusion"
    exit 1
  fi
  
  # Optional: apply enhanced RBAC if provided
  if [[ -f "rbac-enhanced.yaml" ]]; then
    log_info "Applying enhanced RBAC"
    kubectl apply -f rbac-enhanced.yaml || log_warning "Enhanced RBAC failed to apply"
  fi
  
  # Deploy sample CRDs for testing
  deploy_sample_crds
  
  print_summary
  
  log_info "Deployment completed! Check status with: kubectl get pods -A"
}

main "$@"


