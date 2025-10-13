-- +goose Up
-- +goose StatementBegin
ALTER TABLE recurring_schedules ADD COLUMN group_id UUID;

-- Создаем индекс для быстрого поиска по group_id
CREATE INDEX idx_recurring_schedules_group_id ON recurring_schedules(group_id);

-- Обновляем существующие записи: для каждой записи создаем уникальный group_id
-- Это означает, что старые расписания станут "одиночными" группами
UPDATE recurring_schedules 
SET group_id = gen_random_uuid() 
WHERE group_id IS NULL;

-- Делаем поле обязательным
ALTER TABLE recurring_schedules ALTER COLUMN group_id SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_recurring_schedules_group_id;
ALTER TABLE recurring_schedules DROP COLUMN group_id;
-- +goose StatementEnd

