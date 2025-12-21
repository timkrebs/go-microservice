-- Remove delete_at index
DROP INDEX IF EXISTS idx_jobs_delete_at;

-- Remove delete_at column
ALTER TABLE jobs DROP COLUMN delete_at;
