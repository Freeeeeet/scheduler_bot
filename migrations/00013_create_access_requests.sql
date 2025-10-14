-- +goose Up
-- Create access_requests table for managing student requests to private teachers
CREATE TABLE access_requests (
    id BIGSERIAL PRIMARY KEY,
    student_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    teacher_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'approved', 'rejected'
    message TEXT, -- Сообщение от студента
    teacher_response TEXT, -- Ответ учителя
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    
    CONSTRAINT valid_status CHECK (status IN ('pending', 'approved', 'rejected'))
);

-- Create unique constraint for pending requests only
CREATE UNIQUE INDEX unique_pending_request ON access_requests(student_id, teacher_id, status) 
WHERE status = 'pending';

-- Create indexes for efficient lookups
CREATE INDEX idx_requests_student ON access_requests(student_id, status);
CREATE INDEX idx_requests_teacher ON access_requests(teacher_id, status);
CREATE INDEX idx_requests_pending ON access_requests(teacher_id) WHERE status = 'pending';

-- Add comments for documentation
COMMENT ON TABLE access_requests IS 'Заявки студентов на доступ к приватным учителям';

-- +goose Down
-- Drop table and all related objects
DROP TABLE IF EXISTS access_requests;

