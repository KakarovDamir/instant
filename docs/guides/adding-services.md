# Adding New Microservices

Step-by-step guide for adding new microservices to the Instant Platform following Go best practices and the existing architecture patterns.

## Overview

This guide walks through creating a new service from scratch. We'll use a "Comments Service" as our example, but the pattern applies to any service (likes, follow, notifications, etc.).

## Prerequisites

- Platform running locally (see [Getting Started](./getting-started.md))
- Understanding of Go basics
- Familiarity with Gin web framework
- Knowledge of Consul service discovery

## Service Creation Steps

### 1. Create Service Entry Point

Create the main entry point in `cmd/`:

```bash
mkdir -p cmd/comments
touch cmd/comments/main.go
```

**cmd/comments/main.go:**
```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"instant/internal/comments"
	"instant/internal/config"
	"instant/internal/consul"
	"instant/internal/database"
)

func main() {
	// Load configuration
	cfg := loadConfig()

	// Initialize database
	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize service
	commentsService := comments.NewService(db)

	// Setup router
	router := comments.SetupRouter(commentsService)

	// Consul setup
	consulClient := consul.NewClientWithToken(cfg.Consul.Address, cfg.Consul.Token)

	serviceID := fmt.Sprintf("comments-service-%s", getHostname())
	serviceConfig := &consul.ServiceConfig{
		ID:      serviceID,
		Name:    "comments-service",
		Port:    cfg.Service.Port,
		Address: cfg.Service.Host,
		Check: &consul.HealthCheck{
			HTTP:     fmt.Sprintf("http://%s:%d/health", cfg.Service.Host, cfg.Service.Port),
			Interval: "10s",
			Timeout:  "5s",
		},
	}

	// Register with Consul
	registrar := consul.NewServiceRegistrar(consulClient)
	if err := registrar.Register(serviceConfig); err != nil {
		log.Fatalf("Failed to register service: %v", err)
	}
	log.Printf("Registered with Consul as %s", serviceID)

	// Start HTTP server
	addr := fmt.Sprintf(":%d", cfg.Service.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Comments service starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down comments service...")

	// Deregister from Consul
	if err := registrar.Deregister(serviceID); err != nil {
		log.Printf("Failed to deregister: %v", err)
	}

	// Shutdown server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Comments service stopped")
}

func loadConfig() *config.Config {
	return &config.Config{
		Service: config.ServiceConfig{
			Port: getEnvAsInt("COMMENTS_SERVICE_PORT", 8083),
			Host: getEnv("COMMENTS_SERVICE_HOST", "comments-service"),
		},
		Database: config.DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			Database: getEnv("DB_DATABASE", "blueprint"),
			Username: getEnv("DB_USERNAME", "melkey"),
			Password: getEnv("DB_PASSWORD", "password1234"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Consul: config.ConsulConfig{
			Address: getEnv("CONSUL_HTTP_ADDR", "localhost:8500"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	// Implementation omitted for brevity
	return defaultValue
}

func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}
```

### 2. Create Service Logic

Create the service package in `internal/`:

```bash
mkdir -p internal/comments
touch internal/comments/{service.go,handler.go,router.go,models.go}
```

**internal/comments/models.go:**
```go
// Package comments provides comment management functionality for posts.
// It handles comment creation, retrieval, updates, and deletion with
// user authentication and post association.
package comments

import (
	"time"
)

// Comment represents a user comment on a post.
type Comment struct {
	// ID is the unique comment identifier
	ID string `json:"id"`

	// PostID is the identifier of the post this comment belongs to
	PostID string `json:"post_id"`

	// UserID is the identifier of the user who created this comment
	UserID string `json:"user_id"`

	// Content is the comment text
	Content string `json:"content"`

	// CreatedAt is when the comment was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the comment was last modified
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateCommentRequest represents the payload for creating a comment.
type CreateCommentRequest struct {
	PostID  string `json:"post_id" binding:"required"`
	Content string `json:"content" binding:"required,min=1,max=500"`
}

// UpdateCommentRequest represents the payload for updating a comment.
type UpdateCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=500"`
}
```

**internal/comments/service.go:**
```go
package comments

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

