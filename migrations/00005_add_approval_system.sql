-- +goose Up
-- Добавляем настройки учителя для автоматического одобрения
ALTER TABLE users ADD COLUMN IF NOT EXISTS auto_approve_bookings BOOLEAN DEFAULT TRUE;

COMMENT ON COLUMN users.auto_approve_bookings IS 'Автоматически одобрять записи студентов (TRUE) или требовать подтверждения (FALSE)';

-- Добавляем поля для системы запросов на отмену
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS cancellation_requested BOOLEAN DEFAULT FALSE;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS cancellation_requested_at TIMESTAMPTZ;

COMMENT ON COLUMN bookings.cancellation_requested IS 'Студент запросил отмену, ожидается одобрение учителя';
COMMENT ON COLUMN bookings.cancellation_requested_at IS 'Когда был отправлен запрос на отмену';

-- Индекс для быстрого поиска запросов на отмену
CREATE INDEX IF NOT EXISTS idx_bookings_cancellation_requested 
ON bookings(teacher_id, cancellation_requested) 
WHERE cancellation_requested = TRUE;

-- +goose Down
-- Удаляем добавленные поля
DROP INDEX IF EXISTS idx_bookings_cancellation_requested;

ALTER TABLE bookings DROP COLUMN IF EXISTS cancellation_requested_at;
ALTER TABLE bookings DROP COLUMN IF EXISTS cancellation_requested;

ALTER TABLE users DROP COLUMN IF EXISTS auto_approve_bookings;
