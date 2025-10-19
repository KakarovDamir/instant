package server

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // Add your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	r.GET("/", s.HelloWorldHandler)
	r.GET("/health", s.healthHandler)

	// File operations routes
	fileOps := r.Group("/files")
	{
		fileOps.POST("/upload-url", s.generateUploadURLHandler)
		fileOps.POST("/download-url", s.generateDownloadURLHandler)
		fileOps.DELETE("/:key", s.deleteFileHandler)
	}

	return r
}

func (s *Server) HelloWorldHandler(c *gin.Context) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	c.JSON(http.StatusOK, resp)
}

func (s *Server) healthHandler(c *gin.Context) {
	response := make(map[string]interface{})

	// Database health
	response["database"] = s.db.Health()

	// Check storage health if available
	if s.storage != nil {
		storageHealth := make(map[string]string)
		if err := s.storage.Health(c.Request.Context()); err != nil {
			storageHealth["status"] = "down"
			storageHealth["error"] = err.Error()
		} else {
			storageHealth["status"] = "up"
		}
		response["storage"] = storageHealth
	}

	c.JSON(http.StatusOK, response)
}

// Constants for file operations
const (
	MaxFilenameLength = 255
	MaxFileSize       = 100 * 1024 * 1024 // 100MB default
	MinTTL            = 1 * time.Minute
	MaxTTL            = 24 * time.Hour
)

// Allowed content types (whitelist for security)
var allowedContentTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/jpg":	   true,
	"image/gif":       true,
	"image/webp":      true,
	"application/pdf": true,
	"text/plain":      true,
	"application/json": true,
	"video/mp4":       true,
	"audio/mpeg":      true,
}

// Request/Response types for file operations
type GenerateUploadURLRequest struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
	MaxSize     int64  `json:"max_size,omitempty"` // Optional: max file size in bytes
}

type GenerateUploadURLResponse struct {
	UploadURL string `json:"upload_url"`
	FileKey   string `json:"file_key"`
	ExpiresAt int64  `json:"expires_at"` // Unix timestamp
}

type GenerateDownloadURLRequest struct {
	FileKey string `json:"file_key" binding:"required"`
}

type GenerateDownloadURLResponse struct {
	DownloadURL string `json:"download_url"`
	ExpiresAt   int64  `json:"expires_at"` // Unix timestamp
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// validateFilename checks if filename is safe and valid
func validateFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if len(filename) > MaxFilenameLength {
		return fmt.Errorf("filename too long (max %d characters)", MaxFilenameLength)
	}
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return fmt.Errorf("filename contains invalid characters")
	}
	ext := filepath.Ext(filename)
	if ext == "" {
		return fmt.Errorf("filename must have an extension")
	}
	return nil
}

// validateContentType checks if content type is allowed
func validateContentType(contentType string) error {
	if contentType == "" {
		return fmt.Errorf("content type cannot be empty")
	}
	if !allowedContentTypes[contentType] {
		return fmt.Errorf("content type %s is not allowed", contentType)
	}
	return nil
}

// generateUploadURLHandler creates a presigned URL for file upload
func (s *Server) generateUploadURLHandler(c *gin.Context) {
	if s.storage == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Storage service is not available",
			Code:  "STORAGE_UNAVAILABLE",
		})
		return
	}

	var req GenerateUploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	if err := validateFilename(req.Filename); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid filename",
			Code:    "INVALID_FILENAME",
			Details: err.Error(),
		})
		return
	}

	if err := validateContentType(req.ContentType); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid content type",
			Code:    "INVALID_CONTENT_TYPE",
			Details: err.Error(),
		})
		return
	}

	maxSize := req.MaxSize
	if maxSize <= 0 {
		maxSize = MaxFileSize
	}
	if maxSize > MaxFileSize {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   fmt.Sprintf("Max file size cannot exceed %d bytes", MaxFileSize),
			Code:    "FILE_TOO_LARGE",
		})
		return
	}

	fileKey := fmt.Sprintf("%s-%s", uuid.New().String(), req.Filename)

	ttl := 15 * time.Minute

	uploadURL, err := s.storage.GeneratePresignedUploadURL(
		c.Request.Context(),
		fileKey,
		req.ContentType,
		ttl,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate upload URL",
			Code:    "GENERATION_FAILED",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, GenerateUploadURLResponse{
		UploadURL: uploadURL,
		FileKey:   fileKey,
		ExpiresAt: time.Now().Add(ttl).Unix(),
	})
}

// generateDownloadURLHandler creates a presigned URL for file download
func (s *Server) generateDownloadURLHandler(c *gin.Context) {
	if s.storage == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Storage service is not available",
			Code:  "STORAGE_UNAVAILABLE",
		})
		return
	}

	var req GenerateDownloadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	if req.FileKey == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "File key cannot be empty",
			Code:  "INVALID_FILE_KEY",
		})
		return
	}

	ttl := 1 * time.Hour

	downloadURL, err := s.storage.GeneratePresignedDownloadURL(
		c.Request.Context(),
		req.FileKey,
		ttl,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate download URL",
			Code:    "GENERATION_FAILED",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, GenerateDownloadURLResponse{
		DownloadURL: downloadURL,
		ExpiresAt:   time.Now().Add(ttl).Unix(),
	})
}

// deleteFileHandler removes a file from storage
func (s *Server) deleteFileHandler(c *gin.Context) {
	if s.storage == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Storage service is not available",
			Code:  "STORAGE_UNAVAILABLE",
		})
		return
	}

	fileKey := c.Param("key")
	if fileKey == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "File key is required",
			Code:  "INVALID_FILE_KEY",
		})
		return
	}

	if err := s.storage.DeleteFile(c.Request.Context(), fileKey); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete file",
			Code:    "DELETE_FAILED",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "File deleted successfully",
		"file_key": fileKey,
	})
}
