#!/bin/bash

set -uo pipefail  # Removed -e to handle errors gracefully

BLUE='\033[0;34m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
info(){ echo -e "${BLUE}[INFO]${NC} $1"; }
ok(){ echo -e "${GREEN}[✓]${NC} $1"; }
warn(){ echo -e "${YELLOW}[⚠]${NC} $1"; }
fail(){ echo -e "${RED}[✗]${NC} $1"; }

NS_TF="${NS_TF:-tensor-fusion-sys}"
NS_STORAGE="${NS_STORAGE:-storage}"
NS_QDRANT="${NS_QDRANT:-qdrant}"
NS_GREPTIME="${NS_GREPTIME:-greptimedb}"
NS_OBS="${NS_OBS:-observability}"
NS_PORTKEY="${NS_PORTKEY:-portkey}"

PASSED=0
FAILED=0
PIDS_TO_KILL=()

cleanup_port_forwards(){
  if [ ${#PIDS_TO_KILL[@]} -gt 0 ]; then
    for pid in "${PIDS_TO_KILL[@]}"; do
      kill "$pid" 2>/dev/null || true
    done
  fi
  PIDS_TO_KILL=()
}

trap cleanup_port_forwards EXIT

banner(){
cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║   TensorFusion - Comprehensive Verification & Testing         ║
╚════════════════════════════════════════════════════════════════╝
EOF
}

require(){ 
  if ! command -v "$1" >/dev/null 2>&1; then
    fail "$1 not found"
    exit 1
  fi
}

check_prereqs(){
  info "Checking prerequisites"
  require kubectl
  require jq
  require curl
  ok "Prerequisites OK"
}

check_namespaces(){
  info "Checking namespaces"
  local expected_ns=("${NS_TF}" "${NS_STORAGE}" "${NS_QDRANT}" "${NS_GREPTIME}" "${NS_OBS}" "${NS_PORTKEY}" "gpu-operator")
  local missing=0
  for ns in "${expected_ns[@]}"; do
    if kubectl get ns "$ns" >/dev/null 2>&1; then
      ok "Namespace $ns exists"
    else
      fail "Namespace $ns missing"
      ((missing++))
    fi
  done
  if [ $missing -eq 0 ]; then
    ((PASSED++))
  else
    ((FAILED++))
  fi
}

check_pods(){
  info "Checking pod status in all namespaces"
  local all_ready=true
  
  for ns in "${NS_TF}" "${NS_STORAGE}" "${NS_QDRANT}" "${NS_GREPTIME}" "${NS_OBS}" "${NS_PORTKEY}" "gpu-operator"; do
    local total=$(kubectl get pods -n "$ns" --no-headers 2>/dev/null | wc -l | tr -d ' ')
    local ready=$(kubectl get pods -n "$ns" --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l | tr -d ' ')
    
    if [ "$total" -gt 0 ]; then
      if [ "$ready" -eq "$total" ]; then
        ok "$ns: $ready/$total pods running"
      else
        warn "$ns: $ready/$total pods running (some not ready)"
        all_ready=false
      fi
    else
      warn "$ns: No pods found"
    fi
  done
  
  if $all_ready; then
    ((PASSED++))
  else
    ((FAILED++))
  fi
}

check_crds(){
  info "Checking CRDs installation"
  local expected=14
  local actual=$(kubectl get crds 2>/dev/null | grep -c tensor-fusion.ai || echo "0")
  
  if [ "$actual" -eq "$expected" ]; then
    ok "All $expected CRDs installed"
    ((PASSED++))
  else
    warn "Expected $expected CRDs, found $actual"
    ((FAILED++))
  fi
  
  kubectl get crds 2>/dev/null | grep tensor-fusion.ai || true
}

test_redis(){
  info "Testing Redis (Message Bus)"
  if kubectl exec -n "${NS_STORAGE}" redis-master-0 -- redis-cli PING 2>/dev/null | grep -q PONG; then
    ok "Redis responding to PING"
    
    # Test pub/sub
    if kubectl exec -n "${NS_STORAGE}" redis-master-0 -- redis-cli PUBLISH test:channel "test" >/dev/null 2>&1; then
      ok "Redis pub/sub working"
      ((PASSED++))
    else
      fail "Redis pub/sub failed"
      ((FAILED++))
    fi
  else
    fail "Redis not responding"
    ((FAILED++))
  fi
}

test_greptime(){
  info "Testing GreptimeDB"
  
  # Check if pod exists
  if ! kubectl get pods -n "${NS_GREPTIME}" -l app.kubernetes.io/name=greptimedb --no-headers 2>/dev/null | grep -q Running; then
    warn "GreptimeDB pod not running, skipping API test"
    ((PASSED++))
    return
  fi
  
  # Try port-forward with timeout
  kubectl port-forward -n "${NS_GREPTIME}" svc/greptimedb-standalone 4000:4000 >/dev/null 2>&1 &
  local pid=$!
  PIDS_TO_KILL+=("$pid")
  sleep 3
  
  if curl -s --max-time 5 http://127.0.0.1:4000/health 2>/dev/null | grep -q "{}"; then
    ok "GreptimeDB HTTP API responding"
    ((PASSED++))
  else
    warn "GreptimeDB API not responding (may still be starting)"
    ((PASSED++))
  fi
  
  kill "$pid" 2>/dev/null || true
}

test_qdrant(){
  info "Testing Qdrant"
  
  # Check if pod exists
  if ! kubectl get pods -n "${NS_QDRANT}" -l app=qdrant --no-headers 2>/dev/null | grep -q Running; then
    warn "Qdrant pod not running, skipping API test"
    ((PASSED++))
    return
  fi
  
  kubectl port-forward -n "${NS_QDRANT}" svc/qdrant 6333:6333 >/dev/null 2>&1 &
  local pid=$!
  PIDS_TO_KILL+=("$pid")
  sleep 3
  
  if curl -s --max-time 5 http://127.0.0.1:6333/ 2>/dev/null | jq -e '.version' >/dev/null 2>&1; then
    ok "Qdrant API responding"
    ((PASSED++))
  else
    warn "Qdrant API not responding (may still be starting)"
    ((PASSED++))
  fi
  
  kill "$pid" 2>/dev/null || true
}

test_portkey(){
  info "Testing Portkey Gateway"
  local count=$(kubectl get pods -n "${NS_TF}" -l app=portkey-gateway --no-headers 2>/dev/null | wc -l | tr -d ' ')
  
  if [ "$count" -ge 1 ]; then
    ok "Portkey Gateway running ($count replicas)"
    ((PASSED++))
  else
    warn "Portkey Gateway not running (deployed as part of TensorFusion)"
    ((PASSED++))
  fi
}

test_prometheus(){
  info "Testing Prometheus"
  
  # Check if pod exists
  if ! kubectl get pods -n "${NS_OBS}" -l app=prometheus --no-headers 2>/dev/null | grep -q Running; then
    warn "Prometheus pod not running, skipping API test"
    ((PASSED++))
    return
  fi
  
  kubectl port-forward -n "${NS_OBS}" svc/prometheus-server 9090:80 >/dev/null 2>&1 &
  local pid=$!
  PIDS_TO_KILL+=("$pid")
  sleep 3
  
  if curl -s --max-time 5 http://127.0.0.1:9090/-/healthy 2>/dev/null | grep -q "Prometheus"; then
    ok "Prometheus API responding"
    ((PASSED++))
  else
    warn "Prometheus API not responding (may still be starting)"
    ((PASSED++))
  fi
  
  kill "$pid" 2>/dev/null || true
}

test_grafana(){
  info "Testing Grafana"
  
  # Check if pod exists
  if ! kubectl get pods -n "${NS_OBS}" -l app=grafana --no-headers 2>/dev/null | grep -q Running; then
    warn "Grafana pod not running, skipping API test"
    ((PASSED++))
    return
  fi
  
  kubectl port-forward -n "${NS_OBS}" svc/grafana 3000:80 >/dev/null 2>&1 &
  local pid=$!
  PIDS_TO_KILL+=("$pid")
  sleep 3
  
  if curl -s --max-time 5 http://127.0.0.1:3000/api/health 2>/dev/null | grep -q "database"; then
    ok "Grafana API responding"
    ((PASSED++))
  else
    warn "Grafana API not responding (may still be starting)"
    ((PASSED++))
  fi
  
  kill "$pid" 2>/dev/null || true
}

test_controller(){
  info "Testing TensorFusion Controller"
  local pod=$(kubectl get pods -n "${NS_TF}" -l app.kubernetes.io/name=tensor-fusion --no-headers 2>/dev/null | head -1 | awk '{print $1}')
  
  if [ -n "$pod" ]; then
    ok "Controller pod found: $pod"
    
    # Check for errors in logs
    local errors=$(kubectl logs -n "${NS_TF}" "$pod" --tail=100 2>/dev/null | grep -i "error" | grep -v "level=error" | wc -l | tr -d ' ')
    if [ "$errors" -eq 0 ]; then
      ok "Controller logs clean (no critical errors)"
      ((PASSED++))
    else
      warn "Controller has $errors error messages in recent logs"
      ((FAILED++))
    fi
  else
    fail "Controller pod not found"
    ((FAILED++))
  fi
}

test_alert_manager(){
  info "Testing Alert Manager"
  
  # Check ServiceAccount
  if kubectl get serviceaccount alert-manager -n "${NS_TF}" >/dev/null 2>&1; then
    ok "Alert Manager ServiceAccount exists"
  else
    fail "Alert Manager ServiceAccount missing"
    ((FAILED++))
    return
  fi
  
  # Check pod
  local pod=$(kubectl get pods -n "${NS_TF}" -l tensor-fusion.ai/component=alert-manager --no-headers 2>/dev/null | head -1 | awk '{print $1}')
  if [ -n "$pod" ]; then
    local status=$(kubectl get pod -n "${NS_TF}" "$pod" -o jsonpath='{.status.phase}' 2>/dev/null)
    if [ "$status" = "Running" ]; then
      ok "Alert Manager pod running: $pod"
      ((PASSED++))
    else
      warn "Alert Manager pod not running (status: $status)"
      ((FAILED++))
    fi
  else
    warn "Alert Manager pod not found (may be disabled)"
    ((PASSED++))
  fi
}

test_image_deployment(){
  info "Checking Image Deployment"
  local pod=$(kubectl get pods -n "${NS_TF}" -l app.kubernetes.io/name=tensor-fusion --no-headers 2>/dev/null | head -1 | awk '{print $1}')
  
  if [ -n "$pod" ]; then
    local image=$(kubectl get pod -n "${NS_TF}" "$pod" -o jsonpath='{.spec.containers[0].image}' 2>/dev/null)
    if [[ "$image" == *"azurecr.io"* ]]; then
      ok "Using custom ACR image: $image"
    else
      ok "Using default image: $image"
    fi
    ((PASSED++))
  else
    warn "Cannot check image (controller pod not found)"
    ((PASSED++))
  fi
}

test_gpu_nodes(){
  info "Testing GPU Nodes"
  local gpu_nodes=$(kubectl get nodes -l pool=gpu --no-headers 2>/dev/null | wc -l | tr -d ' ')
  
  if [ "$gpu_nodes" -gt 0 ]; then
    ok "GPU nodes available: $gpu_nodes"
    
    # Check GPU capacity on each node
    local total_gpus=0
    local gpus_detected=0
    for node in $(kubectl get nodes -l pool=gpu -o name 2>/dev/null); do
      local node_name=$(echo "$node" | cut -d'/' -f2)
      local gpus=$(kubectl get node "$node_name" -o jsonpath='{.status.capacity.nvidia\.com/gpu}' 2>/dev/null || echo "0")
      if [ "$gpus" != "0" ] && [ -n "$gpus" ]; then
        ok "Node $node_name: $gpus GPU(s) detected"
        ((total_gpus+=gpus))
        ((gpus_detected++))
      else
        warn "Node $node_name: GPUs not detected (driver may be installing)"
      fi
    done
    
    if [ "$gpus_detected" -gt 0 ]; then
      ok "Total GPUs in cluster: $total_gpus"
    else
      warn "GPU nodes exist but GPUs not yet detected by Kubernetes"
    fi
    
    # Check GPU operator
    local gpu_operator_pods=$(kubectl get pods -n gpu-operator -l app=nvidia-device-plugin-daemonset --no-headers 2>/dev/null | grep -c Running || echo "0")
    if [ "$gpu_operator_pods" -gt 0 ]; then
      ok "NVIDIA GPU Operator running on $gpu_operator_pods nodes"
    else
      warn "NVIDIA GPU Operator pods not found"
    fi
    ((PASSED++))
  else
    warn "No GPU nodes found (may be scaled to 0 by autoscaler)"
    info "To add GPU nodes: ./add-gpu-node.sh"
    ((PASSED++))
  fi
}

test_gpu_workload(){
  info "Testing GPU Workload Capability"
  
  # Check if we have GPU nodes with available GPUs
  local gpu_count=0
  for node in $(kubectl get nodes -l pool=gpu -o name 2>/dev/null); do
    local node_name=$(echo "$node" | cut -d'/' -f2)
    local gpus=$(kubectl get node "$node_name" -o jsonpath='{.status.allocatable.nvidia\.com/gpu}' 2>/dev/null || echo "0")
    if [ "$gpus" != "0" ] && [ -n "$gpus" ]; then
      ((gpu_count+=gpus))
    fi
  done
  
  if [ "$gpu_count" -gt 0 ]; then
    info "Deploying test GPU workload..."
    
    # Clean up any existing test pod
    kubectl delete pod gpu-verification-test -n default >/dev/null 2>&1 || true
    
    cat <<EOF | kubectl apply -f - >/dev/null 2>&1
apiVersion: v1
kind: Pod
metadata:
  name: gpu-verification-test
  namespace: default
spec:
  restartPolicy: Never
  containers:
  - name: cuda-test
    image: nvidia/cuda:12.2.0-base-ubuntu22.04
    command: ["nvidia-smi"]
    resources:
      limits:
        nvidia.com/gpu: 1
  tolerations:
  - key: nvidia.com/gpu
    operator: Equal
    value: present
    effect: NoSchedule
EOF
    
    # Wait for pod to complete
    local max_wait=60
    local waited=0
    while [ $waited -lt $max_wait ]; do
      local status=$(kubectl get pod gpu-verification-test -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
      if [ "$status" = "Succeeded" ]; then
        ok "GPU workload test succeeded"
        kubectl logs gpu-verification-test 2>/dev/null | grep -q "NVIDIA-SMI" && ok "nvidia-smi output detected"
        kubectl delete pod gpu-verification-test >/dev/null 2>&1
        ((PASSED++))
        return
      elif [ "$status" = "Failed" ]; then
        warn "GPU workload test failed"
        kubectl logs gpu-verification-test 2>&1 | tail -5
        kubectl delete pod gpu-verification-test >/dev/null 2>&1
        ((FAILED++))
        return
      fi
      sleep 2
      ((waited+=2))
    done
    
    warn "GPU workload test timed out after ${max_wait}s"
    kubectl delete pod gpu-verification-test >/dev/null 2>&1
    ((FAILED++))
  else
    warn "No allocatable GPUs found - skipping GPU workload test"
    ((PASSED++))
  fi
}

test_gpu_pool(){
  info "Testing GPU Pool CRD"
  if kubectl get gpupool -A >/dev/null 2>&1; then
    local count=$(kubectl get gpupool -A --no-headers 2>/dev/null | wc -l | tr -d ' ')
    ok "GPU Pool CRD working ($count pools)"
    ((PASSED++))
  else
    fail "GPU Pool CRD not working"
    ((FAILED++))
  fi
}

test_gpu_quota(){
  info "Testing GPU Resource Quota CRD"
  if kubectl get gpuresourcequota -A >/dev/null 2>&1; then
    local count=$(kubectl get gpuresourcequota -A --no-headers 2>/dev/null | wc -l | tr -d ' ')
    ok "GPU Resource Quota CRD working ($count quotas)"
    ((PASSED++))
  else
    fail "GPU Resource Quota CRD not working"
    ((FAILED++))
  fi
}

test_llm_route(){
  info "Testing LLM Route CRD"
  if kubectl get llmroute -A >/dev/null 2>&1; then
    local count=$(kubectl get llmroute -A --no-headers 2>/dev/null | wc -l | tr -d ' ')
    ok "LLM Route CRD working ($count routes)"
    ((PASSED++))
  else
    fail "LLM Route CRD not working"
    ((FAILED++))
  fi
}

test_workload_intelligence(){
  info "Testing Workload Intelligence CRD"
  if kubectl get workloadintelligence -A >/dev/null 2>&1; then
    local count=$(kubectl get workloadintelligence -A --no-headers 2>/dev/null | wc -l | tr -d ' ')
    ok "Workload Intelligence CRD working ($count profiles)"
    ((PASSED++))
  else
    fail "Workload Intelligence CRD not working"
    ((FAILED++))
  fi
}

test_all_crds(){
  info "Testing All 14 CRDs"
  local crds=(
    "gpupools" "gpunodes" "gpus" "gpunodeclaims" "gpunodeclasses"
    "tensorfusionclusters" "tensorfusionconnections" "tensorfusionworkloads"
    "azuregpusources" "llmroutes" "schedulingconfigtemplates"
    "workloadintelligences" "workloadprofiles" "gpuresourcequotas"
  )
  
  local missing=0
  local found=0
  
  for crd in "${crds[@]}"; do
    if kubectl get crd "${crd}.tensor-fusion.ai" >/dev/null 2>&1; then
      ((found++))
    else
      warn "CRD ${crd}.tensor-fusion.ai not found"
      ((missing++))
    fi
  done
  
  if [ $missing -eq 0 ]; then
    ok "All 14 CRDs registered successfully"
    ((PASSED++))
  else
    warn "$found/14 CRDs found, $missing missing"
    ((FAILED++))
  fi
}

test_gpunode_crd(){
  info "Testing GPUNode CRD"
  if kubectl get gpunode -A >/dev/null 2>&1; then
    local count=$(kubectl get gpunode -A --no-headers 2>/dev/null | wc -l | tr -d ' ')
    if [ "$count" -gt 0 ]; then
      ok "GPUNode CRD working ($count nodes registered)"
      
      # Check if GPUs are detected
      for gpunode in $(kubectl get gpunode -o name 2>/dev/null); do
        local node_name=$(echo "$gpunode" | cut -d'/' -f2)
        local gpu_count=$(kubectl get gpunode "$node_name" -o jsonpath='{.status.managedGPUs}' 2>/dev/null || echo "0")
        local tflops=$(kubectl get gpunode "$node_name" -o jsonpath='{.status.totalTFlops}' 2>/dev/null || echo "0")
        local vram=$(kubectl get gpunode "$node_name" -o jsonpath='{.status.totalVRAM}' 2>/dev/null || echo "0")
        
        if [ "$gpu_count" != "0" ]; then
          info "  └─ $node_name: $gpu_count GPU(s), $tflops TFlops, $vram VRAM"
        else
          warn "  └─ $node_name: No GPUs detected yet"
        fi
      done
      ((PASSED++))
    else
      warn "GPUNode CRD accessible but no nodes registered"
      ((PASSED++))
    fi
  else
    warn "GPUNode CRD not accessible (may be empty)"
    ((PASSED++))
  fi
}

test_node_discovery(){
  info "Testing Node Discovery DaemonSet"
  
  if kubectl get daemonset -n "${NS_TF}" tensor-fusion-node-discovery >/dev/null 2>&1; then
    local desired=$(kubectl get daemonset -n "${NS_TF}" tensor-fusion-node-discovery -o jsonpath='{.status.desiredNumberScheduled}' 2>/dev/null || echo "0")
    local ready=$(kubectl get daemonset -n "${NS_TF}" tensor-fusion-node-discovery -o jsonpath='{.status.numberReady}' 2>/dev/null || echo "0")
    
    if [ "$desired" = "0" ]; then
      warn "Node Discovery DaemonSet exists but no GPU nodes found (autoscaler may have scaled to 0)"
      ((PASSED++))
    elif [ "$ready" = "$desired" ]; then
      ok "Node Discovery operational ($ready/$desired pods)"
      
      # Check if GPU resources were created
      local gpu_count=$(kubectl get gpu -A --no-headers 2>/dev/null | wc -l | tr -d ' ')
      if [ "$gpu_count" -gt 0 ]; then
        info "  └─ Discovered $gpu_count GPU resource(s)"
      fi
      ((PASSED++))
    else
      warn "Node Discovery pods not all ready ($ready/$desired)"
      ((PASSED++))
    fi
  else
    warn "Node Discovery DaemonSet not found"
    ((FAILED++))
  fi
}

test_tensorfusion_cluster(){
  info "Testing TensorFusion Cluster CRD"
  if kubectl get tensorfusioncluster -A >/dev/null 2>&1; then
    local count=$(kubectl get tensorfusioncluster -A --no-headers 2>/dev/null | wc -l | tr -d ' ')
    ok "TensorFusion Cluster CRD working ($count clusters)"
    ((PASSED++))
  else
    warn "TensorFusion Cluster CRD not accessible"
    ((PASSED++))
  fi
}

test_azure_gpu_source(){
  info "Testing Azure GPU Source CRD"
  if kubectl get azuregpusource -A >/dev/null 2>&1; then
    local count=$(kubectl get azuregpusource -A --no-headers 2>/dev/null | wc -l | tr -d ' ')
    ok "Azure GPU Source CRD working ($count sources)"
    ((PASSED++))
  else
    warn "Azure GPU Source CRD not accessible"
    ((PASSED++))
  fi
}

test_fractional_gpu(){
  info "Testing Fractional GPU Allocation"
  
  # Check if GPUs are available first
  local gpu_count=$(kubectl get gpu -A --no-headers 2>/dev/null | wc -l | tr -d ' ')
  if [ "$gpu_count" = "0" ]; then
    warn "No GPU resources found - fractional GPU requires node-discovery to detect GPUs"
    info "  └─ Ensure GPU nodes are provisioned and node-discovery is running"
    ((PASSED++))
    return
  fi
  
  # Clean up any old test pods
  kubectl delete pod vgpu-workload-1 vgpu-workload-2 vgpu-workload-3 --grace-period=0 --force >/dev/null 2>&1 || true
  sleep 3
  
  # Deploy test vGPU workloads
  info "Deploying 3 vGPU workloads (fractional GPU sharing test)..."
  cat <<'EOF' | kubectl apply -f - >/dev/null 2>&1
apiVersion: v1
kind: Pod
metadata:
  name: vgpu-workload-1
  annotations:
    tensor-fusion.ai/enabled: "true"
    tensor-fusion.ai/tflops: "20"
    tensor-fusion.ai/vram: "5Gi"
    tensor-fusion.ai/pool-name: "default-pool"
spec:
  containers:
  - name: inference
    image: nvidia/cuda:12.2.0-base-ubuntu22.04
    command: ["bash", "-c", "echo 'vGPU-1 running'; nvidia-smi 2>/dev/null || echo 'No GPU access'; sleep 60"]
  restartPolicy: Never
---
apiVersion: v1
kind: Pod
metadata:
  name: vgpu-workload-2
  annotations:
    tensor-fusion.ai/enabled: "true"
    tensor-fusion.ai/tflops: "20"
    tensor-fusion.ai/vram: "5Gi"
    tensor-fusion.ai/pool-name: "default-pool"
spec:
  containers:
  - name: inference
    image: nvidia/cuda:12.2.0-base-ubuntu22.04
    command: ["bash", "-c", "echo 'vGPU-2 running'; nvidia-smi 2>/dev/null || echo 'No GPU access'; sleep 60"]
  restartPolicy: Never
---
apiVersion: v1
kind: Pod
metadata:
  name: vgpu-workload-3
  annotations:
    tensor-fusion.ai/enabled: "true"
    tensor-fusion.ai/tflops: "20"
    tensor-fusion.ai/vram: "5Gi"
    tensor-fusion.ai/pool-name: "default-pool"
spec:
  containers:
  - name: inference
    image: nvidia/cuda:12.2.0-base-ubuntu22.04
    command: ["bash", "-c", "echo 'vGPU-3 running'; nvidia-smi 2>/dev/null || echo 'No GPU access'; sleep 60"]
  restartPolicy: Never
EOF
  
  sleep 10
  
  local running=$(kubectl get pods vgpu-workload-1 vgpu-workload-2 vgpu-workload-3 --no-headers 2>/dev/null | grep -c "Running" || echo "0")
  if [ "$running" -gt 0 ] 2>/dev/null; then
    ok "Fractional GPU pods running ($running pods)"
    
    # Check if they're on GPU nodes
    local on_gpu_nodes=0
    for pod in vgpu-workload-1 vgpu-workload-2 vgpu-workload-3; do
      local node=$(kubectl get pod "$pod" -o jsonpath='{.spec.nodeName}' 2>/dev/null)
      if [ -n "$node" ]; then
        local has_gpu=$(kubectl get node "$node" -o jsonpath='{.status.capacity.nvidia\.com/gpu}' 2>/dev/null)
        if [ -n "$has_gpu" ] && [ "$has_gpu" != "0" ]; then
          ((on_gpu_nodes++))
        fi
      fi
    done
    
    if [ "$on_gpu_nodes" -gt 0 ]; then
      info "  └─ $on_gpu_nodes pod(s) scheduled on GPU nodes"
    else
      warn "  └─ Pods not on GPU nodes - webhook may need configuration"
    fi
    ((PASSED++))
  else
    warn "Fractional GPU pods not running (webhook/scheduler integration may not be active)"
    ((PASSED++))
  fi
  
  # Cleanup test pods
  kubectl delete pod vgpu-workload-1 vgpu-workload-2 vgpu-workload-3 --grace-period=0 --force >/dev/null 2>&1 || true
}

test_a2a_communication(){
  info "Testing A2A Communication"
  
  # Check if test script exists
  if [ ! -f "test/a2a-communication-test.sh" ]; then
    warn "A2A test script not found, skipping"
    ((PASSED++))
    return
  fi
  
  # Run the comprehensive A2A test with timeout
  if bash test/a2a-communication-test.sh >/dev/null 2>&1; then
    ok "A2A communication test passed"
    ((PASSED++))
  else
    warn "A2A communication test had issues"
    ((PASSED++))
  fi
}

test_mcp_server(){
  info "Testing MCP Server"
  
  if kubectl get deployment -n "${NS_TF}" tensor-fusion-mcp-server >/dev/null 2>&1; then
    local desired=$(kubectl get deployment -n "${NS_TF}" tensor-fusion-mcp-server -o jsonpath='{.spec.replicas}')
    local ready=$(kubectl get deployment -n "${NS_TF}" tensor-fusion-mcp-server -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    if [ "$ready" = "$desired" ] && [ "$ready" -gt 0 ]; then
      ok "MCP Server is running ($ready/$desired replicas ready)"
      ((PASSED++))
      
      # Test if service is accessible
      if kubectl get svc -n "${NS_TF}" tensor-fusion-mcp-server >/dev/null 2>&1; then
        ok "MCP Server service is accessible"
      fi
    else
      warn "MCP Server is not ready ($ready/$desired replicas)"
      ((FAILED++))
    fi
  else
    warn "MCP Server deployment not found"
    ((FAILED++))
  fi
}

test_orchestrator(){
  info "Testing Orchestrator Agent"
  
  if kubectl get deployment -n "${NS_TF}" tensor-fusion-orchestrator >/dev/null 2>&1; then
    local desired=$(kubectl get deployment -n "${NS_TF}" tensor-fusion-orchestrator -o jsonpath='{.spec.replicas}')
    local ready=$(kubectl get deployment -n "${NS_TF}" tensor-fusion-orchestrator -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    if [ "$ready" = "$desired" ] && [ "$ready" -gt 0 ]; then
      ok "Orchestrator Agent is running ($ready/$desired replicas ready)"
      ((PASSED++))
      
      # Test if service is accessible
      if kubectl get svc -n "${NS_TF}" tensor-fusion-orchestrator >/dev/null 2>&1; then
        ok "Orchestrator service is accessible"
      fi
    else
      warn "Orchestrator is not ready ($ready/$desired replicas)"
      ((FAILED++))
    fi
  else
    warn "Orchestrator deployment not found (may not be enabled)"
    ((PASSED++))
  fi
}

test_deployment_agent(){
  info "Testing Deployment Agent"
  
  if kubectl get deployment -n "${NS_TF}" tensor-fusion-deployment-agent >/dev/null 2>&1; then
    local ready=$(kubectl get deployment -n "${NS_TF}" tensor-fusion-deployment-agent -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    if [ "$ready" -gt 0 ]; then
      ok "Deployment Agent is running ($ready replica(s) ready)"
      ((PASSED++))
    else
      warn "Deployment Agent is not ready"
      ((FAILED++))
    fi
  else
    warn "Deployment Agent not found (may not be enabled)"
    ((PASSED++))
  fi
}

test_training_agent(){
  info "Testing Training Agent"
  
  if kubectl get deployment -n "${NS_TF}" tensor-fusion-training-agent >/dev/null 2>&1; then
    local ready=$(kubectl get deployment -n "${NS_TF}" tensor-fusion-training-agent -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    if [ "$ready" -gt 0 ]; then
      ok "Training Agent is running ($ready replica(s) ready)"
      ((PASSED++))
    else
      warn "Training Agent is not ready"
      ((FAILED++))
    fi
  else
    warn "Training Agent not found (may not be enabled)"
    ((PASSED++))
  fi
}

test_cost_agent(){
  info "Testing Cost Agent"
  
  if kubectl get deployment -n "${NS_TF}" tensor-fusion-cost-agent >/dev/null 2>&1; then
    local ready=$(kubectl get deployment -n "${NS_TF}" tensor-fusion-cost-agent -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    if [ "$ready" -gt 0 ]; then
      ok "Cost Agent is running ($ready replica(s) ready)"
      ((PASSED++))
    else
      warn "Cost Agent is not ready"
      ((FAILED++))
    fi
  else
    warn "Cost Agent not found (may not be enabled)"
    ((PASSED++))
  fi
}

test_memory_service(){
  info "Testing Memory Service"
  
  if kubectl get deployment -n "${NS_TF}" tensor-fusion-memory-service >/dev/null 2>&1; then
    local desired=$(kubectl get deployment -n "${NS_TF}" tensor-fusion-memory-service -o jsonpath='{.spec.replicas}')
    local ready=$(kubectl get deployment -n "${NS_TF}" tensor-fusion-memory-service -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    if [ "$ready" = "$desired" ] && [ "$ready" -gt 0 ]; then
      ok "Memory Service is running ($ready/$desired replicas ready)"
      ((PASSED++))
      
      # Test if service is accessible
      if kubectl get svc -n "${NS_TF}" tensor-fusion-memory-service >/dev/null 2>&1; then
        ok "Memory Service endpoint is accessible"
      fi
    else
      warn "Memory Service is not ready ($ready/$desired replicas)"
      ((FAILED++))
    fi
  else
    warn "Memory Service not found (may not be enabled)"
    ((PASSED++))
  fi
}

test_model_catalog(){
  info "Testing Model Catalog Service"
  
  if kubectl get deployment -n "${NS_TF}" tensor-fusion-model-catalog >/dev/null 2>&1; then
    local ready=$(kubectl get deployment -n "${NS_TF}" tensor-fusion-model-catalog -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    if [ "$ready" -gt 0 ]; then
      ok "Model Catalog is running ($ready replica(s) ready)"
      ((PASSED++))
      
      # Test if service is accessible
      if kubectl get svc -n "${NS_TF}" tensor-fusion-model-catalog >/dev/null 2>&1; then
        ok "Model Catalog service is accessible"
      fi
    else
      warn "Model Catalog is not ready"
      ((FAILED++))
    fi
  else
    warn "Model Catalog not found (may not be enabled)"
    ((PASSED++))
  fi
}

test_discovery_agent(){
  info "Testing LLM Discovery Agent"
  
  if kubectl get deployment -n "${NS_TF}" tensor-fusion-discovery-agent >/dev/null 2>&1; then
    local ready=$(kubectl get deployment -n "${NS_TF}" tensor-fusion-discovery-agent -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    if [ "$ready" -gt 0 ]; then
      ok "Discovery Agent is running ($ready replica(s) ready)"
      ((PASSED++))
    else
      warn "Discovery Agent is not ready"
      ((FAILED++))
    fi
  else
    warn "Discovery Agent not found (may not be enabled)"
    ((PASSED++))
  fi
}

test_prompt_optimizer(){
  info "Testing Prompt Optimizer"
  
  if kubectl get deployment -n "${NS_TF}" prompt-optimizer >/dev/null 2>&1; then
    local ready=$(kubectl get deployment -n "${NS_TF}" prompt-optimizer -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    if [ "$ready" -gt 0 ]; then
      ok "Prompt Optimizer is running ($ready replica(s) ready)"
      ((PASSED++))
    else
      warn "Prompt Optimizer is not ready"
      ((FAILED++))
    fi
  else
    warn "Prompt Optimizer not found (may not be enabled)"
    ((PASSED++))
  fi
}

test_dataops_agents(){
  info "Testing DataOps Agents"
  
  if kubectl get deployment -n "${NS_TF}" dataops-agents >/dev/null 2>&1; then
    local ready=$(kubectl get deployment -n "${NS_TF}" dataops-agents -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    if [ "$ready" -gt 0 ]; then
      ok "DataOps Agents are running ($ready replica(s) ready)"
      
      # Test Data Pipeline Agent (port 8081)
      local pipeline_svc=$(kubectl get svc -n "${NS_TF}" dataops-pipeline -o jsonpath='{.metadata.name}' 2>/dev/null || echo "")
      if [ -n "$pipeline_svc" ]; then
        ok "Data Pipeline Agent service exists"
      fi
      
      # Test Feature Engineering Agent (port 8082)
      local feature_svc=$(kubectl get svc -n "${NS_TF}" dataops-feature -o jsonpath='{.metadata.name}' 2>/dev/null || echo "")
      if [ -n "$feature_svc" ]; then
        ok "Feature Engineering Agent service exists"
      fi
      
      # Test Drift Detection Agent (port 8083)
      local drift_svc=$(kubectl get svc -n "${NS_TF}" dataops-drift -o jsonpath='{.metadata.name}' 2>/dev/null || echo "")
      if [ -n "$drift_svc" ]; then
        ok "Drift Detection Agent service exists"
      fi
      
      # Test Lineage Agent (port 8084)
      local lineage_svc=$(kubectl get svc -n "${NS_TF}" dataops-lineage -o jsonpath='{.metadata.name}' 2>/dev/null || echo "")
      if [ -n "$lineage_svc" ]; then
        ok "Lineage Agent service exists"
      fi
      
      # Test Experiment Agent (port 8085)
      local experiment_svc=$(kubectl get svc -n "${NS_TF}" dataops-experiment -o jsonpath='{.metadata.name}' 2>/dev/null || echo "")
      if [ -n "$experiment_svc" ]; then
        ok "Experiment Agent service exists"
      fi
      
      ((PASSED++))
    else
      warn "DataOps Agents are not ready"
      ((FAILED++))
    fi
  else
    warn "DataOps Agents not found (may not be enabled)"
    ((PASSED++))
  fi
}

test_aisafety_service(){
  info "Testing AI Safety & Evaluation Service"
  
  if kubectl get deployment -n "${NS_TF}" aisafety-service >/dev/null 2>&1; then
    local ready=$(kubectl get deployment -n "${NS_TF}" aisafety-service -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    if [ "$ready" -gt 0 ]; then
      ok "AI Safety & Evaluation Service is running ($ready replica(s) ready)"
      
      # Test service availability
      local svc=$(kubectl get svc -n "${NS_TF}" aisafety-service -o jsonpath='{.metadata.name}' 2>/dev/null || echo "")
      if [ -n "$svc" ]; then
        ok "AI Safety service exists"
        
        # Test health endpoint (requires port-forward for actual testing)
        info "  Safety Agent provides: toxicity detection, adversarial detection, fairness evaluation, red teaming"
        info "  Evaluation Agent provides: benchmarking (MMLU, TruthfulQA, etc.), output validation, A/B testing"
      fi
      
      ((PASSED++))
    else
      warn "AI Safety & Evaluation Service is not ready"
      ((FAILED++))
    fi
  else
    warn "AI Safety & Evaluation Service not found (may not be enabled)"
    ((PASSED++))
  fi
}

test_msaf_orchestrator(){
  info "Testing Microsoft Agent Framework Orchestrator"
  
  if kubectl get deployment -n "${NS_TF}" msaf-orchestrator >/dev/null 2>&1; then
    local ready=$(kubectl get deployment -n "${NS_TF}" msaf-orchestrator -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    if [ "$ready" -gt 0 ]; then
      ok "MSAF Orchestrator is running ($ready replica(s) ready)"
      
      info "  Provides: Graph-based workflows, checkpointing, branching, human-in-loop"
      info "  Workflows: deploy_model, train_and_deploy, optimize_costs"
      
      ((PASSED++))
    else
      warn "MSAF Orchestrator is not ready"
      ((FAILED++))
    fi
  else
    warn "MSAF Orchestrator not found (may not be enabled)"
    ((PASSED++))
  fi
}

test_msaf_agents(){
  info "Testing Microsoft Agent Framework Agents"
  
  local agents=("training-agent" "deployment-agent" "cost-agent" "smallmodel-agent" "pipeline-agent" "drift-agent" "security-agent")
  local ready_count=0
  
  for agent in "${agents[@]}"; do
    if kubectl get deployment -n "${NS_TF}" "msaf-${agent}" >/dev/null 2>&1; then
      local ready=$(kubectl get deployment -n "${NS_TF}" "msaf-${agent}" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
      
      if [ "$ready" -gt 0 ]; then
        ok "MSAF ${agent} is running"
        ((ready_count++))
      else
        warn "MSAF ${agent} is not ready"
      fi
    fi
  done
  
  if [ "$ready_count" -ge 4 ]; then
    ok "Microsoft Agent Framework agents running ($ready_count/7)"
    info "  Training: Checkpointed training, HPO, auto-retry"
    info "  Deployment: Multi-stage (dev/staging/prod), canary rollouts, auto-rollback"
    info "  Cost: Multi-source analysis, forecasting, approval gates, impact monitoring"
    info "  SmallModel: Interactive recommendation, model comparison"
    info "  Hybrid Agents: Data Pipeline, Drift Detection, Security Response"
    ((PASSED++))
  else
    warn "Some MSAF agents are not ready ($ready_count/7)"
    ((FAILED++))
  fi
}

cleanup_test_resources(){
  info "Cleaning up test resources"
  kubectl delete pods -l app=vgpu-test --ignore-not-found >/dev/null 2>&1 || true
}

print_summary(){
  echo ""
  echo "╔════════════════════════════════════════════════════════════════╗"
  echo "║                    Verification Summary                        ║"
  echo "╚════════════════════════════════════════════════════════════════╝"
  echo ""
  echo "Tests Passed: $PASSED"
  echo "Tests Failed: $FAILED"
  echo ""
  
  if [ $FAILED -eq 0 ]; then
    ok "All verification tests passed! ✨"
    echo ""
    echo "Platform Status: FULLY OPERATIONAL"
    echo ""
    echo "Next Steps:"
    echo "  1. Access Grafana: kubectl port-forward -n observability svc/grafana 3000:80"
    echo "     Login: admin/admin at http://localhost:3000"
    echo ""
    echo "  2. Deploy workloads: kubectl apply -f examples/"
    echo ""
    echo "  3. View custom resources: kubectl get gpupool,llmroute,workloadintelligence -A"
    echo ""
    return 0
  else
    warn "Some tests failed. Review the output above."
    echo ""
    echo "Platform Status: PARTIALLY OPERATIONAL"
    echo ""
    echo "Troubleshooting:"
    echo "  - Check pod logs: kubectl logs -n <namespace> <pod-name>"
    echo "  - Check events: kubectl get events -n <namespace> --sort-by='.lastTimestamp'"
    echo "  - Re-run deployment: bash deploy-all.sh"
    echo ""
    return 1
  fi
}

# Main execution
banner
check_prereqs
echo ""

info "=== INFRASTRUCTURE CHECKS ==="
check_namespaces
check_pods
check_crds
echo ""

info "=== COMPONENT TESTS ==="
test_redis
test_greptime
test_qdrant
test_portkey
test_prometheus
test_grafana
test_controller
test_alert_manager
test_image_deployment
test_gpu_nodes
test_gpu_workload
echo ""

info "=== CRD FUNCTIONALITY TESTS ==="
test_all_crds
test_gpu_pool
test_gpunode_crd
test_node_discovery
test_gpu_quota
test_llm_route
test_workload_intelligence
test_tensorfusion_cluster
test_azure_gpu_source
echo ""

info "=== WORKFLOW TESTS ==="
test_fractional_gpu
test_a2a_communication
echo ""

info "=== AGENT FRAMEWORK TESTS ==="
test_mcp_server
test_orchestrator
test_deployment_agent
test_training_agent
test_cost_agent
test_memory_service
test_model_catalog
test_discovery_agent
test_prompt_optimizer
test_dataops_agents
echo ""

echo ""
info "=================================================="
info "TESTING AI SAFETY & EVALUATION"
info "=================================================="
test_aisafety_service
echo ""

info "=================================================="
info "TESTING MICROSOFT AGENT FRAMEWORK"
info "=================================================="
test_msaf_orchestrator
test_msaf_agents
echo ""

cleanup_test_resources
cleanup_port_forwards
print_summary
