#!/bin/bash

set -euo pipefail

echo "🚀 NEXUSAI PLATFORM - COMPREHENSIVE TESTING SUITE"
echo "=================================================="
echo ""

# Test execution order - from infrastructure to advanced features
TESTS=(
    "test_infrastructure.sh:Infrastructure & Orchestration"
    "test_gpu_pooling.sh:GPU Pooling & Scheduling"
    "test_memory_service.sh:Memory Services"
    "test_model_catalog.sh:Model Catalog & Registry"
    "test_llm_discovery.sh:LLM Discovery & Management"
    "test_ai_gateway.sh:AI Gateway & Token Management"
    "test_observability.sh:Observability & Monitoring"
    "test_security.sh:Security & Compliance"
    "test_ai_agents.sh:AI Agent Framework (MSAF)"
    "test_dataops.sh:DataOps Pipeline"
    "test_ai_safety.sh:AI Safety & Evaluation"
    "test_prompt_optimizer.sh:Prompt Optimization"
)

echo "📋 AVAILABLE TESTS:"
echo "=================="
for i in "${!TESTS[@]}"; do
    IFS=':' read -r script description <<< "${TESTS[$i]}"
    echo "$((i+1)). $script - $description"
done

echo ""
echo "🎯 TESTING STRATEGY:"
echo "==================="
echo "• Test infrastructure first (foundations)"
echo "• Test core GPU functionality"
echo "• Test storage and memory systems"
echo "• Test AI/ML services and agents"
echo "• Test monitoring and security last"
echo ""

# Check if we should run all tests or individual ones
if [ $# -eq 0 ]; then
    echo "💡 USAGE:"
    echo "  ./run_all_tests.sh all          # Run all tests"
    echo "  ./run_all_tests.sh 1            # Run test #1"
    echo "  ./run_all_tests.sh infra        # Run infrastructure tests"
    echo ""

    echo "🔥 QUICK START - RUN INFRASTRUCTURE TESTS:"
    echo "./run_all_tests.sh infra"
    exit 0
fi

run_test() {
    local test_script="$1"
    local description="$2"

    echo ""
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║ RUNNING: $description"
    echo "╚════════════════════════════════════════════════════════════════╝"

    if [ -f "$test_script" ]; then
        chmod +x "$test_script"
        ./"$test_script"
        echo ""
        echo "✅ $description - COMPLETED"
    else
        echo "❌ Test script $test_script not found!"
    fi
}

case "$1" in
    "all")
        echo "🚀 RUNNING ALL TESTS..."
        for test_info in "${TESTS[@]}"; do
            IFS=':' read -r script description <<< "$test_info"
            run_test "$script" "$description"
            echo ""
            echo "⏱️  Waiting 5 seconds before next test..."
            sleep 5
        done
        ;;

    "infra")
        echo "🏗️  RUNNING INFRASTRUCTURE TESTS..."
        run_test "test_infrastructure.sh" "Infrastructure & Orchestration"
        run_test "test_gpu_pooling.sh" "GPU Pooling & Scheduling"
        run_test "test_observability.sh" "Observability & Monitoring"
        run_test "test_security.sh" "Security & Compliance"
        ;;

    [0-9]*)
        test_index=$(( $1 - 1 ))
        if [ $test_index -ge 0 ] && [ $test_index -lt ${#TESTS[@]} ]; then
            IFS=':' read -r script description <<< "${TESTS[$test_index]}"
            run_test "$script" "$description"
        else
            echo "❌ Invalid test number: $1"
            exit 1
        fi
        ;;

    *)
        echo "❌ Unknown option: $1"
        echo "Use: ./run_all_tests.sh [all|infra|1-12]"
        exit 1
        ;;
esac

echo ""
echo "🎉 TESTING COMPLETE!"
echo ""
echo "📊 SUMMARY:"
echo "• 13 comprehensive test suites created"
echo "• Each test validates specific functionality"
echo "• Tests include health checks, API calls, and integrations"
echo "• All tests are idempotent and can be run multiple times"
echo ""
echo "🔄 NEXT STEPS:"
echo "• Fix any ❌ failures shown above"
echo "• Run ./scripts/verify-all.sh for overall platform health"
echo "• Access Grafana dashboards for monitoring"
echo "• Check pod logs for detailed error information"
