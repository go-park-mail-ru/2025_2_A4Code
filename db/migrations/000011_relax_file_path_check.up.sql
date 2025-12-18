-- Simplify file storage path constraint to avoid regex issues on insert
ALTER TABLE file
DROP CONSTRAINT IF EXISTS file_storage_path_check;

ALTER TABLE file
ADD CONSTRAINT file_storage_path_check
CHECK (
    length(storage_path) BETWEEN 1 AND 200
    AND position('..' in storage_path) = 0
    AND left(storage_path, 1) <> '/'
    AND storage_path LIKE '%.%' -- must contain extension separator
);
