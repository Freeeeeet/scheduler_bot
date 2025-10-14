-- +goose Up
-- +goose StatementBegin

-- Удаляем старую колонку group_id (UUID)
ALTER TABLE recurring_schedules DROP COLUMN group_id;

-- Добавляем новую колонку group_id (BIGINT)
ALTER TABLE recurring_schedules ADD COLUMN group_id BIGINT;

-- Для существующих записей создаем уникальные group_id
-- Группируем ВСЕ записи (активные и неактивные) по комбинации времени и предмета
DO $$
DECLARE
    next_group_id BIGINT := 1;
    rec RECORD;
BEGIN
    -- Обрабатываем активные записи
    FOR rec IN 
        SELECT DISTINCT teacher_id, subject_id, start_hour, start_minute, duration_minutes
        FROM recurring_schedules
        WHERE is_active = true
        ORDER BY teacher_id, subject_id, start_hour, start_minute
    LOOP
        UPDATE recurring_schedules
        SET group_id = next_group_id
        WHERE teacher_id = rec.teacher_id
          AND subject_id = rec.subject_id
          AND start_hour = rec.start_hour
          AND start_minute = rec.start_minute
          AND duration_minutes = rec.duration_minutes
          AND is_active = true;
        
        next_group_id := next_group_id + 1;
    END LOOP;
    
    -- Обрабатываем неактивные записи (каждая получает свой group_id)
    FOR rec IN 
        SELECT id
        FROM recurring_schedules
        WHERE is_active = false AND group_id IS NULL
    LOOP
        UPDATE recurring_schedules
        SET group_id = next_group_id
        WHERE id = rec.id;
        
        next_group_id := next_group_id + 1;
    END LOOP;
END $$;

-- Делаем поле обязательным
ALTER TABLE recurring_schedules ALTER COLUMN group_id SET NOT NULL;

-- Создаем индекс
CREATE INDEX idx_recurring_schedules_group_id ON recurring_schedules(group_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_recurring_schedules_group_id;
ALTER TABLE recurring_schedules DROP COLUMN group_id;
ALTER TABLE recurring_schedules ADD COLUMN group_id UUID;
CREATE INDEX idx_recurring_schedules_group_id ON recurring_schedules(group_id);

-- +goose StatementEnd

