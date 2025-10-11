-- +goose Up
-- Create users table
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    telegram_id BIGINT NOT NULL UNIQUE,
    username TEXT,
    first_name TEXT NOT NULL,
    last_name TEXT,
    language_code TEXT,
    is_teacher BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_users_telegram_id ON users(telegram_id);
CREATE INDEX idx_users_is_teacher ON users(is_teacher) WHERE is_teacher = true;

-- Add comments for documentation
COMMENT ON TABLE users IS 'Stores both students and teachers';
COMMENT ON COLUMN users.telegram_id IS 'Unique Telegram user ID';
COMMENT ON COLUMN users.is_teacher IS 'Role: false=student, true=teacher';

-- +goose Down
-- Drop users table and related indexes
DROP INDEX IF EXISTS idx_users_telegram_id;
DROP INDEX IF EXISTS idx_users_is_teacher;
DROP TABLE IF EXISTS users;
