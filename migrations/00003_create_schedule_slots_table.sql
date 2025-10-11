-- +goose Up
-- Create schedule_slots table
CREATE TABLE schedule_slots (
    id BIGSERIAL PRIMARY KEY,
    teacher_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id BIGINT NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'free',
    student_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT valid_time_range CHECK (end_time > start_time),
    CONSTRAINT valid_status CHECK (status IN ('free', 'booked', 'canceled')),
    CONSTRAINT student_only_for_booked 
        CHECK ((status = 'booked' AND student_id IS NOT NULL) OR 
               (status != 'booked' AND student_id IS NULL))
);

-- Indexes for query performance
CREATE INDEX idx_schedule_slots_teacher_id ON schedule_slots(teacher_id);
CREATE INDEX idx_schedule_slots_student_id ON schedule_slots(student_id);
CREATE INDEX idx_schedule_slots_status ON schedule_slots(status);
CREATE INDEX idx_schedule_slots_time_range ON schedule_slots(start_time, end_time);

-- Partial indexes for common queries
CREATE INDEX idx_schedule_slots_free ON schedule_slots(teacher_id, start_time) 
WHERE status = 'free';

-- Prevent overlapping slots for the same teacher
CREATE UNIQUE INDEX idx_schedule_slots_no_overlap 
ON schedule_slots (teacher_id, start_time, end_time);

COMMENT ON TABLE schedule_slots IS 'Time slots in teacher schedules';
COMMENT ON COLUMN schedule_slots.status IS 'free, booked, or canceled';

-- +goose Down
-- Drop schedule_slots table and indexes
DROP INDEX IF EXISTS idx_schedule_slots_teacher_id;
DROP INDEX IF EXISTS idx_schedule_slots_student_id;
DROP INDEX IF EXISTS idx_schedule_slots_status;
DROP INDEX IF EXISTS idx_schedule_slots_time_range;
DROP INDEX IF EXISTS idx_schedule_slots_free;
DROP INDEX IF EXISTS idx_schedule_slots_future;
DROP INDEX IF EXISTS idx_schedule_slots_no_overlap;
DROP TABLE IF EXISTS schedule_slots;
