-- Align file size limits with 40MB attachment policy and allow any MIME type
ALTER TABLE file
DROP CONSTRAINT IF EXISTS file_size_limit;

ALTER TABLE file
ADD CONSTRAINT file_size_limit CHECK (size >= 0 AND size <= 41943040); -- 40 MB
