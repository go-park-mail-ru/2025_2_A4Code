ALTER TABLE profile
    DROP CONSTRAINT IF EXISTS profile_surname_check,
    ALTER COLUMN surname DROP NOT NULL;