package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/timkrebs/image-processor/internal/config"
	"github.com/timkrebs/image-processor/internal/database"
	"github.com/timkrebs/image-processor/internal/models"
	"github.com/timkrebs/image-processor/internal/processor"
	"github.com/timkrebs/image-processor/internal/queue"
	"github.com/timkrebs/image-processor/internal/storage"
)

type Worker struct {
	id        string
	jobRepo   *database.JobRepository
	storage   *storage.Storage
	consumer  *queue.Consumer
	processor *processor.Processor
	logger    *slog.Logger
}

func main() {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Generate worker ID
	workerID := fmt.Sprintf("worker-%s", uuid.New().String()[:8])
	logger = logger.With("worker_id", workerID)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Connect to database
	db, err := database.New(cfg.DatabaseURL, cfg.DatabaseMaxConn)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("connected to database")

	// Create job repository
	jobRepo := database.NewJobRepository(db)

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		cancel()
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	cancel()
	logger.Info("connected to redis")

	// Create queue consumer
	consumer := queue.NewConsumer(redisClient, queue.ConsumerConfig{
		StreamName:    cfg.QueueStreamName,
		ConsumerGroup: cfg.QueueConsumerGroup,
		ConsumerName:  workerID,
		PollTimeout:   cfg.WorkerPollTimeout,
	}, logger)

	// Ensure consumer group exists
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	if err := consumer.EnsureGroup(ctx); err != nil {
		cancel()
		logger.Error("failed to ensure consumer group", "error", err)
		os.Exit(1)
	}
	cancel()

	// Connect to MinIO
	storageClient, err := storage.New(storage.Config{
		Endpoint:  cfg.MinIOEndpoint,
		AccessKey: cfg.MinIOAccessKey,
		SecretKey: cfg.MinIOSecretKey,
		Bucket:    cfg.MinIOBucket,
		UseSSL:    cfg.MinIOUseSSL,
	})
	if err != nil {
		logger.Error("failed to create storage client", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to minio", "bucket", cfg.MinIOBucket)

	// Create image processor
	imageProcessor := processor.New()

	// Create worker
	worker := &Worker{
		id:        workerID,
		jobRepo:   jobRepo,
		storage:   storageClient,
		consumer:  consumer,
		processor: imageProcessor,
		logger:    logger,
	}

	// Start health check server
	go startHealthServer(cfg.HTTPPort, logger)

	// Create context for graceful shutdown
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < cfg.WorkerConcurrency; i++ {
		wg.Add(1)
		go func(workerNum int) {
			defer wg.Done()
			worker.run(ctx, workerNum)
		}(i)
	}

	logger.Info("worker started", "concurrency", cfg.WorkerConcurrency)

	// Wait for shutdown signal
	<-quit
	logger.Info("shutting down worker...")
	cancel()

	// Wait for all workers to finish
	wg.Wait()
	logger.Info("worker stopped")
}

func (w *Worker) run(ctx context.Context, workerNum int) {
	logger := w.logger.With("goroutine", workerNum)

	for {
		select {
		case <-ctx.Done():
			logger.Info("worker goroutine stopping")
			return
		default:
			// Consume a message
			msg, err := w.consumer.Consume(ctx)
			if err != nil {
				logger.Error("failed to consume message", "error", err)
				time.Sleep(time.Second)
				continue
			}

			if msg == nil {
				// No message available, continue polling
				continue
			}

			// Process the job
			if err := w.processJob(ctx, msg); err != nil {
				logger.Error("failed to process job", "job_id", msg.Job.JobID, "error", err)
			}

			// Acknowledge the message
			if err := w.consumer.Acknowledge(ctx, msg.ID); err != nil {
				logger.Error("failed to acknowledge message", "error", err)
			}
		}
	}
}

func (w *Worker) processJob(ctx context.Context, msg *queue.Message) error {
	jobID := msg.Job.JobID
	logger := w.logger.With("job_id", jobID)

	logger.Info("starting job processing")

	// Get job from database
	job, err := w.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	if job == nil {
		logger.Warn("job not found, skipping")
		return nil
	}

	// Check if job is canceled
	if job.Status == models.JobStatusCancelled {
		logger.Info("job canceled, skipping")
		return nil
	}

	// Mark job as processing
	if err := w.jobRepo.StartProcessing(ctx, jobID, w.id); err != nil {
		return fmt.Errorf("failed to start processing: %w", err)
	}

	// Download original image
	logger.Info("downloading original image", "key", job.OriginalKey)
	reader, err := w.storage.Download(ctx, job.OriginalKey)
	if err != nil {
		w.jobRepo.FailJob(ctx, jobID, "failed to download image: "+err.Error())
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer reader.Close()

	// Process the image
	logger.Info("processing image", "operations", len(msg.Job.Operations))
	result, err := w.processor.Process(reader, job.ContentType, msg.Job.Operations)
	if err != nil {
		w.jobRepo.FailJob(ctx, jobID, "failed to process image: "+err.Error())
		return fmt.Errorf("failed to process image: %w", err)
	}

	// Generate processed key
	processedKey := fmt.Sprintf("processed/%s/%s", jobID.String(), job.OriginalName)

	// Upload processed image
	logger.Info("uploading processed image", "key", processedKey, "size", len(result.Data))
	if err := w.storage.Upload(ctx, processedKey, bytes.NewReader(result.Data), int64(len(result.Data)), result.ContentType); err != nil {
		w.jobRepo.FailJob(ctx, jobID, "failed to upload processed image: "+err.Error())
		return fmt.Errorf("failed to upload processed image: %w", err)
	}

	// Mark job as completed - set 1 hour retention for cleanup
	if err := w.jobRepo.CompleteJob(ctx, jobID, processedKey, 1); err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	logger.Info("job completed successfully")
	return nil
}

func startHealthServer(port int, logger *slog.Logger) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	addr := fmt.Sprintf(":%d", port)
	logger.Info("starting health server", "addr", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("health server error", "error", err)
	}
}
