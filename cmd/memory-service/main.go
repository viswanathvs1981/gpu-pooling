package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/NexusGPU/tensor-fusion/internal/memory"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	var port int
	var qdrantURL string
	var greptimeURL string

	flag.IntVar(&port, "port", 8090, "Port to listen on")
	flag.StringVar(&qdrantURL, "qdrant-url", "http://qdrant.qdrant.svc.cluster.local:6333", "Qdrant URL")
	flag.StringVar(&greptimeURL, "greptime-url", "http://greptimedb-standalone.greptimedb.svc.cluster.local:4000", "GreptimeDB URL")
	flag.Parse()

	// Setup logger
	log.SetLogger(zap.New(zap.UseDevMode(true)))
	logger := log.Log.WithName("memory-service")

	logger.Info("Starting Memory Service", "port", port, "qdrant", qdrantURL, "greptime", greptimeURL)

	// Get Kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Error(err, "Failed to get in-cluster config")
		os.Exit(1)
	}

	// Create controller-runtime client
	k8sClient, err := client.New(config, client.Options{})
	if err != nil {
		logger.Error(err, "Failed to create k8s client")
		os.Exit(1)
	}

	// Create memory service
	memService, err := memory.NewMemoryService(port, k8sClient, qdrantURL, greptimeURL)
	if err != nil {
		logger.Error(err, "Failed to create memory service")
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

	// Start service
	logger.Info("Memory Service ready")
	if err := memService.Start(ctx); err != nil {
		logger.Error(err, "Memory service stopped with error")
		os.Exit(1)
	}

	logger.Info("Memory Service stopped")
}

