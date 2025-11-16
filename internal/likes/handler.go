package likes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

// Like handles POST /likes
// @Summary Like a post
// @Description Like a post by providing post ID (requires authentication)
// @Tags likes
// @Accept json
// @Produce json
// @Param like body LikeRequest true "Like request data"
// @Success 201 {object} Like
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security SessionAuth
// @Router /api/likes [post]
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

// Unlike handles DELETE /likes/:post_id
// @Summary Unlike a post
// @Description Remove like from a post (requires authentication)
// @Tags likes
// @Produce json
// @Param post_id path string true "Post ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security SessionAuth
// @Router /api/likes/{post_id} [delete]
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

// Count handles GET /posts/:post_id/likes/count
// @Summary Get like count for a post
// @Description Get the total number of likes for a specific post
// @Tags likes
// @Produce json
// @Param post_id path string true "Post ID"
// @Success 200 {object} CountResponse
// @Failure 500 {object} map[string]string
// @Router /api/posts/{post_id}/likes/count [get]
func (h *Handler) Count(c *gin.Context) {
	postID := c.Param("post_id")
	cnt, err := h.svc.Count(c.Request.Context(), postID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count"})
		return
	}
	c.JSON(http.StatusOK, CountResponse{PostID: postID, Count: cnt})
}

// IsLiked handles GET /posts/:post_id/likes/me
// @Summary Check if current user liked a post
// @Description Check if the authenticated user has liked a specific post
// @Tags likes
// @Produce json
// @Param post_id path string true "Post ID"
// @Success 200 {object} LikedResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security SessionAuth
// @Router /api/posts/{post_id}/likes/me [get]
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
