-- +goose Up
-- Allow NULL student_id for booked slots (for teacher marking slots as busy)
ALTER TABLE schedule_slots
DROP CONSTRAINT IF EXISTS student_only_for_booked;

ALTER TABLE schedule_slots
ADD CONSTRAINT student_only_for_booked 
    CHECK ((status = 'booked') OR 
           (status != 'booked' AND student_id IS NULL));

-- +goose Down
-- Restore original constraint
ALTER TABLE schedule_slots
DROP CONSTRAINT IF EXISTS student_only_for_booked;

ALTER TABLE schedule_slots
ADD CONSTRAINT student_only_for_booked 
    CHECK ((status = 'booked' AND student_id IS NOT NULL) OR 
           (status != 'booked' AND student_id IS NULL));

