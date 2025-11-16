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

// POST /
// Create comment
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

// PATCH /:id
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

// DELETE /:id
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

// GET /post/:post_id
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
