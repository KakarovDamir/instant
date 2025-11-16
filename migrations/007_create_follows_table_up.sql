-- Create follows table
CREATE TABLE IF NOT EXISTS follows (
    follow_id BIGSERIAL PRIMARY KEY,
    follower_id UUID NOT NULL,
    following_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Foreign key constraints linking to users table
    CONSTRAINT fk_follows_follower FOREIGN KEY (follower_id)
        REFERENCES users(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
    CONSTRAINT fk_follows_following FOREIGN KEY (following_id)
        REFERENCES users(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
    CONSTRAINT uq_follows UNIQUE (follower_id, following_id)
);

-- Indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_follows_follower_id ON follows(follower_id);
CREATE INDEX IF NOT EXISTS idx_follows_following_id ON follows(following_id);
CREATE INDEX IF NOT EXISTS idx_follows_created_at_desc ON follows(created_at DESC);

-- Comments for documentation
COMMENT ON TABLE follows IS 'Stores user follow relationships';
COMMENT ON COLUMN follows.follow_id IS 'Unique follow identifier (auto-incrementing)';
COMMENT ON COLUMN follows.follower_id IS 'User who follows another user';
COMMENT ON COLUMN follows.following_id IS 'User being followed';
COMMENT ON COLUMN follows.created_at IS 'Timestamp when the follow relationship was created';
