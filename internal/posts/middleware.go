package posts

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthMiddleware extracts user information from headers set by API Gateway
// Gateway sets X-User-ID and X-User-Email after validating the session
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user_id from header (set by gateway after session validation)
		userIDStr := c.GetHeader("X-User-ID")
		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Success: false,
				Error:   "Unauthorized: missing user authentication",
			})
			c.Abort()
			return
		}

		// Parse and validate UUID
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Success: false,
				Error:   "Unauthorized: invalid user ID",
			})
			c.Abort()
			return
		}

		// Get email from header (optional but useful)
		email := c.GetHeader("X-User-Email")

		// Store in context for handlers to access
		c.Set("user_id", userID)
		c.Set("email", email)

		c.Next()
	}
}

// OptionalAuthMiddleware extracts user info if present, but doesn't require it
// Useful for endpoints that can work with or without authentication
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetHeader("X-User-ID")
		if userIDStr != "" {
			userID, err := uuid.Parse(userIDStr)
			if err == nil {
				c.Set("user_id", userID)
				c.Set("email", c.GetHeader("X-User-Email"))
			}
		}
		c.Next()
	}
}

// GetUserID is a helper to extract user_id from context
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	userID, ok := value.(uuid.UUID)
	return userID, ok
}
