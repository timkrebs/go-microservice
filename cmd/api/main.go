package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/timkrebs/image-processor/internal/api"
	"github.com/timkrebs/image-processor/internal/config"
	"github.com/timkrebs/image-processor/internal/database"
	"github.com/timkrebs/image-processor/internal/metrics"
	"github.com/timkrebs/image-processor/internal/queue"
	"github.com/timkrebs/image-processor/internal/storage"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

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
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("failed to close database", "error", err)
		}
	}()
	logger.Info("connected to database")

	// Create job repository
	jobRepo := database.NewJobRepository(db)

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("failed to close redis", "error", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		cancel()
		logger.Error("failed to connect to redis", "error", err)
		return
	}
	cancel()
	logger.Info("connected to redis")

	// Create queue producer
	producer := queue.NewProducer(redisClient, cfg.QueueStreamName)

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

	// Ensure bucket exists
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	if err := storageClient.EnsureBucket(ctx); err != nil {
		cancel()
		logger.Error("failed to ensure bucket", "error", err)
		os.Exit(1)
	}
	cancel()
	logger.Info("connected to minio", "bucket", cfg.MinIOBucket)

	// Initialize metrics
	httpMetrics := metrics.NewHTTPMetrics("image_processor_api")
	jobMetrics := metrics.NewJobMetrics("image_processor_api")
	storageMetrics := metrics.NewStorageMetrics("image_processor_api")
	dbMetrics := metrics.NewDatabaseMetrics("image_processor_api")

	// Inject metrics into storage client
	storageClient.SetMetrics(storageMetrics)

	// Inject metrics into database
	db.SetMetrics(dbMetrics)

	// Create session store with 24 hour TTL
	sessionStore := api.NewSessionStore(24 * time.Hour)

	// Create handlers
	handlers := api.NewHandlers(jobRepo, storageClient, producer, db, cfg.QueueConsumerGroup, logger)
	handlers.SetMetrics(jobMetrics)

	// Create router
	router := api.NewRouter(handlers, httpMetrics, cfg.MaxUploadSize, db, sessionStore, logger)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("starting API server", "port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down server...")

	// Graceful shutdown
	ctx, cancel = context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	logger.Info("server stopped")
}
