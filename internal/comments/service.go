package comments

import (
    "context"
    "fmt"
    "time"

    "instant/internal/database"
)

type Service interface {
    Create(ctx context.Context, userID string, req CreateCommentRequest) (*Comment, error)
    Update(ctx context.Context, userID string, commentID int64, body string) (*Comment, error)
    Delete(ctx context.Context, userID string, commentID int64) error
    ListByPost(ctx context.Context, postID int64) ([]Comment, error)
}

type service struct {
    db database.Service
}

func NewService(db database.Service) Service {
    return &service{db: db}
}

func (s *service) Create(ctx context.Context, userID string, req CreateCommentRequest) (*Comment, error) {
    now := time.Now()

    const q = `
        INSERT INTO comments (post_id, user_id, body, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5)
        RETURNING comment_id, created_at, updated_at
    `

    c := &Comment{
        PostID: req.PostID,
        UserID: userID,
        Body:   req.Body,
    }

    err := s.db.QueryRow(ctx, q, c.PostID, c.UserID, c.Body, now, now).
        Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
    if err != nil {
        return nil, fmt.Errorf("insert comment: %w", err)
    }

    return c, nil
}

func (s *service) Update(ctx context.Context, userID string, commentID int64, body string) (*Comment, error) {
    const q = `
        UPDATE comments
        SET body=$1, updated_at=NOW()
        WHERE comment_id=$2 AND user_id=$3
        RETURNING post_id, created_at, updated_at
    `
    c := &Comment{ID: commentID, UserID: userID, Body: body}
    err := s.db.QueryRow(ctx, q, body, commentID, userID).
        Scan(&c.PostID, &c.CreatedAt, &c.UpdatedAt)
    if err != nil {
        return nil, err
    }
    return c, nil
}

func (s *service) Delete(ctx context.Context, userID string, commentID int64) error {
    const q = `DELETE FROM comments WHERE comment_id=$1 AND user_id=$2`

    _, err := s.db.Exec(ctx, q, commentID, userID)
    return err
}

func (s *service) ListByPost(ctx context.Context, postID int64) ([]Comment, error) {
    const q = `
        SELECT comment_id, post_id, user_id, body, created_at, updated_at
        FROM comments
        WHERE post_id=$1
        ORDER BY created_at ASC
    `
    rows, err := s.db.Query(ctx, q, postID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    out := []Comment{}
    for rows.Next() {
        var c Comment
        if err := rows.Scan(
            &c.ID, &c.PostID, &c.UserID, &c.Body, &c.CreatedAt, &c.UpdatedAt,
        ); err != nil {
            return nil, err
        }
        out = append(out, c)
    }
    return out, nil
}
