-- Migration to fix likes.post_id type from UUID to BIGINT
-- This aligns with posts.post_id which is BIGSERIAL (BIGINT)

-- Drop existing likes table if it exists (safe because service is new)
DROP TABLE IF EXISTS likes;

-- Recreate with correct post_id type
CREATE TABLE IF NOT EXISTS likes (
    id        UUID PRIMARY KEY,
    post_id   BIGINT NOT NULL,
    user_id   UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT likes_user_post_unique UNIQUE (user_id, post_id),
    CONSTRAINT likes_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT likes_post_fk FOREIGN KEY (post_id) REFERENCES posts(post_id) ON DELETE CASCADE
);

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_likes_post_id  ON likes(post_id);
CREATE INDEX IF NOT EXISTS idx_likes_user_id  ON likes(user_id);
CREATE INDEX IF NOT EXISTS idx_likes_created  ON likes(created_at DESC);

-- Add table comment
COMMENT ON TABLE likes IS 'Stores user likes on posts';
COMMENT ON COLUMN likes.post_id IS 'Reference to posts.post_id (BIGINT, not UUID)';
