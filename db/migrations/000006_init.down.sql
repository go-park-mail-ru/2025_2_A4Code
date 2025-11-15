-- +migrate Down

-- Удаляем столбец role из таблицы profile
ALTER TABLE profile DROP COLUMN role;

-- Удаляем тип enum
DROP TYPE user_role;