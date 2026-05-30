BEGIN;

-- Add a dedicated MIME type column. The legacy file_type column keeps storing
-- the extension (e.g. ".pdf"); mime_type stores the resolved content type used
-- for Content-Type / Content-Disposition decisions in the gateway.
ALTER TABLE file_metadata
    ADD COLUMN IF NOT EXISTS mime_type VARCHAR(255);

-- Backfill existing rows from the original file name extension.
UPDATE file_metadata
SET mime_type = CASE
    WHEN file_name ILIKE '%.pdf'  THEN 'application/pdf'
    WHEN file_name ILIKE '%.docx' THEN 'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
    WHEN file_name ILIKE '%.doc'  THEN 'application/msword'
    WHEN file_name ILIKE '%.xlsx' THEN 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    WHEN file_name ILIKE '%.xls'  THEN 'application/vnd.ms-excel'
    WHEN file_name ILIKE '%.pptx' THEN 'application/vnd.openxmlformats-officedocument.presentationml.presentation'
    WHEN file_name ILIKE '%.ppt'  THEN 'application/vnd.ms-powerpoint'
    WHEN file_name ILIKE '%.png'  THEN 'image/png'
    WHEN file_name ILIKE '%.jpg'  THEN 'image/jpeg'
    WHEN file_name ILIKE '%.jpeg' THEN 'image/jpeg'
    WHEN file_name ILIKE '%.gif'  THEN 'image/gif'
    WHEN file_name ILIKE '%.txt'  THEN 'text/plain; charset=utf-8'
    WHEN file_name ILIKE '%.csv'  THEN 'text/csv; charset=utf-8'
    WHEN file_name ILIKE '%.zip'  THEN 'application/zip'
    ELSE 'application/octet-stream'
END
WHERE mime_type IS NULL;

COMMIT;
