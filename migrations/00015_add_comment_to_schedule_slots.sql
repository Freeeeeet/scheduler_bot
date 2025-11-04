-- +goose Up
-- Add comment field to schedule_slots for teacher notes
ALTER TABLE schedule_slots
ADD COLUMN comment TEXT;

COMMENT ON COLUMN schedule_slots.comment IS 'Комментарий преподавателя к слоту (например, причина пометки занятым)';

-- +goose Down
-- Remove comment field
ALTER TABLE schedule_slots
DROP COLUMN IF EXISTS comment;

