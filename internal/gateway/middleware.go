package gateway

import (
	"log"
	"net/http"
	"time"

	"instant/internal/session"

	"github.com/gin-gonic/gin"
)

// SessionAuthMiddleware validates session and injects user context
func SessionAuthMiddleware(sessionMgr session.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session ID from cookie
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized: no session cookie",
			})
			return
		}

		// Validate and get session
		sess, err := sessionMgr.Get(c.Request.Context(), sessionID)
		if err != nil {
			log.Printf("Invalid session %s: %v", sessionID, err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized: invalid session",
			})
			return
		}

		// Double-check expiration (should be caught by Get, but be defensive)
		if time.Now().After(sess.ExpiresAt) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized: session expired",
			})
			return
		}

		// Inject user context for downstream services
		c.Set("user_id", sess.UserID)
		c.Set("email", sess.Email)

		// Add headers for proxied requests
		c.Request.Header.Set("X-User-ID", sess.UserID)
		c.Request.Header.Set("X-User-Email", sess.Email)

		c.Next()
	}
}

// CORSMiddleware handles CORS for the gateway
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// LoggingMiddleware logs all requests passing through the gateway
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		log.Printf("[Gateway] %s %s | Status: %d | Latency: %v",
			method, path, statusCode, latency)
	}
}