var (
	// ErrCommentNotFound indicates the comment does not exist
	ErrCommentNotFound = errors.New("comment not found")

	// ErrUnauthorized indicates user is not authorized for this operation
	ErrUnauthorized = errors.New("unauthorized")
)

// Service defines the comments service interface.
type Service interface {
	// CreateComment creates a new comment on a post.
	CreateComment(ctx context.Context, userID string, req *CreateCommentRequest) (*Comment, error)

	// GetComment retrieves a comment by ID.
	GetComment(ctx context.Context, commentID string) (*Comment, error)

	// GetCommentsByPost retrieves all comments for a post.
	GetCommentsByPost(ctx context.Context, postID string) ([]*Comment, error)

	// UpdateComment updates an existing comment.
	// Only the comment author can update their comment.
	UpdateComment(ctx context.Context, userID, commentID string, req *UpdateCommentRequest) (*Comment, error)

	// DeleteComment deletes a comment.
	// Only the comment author can delete their comment.
	DeleteComment(ctx context.Context, userID, commentID string) error
}

type service struct {
	db *sql.DB
}

// NewService creates a new comments service with database connection.
func NewService(db *sql.DB) Service {
	return &service{db: db}
}

func (s *service) CreateComment(ctx context.Context, userID string, req *CreateCommentRequest) (*Comment, error) {
	comment := &Comment{
		ID:      uuid.New().String(),
		PostID:  req.PostID,
		UserID:  userID,
		Content: req.Content,
	}

	query := `
		INSERT INTO comments (id, post_id, user_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	err := s.db.QueryRowContext(
		ctx,
		query,
		comment.ID,
		comment.PostID,
		comment.UserID,
		comment.Content,
	).Scan(&comment.CreatedAt, &comment.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *service) GetComment(ctx context.Context, commentID string) (*Comment, error) {
	comment := &Comment{}

	query := `
		SELECT id, post_id, user_id, content, created_at, updated_at
		FROM comments
		WHERE id = $1
	`

	err := s.db.QueryRowContext(ctx, query, commentID).Scan(
		&comment.ID,
		&comment.PostID,
		&comment.UserID,
		&comment.Content,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrCommentNotFound
	}
	if err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *service) GetCommentsByPost(ctx context.Context, postID string) ([]*Comment, error) {
	query := `
		SELECT id, post_id, user_id, content, created_at, updated_at
		FROM comments
		WHERE post_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*Comment
	for rows.Next() {
		comment := &Comment{}
		err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&comment.UserID,
			&comment.Content,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}

	return comments, rows.Err()
}

