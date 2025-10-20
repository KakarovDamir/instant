package posts

import (
	"time"

	"github.com/google/uuid"
)

// Post represents a social media post with image and caption
type Post struct {
	PostID    int64     `json:"post_id" db:"post_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Caption   string    `json:"caption" db:"caption"`
	ImageURL  string    `json:"image_url" db:"image_url"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CreatePostRequest represents the request body for creating a new post
// Note: user_id is extracted from authentication context (X-User-ID header), not from request body
type CreatePostRequest struct {
	Caption  string `json:"caption" binding:"required,max=1000"`
	ImageURL string `json:"image_url" binding:"required"` // Can be file_key from MinIO or full URL
}

// UpdatePostRequest represents the request body for updating a post
type UpdatePostRequest struct {
	Caption  *string `json:"caption,omitempty" binding:"omitempty,max=1000"`
	ImageURL *string `json:"image_url,omitempty"` // Can be file_key from MinIO or full URL
}

// PaginatedPostsResponse represents paginated posts response
type PaginatedPostsResponse struct {
	Posts      []Post `json:"posts"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalCount int64  `json:"total_count"`
	TotalPages int    `json:"total_pages"`
}

// PostResponse is a standard response wrapper
type PostResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    *Post  `json:"data,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}
