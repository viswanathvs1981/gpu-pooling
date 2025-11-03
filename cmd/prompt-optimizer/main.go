package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NexusGPU/tensor-fusion/internal/promptopt"
)

func main() {
	log.Println("Starting Prompt Optimizer Service...")

	// Configuration from environment
	redisAddr := getEnv("REDIS_ADDR", "redis:6379")
	enableA2A := getEnv("ENABLE_A2A", "true") == "true"
	httpAddr := getEnv("HTTP_ADDR", ":8080")

	// Create service
	svc, err := promptopt.NewService(redisAddr, enableA2A)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Start A2A listener if enabled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if enableA2A {
		go func() {
			if err := svc.StartA2AListener(ctx); err != nil && err != context.Canceled {
				log.Printf("A2A listener error: %v", err)
			}
		}()
		log.Println("A2A listener started")
	}

	// Start HTTP server
	server := &http.Server{
		Addr:    httpAddr,
		Handler: svc.HTTPHandler(),
	}

	go func() {
		log.Printf("HTTP server listening on %s", httpAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Prompt Optimizer Service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

