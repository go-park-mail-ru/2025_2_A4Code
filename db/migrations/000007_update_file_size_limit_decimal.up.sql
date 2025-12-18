-- Switch file size limit to decimal 40 MB (40,000,000 bytes)
ALTER TABLE file
DROP CONSTRAINT IF EXISTS file_size_limit;

ALTER TABLE file
ADD CONSTRAINT file_size_limit CHECK (size >= 0 AND size <= 40000000);
