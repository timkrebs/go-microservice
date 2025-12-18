package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/timkrebs/image-processor/internal/models"
)

// JobRepository handles job database operations
type JobRepository struct {
	db *DB
}

// NewJobRepository creates a new job repository
func NewJobRepository(db *DB) *JobRepository {
	return &JobRepository{db: db}
}

// Create inserts a new job into the database
func (r *JobRepository) Create(ctx context.Context, job *models.Job) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := job.MarshalOperations(); err != nil {
		return fmt.Errorf("failed to marshal operations: %w", err)
	}

	query := `
		INSERT INTO jobs (id, status, original_key, original_name, content_type, file_size, operations, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		job.ID,
		job.Status,
		job.OriginalKey,
		job.OriginalName,
		job.ContentType,
		job.FileSize,
		job.OperationsJSON,
		job.CreatedAt,
		job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

// GetByID retrieves a job by its ID
func (r *JobRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Job, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `
		SELECT id, status, original_key, processed_key, original_name, content_type,
		       file_size, operations, error, progress, worker_id, created_at, updated_at,
		       started_at, completed_at, processing_time_ms
		FROM jobs
		WHERE id = $1
	`

	job := &models.Job{}
	var processedKey, errorMsg, workerID sql.NullString
	var startedAt, completedAt sql.NullTime
	var processingTime sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID,
		&job.Status,
		&job.OriginalKey,
		&processedKey,
		&job.OriginalName,
		&job.ContentType,
		&job.FileSize,
		&job.OperationsJSON,
		&errorMsg,
		&job.Progress,
		&workerID,
		&job.CreatedAt,
		&job.UpdatedAt,
		&startedAt,
		&completedAt,
		&processingTime,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if processedKey.Valid {
		job.ProcessedKey = processedKey.String
	}
	if errorMsg.Valid {
		job.Error = errorMsg.String
	}
	if workerID.Valid {
		job.WorkerID = workerID.String
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if processingTime.Valid {
		job.ProcessingTime = &processingTime.Int64
	}

	if err := job.UnmarshalOperations(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operations: %w", err)
	}

	return job, nil
}

// List retrieves a paginated list of jobs
func (r *JobRepository) List(ctx context.Context, page, pageSize int) ([]*models.Job, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM jobs`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count jobs: %w", err)
	}

	query := `
		SELECT id, status, original_key, processed_key, original_name, content_type,
		       file_size, operations, error, progress, worker_id, created_at, updated_at,
		       started_at, completed_at, processing_time_ms
		FROM jobs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		job := &models.Job{}
		var processedKey, errorMsg, workerID sql.NullString
		var startedAt, completedAt sql.NullTime
		var processingTime sql.NullInt64

		err := rows.Scan(
			&job.ID,
			&job.Status,
			&job.OriginalKey,
			&processedKey,
			&job.OriginalName,
			&job.ContentType,
			&job.FileSize,
			&job.OperationsJSON,
			&errorMsg,
			&job.Progress,
			&workerID,
			&job.CreatedAt,
			&job.UpdatedAt,
			&startedAt,
			&completedAt,
			&processingTime,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan job: %w", err)
		}

		if processedKey.Valid {
			job.ProcessedKey = processedKey.String
		}
		if errorMsg.Valid {
			job.Error = errorMsg.String
		}
		if workerID.Valid {
			job.WorkerID = workerID.String
		}
		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}
		if processingTime.Valid {
			job.ProcessingTime = &processingTime.Int64
		}

		if err := job.UnmarshalOperations(); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal operations: %w", err)
		}

		jobs = append(jobs, job)
	}

	return jobs, total, nil
}

// UpdateStatus updates the status of a job
func (r *JobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.JobStatus) error {
	query := `UPDATE jobs SET status = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("job not found")
	}

	return nil
}

// UpdateProgress updates the progress of a job
func (r *JobRepository) UpdateProgress(ctx context.Context, id uuid.UUID, progress int) error {
	query := `UPDATE jobs SET progress = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, progress, id)
	return err
}

// StartProcessing marks a job as processing and records the worker ID
func (r *JobRepository) StartProcessing(ctx context.Context, id uuid.UUID, workerID string) error {
	now := time.Now()
	query := `
		UPDATE jobs
		SET status = $1, worker_id = $2, started_at = $3
		WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query, models.JobStatusProcessing, workerID, now, id)
	return err
}

// CompleteJob marks a job as completed
func (r *JobRepository) CompleteJob(ctx context.Context, id uuid.UUID, processedKey string) error {
	now := time.Now()

	// Calculate processing time
	var startedAt time.Time
	err := r.db.QueryRowContext(ctx, `SELECT started_at FROM jobs WHERE id = $1`, id).Scan(&startedAt)
	if err != nil {
		return fmt.Errorf("failed to get started_at: %w", err)
	}
	processingTime := now.Sub(startedAt).Milliseconds()

	query := `
		UPDATE jobs
		SET status = $1, processed_key = $2, progress = 100, completed_at = $3, processing_time_ms = $4
		WHERE id = $5
	`
	_, err = r.db.ExecContext(ctx, query, models.JobStatusCompleted, processedKey, now, processingTime, id)
	return err
}

// FailJob marks a job as failed with an error message
func (r *JobRepository) FailJob(ctx context.Context, id uuid.UUID, errorMsg string) error {
	now := time.Now()
	query := `
		UPDATE jobs
		SET status = $1, error = $2, completed_at = $3
		WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query, models.JobStatusFailed, errorMsg, now, id)
	return err
}

// CancelJob marks a job as canceled
func (r *JobRepository) CancelJob(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE jobs
		SET status = $1
		WHERE id = $2 AND status IN ($3, $4)
	`
	result, err := r.db.ExecContext(ctx, query,
		models.JobStatusCancelled,
		id,
		models.JobStatusPending,
		models.JobStatusQueued,
	)
	if err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("job cannot be canceled (already processing or completed)")
	}

	return nil
}

// GetPendingJobsCount returns the count of pending jobs
func (r *JobRepository) GetPendingJobsCount(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM jobs WHERE status IN ($1, $2)`
	err := r.db.QueryRowContext(ctx, query, models.JobStatusPending, models.JobStatusQueued).Scan(&count)
	return count, err
}
