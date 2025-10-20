package posts

import (
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000", "http://localhost:8080"}, // Add frontend and gateway URLs
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-User-ID", "X-User-Email"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	// Initialize repository, service, and handler
	repo := NewRepository(s.db)

	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := 0

	service := NewService(repo, redisAddr, redisPassword, redisDB)
	handler := NewHandler(service)

	// Health check endpoint (public, no auth required)
	r.GET("/health", handler.Health)

	// Posts API endpoints - all require authentication via Gateway
	postsGroup := r.Group("/posts")
	postsGroup.Use(AuthMiddleware()) // Validate X-User-ID header from gateway
	{
		postsGroup.GET("", handler.GetAllPosts)           // GET /posts?page=1&page_size=20
		postsGroup.POST("", handler.CreatePost)           // POST /posts
		postsGroup.GET("/:id", handler.GetPost)           // GET /posts/:id
		postsGroup.PATCH("/:id", handler.UpdatePost)      // PATCH /posts/:id
		postsGroup.DELETE("/:id", handler.DeletePost)     // DELETE /posts/:id
	}

	// User posts endpoint - requires auth
	users := r.Group("/users")
	users.Use(AuthMiddleware())
	{
		users.GET("/:user_id/posts", handler.GetUserPosts) // GET /users/:user_id/posts?page=1&page_size=20
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
