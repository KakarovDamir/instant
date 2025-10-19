package files

import "time"

// GenerateUploadURLRequest represents request for upload URL generation
type GenerateUploadURLRequest struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
	MaxSize     int64  `json:"max_size,omitempty"` // Optional: max file size in bytes
}

// GenerateUploadURLResponse represents response with presigned upload URL
type GenerateUploadURLResponse struct {
	UploadURL string `json:"upload_url"`
	FileKey   string `json:"file_key"`
	ExpiresAt int64  `json:"expires_at"` // Unix timestamp
}

// GenerateDownloadURLRequest represents request for download URL generation
type GenerateDownloadURLRequest struct {
	FileKey string `json:"file_key" binding:"required"`
}

// GenerateDownloadURLResponse represents response with presigned download URL
type GenerateDownloadURLResponse struct {
	DownloadURL string `json:"download_url"`
	ExpiresAt   int64  `json:"expires_at"` // Unix timestamp
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// Constants for file operations
const (
	MaxFilenameLength = 255
	MaxFileSize       = 100 * 1024 * 1024 // 100MB default
	MinTTL            = 1 * time.Minute
	MaxTTL            = 24 * time.Hour
)

// AllowedContentTypes defines whitelist for security
var AllowedContentTypes = map[string]bool{
	"image/jpeg":       true,
	"image/png":        true,
	"image/jpg":        true,
	"image/gif":        true,
	"image/webp":       true,
	"application/pdf":  true,
	"text/plain":       true,
	"application/json": true,
	"video/mp4":        true,
	"audio/mpeg":       true,
}
