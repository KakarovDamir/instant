-- Rollback username column addition
-- Migration: 002_add_username_column_down.sql
-- Description: Removes username column and associated constraints

BEGIN;

-- Drop index
DROP INDEX IF EXISTS idx_users_username;

-- Drop unique constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_key;

-- Drop column
ALTER TABLE users DROP COLUMN IF EXISTS username;

COMMIT;
