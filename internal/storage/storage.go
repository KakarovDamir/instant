// Package storage provides S3-compatible object storage functionality using MinIO.
// It handles presigned URLs for secure file uploads and downloads, file deletion,
// and storage health checks.
package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Service defines the interface for storage operations
type Service interface {
	// GeneratePresignedUploadURL creates a time-limited presigned URL for uploading a file
	GeneratePresignedUploadURL(ctx context.Context, key string, contentType string, ttl time.Duration) (string, error)

	// GeneratePresignedDownloadURL creates a time-limited presigned URL for downloading a file
	GeneratePresignedDownloadURL(ctx context.Context, key string, ttl time.Duration) (string, error)

	// DeleteFile removes a file from storage
	DeleteFile(ctx context.Context, key string) error

	// EnsureBucketExists creates the bucket if it doesn't exist
	EnsureBucketExists(ctx context.Context) error

	// Health checks if the storage service is accessible
	Health(ctx context.Context) error
}

type service struct {
	client          *s3.Client
	presigner       *s3.PresignClient
	publicPresigner *s3.PresignClient
	bucketName      string
	publicEndpoint  string
	useSSL          bool
}

// New creates a new storage service instance configured for MinIO
func New(ctx context.Context) (Service, error) {
	endpoint := os.Getenv("S3_ENDPOINT")
	publicEndpoint := os.Getenv("S3_PUBLIC_ENDPOINT")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	bucketName := os.Getenv("S3_BUCKET_NAME")
	useSSL := os.Getenv("S3_USE_SSL") == "true"

	// Validate required environment variables
	if endpoint == "" {
		return nil, fmt.Errorf("S3_ENDPOINT environment variable is required")
	}
	if accessKey == "" {
		return nil, fmt.Errorf("S3_ACCESS_KEY environment variable is required")
	}
	if secretKey == "" {
		return nil, fmt.Errorf("S3_SECRET_KEY environment variable is required")
	}
	if bucketName == "" {
		return nil, fmt.Errorf("S3_BUCKET_NAME environment variable is required")
	}

	// Use internal endpoint for presigned URLs if public endpoint not specified
	if publicEndpoint == "" {
		publicEndpoint = endpoint
		log.Printf("[Storage] Using internal endpoint for presigned URLs: %s", endpoint)
	} else {
		log.Printf("[Storage] Using public endpoint for presigned URLs: %s", publicEndpoint)
	}

	protocol := "http"
	if useSSL {
		protocol = "https"
	}
	endpointURL := fmt.Sprintf("%s://%s", protocol, endpoint)

	// Create custom resolver for MinIO endpoint
	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               endpointURL,
				SigningRegion:     "us-east-1",
				HostnameImmutable: true,
			}, nil
		},
	)

	// Load AWS config with MinIO settings
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with path-style addressing (required for MinIO)
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	presigner := s3.NewPresignClient(client)

	// Create separate presigner for public endpoint if different
	var publicPresigner *s3.PresignClient
	if publicEndpoint != endpoint {
		publicEndpointURL := fmt.Sprintf("%s://%s", protocol, publicEndpoint)

		publicResolver := aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               publicEndpointURL,
					SigningRegion:     "us-east-1",
					HostnameImmutable: true,
				}, nil
			},
		)

		publicCfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion("us-east-1"),
			config.WithEndpointResolverWithOptions(publicResolver),
			config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load public AWS config: %w", err)
		}

		publicClient := s3.NewFromConfig(publicCfg, func(o *s3.Options) {
			o.UsePathStyle = true
		})

		publicPresigner = s3.NewPresignClient(publicClient)
	} else {
		publicPresigner = presigner
	}

	s := &service{
		client:          client,
		presigner:       presigner,
		publicPresigner: publicPresigner,
		bucketName:      bucketName,
		publicEndpoint:  publicEndpoint,
		useSSL:          useSSL,
	}

	// Ensure bucket exists on initialization
	if err := s.EnsureBucketExists(ctx); err != nil {
		log.Printf("Warning: failed to ensure bucket exists: %v", err)
	}

	return s, nil
}

// EnsureBucketExists creates the bucket if it doesn't already exist
func (s *service) EnsureBucketExists(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucketName),
	})

	if err == nil {
		return nil // Bucket already exists
	}

	// Create bucket
	_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucketName),
	})

	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	log.Printf("Created S3 bucket: %s", s.bucketName)
	return nil
}

// GeneratePresignedUploadURL creates a presigned URL for uploading
func (s *service) GeneratePresignedUploadURL(ctx context.Context, key string, contentType string, ttl time.Duration) (string, error) {
	if key == "" {
		return "", fmt.Errorf("file key cannot be empty")
	}
	if contentType == "" {
		return "", fmt.Errorf("content type cannot be empty")
	}
	if ttl <= 0 {
		return "", fmt.Errorf("TTL must be positive")
	}

	putObjectInput := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}

	request, err := s.publicPresigner.PresignPutObject(ctx, putObjectInput, func(opts *s3.PresignOptions) {
		opts.Expires = ttl
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL for key %s: %w", key, err)
	}

	return request.URL, nil
}

// GeneratePresignedDownloadURL creates a presigned URL for downloading
func (s *service) GeneratePresignedDownloadURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if key == "" {
		return "", fmt.Errorf("file key cannot be empty")
	}
	if ttl <= 0 {
		return "", fmt.Errorf("TTL must be positive")
	}

	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	request, err := s.publicPresigner.PresignGetObject(ctx, getObjectInput, func(opts *s3.PresignOptions) {
		opts.Expires = ttl
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL for key %s: %w", key, err)
	}

	return request.URL, nil
}

// DeleteFile removes a file from storage
func (s *service) DeleteFile(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("file key cannot be empty")
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete file %s: %w", key, err)
	}

	return nil
}

// Health checks if the storage service is accessible
func (s *service) Health(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucketName),
	})

	if err != nil {
		return fmt.Errorf("storage health check failed: %w", err)
	}

	return nil
}
