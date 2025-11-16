package follow

import (
	
	"net/http"
	"os"
	"time"

	"instant/internal/database"

	
)

type Server struct {
	port string
	db   database.Service
}

// NewServer initializes the server
func NewServer() *http.Server {
	port := getEnv("FOLLOW_SERVICE_PORT", "8085")

	s := &Server{
		port: port,
		db:   database.New(),
	}

	return &http.Server{
		Addr:         ":" + port,
		Handler:      s.RegisterRoutes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// getEnv reads environment variable or default
func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}
