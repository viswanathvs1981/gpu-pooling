FROM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY api/ api/
COPY cmd/aisafety-service/ cmd/aisafety-service/
COPY internal/ internal/

# Build the binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -a -o aisafety-service cmd/aisafety-service/main.go

# Final stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /workspace/aisafety-service .

USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/aisafety-service"]

