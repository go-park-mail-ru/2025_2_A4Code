-- Revert surname/patronymic checks to require length between 1 and 200
ALTER TABLE profile
DROP CONSTRAINT IF EXISTS profile_surname_check;

ALTER TABLE profile
DROP CONSTRAINT IF EXISTS profile_patronymic_check;

ALTER TABLE profile
ADD CONSTRAINT profile_surname_check
CHECK (length(surname) BETWEEN 1 AND 200);

ALTER TABLE profile
ADD CONSTRAINT profile_patronymic_check
CHECK (length(patronymic) BETWEEN 1 AND 200);
