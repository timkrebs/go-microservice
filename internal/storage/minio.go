package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/timkrebs/image-processor/internal/metrics"
)

// Storage provides object storage operations
type Storage struct {
	client     *minio.Client
	metrics    *metrics.StorageMetrics
	bucketName string
}

// Config holds MinIO configuration
type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// New creates a new storage client
func New(cfg Config) (*Storage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &Storage{
		client:     client,
		bucketName: cfg.Bucket,
	}, nil
}

// SetMetrics injects metrics collectors into storage client
func (s *Storage) SetMetrics(m *metrics.StorageMetrics) {
	s.metrics = m
}

// EnsureBucket creates the bucket if it doesn't exist
func (s *Storage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = s.client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}

// Upload uploads a file to storage
func (s *Storage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	start := time.Now()
	status := "success"

	_, err := s.client.PutObject(ctx, s.bucketName, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})

	if err != nil {
		status = "error"
	}

	if s.metrics != nil {
		duration := time.Since(start).Seconds()
		s.metrics.OperationDuration.WithLabelValues("upload", status).Observe(duration)
		s.metrics.OperationsTotal.WithLabelValues("upload", status).Inc()
		if status == "success" {
			s.metrics.BytesTransferred.WithLabelValues("upload").Add(float64(size))
		}
	}

	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}
	return nil
}

// Download downloads a file from storage
func (s *Storage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	start := time.Now()
	status := "success"

	obj, err := s.client.GetObject(ctx, s.bucketName, key, minio.GetObjectOptions{})

	if err != nil {
		status = "error"
	}

	if s.metrics != nil {
		duration := time.Since(start).Seconds()
		s.metrics.OperationDuration.WithLabelValues("download", status).Observe(duration)
		s.metrics.OperationsTotal.WithLabelValues("download", status).Inc()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	return obj, nil
}

// Delete removes a file from storage
func (s *Storage) Delete(ctx context.Context, key string) error {
	start := time.Now()
	status := "success"

	err := s.client.RemoveObject(ctx, s.bucketName, key, minio.RemoveObjectOptions{})

	if err != nil {
		status = "error"
	}

	if s.metrics != nil {
		duration := time.Since(start).Seconds()
		s.metrics.OperationDuration.WithLabelValues("delete", status).Observe(duration)
		s.metrics.OperationsTotal.WithLabelValues("delete", status).Inc()
	}

	return err
}

// GetPresignedURL generates a presigned URL for downloading
func (s *Storage) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, s.bucketName, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url.String(), nil
}

// Stat retrieves object metadata
func (s *Storage) Stat(ctx context.Context, key string) (*minio.ObjectInfo, error) {
	info, err := s.client.StatObject(ctx, s.bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}
	return &info, nil
}

// Health checks if storage is accessible
func (s *Storage) Health(ctx context.Context) error {
	_, err := s.client.BucketExists(ctx, s.bucketName)
	return err
}
