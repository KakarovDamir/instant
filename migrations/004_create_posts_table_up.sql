-- Create posts table
CREATE TABLE IF NOT EXISTS posts (
    post_id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    caption VARCHAR(1000) NOT NULL,
    image_url TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Foreign key constraint linking to users table
    CONSTRAINT fk_posts_user FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

-- Create indexes for performance optimization (critical for high load)
-- Index on user_id for fast lookups of user's posts
CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id);

-- Composite index for pagination queries (ORDER BY created_at DESC with filtering)
CREATE INDEX IF NOT EXISTS idx_posts_created_at_desc ON posts(created_at DESC);

-- Composite index for user's posts ordered by date (common query pattern)
CREATE INDEX IF NOT EXISTS idx_posts_user_created ON posts(user_id, created_at DESC);

-- Add comments for documentation
COMMENT ON TABLE posts IS 'Stores user posts with captions and images';
COMMENT ON COLUMN posts.post_id IS 'Unique post identifier (auto-incrementing)';
COMMENT ON COLUMN posts.user_id IS 'Reference to user who created the post';
COMMENT ON COLUMN posts.caption IS 'Post caption text (max 1000 characters)';
COMMENT ON COLUMN posts.image_url IS 'URL to the post image';
COMMENT ON COLUMN posts.created_at IS 'Timestamp when post was created';
COMMENT ON COLUMN posts.updated_at IS 'Timestamp when post was last updated';

-- Create function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_posts_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to call the function before updates
CREATE TRIGGER trigger_posts_updated_at
    BEFORE UPDATE ON posts
    FOR EACH ROW
    EXECUTE FUNCTION update_posts_updated_at();
