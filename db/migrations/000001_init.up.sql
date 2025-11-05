-- +migrate Up

-- Изменяем поле patronymic: убираем CHECK ограничение и делаем его полностью необязательным
ALTER TABLE profile
    DROP CONSTRAINT IF EXISTS profile_patronymic_check,
    ALTER COLUMN patronymic DROP NOT NULL;


-- Изменяем поле gender: обновляем существующие данные и меняем CHECK ограничение
-- Сначала обновляем существующие данные в нижний регистр
UPDATE profile SET gender = LOWER(gender) WHERE gender IS NOT NULL;

-- Удаляем старое CHECK ограничение для gender
ALTER TABLE profile
    DROP CONSTRAINT IF EXISTS profile_gender_check;

-- Добавляем новое CHECK ограничение для gender с значениями в нижнем регистре
ALTER TABLE profile
    ADD CONSTRAINT profile_gender_check CHECK (gender IN ('male', 'female'));

