#!/bin/bash

set -euo pipefail

echo "🔥 TESTING: GPU POOLING & SCHEDULING SYSTEM"
echo "================================================"

echo ""
echo "1. WHAT: Check Node Discovery DaemonSet"
echo "   HOW: kubectl get daemonset -n tensor-fusion-sys"
kubectl get daemonset -n tensor-fusion-sys | grep node-discovery || echo "❌ Node Discovery not running"

echo ""
echo "2. WHAT: Check GPUNode Custom Resources"
echo "   HOW: kubectl get gpunode"
kubectl get gpunode || echo "❌ No GPUNode resources found"

echo ""
echo "3. WHAT: Check GPU-related CRDs exist"
echo "   HOW: kubectl get crd | grep gpu"
kubectl get crd | grep gpu || echo "❌ GPU CRDs not installed"

echo ""
echo "4. WHAT: Create a GPUPool resource"
echo "   HOW: Apply GPUPool YAML with correct schema"
cat <<EOF | kubectl apply -f -
apiVersion: tensor-fusion.ai/v1
kind: GPUPool
metadata:
  name: test-gpu-pool
  namespace: default
spec:
  capacityConfig:
    oversubscription:
      tflopsOversellRatio: 500
      vramExpandToHostMem: 50
      vramExpandToHostDisk: 70
  nodeManagerConfig:
    provisioningMode: AutoSelect
    nodeSelector:
      nodeSelectorTerms:
      - matchExpressions:
        - key: nvidia.com/gpu.present
          operator: Exists
  qosConfig:
    defaultQoS: medium
EOF

echo ""
echo "5. WHAT: Check GPUPool was created"
echo "   HOW: kubectl get gpupool"
kubectl get gpupool

echo ""
echo "6. WHAT: Create a GPUNodeClaim to test allocation"
echo "   HOW: Apply GPUNodeClaim YAML"
cat <<EOF | kubectl apply -f -
apiVersion: tensor-fusion.ai/v1
kind: GPUNodeClaim
metadata:
  name: test-gpu-node-claim
  namespace: default
spec:
  gpuDeviceOffered: 1
  tflopsOffered: "10"
  vramOffered: "16Gi"
  instanceType: "Standard_NC4as_T4_v3"
  region: "eastus"
  zone: "1"
EOF

echo ""
echo "7. WHAT: Check GPUNodeClaim status"
echo "   HOW: kubectl get gpunodeclaim"
kubectl get gpunodeclaim

echo ""
echo "8. WHAT: Create a pod that requests fractional GPU"
echo "   HOW: Apply test pod YAML"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: test-fractional-gpu-pod
  namespace: default
spec:
  containers:
  - name: gpu-test
    image: nvidia/cuda:11.8-runtime-ubuntu20.04
    command: ["nvidia-smi"]
    resources:
      limits:
        tensor-fusion.ai/gpu-memory: "4Gi"
        tensor-fusion.ai/gpu-count: "0.5"
      requests:
        tensor-fusion.ai/gpu-memory: "4Gi"
        tensor-fusion.ai/gpu-count: "0.5"
EOF

echo ""
echo "9. WHAT: Check if pod gets scheduled"
echo "   HOW: kubectl get pod test-fractional-gpu-pod"
kubectl get pod test-fractional-gpu-pod

echo ""
echo "10. WHAT: Check pod logs (should show GPU info)"
echo "    HOW: kubectl logs test-fractional-gpu-pod"
kubectl logs test-fractional-gpu-pod || echo "❌ Pod not running yet"

echo ""
echo "🎯 EXPECTED RESULTS:"
echo "• Node Discovery DaemonSet running on GPU nodes"
echo "• GPUNode resources created automatically"
echo "• GPUPool created successfully"
echo "• GPUNodeClaim shows allocated status"
echo "• Test pod scheduled on GPU node"
echo "• Pod logs show nvidia-smi output"

echo ""
echo "🧹 CLEANUP:"
echo "kubectl delete gpupool test-gpu-pool"
echo "kubectl delete gpunodeclaim test-gpu-node-claim"
echo "kubectl delete pod test-fractional-gpu-pod"

echo ""
echo "✅ GPU POOLING TEST COMPLETE"
