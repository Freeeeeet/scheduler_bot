-- +goose Up
-- Create bookings table
CREATE TABLE bookings (
    id BIGSERIAL PRIMARY KEY,
    student_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    teacher_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id BIGINT NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    slot_id BIGINT NOT NULL REFERENCES schedule_slots(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'confirmed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT valid_booking_status CHECK (status IN ('confirmed', 'completed', 'canceled'))
);

-- Partial unique index: только одна активная бронь на слот
CREATE UNIQUE INDEX idx_one_active_booking_per_slot ON bookings(slot_id) 
WHERE status = 'confirmed';

-- Indexes
CREATE INDEX idx_bookings_student_id ON bookings(student_id);
CREATE INDEX idx_bookings_teacher_id ON bookings(teacher_id);
CREATE INDEX idx_bookings_slot_id ON bookings(slot_id);
CREATE INDEX idx_bookings_status ON bookings(status);
CREATE INDEX idx_bookings_created_at ON bookings(created_at);

-- Function to update updated_at timestamp
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';
-- +goose StatementEnd

-- Trigger for automatic updated_at
CREATE TRIGGER update_bookings_updated_at 
    BEFORE UPDATE ON bookings 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE bookings IS 'History of student bookings';
COMMENT ON COLUMN bookings.status IS 'confirmed, completed, or canceled';

-- +goose Down
-- Drop bookings table
DROP TRIGGER IF EXISTS update_bookings_updated_at ON bookings;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS bookings;
