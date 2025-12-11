-- drop unique(username) to allow same local-part in different domains
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'base_profile_username_key'
            AND conrelid = 'base_profile'::regclass
    ) THEN
        ALTER TABLE base_profile DROP CONSTRAINT base_profile_username_key;
    END IF;
END;
$$;
