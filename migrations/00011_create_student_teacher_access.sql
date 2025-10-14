-- +goose Up
-- Create student_teacher_access table for managing access to private teachers
CREATE TABLE student_teacher_access (
    id BIGSERIAL PRIMARY KEY,
    student_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    teacher_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_type TEXT NOT NULL, -- 'invited', 'approved', 'subscribed'
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_student_teacher UNIQUE (student_id, teacher_id),
    CONSTRAINT valid_access_type CHECK (access_type IN ('invited', 'approved', 'subscribed'))
);

-- Create indexes for efficient lookups
CREATE INDEX idx_access_student ON student_teacher_access(student_id);
CREATE INDEX idx_access_teacher ON student_teacher_access(teacher_id);

-- Add comments for documentation
COMMENT ON TABLE student_teacher_access IS 'Доступ студентов к приватным учителям';
COMMENT ON COLUMN student_teacher_access.access_type IS 'invited=по коду, approved=одобрена заявка, subscribed=подписка';

-- +goose Down
-- Drop table and all related objects
DROP TABLE IF EXISTS student_teacher_access;

