-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS recurring_schedules (
    id SERIAL PRIMARY KEY,
    teacher_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id BIGINT NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    weekday INTEGER NOT NULL CHECK (weekday >= 0 AND weekday <= 6), -- 0 = Sunday, 6 = Saturday
    start_hour INTEGER NOT NULL CHECK (start_hour >= 0 AND start_hour <= 23),
    start_minute INTEGER NOT NULL CHECK (start_minute >= 0 AND start_minute <= 59),
    duration_minutes INTEGER NOT NULL CHECK (duration_minutes > 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recurring_schedules_teacher ON recurring_schedules(teacher_id);
CREATE INDEX idx_recurring_schedules_subject ON recurring_schedules(subject_id);
CREATE INDEX idx_recurring_schedules_active ON recurring_schedules(is_active);
CREATE INDEX idx_recurring_schedules_weekday ON recurring_schedules(weekday);

-- Добавим триггер для обновления updated_at
CREATE OR REPLACE FUNCTION update_recurring_schedules_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_recurring_schedules_updated_at
    BEFORE UPDATE ON recurring_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_recurring_schedules_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trigger_update_recurring_schedules_updated_at ON recurring_schedules;
DROP FUNCTION IF EXISTS update_recurring_schedules_updated_at();
DROP TABLE IF EXISTS recurring_schedules;
-- +goose StatementEnd

