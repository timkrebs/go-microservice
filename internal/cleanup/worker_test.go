package cleanup

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/timkrebs/image-processor/internal/database"
	"github.com/timkrebs/image-processor/internal/models"
	"github.com/timkrebs/image-processor/internal/storage"
)

func setupCleanupTest(t *testing.T) (*Worker, *database.DB, *storage.Storage, uuid.UUID) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	db, err := database.New(dbURL, 5)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	minioEndpoint := os.Getenv("TEST_MINIO_ENDPOINT")
	if minioEndpoint == "" {
		minioEndpoint = "localhost:9000"
	}

	storageClient, err := storage.New(storage.Config{
		Endpoint:  minioEndpoint,
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Bucket:    "images",
		UseSSL:    false,
	})
	if err != nil {
		t.Fatalf("failed to create storage client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := storageClient.EnsureBucket(ctx); err != nil {
		t.Fatalf("failed to ensure bucket: %v", err)
	}

	systemUserID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
	jobRepo := database.NewJobRepository(db)

	worker := NewWorker(jobRepo, storageClient, Config{
		Interval:  1 * time.Minute,
		BatchSize: 10,
	}, logger)

	return worker, db, storageClient, systemUserID
}

func TestWorker_Cleanup(t *testing.T) {
	worker, db, storageClient, userID := setupCleanupTest(t)
	defer db.Close()

	ctx := context.Background()

	job := &models.Job{
		ID:           uuid.New(),
		UserID:       userID,
		Status:       "completed",
		OriginalKey:  "test/original.jpg",
		OriginalName: "original.jpg",
		ContentType:  "image/jpeg",
		FileSize:     1024,
		Operations:   []models.Operation{},
	}

	err := worker.jobRepo.Create(ctx, job)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Set job to processing to populate started_at
	if err := worker.jobRepo.UpdateStatus(ctx, job.ID, models.JobStatusProcessing); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	processedKey := "test/processed.jpg"
	if err := worker.jobRepo.CompleteJob(ctx, job.ID, processedKey, 1); err != nil {
		t.Fatalf("failed to complete job: %v", err)
	}

	deleteAt := time.Now().Add(-1 * time.Hour)
	_, err = db.ExecContext(ctx, "UPDATE jobs SET delete_at = $1 WHERE id = $2", deleteAt, job.ID)
	if err != nil {
		t.Fatalf("failed to set delete_at: %v", err)
	}

	if err := worker.cleanup(ctx); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	jobs, err := worker.jobRepo.GetJobsToCleanup(ctx, 10)
	if err != nil {
		t.Fatalf("failed to get jobs to cleanup: %v", err)
	}

	for _, j := range jobs {
		if j.ID == job.ID {
			t.Error("job should have been cleaned up")
		}
	}

	_ = storageClient // Avoid unused variable warning
}

