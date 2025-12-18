-- Rollback performance indexes

DROP INDEX IF EXISTS idx_jobs_failed_completed_at;
DROP INDEX IF EXISTS idx_jobs_processing_started_at;
DROP INDEX IF EXISTS idx_jobs_status_created_at;
