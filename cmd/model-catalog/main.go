package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/NexusGPU/tensor-fusion/internal/training/catalog"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 8095, "Port to listen on")
	flag.Parse()

	// Setup logger
	log.SetLogger(zap.New(zap.UseDevMode(true)))
	logger := log.Log.WithName("model-catalog")

	logger.Info("Starting Model Catalog Service", "port", port)

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

	// Create catalog service
	catalogService := catalog.NewCatalogService(port, k8sClient)

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
	logger.Info("Model Catalog Service ready")
	if err := catalogService.Start(ctx); err != nil {
		logger.Error(err, "Catalog service stopped with error")
		os.Exit(1)
	}

	logger.Info("Model Catalog Service stopped")
}

