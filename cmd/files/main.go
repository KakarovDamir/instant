package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"instant/internal/consul"
	"instant/internal/files"
	"instant/internal/storage"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// Load configuration from environment
	port := getEnv("FILES_SERVICE_PORT", "8084")
	host := getEnv("FILES_SERVICE_HOST", "files-service")
	consulAddr := getEnv("CONSUL_HTTP_ADDR", "localhost:8500")
	consulToken := getEnv("CONSUL_HTTP_TOKEN", "")

	log.Println("Starting Files Service...")
	log.Printf("Port: %s", port)
	log.Printf("Host: %s", host)
	log.Printf("Consul: %s", consulAddr)

	// Initialize storage service (MinIO/S3)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	storageService, err := storage.New(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize storage service: %v", err)
	}
	log.Println("Connected to storage (MinIO)")

	// Initialize files service
	filesService := files.NewService(storageService)

	// Setup router
	server := files.NewServer(filesService)
	router := server.RegisterRoutes()

	// Initialize Consul client
	consulClient, err := consul.NewClientWithToken(consulAddr, consulToken)
	if err != nil {
		log.Fatalf("Failed to create Consul client: %v", err)
	}
	log.Println("Connected to Consul")

	// Register service with Consul
	serviceID := fmt.Sprintf("files-service-%s", host)

	// Deregister any existing instance with same ID
	_ = consulClient.Deregister(serviceID)

	err = consulClient.Register(&consul.ServiceConfig{
		ID:      serviceID,
		Name:    "files-service",
		Address: host,
		Port:    mustAtoi(port),
		Tags:    []string{"files", "storage", "uploads", "downloads"},
		Check: &consul.HealthCheck{
			HTTP:     fmt.Sprintf("http://%s:%s/health", host, port),
			Interval: "10s",
			Timeout:  "3s",
		},
	})
	if err != nil {
		log.Fatalf("Failed to register service with Consul: %v", err)
	}
	log.Printf("Registered with Consul as %s", serviceID)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // Longer for file uploads
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Files Service listening on port %s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Files Service...")

	// Deregister from Consul
	if err := consulClient.Deregister(serviceID); err != nil {
		log.Printf("Failed to deregister from Consul: %v", err)
	} else {
		log.Println("Deregistered from Consul")
	}

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Files Service stopped")
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// mustAtoi converts a string to int or panics
func mustAtoi(s string) int {
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		panic(fmt.Sprintf("invalid integer: %s", s))
	}
	return result
}
