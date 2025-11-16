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
// @Summary Generate presigned upload URL
// @Description Generate a presigned URL for uploading a file to S3/MinIO (requires authentication)
// @Tags files
// @Accept json
// @Produce json
// @Param file body GenerateUploadURLRequest true "File upload request"
// @Success 200 {object} GenerateUploadURLResponse
// @Failure 400 {object} ErrorResponse
// @Security SessionAuth
// @Router /api/files/upload-url [post]
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
// @Summary Generate presigned download URL
// @Description Generate a presigned URL for downloading a file from S3/MinIO
// @Tags files
// @Accept json
// @Produce json
// @Param file body GenerateDownloadURLRequest true "File download request"
// @Success 200 {object} GenerateDownloadURLResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/files/download-url [post]
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
// @Summary Delete a file
// @Description Delete a file from S3/MinIO storage (requires authentication)
// @Tags files
// @Produce json
// @Param key path string true "File key"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security SessionAuth
// @Router /api/files/{key} [delete]
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
