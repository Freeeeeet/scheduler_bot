-- +goose Up
-- Add is_public column to users table
ALTER TABLE users ADD COLUMN is_public BOOLEAN DEFAULT FALSE;

-- Add comment for documentation
COMMENT ON COLUMN users.is_public IS 'Публичный учитель (виден всем) или приватный (нужен доступ)';

-- Create partial index for efficient queries on public teachers
CREATE INDEX idx_users_is_public ON users(is_public) WHERE is_public = true AND is_teacher = true;

-- +goose Down
-- Remove index
DROP INDEX IF EXISTS idx_users_is_public;

-- Remove column
ALTER TABLE users DROP COLUMN IF EXISTS is_public;

