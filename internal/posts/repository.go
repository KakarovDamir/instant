package posts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"instant/internal/database"
)

var (
	ErrPostNotFound = errors.New("post not found")
	ErrUnauthorized = errors.New("unauthorized to modify this post")
)

// Repository handles all database operations for posts
type Repository struct {
	db database.Service
}

// NewRepository creates a new posts repository
func NewRepository(db database.Service) *Repository {
	return &Repository{db: db}
}

// Create inserts a new post into the database
func (r *Repository) Create(ctx context.Context, userID uuid.UUID, caption, imageURL string) (*Post, error) {
	query := `
		INSERT INTO posts (user_id, caption, image_url, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING post_id, user_id, caption, image_url, created_at, updated_at
	`

	post := &Post{}
	err := r.db.QueryRow(ctx, query, userID, caption, imageURL).Scan(
		&post.PostID,
		&post.UserID,
		&post.Caption,
		&post.ImageURL,
		&post.CreatedAt,
		&post.UpdatedAt,
	)

	if err != nil {
		log.Printf("Error creating post: %v", err)
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	return post, nil
}

// GetByID retrieves a single post by ID
func (r *Repository) GetByID(ctx context.Context, postID int64) (*Post, error) {
	query := `
		SELECT post_id, user_id, caption, image_url, created_at, updated_at
		FROM posts
		WHERE post_id = $1
	`

	post := &Post{}
	err := r.db.QueryRow(ctx, query, postID).Scan(
		&post.PostID,
		&post.UserID,
		&post.Caption,
		&post.ImageURL,
		&post.CreatedAt,
		&post.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrPostNotFound
	}
	if err != nil {
		log.Printf("Error getting post by ID: %v", err)
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	return post, nil
}

// GetAll retrieves all posts with pagination (ordered by newest first)
func (r *Repository) GetAll(ctx context.Context, page, pageSize int) ([]Post, int64, error) {
	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20 // Default page size
	}

	offset := (page - 1) * pageSize

	// Get total count
	var totalCount int64
	countQuery := `SELECT COUNT(*) FROM posts`
	err := r.db.QueryRow(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		log.Printf("Error counting posts: %v", err)
		return nil, 0, fmt.Errorf("failed to count posts: %w", err)
	}

	// Get paginated posts
	query := `
		SELECT post_id, user_id, caption, image_url, created_at, updated_at
		FROM posts
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.queryRows(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	return rows, totalCount, nil
}

// GetByUserID retrieves all posts by a specific user with pagination
func (r *Repository) GetByUserID(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]Post, int64, error) {
	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Get total count for user
	var totalCount int64
	countQuery := `SELECT COUNT(*) FROM posts WHERE user_id = $1`
	err := r.db.QueryRow(ctx, countQuery, userID).Scan(&totalCount)
	if err != nil {
		log.Printf("Error counting user posts: %v", err)
		return nil, 0, fmt.Errorf("failed to count user posts: %w", err)
	}

	// Get paginated user posts
	query := `
		SELECT post_id, user_id, caption, image_url, created_at, updated_at
		FROM posts
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.queryRows(ctx, query, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	return rows, totalCount, nil
}

// Update modifies an existing post (only if user owns it)
func (r *Repository) Update(ctx context.Context, postID int64, userID uuid.UUID, caption *string, imageURL *string) (*Post, error) {
	// First, verify the post exists and belongs to the user
	existing, err := r.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	if existing.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Build dynamic update query based on provided fields
	updates := make(map[string]interface{})
	if caption != nil {
		updates["caption"] = *caption
	}
	if imageURL != nil {
		updates["image_url"] = *imageURL
	}

	if len(updates) == 0 {
		return existing, nil // Nothing to update
	}

	// Construct query dynamically
	query := `UPDATE posts SET `
	args := []interface{}{}
	argPos := 1

	for field, value := range updates {
		if argPos > 1 {
			query += ", "
		}
		query += fmt.Sprintf("%s = $%d", field, argPos)
		args = append(args, value)
		argPos++
	}

	query += fmt.Sprintf(`, updated_at = NOW() WHERE post_id = $%d AND user_id = $%d
		RETURNING post_id, user_id, caption, image_url, created_at, updated_at`, argPos, argPos+1)
	args = append(args, postID, userID)

	post := &Post{}
	err = r.db.QueryRow(ctx, query, args...).Scan(
		&post.PostID,
		&post.UserID,
		&post.Caption,
		&post.ImageURL,
		&post.CreatedAt,
		&post.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrPostNotFound
	}
	if err != nil {
		log.Printf("Error updating post: %v", err)
		return nil, fmt.Errorf("failed to update post: %w", err)
	}

	return post, nil
}

// Delete removes a post (only if user owns it)
func (r *Repository) Delete(ctx context.Context, postID int64, userID uuid.UUID) error {
	// First verify ownership
	existing, err := r.GetByID(ctx, postID)
	if err != nil {
		return err
	}

	if existing.UserID != userID {
		return ErrUnauthorized
	}

	query := `DELETE FROM posts WHERE post_id = $1 AND user_id = $2`
	result, err := r.db.Exec(ctx, query, postID, userID)
	if err != nil {
		log.Printf("Error deleting post: %v", err)
		return fmt.Errorf("failed to delete post: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrPostNotFound
	}

	return nil
}

// Helper method to scan multiple rows
func (r *Repository) queryRows(ctx context.Context, query string, args ...interface{}) ([]Post, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		log.Printf("Error querying posts: %v", err)
		return nil, fmt.Errorf("failed to query posts: %w", err)
	}
	defer rows.Close()

	posts := []Post{}
	for rows.Next() {
		var post Post
		err := rows.Scan(
			&post.PostID,
			&post.UserID,
			&post.Caption,
			&post.ImageURL,
			&post.CreatedAt,
			&post.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning post row: %v", err)
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating posts: %v", err)
		return nil, fmt.Errorf("failed to iterate posts: %w", err)
	}

	return posts, nil
}
