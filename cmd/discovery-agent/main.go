package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/NexusGPU/tensor-fusion/internal/agents"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	flag.Parse()

	// Setup logger
	log.SetLogger(zap.New(zap.UseDevMode(true)))
	logger := log.Log.WithName("discovery-agent")

	logger.Info("Starting LLM Discovery Agent")

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

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed to create clientset")
		os.Exit(1)
	}

	// Create discovery agent
	agent := agents.NewDiscoveryAgent(k8sClient, clientset)

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
	logger.Info("LLM Discovery Agent ready")
	if err := agent.Start(ctx); err != nil {
		logger.Error(err, "Discovery agent stopped with error")
		os.Exit(1)
	}

	logger.Info("LLM Discovery Agent stopped")
}

