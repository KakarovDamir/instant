-- Migration to create follow table for user follows system

DROP TABLE IF EXISTS follow;

CREATE TABLE IF NOT EXISTS follow (
    id          UUID PRIMARY KEY,
    follower_id UUID NOT NULL,
    followee_id UUID NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT follow_unique UNIQUE (follower_id, followee_id),

    CONSTRAINT follow_follower_fk FOREIGN KEY (follower_id)
        REFERENCES users(id) ON DELETE CASCADE,

    CONSTRAINT follow_followee_fk FOREIGN KEY (followee_id)
        REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_follow_follower ON follow(follower_id);
CREATE INDEX IF NOT EXISTS idx_follow_followee ON follow(followee_id);
CREATE INDEX IF NOT EXISTS idx_follow_created  ON follow(created_at DESC);

COMMENT ON TABLE follow IS 'Stores user follow relations (follower → followee)';
