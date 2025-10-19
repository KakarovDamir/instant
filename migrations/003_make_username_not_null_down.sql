-- Rollback: Make username column nullable again
-- Migration: 003_make_username_not_null_down.sql
-- Description: Reverts username field to be optional

BEGIN;

-- Make the column nullable again
ALTER TABLE users ALTER COLUMN username DROP NOT NULL;

-- Update documentation
COMMENT ON COLUMN users.username IS 'Unique username (3-50 characters, alphanumeric + underscore/dash). Optional field.';

COMMIT;

