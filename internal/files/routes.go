package files

import (
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Server holds dependencies for files service
type Server struct {
	service *Service
}

// NewServer creates a new files server
func NewServer(service *Service) *Server {
	return &Server{service: service}
}

// RegisterRoutes sets up HTTP routes for files service
func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000", "http://localhost:8080"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-User-ID", "X-User-Email"},
		AllowCredentials: true,
	}))

	handler := NewHandler(s.service)

	// Health check endpoint (public)
	r.GET("/health", handler.Health)

	// File operations endpoints
	// Note: These should be protected via Gateway in production
	filesGroup := r.Group("/files")
	{
		filesGroup.POST("/upload-url", handler.GenerateUploadURL)     // Generate presigned upload URL
		filesGroup.POST("/download-url", handler.GenerateDownloadURL) // Generate presigned download URL
		filesGroup.DELETE("/:key", handler.DeleteFile)                // Delete file
	}

	return r
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
