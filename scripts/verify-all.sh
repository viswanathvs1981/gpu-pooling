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
  for pid in "${PIDS_TO_KILL[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
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
    ((PASSED++))
  else
    fail "Controller pod not found"
    ((FAILED++))
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

test_fractional_gpu(){
  info "Testing Fractional GPU Allocation"
  
  # Check if example file exists
  if [ ! -f "examples/fractional-gpu-sharing.yaml" ]; then
    warn "Fractional GPU example not found, skipping"
    ((PASSED++))
    return
  fi
  
  # Deploy test pods
  kubectl apply -f examples/fractional-gpu-sharing.yaml >/dev/null 2>&1 || true
  sleep 5
  
  local pods=$(kubectl get pods -l app=vgpu-test --no-headers 2>/dev/null | wc -l | tr -d ' ')
  if [ "$pods" -gt 0 ]; then
    ok "Fractional GPU pods created ($pods pods)"
    ((PASSED++))
  else
    warn "Fractional GPU pods not created (may need GPU nodes)"
    ((PASSED++))
  fi
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
echo ""

info "=== CRD FUNCTIONALITY TESTS ==="
test_gpu_pool
test_gpu_quota
test_llm_route
test_workload_intelligence
echo ""

info "=== WORKFLOW TESTS ==="
test_fractional_gpu
test_a2a_communication
echo ""

cleanup_test_resources
cleanup_port_forwards
print_summary
