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
	"instant/internal/posts"

	_ "github.com/joho/godotenv/autoload"
)

func gracefulShutdown(apiServer *http.Server, consulClient *consul.Client, serviceID string, done chan bool) {
	// Create context that listens for the interrupt signal from the OS
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal
	<-ctx.Done()

	log.Println("Shutting down gracefully, press Ctrl+C again to force")
	stop() // Allow Ctrl+C to force shutdown

	// Deregister from Consul
	if err := consulClient.Deregister(serviceID); err != nil {
		log.Printf("Failed to deregister from Consul: %v", err)
	} else {
		log.Println("Deregistered from Consul")
	}

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown with error: %v", err)
	}

	log.Println("Server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

func main() {
	// Load configuration
	port := getEnv("PORT", "8082")
	host := getEnv("POSTS_SERVICE_HOST", "localhost")
	consulAddr := getEnv("CONSUL_HTTP_ADDR", "localhost:8500")
	consulToken := getEnv("CONSUL_HTTP_TOKEN", "")

	log.Println("Starting Posts Service...")
	log.Printf("Port: %s", port)
	log.Printf("Host: %s", host)
	log.Printf("Consul: %s", consulAddr)

	// Initialize Consul client
	consulClient, err := consul.NewClientWithToken(consulAddr, consulToken)
	if err != nil {
		log.Fatalf("Failed to create Consul client: %v", err)
	}
	log.Println("Connected to Consul")

	// Register service with Consul
	// Use static service ID to prevent duplicate registrations on restart
	serviceID := fmt.Sprintf("posts-service-%s", host)

	// Deregister any existing instance with same ID (cleanup from previous crashes)
	_ = consulClient.Deregister(serviceID)

	err = consulClient.Register(&consul.ServiceConfig{
		ID:      serviceID,
		Name:    "posts-service",
		Address: host,
		Port:    mustAtoi(port),
		Tags:    []string{"posts", "content", "api"},
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

	// Create server
	apiServer := posts.NewServer()

	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)

	// Run graceful shutdown in a separate goroutine
	go gracefulShutdown(apiServer, consulClient, serviceID, done)

	log.Printf("Posts Service listening on port %s", port)
	err = apiServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	// Wait for the graceful shutdown to complete
	<-done
	log.Println("Graceful shutdown complete.")
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
