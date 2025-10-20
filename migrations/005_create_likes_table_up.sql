CREATE TABLE IF NOT EXISTS likes (
    id        UUID PRIMARY KEY,
    post_id   BIGINT NOT NULL,
    user_id   UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT likes_user_post_unique UNIQUE (user_id, post_id),
    CONSTRAINT likes_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT likes_post_fk FOREIGN KEY (post_id) REFERENCES posts(post_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_likes_post_id  ON likes(post_id);
CREATE INDEX IF NOT EXISTS idx_likes_user_id  ON likes(user_id);
CREATE INDEX IF NOT EXISTS idx_likes_created  ON likes(created_at DESC);