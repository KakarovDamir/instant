package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"instant/internal/consul"
	"instant/internal/email"
	"instant/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// Initialize logger
	lgr := logger.New()
	lgr.Info("Starting Email Service...")

	// Load configuration from environment
	port := getEnv("EMAIL_SERVICE_PORT", "8085")
	host := getEnv("EMAIL_SERVICE_HOST", "localhost")
	consulAddr := getEnv("CONSUL_HTTP_ADDR", "localhost:8500")
	consulToken := getEnv("CONSUL_HTTP_TOKEN", "")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := 0

	// Kafka configuration
	kafkaBrokers := getEnv("KAFKA_BROKERS", "13.48.120.205:32100")
	kafkaTopic := getEnv("KAFKA_TOPIC_EMAIL_EVENTS", "email-events")
	kafkaDLQTopic := getEnv("KAFKA_TOPIC_EMAIL_DLQ", "email-events-dlq")
	kafkaConsumerGroup := getEnv("KAFKA_CONSUMER_GROUP", "email-service-group")

	lgr.Info("Configuration loaded",
		"port", port,
		"host", host,
		"consul", consulAddr,
		"redis", redisAddr,
		"kafka", kafkaBrokers,
		"topic", kafkaTopic)

	// Initialize Redis for idempotency store
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		lgr.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	lgr.Info("Connected to Redis")

	// Initialize idempotency store
	idempotencyStore := email.NewIdempotencyStore(redisClient, lgr)
	lgr.Info("Idempotency store initialized")

	// Initialize email sender
	emailConfig := email.NewConfig()
	emailSender := email.NewSender(emailConfig)
	lgr.Info("Email sender initialized", "mode", emailConfig.Mode)

	// Initialize Kafka consumer
	consumerConfig := &email.ConsumerConfig{
		Brokers:       kafkaBrokers,
		Topic:         kafkaTopic,
		DLQTopic:      kafkaDLQTopic,
		ConsumerGroup: kafkaConsumerGroup,
		MaxRetries:    3,
	}

	consumer, err := email.NewConsumer(consumerConfig, emailSender, idempotencyStore, lgr)
	if err != nil {
		lgr.Error("Failed to create Kafka consumer", "error", err)
		os.Exit(1)
	}
	defer consumer.Close()

	// Start consumer in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		lgr.Info("Starting Kafka consumer...")
		if err := consumer.Start(ctx); err != nil {
			lgr.Error("Consumer error", "error", err)
		}
	}()

	// Setup HTTP server for health checks
	r := gin.Default()

	handler := email.NewHandler(redisClient, idempotencyStore, lgr)
	r.GET("/health", handler.HealthCheck)
	r.GET("/stats", handler.Stats)

	// Initialize Consul client
	consulClient, err := consul.NewClientWithToken(consulAddr, consulToken)
	if err != nil {
		lgr.Error("Failed to create Consul client", "error", err)
		os.Exit(1)
	}
	lgr.Info("Connected to Consul")

	// Register service with Consul
	serviceID := fmt.Sprintf("email-service-%s", host)

	// Deregister any existing instance with same ID (cleanup from previous crashes)
	_ = consulClient.Deregister(serviceID)

	err = consulClient.Register(&consul.ServiceConfig{
		ID:      serviceID,
		Name:    "email-service",
		Address: host,
		Port:    mustAtoi(port),
		Tags:    []string{"email", "notifications", "kafka-consumer"},
		Check: &consul.HealthCheck{
			HTTP:     fmt.Sprintf("http://%s:%s/health", host, port),
			Interval: "10s",
			Timeout:  "3s",
		},
	})
	if err != nil {
		lgr.Error("Failed to register with Consul", "error", err)
		os.Exit(1)
	}
	lgr.Info("Registered with Consul", "serviceID", serviceID)

	// Start HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	// Run server in goroutine
	go func() {
		lgr.Info("HTTP server started", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			lgr.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	lgr.Info("Shutting down Email Service...")

	// Deregister from Consul
	if err := consulClient.Deregister(serviceID); err != nil {
		lgr.Error("Failed to deregister from Consul", "error", err)
	}

	// Cancel consumer context
	cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		lgr.Error("HTTP server forced to shutdown", "error", err)
	}

	lgr.Info("Email Service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func mustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("Invalid number: %s", s)
	}
	return i
}
