-- +migrate Down

-- Удаляем триггер
DROP TRIGGER IF EXISTS appeal_update_trigger ON appeal;

-- Удаляем новый внешний ключ
ALTER TABLE appeal DROP CONSTRAINT IF EXISTS appeal_base_profile_id_fkey;

-- Возвращаем старое имя столбца
ALTER TABLE appeal RENAME COLUMN base_profile_id TO profile_id;

-- Восстанавливаем старый внешний ключ
ALTER TABLE appeal 
ADD CONSTRAINT appeal_profile_id_fkey 
FOREIGN KEY (profile_id) REFERENCES profile(id) ON DELETE NO ACTION;

-- Восстанавливаем триггер
CREATE TRIGGER appeal_update_trigger
BEFORE UPDATE ON appeal
FOR EACH ROW EXECUTE PROCEDURE update_updated_at();