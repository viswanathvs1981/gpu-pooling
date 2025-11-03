#!/bin/bash

# NexusAI Platform - Port Forwarding Setup Script
# This script sets up port forwarding for all AI services to enable testing

set -e

NAMESPACE="tensor-fusion-sys"
echo "🚀 Setting up port forwarding for NexusAI Platform services..."

# Function to start port forwarding in background
start_port_forward() {
    local service=$1
    local port=$2
    local local_port=$3

    echo "📡 Starting port forward: $service:$port -> localhost:$local_port"
    kubectl port-forward -n $NAMESPACE svc/$service $local_port:$port >/dev/null 2>&1 &
    echo $! > /tmp/port-forward-$service.pid

    # Wait a moment for port forward to establish
    sleep 2

    # Test if port forward is working
    if curl -s http://localhost:$local_port/health >/dev/null 2>&1; then
        echo "✅ $service port forward established on localhost:$local_port"
    else
        echo "⚠️  $service port forward may not be ready yet"
    fi
}

# Clean up any existing port forwards
echo "🧹 Cleaning up existing port forwards..."
pkill -f "kubectl port-forward" || true
rm -f /tmp/port-forward-*.pid
sleep 2

# Start port forwarding for all services
echo ""
echo "🌐 Starting port forwards..."

# AI Services
start_port_forward "aisafety-service" 80 8080
start_port_forward "tensor-fusion-memory-service" 8090 8090
start_port_forward "tensor-fusion-model-catalog" 8095 8095
start_port_forward "tensor-fusion-discovery-agent" 8080 8081  # Check actual port
start_port_forward "prompt-optimizer" 8080 8082
start_port_forward "dataops-agents" 8081 8083  # Main dataops port

# Infrastructure services (for completeness)
start_port_forward "redis" 6379 6379
start_port_forward "postgresql" 5432 5432
start_port_forward "minio" 9000 9000
start_port_forward "greptimedb" 4000 4000
start_port_forward "qdrant" 6333 6333

# TensorFusion controller
start_port_forward "tensor-fusion" 8080 8084

echo ""
echo "🎯 PORT FORWARDING SETUP COMPLETE"
echo "=================================="
echo ""
echo "📋 Service Endpoints (localhost):"
echo "• AI Safety: http://localhost:8080"
echo "• Memory Service: http://localhost:8090"
echo "• Model Catalog: http://localhost:8095"
echo "• Discovery Agent: http://localhost:8081"
echo "• Prompt Optimizer: http://localhost:8082"
echo "• DataOps Agents: http://localhost:8083"
echo "• TensorFusion Controller: http://localhost:8084"
echo ""
echo "🔧 Infrastructure Services:"
echo "• Redis: localhost:6379"
echo "• PostgreSQL: localhost:5432"
echo "• MinIO: localhost:9000"
echo "• GreptimeDB: localhost:4000"
echo "• Qdrant: localhost:6333"
echo ""
echo "🧪 Now you can run the test scripts!"
echo "Example: ./test_ai_safety.sh"
echo ""
echo "🛑 To stop port forwarding: ./scripts/stop-port-forwarding.sh"
echo ""
echo "📝 Port forward PIDs saved in /tmp/port-forward-*.pid"
