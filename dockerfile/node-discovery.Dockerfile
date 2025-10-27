# Build the Node Discovery Agent binary
# This agent runs as a DaemonSet to discover GPU hardware on each node
FROM golang:1.25 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG GO_LDFLAGS

WORKDIR /workspace

# Copy Go module files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy only the necessary source code for node discovery
COPY cmd/nodediscovery/ cmd/nodediscovery/
COPY api/ api/
COPY internal/constants/ internal/constants/
COPY internal/metrics/ internal/metrics/
COPY internal/utils/ internal/utils/
COPY internal/config/ internal/config/

# Build with CGO enabled (required for NVIDIA driver interaction)
# Cross-compilation support for different architectures
RUN CGO_ENABLED=1 \
    GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH} \
    go build \
    -ldflags="${GO_LDFLAGS} -s -w" \
    -a -o nodediscovery \
    cmd/nodediscovery/main.go

# Use NVIDIA CUDA base image which has proper NVML setup
FROM nvidia/cuda:12.2.0-base-ubuntu22.04

# Install minimal dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /
COPY --from=builder /workspace/nodediscovery .

# Run as root for GPU access
USER root

ENTRYPOINT ["/nodediscovery"]
