package likes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"instant/internal/database"

	"github.com/google/uuid"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrInvalidInput = errors.New("invalid input")
)

type Service interface {
	Like(ctx context.Context, userID, postID string) (*Like, error)
	Unlike(ctx context.Context, userID, postID string) (int64, error)
	Count(ctx context.Context, postID string) (int64, error)
	IsLiked(ctx context.Context, userID, postID string) (bool, error)
}

type service struct {
	db database.Service
}

func NewService(db database.Service) Service {
	return &service{db: db}
}

func (s *service) Like(ctx context.Context, userID, postID string) (*Like, error) {
	if userID == "" || postID == "" {
		return nil, ErrInvalidInput
	}
	l := &Like{
		ID:        uuid.New().String(),
		PostID:    postID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	const q = `
		INSERT INTO likes (id, post_id, user_id, created_at)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (user_id, post_id) DO NOTHING
	`
	if _, err := s.db.Exec(ctx, q, l.ID, l.PostID, l.UserID, l.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert like: %w", err)
	}
	return l, nil
}

func (s *service) Unlike(ctx context.Context, userID, postID string) (int64, error) {
	if userID == "" || postID == "" {
		return 0, ErrInvalidInput
	}
	const q = `DELETE FROM likes WHERE user_id=$1 AND post_id=$2`
	res, err := s.db.Exec(ctx, q, userID, postID)
	if err != nil {
		return 0, fmt.Errorf("delete like: %w", err)
	}
	return res.RowsAffected()
}

func (s *service) Count(ctx context.Context, postID string) (int64, error) {
	const q = `SELECT COUNT(*) FROM likes WHERE post_id=$1`
	var cnt int64
	if err := s.db.QueryRow(ctx, q, postID).Scan(&cnt); err != nil {
		return 0, fmt.Errorf("count likes: %w", err)
	}
	return cnt, nil
}

func (s *service) IsLiked(ctx context.Context, userID, postID string) (bool, error) {
	const q = `SELECT 1 FROM likes WHERE user_id=$1 AND post_id=$2 LIMIT 1`
	var one int
	err := s.db.QueryRow(ctx, q, userID, postID).Scan(&one)
	if err != nil {
		return false, nil
	}
	return true, nil
}
