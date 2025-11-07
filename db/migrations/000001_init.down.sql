-- +migrate Down

-- Возвращаем CHECK ограничение для gender в исходное состояние
ALTER TABLE profile
    DROP CONSTRAINT IF EXISTS profile_gender_check;

-- Обновляем данные обратно в исходный регистр (с заглавной буквы)
UPDATE profile SET gender = INITCAP(gender) WHERE gender IS NOT NULL;

-- Возвращаем CHECK ограничение для gender
ALTER TABLE profile
    ADD CONSTRAINT profile_gender_check CHECK (gender IN ('Male', 'Female'));

-- Возвращаем CHECK ограничение для patronymic (опционально, если хотите полный откат)
-- ALTER TABLE profile
-- ADD CONSTRAINT profile_patronymic_check CHECK (LENGTH(patronymic) BETWEEN 1 AND 200);