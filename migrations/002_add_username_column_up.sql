-- Add username column to users table
-- Migration: 002_add_username_column_up.sql
-- Description: Adds unique username field to support user profiles

BEGIN;

-- Add username column (nullable initially for backward compatibility)
ALTER TABLE users ADD COLUMN IF NOT EXISTS username VARCHAR(50);

-- Create unique constraint on username
ALTER TABLE users ADD CONSTRAINT users_username_key UNIQUE (username);

-- Create partial index for performance (only indexes non-null values)
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE username IS NOT NULL;

-- Add documentation
COMMENT ON COLUMN users.username IS 'Unique username (3-50 characters, alphanumeric + underscore/dash). Optional field.';

COMMIT;
