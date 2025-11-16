package comments

import "github.com/gin-gonic/gin"

func SetupRouter(svc Service) *gin.Engine {
    r := gin.Default()
    h := NewHandler(svc)

    r.GET("/health", h.Health)

    r.POST("/", h.Create)
    r.PATCH("/:id", h.Update)
    r.DELETE("/:id", h.Delete)
    r.GET("/post/:post_id", h.List)

    return r
}
