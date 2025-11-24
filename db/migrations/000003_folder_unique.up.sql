-- Ensure unique folder names per profile (case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS idx_folder_profile_lower_name
    ON folder (profile_id, lower(folder_name));

