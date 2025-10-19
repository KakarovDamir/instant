package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"instant/internal/database"
	"instant/internal/storage"
)

// Server holds the dependencies for the HTTP server
type Server struct {
	port int

	db      database.Service
	storage storage.Service
}

// Config holds server configuration
type Config struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// LoadConfigFromEnv loads server configuration from environment variables
func LoadConfigFromEnv() *Config {
	port, _ := strconv.Atoi(getEnv("PORT", "8080"))

	return &Config{
		Port:         port,
		ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 15*time.Second),
		WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 60*time.Second),
		IdleTimeout:  getEnvDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
	}
}

// NewServer creates and configures a new HTTP server
func NewServer() *http.Server {
	cfg := LoadConfigFromEnv()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	storageService, err := storage.New(ctx)
	if err != nil {
		log.Printf("[Server] Warning: failed to initialize storage service: %v", err)
	} else {
		log.Printf("[Server] Storage service initialized successfully")
	}

	// Initialize database
	dbService := database.New()
	log.Printf("[Server] Database service initialized")

	appServer := &Server{
		port:    cfg.Port,
		db:      dbService,
		storage: storageService,
	}

	// Configure HTTP server with optimized settings for high-load
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           appServer.RegisterRoutes(),
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	log.Printf("[Server] HTTP server configured on port %d", cfg.Port)
	return server
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
