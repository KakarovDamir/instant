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
	"instant/internal/database"
	"instant/internal/likes"
)

func main() {
	// ENV
	port := getEnv("LIKES_SERVICE_PORT", "8084")
	host := getEnv("LIKES_SERVICE_HOST", "likes-service")
	consulAddr := getEnv("CONSUL_HTTP_ADDR", "localhost:8500")
	consulToken := getEnv("CONSUL_HTTP_TOKEN", "")

	log.Println("Starting Likes Service...")
	log.Printf("Host: %s Port: %s Consul: %s", host, port, consulAddr)

	db := database.New()
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("db close error: %v", err)
		}
	}()

	svc := likes.NewService(db)
	router := likes.SetupRouter(svc)

	// Consul
	consulClient, err := consul.NewClientWithToken(consulAddr, consulToken)
	if err != nil {
		log.Fatalf("consul client error: %v", err)
	}
	serviceID := fmt.Sprintf("likes-service-%s", host)
	_ = consulClient.Deregister(serviceID)

	if err := consulClient.Register(&consul.ServiceConfig{
		ID:      serviceID,
		Name:    "likes-service",
		Address: host,
		Port:    mustAtoi(port),
		Tags:    []string{"likes", "social"},
		Check: &consul.HealthCheck{
			HTTP:     fmt.Sprintf("http://%s:%s/health", host, port),
			Interval: "10s",
			Timeout:  "3s",
		},
	}); err != nil {
		log.Fatalf("consul register error: %v", err)
	}
	log.Printf("Registered in Consul as %s", serviceID)

	// HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run
	go func() {
		log.Printf("Likes Service listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down Likes Service...")

	if err := consulClient.Deregister(serviceID); err != nil {
		log.Printf("Consul deregister error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("Likes Service stopped")
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func mustAtoi(s string) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		panic("invalid int: " + s)
	}
	return n
}
