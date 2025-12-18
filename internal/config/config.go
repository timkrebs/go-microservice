package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the application
type Config struct {
	// HTTP server settings
	HTTPPort         int           `envconfig:"HTTP_PORT" default:"8080"`
	ReadTimeout      time.Duration `envconfig:"READ_TIMEOUT" default:"30s"`
	WriteTimeout     time.Duration `envconfig:"WRITE_TIMEOUT" default:"30s"`
	ShutdownTimeout  time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"10s"`
	MaxUploadSize    int64         `envconfig:"MAX_UPLOAD_SIZE" default:"52428800"` // 50MB

	// Database settings
	DatabaseURL     string `envconfig:"DATABASE_URL" default:"postgres://postgres:postgres@localhost:5432/imageprocessor?sslmode=disable"`
	DatabaseMaxConn int    `envconfig:"DATABASE_MAX_CONN" default:"25"`

	// Redis settings
	RedisAddr     string `envconfig:"REDIS_ADDR" default:"localhost:6379"`
	RedisPassword string `envconfig:"REDIS_PASSWORD" default:""`
	RedisDB       int    `envconfig:"REDIS_DB" default:"0"`

	// MinIO settings
	MinIOEndpoint  string `envconfig:"MINIO_ENDPOINT" default:"localhost:9000"`
	MinIOAccessKey string `envconfig:"MINIO_ACCESS_KEY" default:"minioadmin"`
	MinIOSecretKey string `envconfig:"MINIO_SECRET_KEY" default:"minioadmin"`
	MinIOBucket    string `envconfig:"MINIO_BUCKET" default:"images"`
	MinIOUseSSL    bool   `envconfig:"MINIO_USE_SSL" default:"false"`

	// Worker settings
	WorkerConcurrency int           `envconfig:"WORKER_CONCURRENCY" default:"4"`
	WorkerPollTimeout time.Duration `envconfig:"WORKER_POLL_TIMEOUT" default:"5s"`

	// Queue settings
	QueueStreamName    string `envconfig:"QUEUE_STREAM_NAME" default:"image-jobs"`
	QueueConsumerGroup string `envconfig:"QUEUE_CONSUMER_GROUP" default:"workers"`

	// Logging
	LogLevel  string `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat string `envconfig:"LOG_FORMAT" default:"json"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
