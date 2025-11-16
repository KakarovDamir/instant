package email

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Handler handles HTTP requests for the email service
type Handler struct {
	redis  *redis.Client
	store  *IdempotencyStore
	logger *slog.Logger
}

// NewHandler creates a new email service handler
func NewHandler(redisClient *redis.Client, store *IdempotencyStore, logger *slog.Logger) *Handler {
	return &Handler{
		redis:  redisClient,
		store:  store,
		logger: logger,
	}
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	ctx := context.Background()

	// Check Redis connection
	redisStatus := "connected"
	if err := h.redis.Ping(ctx).Err(); err != nil {
		redisStatus = "disconnected"
		h.logger.Error("Redis health check failed", "error", err)
	}

	// Get idempotency store stats
	recordCount, err := h.store.Clean(ctx)
	if err != nil {
		h.logger.Error("Failed to get idempotency stats", "error", err)
		recordCount = -1
	}

	status := "healthy"
	httpStatus := http.StatusOK
	if redisStatus != "connected" {
		status = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"status":                status,
		"service":               "email-service",
		"redis":                 redisStatus,
		"idempotency_records":   recordCount,
		"timestamp":             c.GetTime("timestamp"),
	})
}

// Stats handles GET /stats (optional endpoint for monitoring)
func (h *Handler) Stats(c *gin.Context) {
	ctx := context.Background()

	// Get idempotency store stats
	recordCount, err := h.store.Clean(ctx)
	if err != nil {
		h.logger.Error("Failed to get stats", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve stats",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"idempotency_records": recordCount,
		"ttl_hours":           24,
	})
}
