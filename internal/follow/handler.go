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

// Follow handles POST /follow
// @Summary Follow a user
// @Description Follow another user by providing their ID (requires authentication)
// @Tags follow
// @Accept json
// @Produce json
// @Param follow body FollowRequest true "Follow request data"
// @Success 200 {object} FollowResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security SessionAuth
// @Router /api/follow [post]
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

// Unfollow handles DELETE /follow/:user_id
// @Summary Unfollow a user
// @Description Unfollow a user by their ID (requires authentication)
// @Tags follow
// @Produce json
// @Param user_id path string true "User ID (UUID) to unfollow"
// @Success 200 {object} FollowResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security SessionAuth
// @Router /api/follow/{user_id} [delete]
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

// GetFollowers handles GET /follow/:user_id/followers
// @Summary Get user's followers
// @Description Retrieve the list of users who follow the specified user
// @Tags follow
// @Produce json
// @Param user_id path string true "User ID (UUID)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/follow/{user_id}/followers [get]
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

// GetFollowing handles GET /follow/:user_id/following
// @Summary Get users being followed
// @Description Retrieve the list of users that the specified user is following
// @Tags follow
// @Produce json
// @Param user_id path string true "User ID (UUID)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/follow/{user_id}/following [get]
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
