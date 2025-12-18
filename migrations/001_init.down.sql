-- Drop trigger and function
DROP TRIGGER IF EXISTS update_jobs_updated_at ON jobs;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_jobs_status;
DROP INDEX IF EXISTS idx_jobs_created_at;
DROP INDEX IF EXISTS idx_jobs_worker_id;

-- Drop table
DROP TABLE IF EXISTS jobs;
