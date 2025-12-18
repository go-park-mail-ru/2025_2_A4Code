-- Restore previous binary-based 40MB limit (41943040 bytes)
ALTER TABLE file
DROP CONSTRAINT IF EXISTS file_size_limit;

ALTER TABLE file
ADD CONSTRAINT file_size_limit CHECK (size >= 0 AND size <= 41943040);
