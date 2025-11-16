package follow

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Follow(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	var req FollowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}

	err := h.service.Follow(c.Request.Context(), userID, req.TargetUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, FollowResponse{
		Success: true,
		Message: "Followed successfully",
	})
}

func (h *Handler) Unfollow(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	targetID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid user ID",
		})
		return
	}

	err = h.service.Unfollow(c.Request.Context(), userID, targetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, FollowResponse{
		Success: true,
		Message: "Unfollowed successfully",
	})
}


// GetUserID extracts the authenticated user's ID from context
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("userID") // "userID" should be set by your auth middleware
	if !exists {
		return uuid.Nil, false
	}

	idStr, ok := val.(string)
	if !ok {
		return uuid.Nil, false
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, false
	}

	return id, true
}

func (h *Handler) GetFollowers(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid user ID"})
		return
	}

	followers, err := h.service.ListFollowers(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "followers": followers})
}

func (h *Handler) GetFollowing(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid user ID"})
		return
	}

	following, err := h.service.ListFollowing(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "following": following})
}	
