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

	"instant/internal/auth"
	"instant/internal/consul"
	"instant/internal/database"
	"instant/internal/email"
	kafkapkg "instant/internal/kafka"
	"instant/internal/logger"
	"instant/internal/session"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// Load configuration from environment
	port := getEnv("AUTH_SERVICE_PORT", "8081")
	host := getEnv("AUTH_SERVICE_HOST", "localhost")
	consulAddr := getEnv("CONSUL_HTTP_ADDR", "localhost:8500")
	consulToken := getEnv("CONSUL_HTTP_TOKEN", "")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := 0

	log.Println("Starting Auth Service...")
	log.Printf("Port: %s", port)
	log.Printf("Host: %s", host)
	log.Printf("Consul: %s", consulAddr)
	log.Printf("Redis: %s", redisAddr)

	// Initialize database
	db := database.New()
	log.Println("Connected to database")

	// Initialize Redis for verification codes and sessions
	store := session.NewRedisStore(redisAddr, redisPassword, redisDB)
	sessionMgr := session.NewManager(store)
	log.Println("Connected to Redis")

	// Initialize logger
	lgr := logger.New()

	// Initialize email sender
	emailConfig := email.NewConfig()
	emailSender := email.NewSender(emailConfig)
	log.Printf("Email mode: %s", emailConfig.Mode)

	// Initialize Kafka producer (optional)
	var kafkaProducer *kafkapkg.Producer
	var authService auth.Service

	kafkaBrokers := getEnv("KAFKA_BROKERS", "")
	enableKafka := getEnv("ENABLE_KAFKA", "true") // Enable by default

	if kafkaBrokers != "" && enableKafka == "true" {
		kafkaConfig, err := kafkapkg.LoadConfig()
		if err != nil {
			log.Printf("Failed to load Kafka config, using direct email: %v", err)
			authService = auth.NewService(db, store, emailSender)
		} else {
			kafkaProducer, err = kafkapkg.NewProducer(kafkaConfig, lgr)
			if err != nil {
				log.Printf("Failed to create Kafka producer, using direct email: %v", err)
				authService = auth.NewService(db, store, emailSender)
			} else {
				log.Printf("Kafka producer initialized: %s", kafkaBrokers)
				authService = auth.NewServiceWithKafka(db, store, emailSender, kafkaProducer)
				defer kafkaProducer.Close()
			}
		}
	} else {
		log.Println("Kafka disabled, using direct email")
		authService = auth.NewService(db, store, emailSender)
	}

	authHandler := auth.NewHandler(authService, sessionMgr)

	// Setup Gin router
	r := gin.Default()

	// Public auth endpoints
	r.POST("/request-code", authHandler.RequestCode)
	r.POST("/verify-code", authHandler.VerifyCode)
	r.POST("/logout", authHandler.Logout)
	r.GET("/health", authHandler.Health)

	// Protected user management endpoints (require session)
	users := r.Group("/users")
	users.Use(sessionAuthMiddleware(sessionMgr))
	{
		users.PATCH("/:id", authHandler.UpdateUser)
		users.GET("/:id/request-delete-code", authHandler.RequestDeleteCode)
		users.POST("/:id/delete", authHandler.DeleteUser)
	}

	// Initialize Consul client
	consulClient, err := consul.NewClientWithToken(consulAddr, consulToken)
	if err != nil {
		log.Fatalf("Failed to create Consul client: %v", err)
	}
	log.Println("Connected to Consul")

	// Register service with Consul
	// Use static service ID to prevent duplicate registrations on restart
	serviceID := fmt.Sprintf("auth-service-%s", host)

	// Deregister any existing instance with same ID (cleanup from previous crashes)
	_ = consulClient.Deregister(serviceID)

	err = consulClient.Register(&consul.ServiceConfig{
		ID:      serviceID,
		Name:    "auth-service",
		Address: host,
		Port:    mustAtoi(port),
		Tags:    []string{"auth", "authentication", "passwordless"},
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
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Auth Service listening on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Auth Service...")

	// Deregister from Consul
	if err := consulClient.Deregister(serviceID); err != nil {
		log.Printf("Failed to deregister from Consul: %v", err)
	} else {
		log.Println("Deregistered from Consul")
	}

	// Close database connection
	if err := db.Close(); err != nil {
		log.Printf("Failed to close database: %v", err)
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Auth Service stopped")
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getHostname returns the hostname or a default value
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// mustAtoi converts a string to int or panics
func mustAtoi(s string) int {
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		panic(fmt.Sprintf("invalid integer: %s", s))
	}
	return result
}

// sessionAuthMiddleware validates session and injects user context
func sessionAuthMiddleware(sessionMgr session.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session ID from cookie
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized: no session cookie",
			})
			return
		}

		// Validate and get session
		sess, err := sessionMgr.Get(c.Request.Context(), sessionID)
		if err != nil {
			log.Printf("Invalid session %s: %v", sessionID, err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized: invalid session",
			})
			return
		}

		// Double-check expiration
		if time.Now().After(sess.ExpiresAt) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized: session expired",
			})
			return
		}

		// Inject user context into Gin context
		c.Set("user_id", sess.UserID)
		c.Set("email", sess.Email)

		c.Next()
	}
}
