package likes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

// POST /likes  {post_id}
func (h *Handler) Like(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req LikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	like, err := h.svc.Like(c.Request.Context(), userID, req.PostID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to like"})
		return
	}
	c.JSON(http.StatusCreated, like)
}

// DELETE /likes/:post_id
func (h *Handler) Unlike(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	postID := c.Param("post_id")
	_, err := h.svc.Unlike(c.Request.Context(), userID, postID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unlike"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// GET /posts/:post_id/likes/count
func (h *Handler) Count(c *gin.Context) {
	postID := c.Param("post_id")
	cnt, err := h.svc.Count(c.Request.Context(), postID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count"})
		return
	}
	c.JSON(http.StatusOK, CountResponse{PostID: postID, Count: cnt})
}

// GET /posts/:post_id/likes/me
func (h *Handler) IsLiked(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	postID := c.Param("post_id")
	ok, err := h.svc.IsLiked(c.Request.Context(), userID, postID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}
	c.JSON(http.StatusOK, LikedResponse{PostID: postID, Liked: ok})
}

// GET /health
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "likes-service",
	})
}
