package follow

import "time"

type Follow struct {
	ID         string    `json:"id"`
	FollowerID string    `json:"follower_id"`
	FolloweeID string    `json:"followee_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type FollowRequest struct {
	FolloweeID string `json:"followee_id" binding:"required"`
}

type CountResponse struct {
	UserID string `json:"user_id"`
	Count  int64  `json:"count"`
}

type FollowingResponse struct {
	UserID   string `json:"user_id"`
	Following bool  `json:"following"`
}
