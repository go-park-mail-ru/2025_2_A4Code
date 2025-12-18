-- Make surname and patronymic optional; keep reasonable max lengths
ALTER TABLE profile
DROP CONSTRAINT IF EXISTS profile_surname_check;

ALTER TABLE profile
DROP CONSTRAINT IF EXISTS profile_patronymic_check;

ALTER TABLE profile
ADD CONSTRAINT profile_surname_check
CHECK (surname IS NULL OR length(surname) <= 200);

ALTER TABLE profile
ADD CONSTRAINT profile_patronymic_check
CHECK (patronymic IS NULL OR length(patronymic) <= 200);
