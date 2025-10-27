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
	var port int
	var mcpServerURL string

	flag.IntVar(&port, "port", 9000, "Port to listen on")
	flag.StringVar(&mcpServerURL, "mcp-url", "http://tensor-fusion-mcp-server.tensor-fusion-sys.svc.cluster.local:8080", "MCP server URL")
	flag.Parse()

	// Setup logger
	log.SetLogger(zap.New(zap.UseDevMode(true)))
	logger := log.Log.WithName("orchestrator")

	logger.Info("Starting Orchestrator Agent", "port", port, "mcp_url", mcpServerURL)

	// Create orchestrator
	orchestrator := agents.NewOrchestratorAgent(port, mcpServerURL)

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

	// Start orchestrator
	logger.Info("Orchestrator Agent ready")
	if err := orchestrator.Start(ctx); err != nil {
		logger.Error(err, "Orchestrator stopped with error")
		os.Exit(1)
	}

	logger.Info("Orchestrator stopped")
}

