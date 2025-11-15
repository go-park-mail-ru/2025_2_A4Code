-- +migrate Up

-- Создаем тип enum для ролей
CREATE TYPE user_role AS ENUM ('user', 'support');

-- Добавляем столбец role в таблицу profile
ALTER TABLE profile 
ADD COLUMN role user_role NOT NULL DEFAULT 'user';

