package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/NexusGPU/tensor-fusion/internal/mcp"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 8080, "Port to listen on")
	flag.Parse()

	// Setup logger
	log.SetLogger(zap.New(zap.UseDevMode(true)))
	logger := log.Log.WithName("mcp-server")

	logger.Info("Starting TensorFusion MCP Server", "port", port)

	// Create server
	server, err := mcp.NewPlatformServer(port)
	if err != nil {
		logger.Error(err, "Failed to create MCP server")
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

	// Start server
	logger.Info("MCP Server ready")
	if err := server.Start(ctx); err != nil {
		logger.Error(err, "MCP server stopped with error")
		os.Exit(1)
	}

	logger.Info("MCP Server stopped")
}

