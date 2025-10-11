-- +goose Up
-- Create subjects table
CREATE TABLE subjects (
    id BIGSERIAL PRIMARY KEY,
    teacher_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    price INTEGER DEFAULT 0,
    duration INTEGER NOT NULL DEFAULT 60,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_teacher_subject UNIQUE (teacher_id, name),
    CONSTRAINT positive_price CHECK (price >= 0),
    CONSTRAINT valid_duration CHECK (duration > 0 AND duration <= 480)
);

-- Indexes
CREATE INDEX idx_subjects_teacher_id ON subjects(teacher_id);
CREATE INDEX idx_subjects_is_active ON subjects(is_active) WHERE is_active = true;

COMMENT ON TABLE subjects IS 'Subjects that teachers offer';
COMMENT ON COLUMN subjects.price IS 'Price in smallest currency unit (cents/kopecks)';
COMMENT ON COLUMN subjects.duration IS 'Duration in minutes';

-- +goose Down
-- Drop subjects table
DROP TABLE IF EXISTS subjects;
