#!/bin/bash

set -uo pipefail

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

success() { echo -e "${GREEN}✅ $1${NC}"; }
info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
warn() { echo -e "${YELLOW}⚠️  $1${NC}"; }
error() { echo -e "${RED}❌ $1${NC}"; }

DELAY=3
SKIP_LIST=""

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --delay)
      DELAY="$2"
      shift 2
      ;;
    --skip)
      SKIP_LIST="$2"
      shift 2
      ;;
    --help)
      cat <<EOF
Usage: $0 [OPTIONS]

Run all TensorFusion use case demos in sequence.

OPTIONS:
  --delay SECONDS    Delay between demos (default: 3)
  --skip LIST        Comma-separated list of use cases to skip (e.g., "2,9")
  --help             Show this help message

EXAMPLES:
  $0                      # Run all demos with 3s delay
  $0 --delay 10           # Run all with 10s delay
  $0 --skip 2,9           # Skip use cases 2 and 9

AVAILABLE USE CASES:
  1. Multi-Tenant GPU Quotas
  2. Cost-Optimized Auto-Scaling
  3. Fractional GPU Sharing
  5. Distributed Training (A2A)

EOF
      exit 0
      ;;
    *)
      error "Unknown option: $1"
      echo "Use --help for usage information"
      exit 1
      ;;
  esac
done

banner() {
cat <<'EOF'
╔════════════════════════════════════════════════════════════════╗
║                                                                 ║
║     TensorFusion Platform - Complete Demo Suite                ║
║                                                                 ║
║     Running all use cases to demonstrate platform              ║
║     capabilities and real-world problem solving                ║
║                                                                 ║
╚════════════════════════════════════════════════════════════════╝
EOF
}

should_skip() {
  local use_case=$1
  if [[ ",$SKIP_LIST," == *",$use_case,"* ]]; then
    return 0
  fi
  return 1
}

run_demo() {
  local num=$1
  local script=$2
  local title=$3
  
  if should_skip "$num"; then
    warn "Skipping USE CASE $num: $title"
    echo ""
    return
  fi
  
  if [ ! -f "$script" ]; then
    warn "USE CASE $num script not found: $script"
    echo ""
    return
  fi
  
  info "═══════════════════════════════════════════════════════════════"
  info "Starting USE CASE $num: $title"
  info "═══════════════════════════════════════════════════════════════"
  echo ""
  
  bash "$script"
  local exit_code=$?
  
  echo ""
  if [ $exit_code -eq 0 ]; then
    success "USE CASE $num completed successfully"
  else
    error "USE CASE $num failed with exit code $exit_code"
  fi
  
  echo ""
  info "Waiting ${DELAY} seconds before next demo..."
  sleep "$DELAY"
  echo ""
  echo ""
}

banner
echo ""
info "Demo Configuration:"
echo "   • Delay between demos: ${DELAY} seconds"
if [ -n "$SKIP_LIST" ]; then
  echo "   • Skipping use cases: $SKIP_LIST"
fi
echo ""
sleep 2

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

START_TIME=$(date +%s)

# Run all demos
run_demo 1 "01-multi-tenant-quotas.sh" "Multi-Tenant GPU Quotas"
run_demo 2 "02-cost-optimization.sh" "Cost-Optimized Auto-Scaling"
run_demo 3 "03-fractional-gpu.sh" "Fractional GPU Sharing"
run_demo 5 "05-distributed-training.sh" "Distributed Training with A2A"

# Calculate total time
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
MINUTES=$((DURATION / 60))
SECONDS=$((DURATION % 60))

echo ""
echo "═══════════════════════════════════════════════════════════════"
success "🎉 ALL DEMOS COMPLETED!"
echo "═══════════════════════════════════════════════════════════════"
echo ""
info "Summary:"
echo "   • Total time: ${MINUTES}m ${SECONDS}s"
echo "   • Demos run: 4"
echo "   • Platform: TensorFusion"
echo ""
success "Platform capabilities demonstrated successfully! 🚀"
echo ""
info "💡 Next steps:"
echo "   • Review individual demo scripts for details"
echo "   • Run 'bash cleanup-all-demos.sh' to clean up"
echo "   • Check 'kubectl get pods -A' to see running workloads"
echo ""

