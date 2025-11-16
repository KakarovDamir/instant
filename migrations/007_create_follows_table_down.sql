-- Drop trigger and function if any (optional, if you implement updated_at trigger)
-- DROP TRIGGER IF EXISTS trigger_follows_updated_at ON follows;
-- DROP FUNCTION IF EXISTS update_follows_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_follows_follower_id;
DROP INDEX IF EXISTS idx_follows_following_id;
DROP INDEX IF EXISTS idx_follows_created_at_desc;

-- Drop table
DROP TABLE IF EXISTS follows CASCADE;
