package posts

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for posts
type Handler struct {
	service *Service
}

// NewHandler creates a new posts handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CreatePost handles POST /posts
// @Summary Create a new post
// @Description Create a new post with caption and image URL (requires authentication)
// @Tags posts
// @Accept json
// @Produce json
// @Param post body CreatePostRequest true "Post creation data"
// @Success 201 {object} PostResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security SessionAuth
// @Router /api/posts [post]
func (h *Handler) CreatePost(c *gin.Context) {
	// Get authenticated user ID from context (set by AuthMiddleware)
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Unauthorized: user not authenticated",
		})
		return
	}

	// Simplified request - no need to send user_id in body
	type CreatePostBody struct {
		Caption  string `json:"caption" binding:"required,max=1000"`
		ImageURL string `json:"image_url" binding:"required"` // Can be file_key or full URL
	}

	var req CreatePostBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	post, err := h.service.CreatePost(c.Request.Context(), userID, req.Caption, req.ImageURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to create post: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, PostResponse{
		Success: true,
		Message: "Post created successfully",
		Data:    post,
	})
}

// GetPost handles GET /posts/:id
// @Summary Get a post by ID
// @Description Retrieve a single post by its ID
// @Tags posts
// @Produce json
// @Param id path int true "Post ID"
// @Success 200 {object} PostResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/posts/{id} [get]
func (h *Handler) GetPost(c *gin.Context) {
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid post ID",
		})
		return
	}

	post, err := h.service.GetPost(c.Request.Context(), postID)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error:   "Post not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to retrieve post",
		})
		return
	}

	c.JSON(http.StatusOK, PostResponse{
		Success: true,
		Data:    post,
	})
}

// GetAllPosts handles GET /posts with pagination
// @Summary Get all posts
// @Description Retrieve all posts with pagination support
// @Tags posts
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /api/posts [get]
func (h *Handler) GetAllPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	posts, err := h.service.GetAllPosts(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to retrieve posts",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    posts,
	})
}

// GetUserPosts handles GET /users/:user_id/posts
// @Summary Get posts by user
// @Description Retrieve all posts created by a specific user with pagination
// @Tags posts
// @Produce json
// @Param user_id path string true "User ID (UUID)"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/users/{user_id}/posts [get]
func (h *Handler) GetUserPosts(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid user ID",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	posts, err := h.service.GetUserPosts(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to retrieve user posts",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    posts,
	})
}

// UpdatePost handles PATCH /posts/:id
// @Summary Update a post
// @Description Update post caption and/or image URL (requires authentication and ownership)
// @Tags posts
// @Accept json
// @Produce json
// @Param id path int true "Post ID"
// @Param post body UpdatePostRequest true "Post update data"
// @Success 200 {object} PostResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security SessionAuth
// @Router /api/posts/{id} [patch]
func (h *Handler) UpdatePost(c *gin.Context) {
	// Get authenticated user ID from context
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Unauthorized: user not authenticated",
		})
		return
	}

	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid post ID",
		})
		return
	}

	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	post, err := h.service.UpdatePost(c.Request.Context(), postID, userID, req.Caption, req.ImageURL)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error:   "Post not found",
			})
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Success: false,
				Error:   "You are not authorized to update this post",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to update post",
		})
		return
	}

	c.JSON(http.StatusOK, PostResponse{
		Success: true,
		Message: "Post updated successfully",
		Data:    post,
	})
}

// DeletePost handles DELETE /posts/:id
// @Summary Delete a post
// @Description Delete a post by ID (requires authentication and ownership)
// @Tags posts
// @Produce json
// @Param id path int true "Post ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security SessionAuth
// @Router /api/posts/{id} [delete]
func (h *Handler) DeletePost(c *gin.Context) {
	// Get authenticated user ID from context
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Unauthorized: user not authenticated",
		})
		return
	}

	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid post ID",
		})
		return
	}

	err = h.service.DeletePost(c.Request.Context(), postID, userID)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error:   "Post not found",
			})
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Success: false,
				Error:   "You are not authorized to delete this post",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to delete post",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Post deleted successfully",
	})
}

// Health handles GET /health
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "posts-service",
	})
}
