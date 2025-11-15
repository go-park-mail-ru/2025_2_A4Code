-- +migrate Up

-- Удаляем триггер
DROP TRIGGER IF EXISTS appeal_update_trigger ON appeal;

-- Удаляем старый внешний ключ
ALTER TABLE appeal DROP CONSTRAINT IF EXISTS appeal_profile_id_fkey;

-- Переименовываем столбец
ALTER TABLE appeal RENAME COLUMN profile_id TO base_profile_id;

-- Добавляем новый внешний ключ на base_profile(id)
ALTER TABLE appeal 
ADD CONSTRAINT appeal_base_profile_id_fkey 
FOREIGN KEY (base_profile_id) REFERENCES base_profile(id) ON DELETE NO ACTION;

-- Восстанавливаем триггер
CREATE TRIGGER appeal_update_trigger
BEFORE UPDATE ON appeal
FOR EACH ROW EXECUTE PROCEDURE update_updated_at();