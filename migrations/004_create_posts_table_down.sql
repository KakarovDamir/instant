-- Drop trigger first
DROP TRIGGER IF EXISTS trigger_posts_updated_at ON posts;

-- Drop function
DROP FUNCTION IF EXISTS update_posts_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_posts_user_created;
DROP INDEX IF EXISTS idx_posts_created_at_desc;
DROP INDEX IF EXISTS idx_posts_user_id;

-- Drop table (CASCADE will also drop foreign key constraints)
DROP TABLE IF EXISTS posts CASCADE;
