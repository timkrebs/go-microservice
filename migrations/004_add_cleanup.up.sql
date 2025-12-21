-- Add delete_at timestamp for automatic cleanup
ALTER TABLE jobs ADD COLUMN delete_at TIMESTAMP WITH TIME ZONE;

-- Create index for efficient cleanup queries
CREATE INDEX idx_jobs_delete_at ON jobs(delete_at) WHERE delete_at IS NOT NULL;

-- Add comment explaining the field
COMMENT ON COLUMN jobs.delete_at IS 'Timestamp when this job and its associated files should be automatically deleted. Set to completed_at + retention period.';
