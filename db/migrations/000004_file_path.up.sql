-- +migrate Up

ALTER TABLE file
DROP CONSTRAINT IF EXISTS file_message_id_fkey;

ALTER TABLE file
ALTER COLUMN message_id DROP NOT NULL;

ALTER TABLE file
ADD CONSTRAINT file_storage_path_unique UNIQUE (storage_path);

CREATE INDEX IF NOT EXISTS idx_file_storage_path ON file(storage_path);

CREATE INDEX IF NOT EXISTS idx_file_message_id ON file(message_id) WHERE message_id IS NOT NULL;
