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

  if az aks nodepool show -g "${RESOURCE_GROUP}" --cluster-name "${AKS_CLUSTER}" --name gpunodes >/dev/null 2>&1; then
    log_success "GPU nodepool exists"
  else
    if [[ "${ENABLE_GPU_POOL}" == "true" ]]; then
      log_info "Adding GPU nodepool gpunodes (${GPU_NODE_SIZE})"
      set +e
      az aks nodepool add \
        -g "${RESOURCE_GROUP}" --cluster-name "${AKS_CLUSTER}" \
        --name gpunodes \
        --node-vm-size "${GPU_NODE_SIZE}" \
        --enable-cluster-autoscaler \
        --min-count "${GPU_NODE_MIN}" --max-count "${GPU_NODE_MAX}" \
        --node-taints nvidia.com/gpu=present:NoSchedule \
        --labels pool=gpu -o none
      rc=$?
      set -e
      if [[ $rc -ne 0 ]]; then
        log_warning "GPU nodepool could not be created (likely quota). Skipping and continuing CPU-only deployment."
      else
        log_success "GPU nodepool added"
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

build_and_push_images() {
  if [[ "${BUILD_IMAGES}" != "true" ]]; then
    log_warning "Skipping image build (BUILD_IMAGES=false)"
    return 0
  fi
  log_info "Building images in ACR"
  set +e
  az acr build --registry "${ACR_NAME}" --image tensor-fusion/operator:latest --file dockerfile/operator.Dockerfile .
  rc1=$?
  az acr build --registry "${ACR_NAME}" --image tensor-fusion/node-discovery:latest --file dockerfile/node-discovery.Dockerfile .
  rc2=$?
  set -e
  if [[ $rc1 -ne 0 || $rc2 -ne 0 ]]; then
    log_warning "Image build failed; will use chart default images."
    return 1
  fi
}

deploy_tensorfusion() {
  log_info "Deploying TensorFusion via Helm"
  local ACR_LOGIN_SERVER="${ACR_NAME}.azurecr.io"
  # Always configure service endpoints; optionally override images when built
  cat > /tmp/tf-values.yaml <<EOF
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

  if [[ "${BUILD_IMAGES}" == "true" ]]; then
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
EOF
  fi

  helm upgrade --install tensor-fusion ./charts/tensor-fusion \
    -n "${NS_TF}" --create-namespace \
    --values /tmp/tf-values.yaml \
    --wait --timeout 15m
  log_success "TensorFusion deployed"
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
  if ! build_and_push_images; then
    BUILD_IMAGES=false
  fi
  deploy_tensorfusion
  # Optional: apply enhanced RBAC if provided
  if [[ -f "rbac-enhanced.yaml" ]]; then
    log_info "Applying enhanced RBAC"
    kubectl apply -f rbac-enhanced.yaml || true
  fi
  # Deploy sample CRDs for testing
  deploy_sample_crds
  print_summary
}

main "$@"


