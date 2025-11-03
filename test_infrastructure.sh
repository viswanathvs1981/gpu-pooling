#!/bin/bash

set -euo pipefail

echo "🏗️ TESTING: INFRASTRUCTURE & ORCHESTRATION"
echo "==========================================="

echo ""
echo "1. WHAT: Check core services status"
echo "   HOW: kubectl get pods -n tensor-fusion-sys --no-headers | wc -l"
kubectl get pods -n tensor-fusion-sys --no-headers | wc -l
kubectl get pods -n tensor-fusion-sys

echo ""
echo "2. WHAT: Test Redis connectivity"
echo "   HOW: kubectl exec -n tensor-fusion-sys deployment/redis -- redis-cli ping"
kubectl exec -n tensor-fusion-sys deployment/redis -- redis-cli ping 2>/dev/null || echo "❌ Redis not accessible"

echo ""
echo "3. WHAT: Test PostgreSQL connectivity"
echo "   HOW: kubectl exec -n tensor-fusion-sys deployment/postgresql -- psql -U tensorfusion -d tensorfusion -c 'SELECT 1'"
kubectl exec -n tensor-fusion-sys deployment/postgresql -- psql -U tensorfusion -d tensorfusion -c 'SELECT 1' 2>/dev/null || echo "❌ PostgreSQL not accessible"

echo ""
echo "4. WHAT: Test MinIO object storage"
echo "   HOW: kubectl exec -n tensor-fusion-sys deployment/minio -- curl -f http://localhost:9000/minio/health/live"
kubectl exec -n tensor-fusion-sys deployment/minio -- curl -f http://localhost:9000/minio/health/live 2>/dev/null || echo "❌ MinIO not accessible"

echo ""
echo "5. WHAT: Test Qdrant vector database"
echo "   HOW: kubectl exec -n tensor-fusion-sys deployment/qdrant -- curl -f http://localhost:6333/health"
kubectl exec -n tensor-fusion-sys deployment/qdrant -- curl -f http://localhost:6333/health 2>/dev/null || echo "❌ Qdrant not accessible"

echo ""
echo "6. WHAT: Test GreptimeDB time series"
echo "   HOW: kubectl exec -n tensor-fusion-sys deployment/greptimedb -- curl -f http://localhost:4000/health"
kubectl exec -n tensor-fusion-sys deployment/greptimedb -- curl -f http://localhost:4000/health 2>/dev/null || echo "❌ GreptimeDB not accessible"

echo ""
echo "7. WHAT: Check GPU Operator status"
echo "   HOW: kubectl get pods -n gpu-operator"
kubectl get pods -n gpu-operator 2>/dev/null || echo "❌ GPU operator not running"

echo ""
echo "8. WHAT: Test Kubernetes API access"
echo "   HOW: kubectl get nodes"
kubectl get nodes

echo ""
echo "9. WHAT: Check custom resource definitions"
echo "   HOW: kubectl get crd | grep tensor-fusion | wc -l"
kubectl get crd | grep tensor-fusion | wc -l

echo ""
echo "10. WHAT: Test service discovery"
echo "    HOW: kubectl get svc -n tensor-fusion-sys | head -10"
kubectl get svc -n tensor-fusion-sys | head -10

echo ""
echo "🎯 EXPECTED RESULTS:"
echo "• All core services running (Redis, PostgreSQL, MinIO, Qdrant, GreptimeDB)"
echo "• GPU operator deployed and functional"
echo "• Kubernetes API accessible"
echo "• Custom resource definitions installed"
echo "• Service discovery working"
echo "• All infrastructure components healthy"

echo ""
echo "🧹 CLEANUP:"
echo "# Infrastructure services remain running"

echo ""
echo "✅ INFRASTRUCTURE & ORCHESTRATION TEST COMPLETE"
