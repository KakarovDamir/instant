package likes

import "github.com/gin-gonic/gin"

func SetupRouter(svc Service) *gin.Engine {
	r := gin.Default()
	h := NewHandler(svc)

	// Health
	r.GET("/health", h.Health)

	// Likes
	r.POST("/", h.Like)
	r.DELETE("/:post_id", h.Unlike)
	r.GET("/:post_id/likes/count", h.Count)
	r.GET("/:post_id/likes/me", h.IsLiked)

	return r
}
