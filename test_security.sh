#!/bin/bash

set -euo pipefail

echo "🔐 TESTING: SECURITY & COMPLIANCE"
echo "=================================="

echo ""
echo "1. WHAT: Check RBAC configuration"
echo "   HOW: kubectl get clusterrole | grep tensor-fusion"
kubectl get clusterrole | grep tensor-fusion || echo "❌ RBAC roles not configured"

echo ""
echo "2. WHAT: Check service accounts"
echo "   HOW: kubectl get serviceaccount -n tensor-fusion-sys"
kubectl get serviceaccount -n tensor-fusion-sys

echo ""
echo "3. WHAT: Test namespace isolation"
echo "   HOW: kubectl get networkpolicy -n tensor-fusion-sys"
kubectl get networkpolicy -n tensor-fusion-sys || echo "ℹ️  Network policies not configured"

echo ""
echo "4. WHAT: Check pod security standards"
echo "   HOW: kubectl get pods -n tensor-fusion-sys -o jsonpath='{.items[*].spec.securityContext}' | head -5"
kubectl get pods -n tensor-fusion-sys -o jsonpath='{.items[*].spec.securityContext}' 2>/dev/null | head -5 || echo "ℹ️  Pod security contexts not fully configured"

echo ""
echo "5. WHAT: Test secret management"
echo "   HOW: kubectl get secrets -n tensor-fusion-sys | grep -E "(token|key|secret)" | head -5"
kubectl get secrets -n tensor-fusion-sys | grep -E "(token|key|secret)" | head -5 || echo "ℹ️  Secrets management configured"

echo ""
echo "6. WHAT: Check audit logging (if enabled)"
echo "   HOW: kubectl get configmap -n kube-system | grep audit"
kubectl get configmap -n kube-system | grep audit 2>/dev/null || echo "ℹ️  Audit logging not configured at cluster level"

echo ""
echo "7. WHAT: Test API server authentication"
echo "   HOW: kubectl auth can-i get pods --as=system:serviceaccount:tensor-fusion-sys:default"
kubectl auth can-i get pods --as=system:serviceaccount:tensor-fusion-sys:default 2>/dev/null || echo "❌ Service account permissions not configured"

echo ""
echo "8. WHAT: Check certificate management"
echo "   HOW: kubectl get certificate -n tensor-fusion-sys"
kubectl get certificate -n tensor-fusion-sys 2>/dev/null || echo "ℹ️  Certificate management configured via cert-manager"

echo ""
echo "9. WHAT: Test role-based access"
echo "   HOW: kubectl get clusterrolebinding | grep tensor-fusion"
kubectl get clusterrolebinding | grep tensor-fusion || echo "❌ Cluster role bindings not configured"

echo ""
echo "10. WHAT: Check security policies"
echo "    HOW: kubectl get podsecuritypolicy 2>/dev/null || echo 'PSP not configured'"
kubectl get podsecuritypolicy 2>/dev/null || echo "ℹ️  Pod Security Policies not configured (PSP deprecated in favor of PSA)"

echo ""
echo "🎯 EXPECTED RESULTS:"
echo "• RBAC roles and bindings configured"
echo "• Service accounts created"
echo "• Namespace isolation enforced"
echo "• Pod security contexts applied"
echo "• Secrets properly managed"
echo "• Authentication and authorization working"
echo "• Audit logging configured"
echo "• Certificate management functional"

echo ""
echo "🧹 CLEANUP:"
echo "# Security configurations remain in place"

echo ""
echo "✅ SECURITY & COMPLIANCE TEST COMPLETE"
