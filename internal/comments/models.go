package comments

import "time"

type Comment struct {
    ID        int64     `json:"id"`
    PostID    int64     `json:"post_id"`
    UserID    string    `json:"user_id"`
    Body      string    `json:"body"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type CreateCommentRequest struct {
    PostID int64  `json:"post_id" binding:"required"`
    Body   string `json:"body" binding:"required"`
}

type UpdateCommentRequest struct {
    Body string `json:"body" binding:"required"`
}
