package gateway

import (
	"log/slog"
	"net/http"
	"time"

	"instant/internal/session"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
			slog.Warn("Invalid session",
				"session_id", sessionID,
				"error", err.Error(),
				"request_id", c.GetString("request_id"),
			)
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

// RequestIDMiddleware generates a unique request ID for distributed tracing
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate unique request ID
		requestID := uuid.New().String()

		// Store in context for downstream use
		c.Set("request_id", requestID)

		// Add to response headers for client correlation
		c.Writer.Header().Set("X-Request-ID", requestID)

		c.Next()
	}
}

// LoggingMiddleware logs all requests passing through the gateway with structured JSON
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Wrap the response writer to capture response size
		rw := newResponseWriter(c.Writer)
		c.Writer = rw

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		latencyMs := float64(latency.Milliseconds())

		// Get request details
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		// Use Gin's writer Status() which handles aborted requests correctly
		status := c.Writer.Status()
		responseSize := rw.Size()
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()
		requestID := c.GetString("request_id")

		// Build structured log attributes
		attrs := []any{
			"request_id", requestID,
			"method", method,
			"path", path,
			"status", status,
			"latency_ms", latencyMs,
			"client_ip", clientIP,
			"user_agent", userAgent,
			"response_size", responseSize,
		}

		// Add query string if present
		if query != "" {
			attrs = append(attrs, "query", query)
		}

		// Add user context if authenticated
		if userID, exists := c.Get("user_id"); exists {
			attrs = append(attrs, "user_id", userID)
		}
		if email, exists := c.Get("email"); exists {
			attrs = append(attrs, "email", email)
		}

		// Add upstream service if this was a proxied request
		if upstreamService, exists := c.Get("upstream_service"); exists {
			attrs = append(attrs, "upstream_service", upstreamService)
		}

		// Add error details if present
		if len(c.Errors) > 0 {
			attrs = append(attrs, "error", c.Errors.String())
		}

		// Log with appropriate level based on status code
		switch {
		case status >= 500:
			// Server errors - ERROR level
			slog.Error("Request failed - server error", attrs...)
		case status >= 400:
			// Client errors - WARN level
			slog.Warn("Request failed - client error", attrs...)
		default:
			// Success - INFO level
			slog.Info("Request completed", attrs...)
		}
	}
}
