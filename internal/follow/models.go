package follow

import (
	"time"

	"github.com/google/uuid"
)

type Follow struct {
	FollowID    int64     `json:"follow_id" db:"follow_id"`
	FollowerID  uuid.UUID `json:"follower_id" db:"follower_id"`    // user who follows
	FollowingID uuid.UUID `json:"following_id" db:"following_id"` // user being followed
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type FollowRequest struct {
	TargetUserID uuid.UUID `json:"target_user_id" binding:"required"`
}

type FollowResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}
