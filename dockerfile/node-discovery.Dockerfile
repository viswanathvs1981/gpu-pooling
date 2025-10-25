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

# Create minimal runtime image
FROM ubuntu:24.04

# Install NVIDIA driver dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    libnvidia-compute-535 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /
COPY --from=builder /workspace/nodediscovery .

# Run as non-root user for security
USER 65532:65532

ENTRYPOINT ["/nodediscovery"]
