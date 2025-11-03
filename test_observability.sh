#!/bin/bash

set -euo pipefail

echo "📊 TESTING: OBSERVABILITY & MONITORING"
echo "======================================="

echo ""
echo "1. WHAT: Check Prometheus deployment"
echo "   HOW: kubectl get deployment -n tensor-fusion-sys | grep prometheus"
kubectl get deployment -n tensor-fusion-sys | grep prometheus || echo "❌ Prometheus not deployed"

echo ""
echo "2. WHAT: Check Grafana deployment"
echo "   HOW: kubectl get deployment -n tensor-fusion-sys | grep grafana"
kubectl get deployment -n tensor-fusion-sys | grep grafana || echo "❌ Grafana not deployed"

echo ""
echo "3. WHAT: Test Prometheus metrics endpoint"
echo "   HOW: curl http://prometheus.tensor-fusion-sys/api/v1/query?query=up"
curl -s "http://prometheus.tensor-fusion-sys/api/v1/query?query=up" | head -10 || echo "❌ Prometheus metrics not accessible"

echo ""
echo "4. WHAT: Test Grafana health"
echo "   HOW: curl http://prometheus-grafana.tensor-fusion-sys/api/health"
curl -s http://prometheus-grafana.tensor-fusion-sys/api/health || echo "❌ Grafana not accessible"

echo ""
echo "5. WHAT: Check GPU metrics collection"
echo "   HOW: curl http://prometheus.tensor-fusion-sys/api/v1/query?query=gpu_utilization"
curl -s "http://prometheus.tensor-fusion-sys/api/v1/query?query=gpu_utilization" || echo "❌ GPU metrics not collected"

echo ""
echo "6. WHAT: Check service health metrics"
echo "   HOW: curl http://prometheus.tensor-fusion-sys/api/v1/query?query=kube_pod_container_status_running"
curl -s "http://prometheus.tensor-fusion-sys/api/v1/query?query=kube_pod_container_status_running" || echo "❌ Service health metrics missing"

echo ""
echo "7. WHAT: Test alerting rules"
echo "   HOW: curl http://prometheus.tensor-fusion-sys/api/v1/rules"
curl -s http://prometheus.tensor-fusion-sys/api/v1/rules | head -10 || echo "❌ Alerting rules not configured"

echo ""
echo "8. WHAT: Check alert manager status"
echo "   HOW: kubectl get deployment -n tensor-fusion-sys | grep alertmanager"
kubectl get deployment -n tensor-fusion-sys | grep alertmanager || echo "❌ Alertmanager not deployed"

echo ""
echo "9. WHAT: Test distributed tracing (if available)"
echo "   HOW: Check if Jaeger or similar is deployed"
kubectl get deployment -n tensor-fusion-sys | grep jaeger || echo "ℹ️  Distributed tracing not configured"

echo ""
echo "🎯 EXPECTED RESULTS:"
echo "• Prometheus and Grafana deployed and accessible"
echo "• GPU and service metrics collected"
echo "• Alerting rules configured"
echo "• Alert manager functional"
echo "• Metrics queries return data"
echo "• Dashboards accessible"

echo ""
echo "🧹 CLEANUP:"
echo "# Monitoring services remain running"

echo ""
echo "✅ OBSERVABILITY & MONITORING TEST COMPLETE"
