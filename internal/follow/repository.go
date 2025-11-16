package follow

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"instant/internal/database"
)

var (
	ErrAlreadyFollowing = errors.New("already following this user")
	ErrNotFollowing     = errors.New("not following this user")
)

type Repository struct {
	db database.Service
}

func NewRepository(db database.Service) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Follow(ctx context.Context, followerID, followingID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO follows (follower_id, following_id)
		VALUES ($1, $2)
		ON CONFLICT (follower_id, following_id) DO NOTHING
	`, followerID, followingID)

	if err != nil {
		return err
	}

	// Check if row was created
	var exists bool
	err = r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM follows
			WHERE follower_id = $1 AND following_id = $2
		)
	`, followerID, followingID).Scan(&exists)

	if !exists {
		return ErrAlreadyFollowing
	}

	return nil
}

func (r *Repository) Unfollow(ctx context.Context, followerID, followingID uuid.UUID) error {
	res, err := r.db.Exec(ctx, `
		DELETE FROM follows
		WHERE follower_id = $1 AND following_id = $2
	`, followerID, followingID)

	if err != nil {
		return err
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrNotFollowing
	}

	return nil
}

func (r *Repository) ListFollowers(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT follower_id
		FROM follows
		WHERE following_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var followers []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		followers = append(followers, id)
	}
	return followers, nil
}

func (r *Repository) ListFollowing(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT following_id
		FROM follows
		WHERE follower_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var following []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		following = append(following, id)
	}
	return following, nil
}
