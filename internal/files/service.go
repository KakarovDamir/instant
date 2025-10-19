package files

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"instant/internal/storage"
)

// Service handles business logic for file operations
type Service struct {
	storage storage.Service
}

// NewService creates a new files service
func NewService(storage storage.Service) *Service {
	return &Service{
		storage: storage,
	}
}

// ValidateFilename checks if filename is safe and valid
func ValidateFilename(filename string) error {
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

// ValidateContentType checks if content type is allowed
func ValidateContentType(contentType string) error {
	if contentType == "" {
		return fmt.Errorf("content type cannot be empty")
	}
	if !AllowedContentTypes[contentType] {
		return fmt.Errorf("content type %s is not allowed", contentType)
	}
	return nil
}

// GenerateUploadURL creates a presigned URL for file upload
func (s *Service) GenerateUploadURL(ctx context.Context, req *GenerateUploadURLRequest) (*GenerateUploadURLResponse, error) {
	// Validate filename
	if err := ValidateFilename(req.Filename); err != nil {
		return nil, fmt.Errorf("invalid filename: %w", err)
	}

	// Validate content type
	if err := ValidateContentType(req.ContentType); err != nil {
		return nil, fmt.Errorf("invalid content type: %w", err)
	}

	// Validate file size
	maxSize := req.MaxSize
	if maxSize <= 0 {
		maxSize = MaxFileSize
	}
	if maxSize > MaxFileSize {
		return nil, fmt.Errorf("max file size cannot exceed %d bytes", MaxFileSize)
	}

	// Generate unique file key
	fileKey := fmt.Sprintf("%s-%s", uuid.New().String(), req.Filename)

	// TTL for upload URL (15 minutes)
	ttl := 15 * time.Minute

	// Generate presigned upload URL
	uploadURL, err := s.storage.GeneratePresignedUploadURL(ctx, fileKey, req.ContentType, ttl)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload URL: %w", err)
	}

	return &GenerateUploadURLResponse{
		UploadURL: uploadURL,
		FileKey:   fileKey,
		ExpiresAt: time.Now().Add(ttl).Unix(),
	}, nil
}

// GenerateDownloadURL creates a presigned URL for file download
func (s *Service) GenerateDownloadURL(ctx context.Context, req *GenerateDownloadURLRequest) (*GenerateDownloadURLResponse, error) {
	if req.FileKey == "" {
		return nil, fmt.Errorf("file key cannot be empty")
	}

	// TTL for download URL (1 hour)
	ttl := 1 * time.Hour

	// Generate presigned download URL
	downloadURL, err := s.storage.GeneratePresignedDownloadURL(ctx, req.FileKey, ttl)
	if err != nil {
		return nil, fmt.Errorf("failed to generate download URL: %w", err)
	}

	return &GenerateDownloadURLResponse{
		DownloadURL: downloadURL,
		ExpiresAt:   time.Now().Add(ttl).Unix(),
	}, nil
}

// DeleteFile removes a file from storage
func (s *Service) DeleteFile(ctx context.Context, fileKey string) error {
	if fileKey == "" {
		return fmt.Errorf("file key cannot be empty")
	}

	if err := s.storage.DeleteFile(ctx, fileKey); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// HealthCheck checks storage service health
func (s *Service) HealthCheck(ctx context.Context) error {
	return s.storage.Health(ctx)
}
