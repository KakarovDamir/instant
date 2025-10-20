package likes

import "time"

type Like struct {
	ID        string    `json:"id"`
	PostID    string    `json:"post_id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type LikeRequest struct {
	PostID string `json:"post_id" binding:"required"`
}

type CountResponse struct {
	PostID string `json:"post_id"`
	Count  int64  `json:"count"`
}

type LikedResponse struct {
	PostID string `json:"post_id"`
	Liked  bool   `json:"liked"`
}
