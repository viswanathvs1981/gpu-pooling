# Build the TensorFusion Operator binary
# This is the main controller that manages GPU virtualization, pooling, and workload scheduling
FROM golang:1.25 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG GO_LDFLAGS

WORKDIR /workspace

# Copy Go module files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code (operator needs all components)
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/
COPY scripts/ scripts/
COPY patches/ patches/

# Apply Kubernetes scheduler patches and vendor dependencies
RUN go mod vendor && \
    bash ./scripts/patch-scheduler.sh

# Build a static binary (no CGO required for operator)
# Cross-compilation support for different architectures
RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH} \
    go build \
    -ldflags="${GO_LDFLAGS} -s -w" \
    -a -o manager \
    ./cmd

# Create minimal runtime image
FROM ubuntu:24.04

# Install CA certificates for HTTPS connections
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /

# Copy only the binary
COPY --from=builder /workspace/manager .

# Run as non-root user for security
USER 65532:65532

ENTRYPOINT ["/manager"]

# Example: Build locally with version info
# docker build \
#   --build-arg TARGETOS=linux \
#   --build-arg TARGETARCH=amd64 \
#   --build-arg GO_LDFLAGS="-X github.com/NexusGPU/tensor-fusion/internal/version.BuildVersion=v1.0.0" \
#   -t tensorfusion/tensor-fusion-operator:v1.0.0 \
#   -f dockerfile/operator.Dockerfile \
#   .