func (s *service) UpdateComment(ctx context.Context, userID, commentID string, req *UpdateCommentRequest) (*Comment, error) {
	// Verify ownership
	existing, err := s.GetComment(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if existing.UserID != userID {
		return nil, ErrUnauthorized
	}

	query := `
		UPDATE comments
		SET content = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, post_id, user_id, content, created_at, updated_at
	`

	comment := &Comment{}
	err = s.db.QueryRowContext(ctx, query, req.Content, commentID).Scan(
		&comment.ID,
		&comment.PostID,
		&comment.UserID,
		&comment.Content,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *service) DeleteComment(ctx context.Context, userID, commentID string) error {
	// Verify ownership
	existing, err := s.GetComment(ctx, commentID)
	if err != nil {
		return err
	}
	if existing.UserID != userID {
		return ErrUnauthorized
	}

	query := `DELETE FROM comments WHERE id = $1`
	_, err = s.db.ExecContext(ctx, query, commentID)
	return err
}
```

**internal/comments/handler.go:**
```go
package comments

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for the comments service.
type Handler struct {
	service Service
}

// NewHandler creates a new comments HTTP handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// CreateComment handles POST /comments
func (h *Handler) CreateComment(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	comment, err := h.service.CreateComment(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create comment"})
		return
	}

	c.JSON(http.StatusCreated, comment)
}

// GetComment handles GET /comments/:id
func (h *Handler) GetComment(c *gin.Context) {
	commentID := c.Param("id")

	comment, err := h.service.GetComment(c.Request.Context(), commentID)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get comment"})
		return
	}

	c.JSON(http.StatusOK, comment)
}

// GetPostComments handles GET /posts/:post_id/comments
func (h *Handler) GetPostComments(c *gin.Context) {
	postID := c.Param("post_id")

	comments, err := h.service.GetCommentsByPost(c.Request.Context(), postID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get comments"})
		return
	}

	c.JSON(http.StatusOK, comments)
}

// UpdateComment handles PUT /comments/:id
func (h *Handler) UpdateComment(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	commentID := c.Param("id")

	var req UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	comment, err := h.service.UpdateComment(c.Request.Context(), userID, commentID, &req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update comment"})
		return
	}

	c.JSON(http.StatusOK, comment)
}

// DeleteComment handles DELETE /comments/:id
func (h *Handler) DeleteComment(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	commentID := c.Param("id")

	err := h.service.DeleteComment(c.Request.Context(), userID, commentID)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "comment deleted"})
}

// Health handles GET /health
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "comments-service",
	})
}
```

**internal/comments/router.go:**
```go
package comments

import (
	"github.com/gin-gonic/gin"
)

// SetupRouter configures the Gin router for the comments service.
func SetupRouter(service Service) *gin.Engine {
	router := gin.Default()

	handler := NewHandler(service)

	// Health check (no auth required)
	router.GET("/health", handler.Health)

	// Comment routes (auth required via gateway X-User-ID header)
	router.POST("/comments", handler.CreateComment)
	router.GET("/comments/:id", handler.GetComment)
	router.PUT("/comments/:id", handler.UpdateComment)
	router.DELETE("/comments/:id", handler.DeleteComment)

	// Post-specific comments
	router.GET("/posts/:post_id/comments", handler.GetPostComments)

	return router
}
```

### 3. Database Migration

Create migration files:

```bash
touch migrations/003_create_comments_table.sql
```

**migrations/003_create_comments_table.sql:**
```sql
CREATE TABLE comments (
    id UUID PRIMARY KEY,
    post_id UUID NOT NULL,
    user_id UUID NOT NULL,
    content TEXT NOT NULL CHECK (char_length(content) BETWEEN 1 AND 500),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_comments_post_id ON comments(post_id);
CREATE INDEX idx_comments_user_id ON comments(user_id);
CREATE INDEX idx_comments_created_at ON comments(created_at DESC);
```

Apply migration:
```bash
docker exec -it instant-psql_bp-1 psql -U melkey -d blueprint -f /migrations/003_create_comments_table.sql
```

### 4. Update Gateway Router

Edit `internal/gateway/router.go` to add routes for the new service:

```go
// Add to protected routes
api := r.Group("/api")
api.Use(authMiddleware)
{
    // Existing routes...
    api.Any("/posts/*path", reverseProxy("posts-service"))

    // Add new comments routes
    api.Any("/comments/*path", reverseProxy("comments-service"))
}
```

### 5. Update Docker Compose

Edit `docker-compose.yml` to add the new service:

```yaml
services:
  # ... existing services ...

  comments-service:
    build:
      context: .
      dockerfile: Dockerfile
    command: ["./bin/comments"]
    ports:
      - "8083:8083"
    environment:
      - COMMENTS_SERVICE_PORT=8083
      - COMMENTS_SERVICE_HOST=comments-service
      - CONSUL_HTTP_ADDR=consul:8500
      - DB_HOST=psql_bp
      - DB_PORT=5432
      - DB_DATABASE=blueprint
      - DB_USERNAME=melkey
      - DB_PASSWORD=password1234
    depends_on:
      - consul
      - psql_bp
    networks:
      - instant_network
```

### 6. Update Dockerfile

Edit `Dockerfile` to build the new service:

```dockerfile
# Add to build stage
RUN go build -o bin/comments cmd/comments/main.go

# Service will be started via docker-compose command
```

### 7. Update Makefile

Add build and run targets:

```makefile
build-comments:
	@echo "Building comments service..."
	@go build -o bin/comments cmd/comments/main.go

run-comments:
	@echo "Running comments service..."
	@go run cmd/comments/main.go

# Update build-all target
build: build-gateway build-auth build-posts build-comments
```

### 8. Update .env

Add configuration:

```bash
# Comments Service
COMMENTS_SERVICE_PORT=8083
COMMENTS_SERVICE_HOST=comments-service
```

### 9. Test the New Service

```bash
# Build and start
docker-compose up --build comments-service

# Or locally
make build-comments
make run-comments

# Health check
curl http://localhost:8083/health

# Via gateway (requires authentication)
curl -X POST http://localhost:8080/api/comments \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"post_id":"post-uuid","content":"Great post!"}'

# Get comments for a post
curl http://localhost:8080/api/posts/post-uuid/comments \
  -b cookies.txt
```

## Testing the New Service

Create test file `internal/comments/service_test.go`:

```go
package comments_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"instant/internal/comments"
	"instant/internal/database"
)

