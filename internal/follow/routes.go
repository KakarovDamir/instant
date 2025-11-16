package follow

import (

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (s *Server) RegisterRoutes() *gin.Engine {
	r := gin.Default()

	// --- CORS ---
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:5173",
			"http://localhost:3000",
			"http://localhost:8080",
		},
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-User-ID"},
		AllowCredentials: true,
	}))

	// --- Dependencies ---
	repo := NewRepository(s.db)
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := 0

	service := NewService(repo, redisAddr, redisPassword, redisDB)
	handler := NewHandler(service)

	// --- HEALTH ---
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// --- FOLLOW API ---
	followGroup := r.Group("/follow")

	// Add auth if you want:
	// followGroup.Use(AuthMiddleware())

	{
		followGroup.POST("", handler.Follow)              // POST /follow
		followGroup.DELETE("/:user_id", handler.Unfollow) // DELETE /follow/:user_id
		followGroup.GET("/followers/:user_id", handler.GetFollowers)
		followGroup.GET("/following/:user_id", handler.GetFollowing)
	}

	return r
}

