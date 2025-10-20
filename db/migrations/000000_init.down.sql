-- +migrate Down
DROP TRIGGER IF EXISTS settings_update_trigger ON settings;
DROP TABLE IF EXISTS settings CASCADE;

DROP TRIGGER IF EXISTS profile_message_update_trigger ON profile_message;
DROP TABLE IF EXISTS profile_message CASCADE;

DROP TRIGGER IF EXISTS file_update_trigger ON file;
DROP TABLE IF EXISTS file CASCADE;

DROP TABLE IF EXISTS message CASCADE;  -- После file, так как file зависит от message

DROP TRIGGER IF EXISTS thread_update_trigger ON thread;
DROP TABLE IF EXISTS thread CASCADE;

DROP TRIGGER IF EXISTS folder_update_trigger ON folder;
DROP TABLE IF EXISTS folder CASCADE;

DROP TRIGGER IF EXISTS profile_update_trigger ON profile;
DROP TABLE IF EXISTS profile CASCADE;

DROP TRIGGER IF EXISTS base_profile_update_trigger ON base_profile;
DROP TABLE IF EXISTS base_profile CASCADE;

DROP FUNCTION IF EXISTS update_updated_at CASCADE;