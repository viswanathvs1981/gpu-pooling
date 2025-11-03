FROM golang:1.25-alpine AS builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /dataops-agents ./cmd/dataops-agents

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /dataops-agents .

EXPOSE 8080 8081 8082 8083 8084 8085

CMD ["./dataops-agents"]

