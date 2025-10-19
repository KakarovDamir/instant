package files

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for files service
type Handler struct {
	service *Service
}

// NewHandler creates a new files handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GenerateUploadURL handles POST /files/upload-url
func (h *Handler) GenerateUploadURL(c *gin.Context) {
	var req GenerateUploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	response, err := h.service.GenerateUploadURL(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Failed to generate upload URL",
			Code:    "GENERATION_FAILED",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GenerateDownloadURL handles POST /files/download-url
func (h *Handler) GenerateDownloadURL(c *gin.Context) {
	var req GenerateDownloadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	response, err := h.service.GenerateDownloadURL(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to generate download URL",
			Code:    "GENERATION_FAILED",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteFile handles DELETE /files/:key
func (h *Handler) DeleteFile(c *gin.Context) {
	fileKey := c.Param("key")
	if fileKey == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "File key is required",
			Code:    "INVALID_FILE_KEY",
		})
		return
	}

	if err := h.service.DeleteFile(c.Request.Context(), fileKey); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to delete file",
			Code:    "DELETE_FAILED",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "File deleted successfully",
		"file_key": fileKey,
	})
}

// Health handles GET /health
func (h *Handler) Health(c *gin.Context) {
	if err := h.service.HealthCheck(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"service": "files-service",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "files-service",
	})
}
