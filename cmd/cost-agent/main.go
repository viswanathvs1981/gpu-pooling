package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/NexusGPU/tensor-fusion/internal/agents"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	var redisAddr string
	var mcpServerURL string

	flag.StringVar(&redisAddr, "redis-addr", "redis.redis.svc.cluster.local:6379", "Redis address")
	flag.StringVar(&mcpServerURL, "mcp-url", "http://tensor-fusion-mcp-server.tensor-fusion-sys.svc.cluster.local:8080", "MCP server URL")
	flag.Parse()

	// Setup logger
	log.SetLogger(zap.New(zap.UseDevMode(true)))
	logger := log.Log.WithName("cost-agent")

	logger.Info("Starting Cost Agent", "redis", redisAddr, "mcp_url", mcpServerURL)

	// Create agent
	agent, err := agents.NewCostAgent(redisAddr, mcpServerURL)
	if err != nil {
		logger.Error(err, "Failed to create cost agent")
		os.Exit(1)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal")
		cancel()
	}()

	// Start agent
	logger.Info("Cost Agent ready")
	if err := agent.Start(ctx); err != nil {
		logger.Error(err, "Cost Agent stopped with error")
		os.Exit(1)
	}

	logger.Info("Cost Agent stopped")
}

