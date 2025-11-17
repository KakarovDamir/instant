package follow

import "github.com/gin-gonic/gin"

func SetupRouter(svc Service) *gin.Engine {
    r := gin.Default()
    h := NewHandler(svc)

    // Health
    r.GET("/health", h.Health)

    // Follow / unfollow
    r.POST("/", h.Follow)
    r.DELETE("/:user_id", h.Unfollow)

    // Counts
    r.GET("/:user_id/followers/count", h.FollowersCount)
    r.GET("/:user_id/following/count", h.FollowingCount)

    // Check if I follow user
    r.GET("/:user_id/following/me", h.IsFollowing)

    return r
}