func TestCreateComment(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	defer db.Close()

	svc := comments.NewService(db)

	req := &comments.CreateCommentRequest{
		PostID:  "test-post-id",
		Content: "Test comment",
	}

	comment, err := svc.CreateComment(context.Background(), "user-id", req)
	assert.NoError(t, err)
	assert.NotEmpty(t, comment.ID)
	assert.Equal(t, "Test comment", comment.Content)
}
```

Run tests:
```bash
go test ./internal/comments -v
```

## Best Practices Checklist

When adding a new service, ensure:

- [ ] Package-level godoc comment explains service purpose
- [ ] All exported types and functions have doc comments
- [ ] Service implements graceful shutdown
- [ ] Service registers with Consul on startup
- [ ] Service deregisters from Consul on shutdown
- [ ] Health check endpoint returns 200 OK
- [ ] All database queries use context
- [ ] Errors are properly defined and handled
- [ ] User authorization is checked via X-User-ID header
- [ ] Tests cover core functionality
- [ ] Database migrations are included
- [ ] Gateway routes are updated
- [ ] Docker Compose configuration is added
- [ ] Environment variables are documented
- [ ] Makefile targets are added

## Common Patterns

### Authentication

Services receive user context via headers from the gateway:
```go
userID := c.GetHeader("X-User-ID")
email := c.GetHeader("X-User-Email")
```

### Error Handling

Define package-level errors:
```go
var (
    ErrNotFound = errors.New("resource not found")
    ErrUnauthorized = errors.New("unauthorized")
)
```

Use `errors.Is()` for checking:
```go
if errors.Is(err, comments.ErrNotFound) {
    c.JSON(404, gin.H{"error": "not found"})
}
```

### Database Queries

Always use context and prepared statements:
```go
err := db.QueryRowContext(ctx, "SELECT * FROM table WHERE id = $1", id).Scan(&result)
```

### Consul Registration

Follow the existing pattern:
```go
serviceConfig := &consul.ServiceConfig{
    ID:      fmt.Sprintf("service-name-%s", hostname),
    Name:    "service-name",
    Port:    port,
    Check:   &consul.HealthCheck{...},
}
```

## Related Documentation

- [Getting Started](./getting-started.md) - Development setup
- [Testing Guide](./testing.md) - Writing tests
- [API Reference](../api/) - API documentation patterns
- [Architecture](../../ARCHITECTURE.md) - System architecture

## Next Steps

- Add integration tests
- Add API documentation
- Monitor service in Consul UI
- Test with load balancing (multiple instances)
- Add metrics and logging
