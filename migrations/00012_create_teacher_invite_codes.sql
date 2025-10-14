-- +goose Up
-- Create teacher_invite_codes table for managing invite codes
CREATE TABLE teacher_invite_codes (
    id BIGSERIAL PRIMARY KEY,
    teacher_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code TEXT NOT NULL UNIQUE,
    max_uses INTEGER DEFAULT NULL, -- NULL = безлимит
    current_uses INTEGER DEFAULT 0,
    expires_at TIMESTAMPTZ DEFAULT NULL, -- NULL = не истекает
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_max_uses CHECK (max_uses IS NULL OR max_uses > 0),
    CONSTRAINT valid_current_uses CHECK (current_uses >= 0)
);

-- Create indexes for efficient lookups
CREATE INDEX idx_invite_codes_teacher ON teacher_invite_codes(teacher_id);
CREATE INDEX idx_invite_codes_code ON teacher_invite_codes(code) WHERE is_active = true;

-- Add comments for documentation
COMMENT ON TABLE teacher_invite_codes IS 'Пригласительные коды для доступа к приватным учителям';

-- +goose Down
-- Drop table and all related objects
DROP TABLE IF EXISTS teacher_invite_codes;

