package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear relevant environment variables to test defaults
	envVars := []string{
		"DATABASE_URL", "REDIS_ADDR", "REDIS_PASSWORD", "REDIS_DB",
		"MINIO_ENDPOINT", "MINIO_ACCESS_KEY", "MINIO_SECRET_KEY", "MINIO_BUCKET", "MINIO_USE_SSL",
		"QUEUE_STREAM_NAME", "QUEUE_CONSUMER_GROUP",
		"LOG_LEVEL", "LOG_FORMAT",
		"READ_TIMEOUT", "WRITE_TIMEOUT", "SHUTDOWN_TIMEOUT",
		"WORKER_POLL_TIMEOUT", "MAX_UPLOAD_SIZE", "HTTP_PORT",
		"DATABASE_MAX_CONN", "WORKER_CONCURRENCY",
	}

	// Save and clear env vars
	savedEnv := make(map[string]string)
	for _, key := range envVars {
		savedEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}

	// Restore env vars after test
	defer func() {
		for key, value := range savedEnv {
			if value != "" {
				os.Setenv(key, value)
			}
		}
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	// Test database defaults
	if cfg.DatabaseURL != "postgres://postgres:postgres@localhost:5432/imageprocessor?sslmode=disable" {
		t.Errorf("DatabaseURL = %q, want default value", cfg.DatabaseURL)
	}
	if cfg.DatabaseMaxConn != 25 {
		t.Errorf("DatabaseMaxConn = %d, want 25", cfg.DatabaseMaxConn)
	}

	// Test Redis defaults
	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("RedisAddr = %q, want localhost:6379", cfg.RedisAddr)
	}
	if cfg.RedisPassword != "" {
		t.Errorf("RedisPassword = %q, want empty", cfg.RedisPassword)
	}
	if cfg.RedisDB != 0 {
		t.Errorf("RedisDB = %d, want 0", cfg.RedisDB)
	}

	// Test MinIO defaults
	if cfg.MinIOEndpoint != "localhost:9000" {
		t.Errorf("MinIOEndpoint = %q, want localhost:9000", cfg.MinIOEndpoint)
	}
	if cfg.MinIOAccessKey != "minioadmin" {
		t.Errorf("MinIOAccessKey = %q, want minioadmin", cfg.MinIOAccessKey)
	}
	if cfg.MinIOSecretKey != "minioadmin" {
		t.Errorf("MinIOSecretKey = %q, want minioadmin", cfg.MinIOSecretKey)
	}
	if cfg.MinIOBucket != "images" {
		t.Errorf("MinIOBucket = %q, want images", cfg.MinIOBucket)
	}
	if cfg.MinIOUseSSL != false {
		t.Errorf("MinIOUseSSL = %v, want false", cfg.MinIOUseSSL)
	}

	// Test Queue defaults
	if cfg.QueueStreamName != "image-jobs" {
		t.Errorf("QueueStreamName = %q, want image-jobs", cfg.QueueStreamName)
	}
	if cfg.QueueConsumerGroup != "workers" {
		t.Errorf("QueueConsumerGroup = %q, want workers", cfg.QueueConsumerGroup)
	}

	// Test Logging defaults
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want json", cfg.LogFormat)
	}

	// Test HTTP server defaults
	if cfg.ReadTimeout != 30*time.Second {
		t.Errorf("ReadTimeout = %v, want 30s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 30*time.Second {
		t.Errorf("WriteTimeout = %v, want 30s", cfg.WriteTimeout)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 10s", cfg.ShutdownTimeout)
	}
	if cfg.HTTPPort != 8080 {
		t.Errorf("HTTPPort = %d, want 8080", cfg.HTTPPort)
	}

	// Test Worker defaults
	if cfg.WorkerPollTimeout != 5*time.Second {
		t.Errorf("WorkerPollTimeout = %v, want 5s", cfg.WorkerPollTimeout)
	}
	if cfg.MaxUploadSize != 52428800 {
		t.Errorf("MaxUploadSize = %d, want 52428800", cfg.MaxUploadSize)
	}
	if cfg.WorkerConcurrency != 4 {
		t.Errorf("WorkerConcurrency = %d, want 4", cfg.WorkerConcurrency)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	// Set custom environment variables
	testEnv := map[string]string{
		"DATABASE_URL":         "postgres://user:pass@db:5432/testdb",
		"REDIS_ADDR":           "redis:6380",
		"REDIS_PASSWORD":       "secret",
		"REDIS_DB":             "1",
		"MINIO_ENDPOINT":       "minio:9001",
		"MINIO_ACCESS_KEY":     "testkey",
		"MINIO_SECRET_KEY":     "testsecret",
		"MINIO_BUCKET":         "testbucket",
		"MINIO_USE_SSL":        "true",
		"QUEUE_STREAM_NAME":    "test-jobs",
		"QUEUE_CONSUMER_GROUP": "test-workers",
		"LOG_LEVEL":            "debug",
		"LOG_FORMAT":           "text",
		"READ_TIMEOUT":         "60s",
		"WRITE_TIMEOUT":        "120s",
		"SHUTDOWN_TIMEOUT":     "30s",
		"WORKER_POLL_TIMEOUT":  "10s",
		"MAX_UPLOAD_SIZE":      "104857600",
		"HTTP_PORT":            "9090",
		"DATABASE_MAX_CONN":    "50",
		"WORKER_CONCURRENCY":   "8",
	}

	// Save current env and set test values
	savedEnv := make(map[string]string)
	for key, value := range testEnv {
		savedEnv[key] = os.Getenv(key)
		os.Setenv(key, value)
	}

	// Restore original env after test
	defer func() {
		for key, value := range savedEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	// Verify custom values
	if cfg.DatabaseURL != "postgres://user:pass@db:5432/testdb" {
		t.Errorf("DatabaseURL = %q, want custom value", cfg.DatabaseURL)
	}
	if cfg.RedisAddr != "redis:6380" {
		t.Errorf("RedisAddr = %q, want redis:6380", cfg.RedisAddr)
	}
	if cfg.RedisPassword != "secret" {
		t.Errorf("RedisPassword = %q, want secret", cfg.RedisPassword)
	}
	if cfg.RedisDB != 1 {
		t.Errorf("RedisDB = %d, want 1", cfg.RedisDB)
	}
	if cfg.MinIOEndpoint != "minio:9001" {
		t.Errorf("MinIOEndpoint = %q, want minio:9001", cfg.MinIOEndpoint)
	}
	if cfg.MinIOAccessKey != "testkey" {
		t.Errorf("MinIOAccessKey = %q, want testkey", cfg.MinIOAccessKey)
	}
	if cfg.MinIOSecretKey != "testsecret" {
		t.Errorf("MinIOSecretKey = %q, want testsecret", cfg.MinIOSecretKey)
	}
	if cfg.MinIOBucket != "testbucket" {
		t.Errorf("MinIOBucket = %q, want testbucket", cfg.MinIOBucket)
	}
	if cfg.MinIOUseSSL != true {
		t.Errorf("MinIOUseSSL = %v, want true", cfg.MinIOUseSSL)
	}
	if cfg.QueueStreamName != "test-jobs" {
		t.Errorf("QueueStreamName = %q, want test-jobs", cfg.QueueStreamName)
	}
	if cfg.QueueConsumerGroup != "test-workers" {
		t.Errorf("QueueConsumerGroup = %q, want test-workers", cfg.QueueConsumerGroup)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("LogFormat = %q, want text", cfg.LogFormat)
	}
	if cfg.ReadTimeout != 60*time.Second {
		t.Errorf("ReadTimeout = %v, want 60s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 120*time.Second {
		t.Errorf("WriteTimeout = %v, want 120s", cfg.WriteTimeout)
	}
	if cfg.ShutdownTimeout != 30*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 30s", cfg.ShutdownTimeout)
	}
	if cfg.WorkerPollTimeout != 10*time.Second {
		t.Errorf("WorkerPollTimeout = %v, want 10s", cfg.WorkerPollTimeout)
	}
	if cfg.MaxUploadSize != 104857600 {
		t.Errorf("MaxUploadSize = %d, want 104857600", cfg.MaxUploadSize)
	}
	if cfg.HTTPPort != 9090 {
		t.Errorf("HTTPPort = %d, want 9090", cfg.HTTPPort)
	}
	if cfg.DatabaseMaxConn != 50 {
		t.Errorf("DatabaseMaxConn = %d, want 50", cfg.DatabaseMaxConn)
	}
	if cfg.WorkerConcurrency != 8 {
		t.Errorf("WorkerConcurrency = %d, want 8", cfg.WorkerConcurrency)
	}
}

func TestLoad_InvalidDuration(t *testing.T) {
	savedTimeout := os.Getenv("READ_TIMEOUT")
	os.Setenv("READ_TIMEOUT", "invalid")
	defer func() {
		if savedTimeout == "" {
			os.Unsetenv("READ_TIMEOUT")
		} else {
			os.Setenv("READ_TIMEOUT", savedTimeout)
		}
	}()

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error for invalid duration")
	}
}

func TestLoad_InvalidInt(t *testing.T) {
	savedPort := os.Getenv("HTTP_PORT")
	os.Setenv("HTTP_PORT", "not-a-number")
	defer func() {
		if savedPort == "" {
			os.Unsetenv("HTTP_PORT")
		} else {
			os.Setenv("HTTP_PORT", savedPort)
		}
	}()

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error for invalid integer")
	}
}

func TestLoad_InvalidBool(t *testing.T) {
	savedSSL := os.Getenv("MINIO_USE_SSL")
	os.Setenv("MINIO_USE_SSL", "not-a-bool")
	defer func() {
		if savedSSL == "" {
			os.Unsetenv("MINIO_USE_SSL")
		} else {
			os.Setenv("MINIO_USE_SSL", savedSSL)
		}
	}()

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error for invalid boolean")
	}
}
