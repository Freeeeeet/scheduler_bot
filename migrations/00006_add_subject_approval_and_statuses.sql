-- +goose Up
-- Добавляем настройку одобрения на уровне предмета
ALTER TABLE subjects ADD COLUMN IF NOT EXISTS requires_booking_approval BOOLEAN DEFAULT FALSE;

COMMENT ON COLUMN subjects.requires_booking_approval IS 'Требуется ли одобрение учителя для записи на этот предмет';

-- Обновляем ENUM для статусов бронирований (добавляем pending и rejected)
-- Сначала создаем новый тип
-- +goose StatementBegin
DO $$ 
BEGIN
    CREATE TYPE booking_status_new AS ENUM ('pending', 'confirmed', 'completed', 'canceled', 'rejected');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;
-- +goose StatementEnd

-- Добавляем временную колонку с новым типом
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS status_new booking_status_new;

-- Копируем данные из старой колонки в новую
UPDATE bookings SET status_new = status::text::booking_status_new;

-- Удаляем старую колонку и переименовываем новую
ALTER TABLE bookings DROP COLUMN status;
ALTER TABLE bookings RENAME COLUMN status_new TO status;

-- Устанавливаем значение по умолчанию и NOT NULL
ALTER TABLE bookings ALTER COLUMN status SET DEFAULT 'confirmed';
ALTER TABLE bookings ALTER COLUMN status SET NOT NULL;

-- Удаляем старый тип и переименовываем новый
DROP TYPE IF EXISTS booking_status_old CASCADE;
ALTER TYPE booking_status_new RENAME TO booking_status;

-- Индекс для быстрого поиска pending бронирований учителя
CREATE INDEX IF NOT EXISTS idx_bookings_teacher_pending 
ON bookings(teacher_id, status) 
WHERE status = 'pending';

COMMENT ON COLUMN bookings.status IS 'Статус бронирования: pending (ожидает одобрения), confirmed (подтверждено), completed (завершено), canceled (отменено), rejected (отклонено)';

-- +goose Down
-- Откатываем изменения

DROP INDEX IF EXISTS idx_bookings_teacher_pending;

-- Возвращаем старый ENUM (только основные статусы)
-- +goose StatementBegin
DO $$ 
BEGIN
    CREATE TYPE booking_status_old AS ENUM ('confirmed', 'completed', 'canceled');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;
-- +goose StatementEnd

ALTER TABLE bookings ADD COLUMN IF NOT EXISTS status_old booking_status_old;

-- Копируем данные, pending и rejected переводим в confirmed и canceled
UPDATE bookings SET status_old = 
    CASE 
        WHEN status::text = 'pending' THEN 'confirmed'::booking_status_old
        WHEN status::text = 'rejected' THEN 'canceled'::booking_status_old
        ELSE status::text::booking_status_old
    END;

ALTER TABLE bookings DROP COLUMN status;
ALTER TABLE bookings RENAME COLUMN status_old TO status;

ALTER TABLE bookings ALTER COLUMN status SET DEFAULT 'confirmed';
ALTER TABLE bookings ALTER COLUMN status SET NOT NULL;

DROP TYPE IF EXISTS booking_status CASCADE;
ALTER TYPE booking_status_old RENAME TO booking_status;

-- Удаляем поле из subjects
ALTER TABLE subjects DROP COLUMN IF EXISTS requires_booking_approval;

