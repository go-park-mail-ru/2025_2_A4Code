-- Restore previous per-type size limits
ALTER TABLE file
DROP CONSTRAINT IF EXISTS file_size_limit;

ALTER TABLE file
ADD CONSTRAINT file_size_limit CHECK (
    (file_type IN ('image', 'avatar') AND size <= 10485760) OR
    (file_type = 'document' AND size <= 52428800) OR
    (file_type = 'video' AND size <= 1073741824) OR
    (file_type NOT IN ('image', 'avatar', 'document', 'video') AND size <= 10485760)
);
