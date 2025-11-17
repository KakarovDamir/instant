package follow

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Follow(c *gin.Context) {
	followerID := c.GetHeader("X-User-ID")
	if followerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req FollowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	f, err := h.svc.Follow(c, followerID, req.FolloweeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}

	c.JSON(http.StatusCreated, f)
}

func (h *Handler) Unfollow(c *gin.Context) {
	followerID := c.GetHeader("X-User-ID")
	if followerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	followeeID := c.Param("user_id")

	_, err := h.svc.Unfollow(c, followerID, followeeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *Handler) FollowersCount(c *gin.Context) {
	userID := c.Param("user_id")

	cnt, err := h.svc.FollowersCount(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}

	c.JSON(http.StatusOK, CountResponse{UserID: userID, Count: cnt})
}

func (h *Handler) FollowingCount(c *gin.Context) {
	userID := c.Param("user_id")

	cnt, err := h.svc.FollowingCount(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}

	c.JSON(http.StatusOK, CountResponse{UserID: userID, Count: cnt})
}

func (h *Handler) IsFollowing(c *gin.Context) {
	followerID := c.GetHeader("X-User-ID")
	if followerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	followeeID := c.Param("user_id")

	ok, err := h.svc.IsFollowing(c, followerID, followeeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}

	c.JSON(http.StatusOK, FollowingResponse{UserID: followeeID, Following: ok})
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "follow-service",
	})
}
