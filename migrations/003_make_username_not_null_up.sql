-- Make username column NOT NULL
-- Migration: 003_make_username_not_null_up.sql
-- Description: Makes username field required for all users

BEGIN;

-- First, update any existing users with NULL username to have a default username
-- This generates a username based on email prefix + random suffix
UPDATE users 
SET username = CONCAT(
    SPLIT_PART(email, '@', 1), 
    '_', 
    EXTRACT(EPOCH FROM NOW())::bigint % 10000
)
WHERE username IS NULL;

-- Now make the column NOT NULL
ALTER TABLE users ALTER COLUMN username SET NOT NULL;

-- Add documentation
COMMENT ON COLUMN users.username IS 'Required unique username (3-50 characters, alphanumeric + underscore/dash)';

COMMIT;

