-- Add composite indexes for improved query performance

-- Composite index for status and created_at (used in job listing and dashboard queries)
CREATE INDEX IF NOT EXISTS idx_jobs_status_created_at ON jobs(status, created_at DESC);

-- Index for finding stuck processing jobs
CREATE INDEX IF NOT EXISTS idx_jobs_processing_started_at ON jobs(status, started_at) 
WHERE status = 'processing';

-- Index for finding failed jobs that need cleanup
CREATE INDEX IF NOT EXISTS idx_jobs_failed_completed_at ON jobs(status, completed_at)
WHERE status = 'failed';
