-- Restore stricter regex-based storage path constraint
ALTER TABLE file
DROP CONSTRAINT IF EXISTS file_storage_path_check;

ALTER TABLE file
ADD CONSTRAINT file_storage_path_check
CHECK (
    length(storage_path) BETWEEN 1 AND 200 AND
    storage_path ~ '^[\\w\\-./]+[\\w\\-]$' AND
    storage_path !~ '\\.\\.' AND
    storage_path !~ '^/' AND
    storage_path ~ '\\.\\w{1,10}$'
);
