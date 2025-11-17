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
	"instant/internal/follow"
)

func main() {
	port := getEnv("FOLLOW_SERVICE_PORT", "8087")
	host := getEnv("FOLLOW_SERVICE_HOST", "follow-service")
	consulAddr := getEnv("CONSUL_HTTP_ADDR", "localhost:8500")
	consulToken := getEnv("CONSUL_HTTP_TOKEN", "")

	log.Println("Starting Follow Service...")
	log.Printf("Host: %s Port: %s Consul: %s", host, port, consulAddr)

	db := database.New()
	defer db.Close()

	svc := follow.NewService(db)
	router := follow.SetupRouter(svc)

	// CONSUL
	consulClient, err := consul.NewClientWithToken(consulAddr, consulToken)
	if err != nil {
		log.Fatalf("consul client error: %v", err)
	}
	serviceID := fmt.Sprintf("follow-service-%s", host)
	_ = consulClient.Deregister(serviceID)

	if err := consulClient.Register(&consul.ServiceConfig{
		ID:      serviceID,
		Name:    "follow-service",
		Address: host,
		Port:    mustAtoi(port),
		Tags:    []string{"follow", "social"},
		Check: &consul.HealthCheck{
			HTTP:     fmt.Sprintf("http://%s:%s/health", host, port),
			Interval: "10s",
			Timeout:  "3s",
		},
	}); err != nil {
		log.Fatalf("consul register error: %v", err)
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Follow Service listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if err := consulClient.Deregister(serviceID); err != nil {
		log.Printf("Consul deregister error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)

	log.Println("Follow Service stopped")
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustAtoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
