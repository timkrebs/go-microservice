package cleanup

import (
	"context"
	"log/slog"
	"time"

	"github.com/timkrebs/image-processor/internal/database"
	"github.com/timkrebs/image-processor/internal/models"
	"github.com/timkrebs/image-processor/internal/storage"
)

// Worker handles periodic cleanup of expired jobs and their associated files
type Worker struct {
	jobRepo   *database.JobRepository
	storage   *storage.Storage
	logger    *slog.Logger
	interval  time.Duration
	batchSize int
}

// Config holds cleanup worker configuration
type Config struct {
	Interval  time.Duration
	BatchSize int
}

// NewWorker creates a new cleanup worker
func NewWorker(jobRepo *database.JobRepository, storage *storage.Storage, cfg Config, logger *slog.Logger) *Worker {
	if cfg.Interval == 0 {
		cfg.Interval = 5 * time.Minute
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}

	return &Worker{
		jobRepo:   jobRepo,
		storage:   storage,
		logger:    logger,
		interval:  cfg.Interval,
		batchSize: cfg.BatchSize,
	}
}

// Start begins the cleanup worker in a goroutine
func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("cleanup worker started", "interval", w.interval, "batch_size", w.batchSize)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("cleanup worker stopped")
			return
		case <-ticker.C:
			if err := w.cleanup(ctx); err != nil {
				w.logger.Error("cleanup failed", "error", err)
			}
		}
	}
}

// cleanup performs a single cleanup cycle
func (w *Worker) cleanup(ctx context.Context) error {
	startTime := time.Now()
	w.logger.Info("starting cleanup cycle")

	jobs, err := w.jobRepo.GetJobsToCleanup(ctx, w.batchSize)
	if err != nil {
		return err
	}

	if len(jobs) == 0 {
		w.logger.Info("no jobs to cleanup")
		return nil
	}

	w.logger.Info("found jobs to cleanup", "count", len(jobs))

	cleanedCount := 0
	errorCount := 0

	for i := range jobs {
		if err := w.cleanupJob(ctx, jobs[i]); err != nil {
			w.logger.Error("failed to cleanup job",
				"job_id", jobs[i].ID,
				"error", err,
			)
			errorCount++
			continue
		}
		cleanedCount++
	}

	duration := time.Since(startTime)
	w.logger.Info("cleanup cycle completed",
		"duration_ms", duration.Milliseconds(),
		"cleaned", cleanedCount,
		"errors", errorCount,
		"total", len(jobs),
	)

	return nil
}

// cleanupJob removes a single job and its associated files
func (w *Worker) cleanupJob(ctx context.Context, job *models.Job) error {
	logger := w.logger.With("job_id", job.ID)

	if job.OriginalKey != "" {
		logger.Info("deleting original file", "key", job.OriginalKey)
		if err := w.storage.Delete(ctx, job.OriginalKey); err != nil {
			logger.Error("failed to delete original file",
				"key", job.OriginalKey,
				"error", err,
			)
		}
	}

	if job.ProcessedKey != "" {
		logger.Info("deleting processed file", "key", job.ProcessedKey)
		if err := w.storage.Delete(ctx, job.ProcessedKey); err != nil {
			logger.Error("failed to delete processed file",
				"key", job.ProcessedKey,
				"error", err,
			)
		}
	}

	logger.Info("deleting job record")
	if err := w.jobRepo.DeleteJob(ctx, job.ID); err != nil {
		return err
	}

	logger.Info("job cleaned up successfully")
	return nil
}
