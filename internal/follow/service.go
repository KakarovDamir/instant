package follow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"instant/internal/database"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput = errors.New("invalid input")
)

type Service interface {
	Follow(ctx context.Context, followerID, followeeID string) (*Follow, error)
	Unfollow(ctx context.Context, followerID, followeeID string) (int64, error)
	FollowersCount(ctx context.Context, userID string) (int64, error)
	FollowingCount(ctx context.Context, userID string) (int64, error)
	IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error)
}

type service struct {
	db database.Service
}

func NewService(db database.Service) Service {
	return &service{db: db}
}

func (s *service) Follow(ctx context.Context, followerID, followeeID string) (*Follow, error) {
	if followerID == "" || followeeID == "" {
		return nil, ErrInvalidInput
	}

	f := &Follow{
		ID:         uuid.New().String(),
		FollowerID: followerID,
		FolloweeID: followeeID,
		CreatedAt:  time.Now(),
	}

	const q = `
		INSERT INTO follow (id, follower_id, followee_id, created_at)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (follower_id, followee_id) DO NOTHING
	`

	if _, err := s.db.Exec(ctx, q, f.ID, f.FollowerID, f.FolloweeID, f.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert follow: %w", err)
	}

	return f, nil
}

func (s *service) Unfollow(ctx context.Context, followerID, followeeID string) (int64, error) {
	const q = `DELETE FROM follow WHERE follower_id=$1 AND followee_id=$2`

	res, err := s.db.Exec(ctx, q, followerID, followeeID)
	if err != nil {
		return 0, fmt.Errorf("delete follow: %w", err)
	}

	return res.RowsAffected()
}

func (s *service) FollowersCount(ctx context.Context, userID string) (int64, error) {
	const q = `SELECT COUNT(*) FROM follow WHERE followee_id=$1`

	var cnt int64
	err := s.db.QueryRow(ctx, q, userID).Scan(&cnt)
	return cnt, err
}

func (s *service) FollowingCount(ctx context.Context, userID string) (int64, error) {
	const q = `SELECT COUNT(*) FROM follow WHERE follower_id=$1`

	var cnt int64
	err := s.db.QueryRow(ctx, q, userID).Scan(&cnt)
	return cnt, err
}

func (s *service) IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error) {
	const q = `SELECT 1 FROM follow WHERE follower_id=$1 AND followee_id=$2 LIMIT 1`

	var one int
	err := s.db.QueryRow(ctx, q, followerID, followeeID).Scan(&one)

	if err != nil {
		return false, nil
	}
	return true, nil
}
