-- +migrate Down

ALTER TABLE file
DROP CONSTRAINT IF EXISTS file_storage_path_unique;

DROP INDEX IF EXISTS idx_file_storage_path;
DROP INDEX IF EXISTS idx_file_message_id;

ALTER TABLE file
ALTER COLUMN message_id SET NOT NULL;

ALTER TABLE file
ADD CONSTRAINT file_message_id_fkey FOREIGN KEY (message_id) REFERENCES message(id) ON DELETE CASCADE;
