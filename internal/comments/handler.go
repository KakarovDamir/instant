package comments

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
)

type Handler struct {
    svc Service
}

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

func getUserID(c *gin.Context) string {
    return c.GetHeader("X-User-ID")
}

// Create handles POST /
// @Summary Create a comment
// @Description Create a new comment on a post (requires authentication)
// @Tags comments
// @Accept json
// @Produce json
// @Param comment body CreateCommentRequest true "Comment creation data"
// @Success 201 {object} Comment
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security SessionAuth
// @Router /api/comments [post]
func (h *Handler) Create(c *gin.Context) {
    userID := getUserID(c)
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }

    var req CreateCommentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
        return
    }

    comment, err := h.svc.Create(c.Request.Context(), userID, req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, comment)
}

// Update handles PATCH /:id
// @Summary Update a comment
// @Description Update comment body (requires authentication and ownership)
// @Tags comments
// @Accept json
// @Produce json
// @Param id path int true "Comment ID"
// @Param comment body UpdateCommentRequest true "Comment update data"
// @Success 200 {object} Comment
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security SessionAuth
// @Router /api/comments/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
    userID := getUserID(c)
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }

    id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

    var req UpdateCommentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
        return
    }

    comment, err := h.svc.Update(c.Request.Context(), userID, id, req.Body)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, comment)
}

// Delete handles DELETE /:id
// @Summary Delete a comment
// @Description Delete a comment by ID (requires authentication and ownership)
// @Tags comments
// @Produce json
// @Param id path int true "Comment ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security SessionAuth
// @Router /api/comments/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
    userID := getUserID(c)
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }

    id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

    if err := h.svc.Delete(c.Request.Context(), userID, id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// List handles GET /post/:post_id
// @Summary Get comments for a post
// @Description Retrieve all comments for a specific post
// @Tags comments
// @Produce json
// @Param post_id path int true "Post ID"
// @Success 200 {array} Comment
// @Failure 500 {object} map[string]string
// @Router /api/comments/post/{post_id} [get]
func (h *Handler) List(c *gin.Context) {
    postID, _ := strconv.ParseInt(c.Param("post_id"), 10, 64)

    comments, err := h.svc.ListByPost(c.Request.Context(), postID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, comments)
}

// GET /health
func (h *Handler) Health(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status":  "healthy",
        "service": "comments-service",
    })
}
