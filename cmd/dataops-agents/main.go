package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/NexusGPU/tensor-fusion/internal/dataops"
)

func main() {
	log.Println("Starting DataOps Agents...")

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get agent type from environment
	agentType := getEnv("AGENT_TYPE", "all")
	port := getEnv("HTTP_ADDR", ":8080")

	// Start appropriate agent(s)
	switch agentType {
	case "pipeline":
		agent := dataops.NewDataPipelineAgent()
		go agent.Start(ctx, port)
	case "feature":
		agent := dataops.NewFeatureEngineeringAgent()
		go agent.Start(ctx, port)
	case "drift":
		agent := dataops.NewDriftDetectionAgent()
		go agent.Start(ctx, port)
	case "lineage":
		agent := dataops.NewLineageAgent()
		go agent.Start(ctx, port)
	case "experiment":
		agent := dataops.NewExperimentAgent()
		go agent.Start(ctx, port)
	case "all":
		// Start all agents on different ports
		go dataops.NewDataPipelineAgent().Start(ctx, ":8081")
		go dataops.NewFeatureEngineeringAgent().Start(ctx, ":8082")
		go dataops.NewDriftDetectionAgent().Start(ctx, ":8083")
		go dataops.NewLineageAgent().Start(ctx, ":8084")
		go dataops.NewExperimentAgent().Start(ctx, ":8085")
		log.Println("All DataOps agents started")
	default:
		log.Fatalf("Unknown agent type: %s", agentType)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down DataOps agents...")
	cancel()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