func TestWorker_CleanupWithFiles(t *testing.T) {
	worker, db, storageClient, userID := setupCleanupTest(t)
	defer db.Close()

	ctx := context.Background()

	originalKey := "cleanup-test/original-" + uuid.New().String() + ".txt"
	processedKey := "cleanup-test/processed-" + uuid.New().String() + ".txt"

	originalContent := []byte("original content")
	err := storageClient.Upload(ctx, originalKey, bytes.NewReader(originalContent), int64(len(originalContent)), "text/plain")
	if err != nil {
		t.Fatalf("failed to upload original file: %v", err)
	}

	processedContent := []byte("processed content")
	err = storageClient.Upload(ctx, processedKey, bytes.NewReader(processedContent), int64(len(processedContent)), "text/plain")
	if err != nil {
		t.Fatalf("failed to upload processed file: %v", err)
	}

	job := &models.Job{
		ID:           uuid.New(),
		UserID:       userID,
		Status:       "completed",
		OriginalKey:  originalKey,
		OriginalName: "test.txt",
		ContentType:  "text/plain",
		FileSize:     16,
		Operations:   []models.Operation{},
	}

	err = worker.jobRepo.Create(ctx, job)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Set job to processing to populate started_at
	if err := worker.jobRepo.UpdateStatus(ctx, job.ID, models.JobStatusProcessing); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	if err := worker.jobRepo.CompleteJob(ctx, job.ID, processedKey, 1); err != nil {
		t.Fatalf("failed to complete job: %v", err)
	}

	deleteAt := time.Now().Add(-1 * time.Hour)
	_, err = db.ExecContext(ctx, "UPDATE jobs SET delete_at = $1 WHERE id = $2", deleteAt, job.ID)
	if err != nil {
		t.Fatalf("failed to set delete_at: %v", err)
	}

	// Debug: Verify job is ready for cleanup and has ProcessedKey set
	jobsToCleanup, err := worker.jobRepo.GetJobsToCleanup(ctx, 10)
	if err != nil {
		t.Fatalf("failed to get jobs to cleanup: %v", err)
	}
	t.Logf("Found %d jobs ready for cleanup", len(jobsToCleanup))
	found := false
	for _, j := range jobsToCleanup {
		if j.ID == job.ID {
			found = true
			t.Logf("Job %s found: OriginalKey=%q, ProcessedKey=%q", j.ID, j.OriginalKey, j.ProcessedKey)
			if j.ProcessedKey == "" {
				t.Error("ProcessedKey is empty in job retrieved for cleanup")
			}
		}
	}
	if !found {
		t.Error("job not found in cleanup list")
	}

	t.Logf("Calling cleanup()...")
	if err := worker.cleanup(ctx); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	t.Logf("Cleanup completed successfully")

	// Wait a moment for MinIO to process deletions (eventual consistency)
	time.Sleep(100 * time.Millisecond)

	// Verify files were deleted - use StatObject for more reliable checking
	t.Logf("Checking if original file %q was deleted...", originalKey)
	_, err = storageClient.Download(ctx, originalKey)
	if err == nil {
		t.Error("original file should have been deleted")
	} else {
		t.Logf("Original file deletion verified: %v", err)
	}

	t.Logf("Checking if processed file %q was deleted...", processedKey)
	_, err = storageClient.Download(ctx, processedKey)
	if err == nil {
		t.Error("processed file should have been deleted")
	} else {
		t.Logf("Processed file deletion verified: %v", err)
	}

	jobs, err := worker.jobRepo.GetJobsToCleanup(ctx, 10)
	if err != nil {
		t.Fatalf("failed to get jobs to cleanup: %v", err)
	}

	for _, j := range jobs {
		if j.ID == job.ID {
			t.Error("job should have been cleaned up")
		}
	}
}

func TestWorker_CleanupNoJobsReady(t *testing.T) {
	worker, db, _, _ := setupCleanupTest(t)
	defer db.Close()

	ctx := context.Background()

	if err := worker.cleanup(ctx); err != nil {
		t.Fatalf("cleanup should not fail when no jobs ready: %v", err)
	}
}

func TestWorker_CleanupBatchSize(t *testing.T) {
	worker, db, _, userID := setupCleanupTest(t)
	defer db.Close()

	worker.batchSize = 5
	ctx := context.Background()

	// Create 10 jobs with delete_at in the past
	for i := 0; i < 10; i++ {
		job := &models.Job{
			ID:           uuid.New(),
			UserID:       userID,
			Status:       "completed",
			OriginalKey:  "test/file-" + uuid.New().String() + ".txt",
			OriginalName: "file.txt",
			ContentType:  "text/plain",
			FileSize:     100,
			Operations:   []models.Operation{},
		}

		err := worker.jobRepo.Create(ctx, job)
		if err != nil {
			t.Fatalf("failed to create job: %v", err)
		}

		// Set job to processing to populate started_at
		if err := worker.jobRepo.UpdateStatus(ctx, job.ID, models.JobStatusProcessing); err != nil {
			t.Fatalf("failed to update status: %v", err)
		}

		if err := worker.jobRepo.CompleteJob(ctx, job.ID, "test/processed-"+job.ID.String()+".txt", 1); err != nil {
			t.Fatalf("failed to complete job: %v", err)
		}

		deleteAt := time.Now().Add(-1 * time.Hour)
		_, err = db.ExecContext(ctx, "UPDATE jobs SET delete_at = $1 WHERE id = $2", deleteAt, job.ID)
		if err != nil {
			t.Fatalf("failed to set delete_at: %v", err)
		}
	}

	// Run cleanup - should only process 5 jobs due to batch size
	if err := worker.cleanup(ctx); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// Check that 5 jobs remain
	remainingJobs, err := worker.jobRepo.GetJobsToCleanup(ctx, 100)
	if err != nil {
		t.Fatalf("failed to get remaining jobs: %v", err)
	}

	if len(remainingJobs) != 5 {
		t.Errorf("expected 5 remaining jobs, got %d", len(remainingJobs))
	}
}

func TestNewWorker_DefaultConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	worker := NewWorker(nil, nil, Config{}, logger)

	if worker.interval != 5*time.Minute {
		t.Errorf("expected default interval 5m, got %v", worker.interval)
	}

	if worker.batchSize != 100 {
		t.Errorf("expected default batch size 100, got %d", worker.batchSize)
	}
}
