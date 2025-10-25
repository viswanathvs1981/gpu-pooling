#!/bin/bash

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║   TensorFusion vLLM Deployment                                 ║"
echo "╚════════════════════════════════════════════════════════════════╝"

# Configuration
NAMESPACE="${NAMESPACE:-vllm}"
MODEL_NAME="${MODEL_NAME:-llama-3-8b}"
BASE_MODEL="${BASE_MODEL:-meta-llama/Meta-Llama-3-8B}"
GPU_COUNT="${GPU_COUNT:-1}"
VGPU_RESOURCES="${VGPU_RESOURCES:-1.0}"
IMAGE_REPO="${IMAGE_REPO:-vllm/vllm-openai}"
IMAGE_TAG="${IMAGE_TAG:-latest}"

log_info "Creating namespace..."
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

log_info "Creating PVC for LoRA adapters..."
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ${MODEL_NAME}-lora-storage
  namespace: $NAMESPACE
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 50Gi
EOF

log_info "Deploying vLLM for model: $BASE_MODEL..."
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $MODEL_NAME
  namespace: $NAMESPACE
  labels:
    app: vllm
    model: $MODEL_NAME
spec:
  replicas: 1
  selector:
    matchLabels:
      app: vllm
      model: $MODEL_NAME
  template:
    metadata:
      labels:
        app: vllm
        model: $MODEL_NAME
    spec:
      containers:
      - name: vllm
        image: ${IMAGE_REPO}:${IMAGE_TAG}
        args:
          - --model
          - $BASE_MODEL
          - --host
          - "0.0.0.0"
          - --port
          - "8000"
          - --tensor-parallel-size
          - "$GPU_COUNT"
          - --enable-lora
          - --lora-modules
          - /lora-adapters
          - --max-lora-rank
          - "64"
          - --served-model-name
          - $MODEL_NAME
        ports:
        - containerPort: 8000
          name: http
        resources:
          requests:
            tensor-fusion.ai/vgpu: "$VGPU_RESOURCES"
            memory: 16Gi
            cpu: "4"
          limits:
            tensor-fusion.ai/vgpu: "$VGPU_RESOURCES"
            memory: 32Gi
        livenessProbe:
          httpGet:
            path: /health
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /v1/models
            port: 8000
          initialDelaySeconds: 15
          periodSeconds: 5
        volumeMounts:
        - name: lora-adapters
          mountPath: /lora-adapters
        - name: cache
          mountPath: /root/.cache/huggingface
        env:
        - name: VLLM_ENGINE_ITERATION_TIMEOUT_S
          value: "60"
        - name: VLLM_RPC_TIMEOUT
          value: "10000"
        - name: HF_HOME
          value: /root/.cache/huggingface
      volumes:
      - name: lora-adapters
        persistentVolumeClaim:
          claimName: ${MODEL_NAME}-lora-storage
      - name: cache
        emptyDir:
          sizeLimit: 50Gi
---
apiVersion: v1
kind: Service
metadata:
  name: $MODEL_NAME
  namespace: $NAMESPACE
  labels:
    app: vllm
    model: $MODEL_NAME
spec:
  type: ClusterIP
  ports:
  - port: 8000
    targetPort: 8000
    name: http
  selector:
    app: vllm
    model: $MODEL_NAME
EOF

log_success "vLLM deployment created!"

log_info "Waiting for vLLM to be ready..."
kubectl wait --for=condition=ready pod -l app=vllm,model=$MODEL_NAME -n $NAMESPACE --timeout=600s

log_success "vLLM is ready!"

log_info "Testing vLLM endpoint..."
POD_NAME=$(kubectl get pods -n $NAMESPACE -l app=vllm,model=$MODEL_NAME -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n $NAMESPACE pod/$POD_NAME 8000:8000 &
PF_PID=$!

sleep 5

curl -s http://localhost:8000/v1/models | jq .

kill $PF_PID 2>/dev/null || true

log_success "Deployment complete!"
echo ""
echo "To access vLLM:"
echo "  kubectl port-forward -n $NAMESPACE svc/$MODEL_NAME 8000:8000"
echo ""
echo "Test with:"
echo "  curl http://localhost:8000/v1/models"



