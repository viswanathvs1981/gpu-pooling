package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/NexusGPU/tensor-fusion/internal/aisafety"
)

func main() {
	log.Println("Starting NexusAI Safety & Evaluation Service...")

	// Get service port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Determine which service to run
	serviceType := os.Getenv("SERVICE_TYPE")
	if serviceType == "" {
		serviceType = "all" // Run all services by default
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received, stopping...")
		cancel()
	}()

	switch serviceType {
	case "safety":
		runSafetyAgent(ctx, port)
	case "evaluation":
		runEvaluationAgent(ctx, port)
	case "all":
		runAllServices(ctx, port)
	default:
		log.Fatalf("Unknown SERVICE_TYPE: %s", serviceType)
	}
}

func runSafetyAgent(ctx context.Context, port string) {
	agent := aisafety.NewSafetyAgent()
	addr := fmt.Sprintf(":%s", port)
	
	log.Printf("Starting AI Safety Agent on %s", addr)
	if err := agent.Start(ctx, addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Safety agent failed: %v", err)
	}
}

func runEvaluationAgent(ctx context.Context, port string) {
	agent := aisafety.NewEvaluationAgent()
	addr := fmt.Sprintf(":%s", port)
	
	log.Printf("Starting Model Evaluation Agent on %s", addr)
	if err := agent.Start(ctx, addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Evaluation agent failed: %v", err)
	}
}

func runAllServices(ctx context.Context, port string) {
	// Run both services behind a multiplexer
	mux := http.NewServeMux()
	
	// Safety agent on /safety/*
	safetyAgent := aisafety.NewSafetyAgent()
	mux.Handle("/safety/", http.StripPrefix("/safety", safetyAgent.HTTPHandler()))
	
	// Evaluation agent on /evaluation/*
	evalAgent := aisafety.NewEvaluationAgent()
	mux.Handle("/evaluation/", http.StripPrefix("/evaluation", evalAgent.HTTPHandler()))
	
	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	addr := fmt.Sprintf(":%s", port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	log.Printf("AI Safety & Evaluation Service listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

