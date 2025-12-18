package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the current state of a processing job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusQueued     JobStatus = "queued"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "canceled"
)

// OperationType represents the type of image processing operation
type OperationType string

const (
	OperationResize     OperationType = "resize"
	OperationThumbnail  OperationType = "thumbnail"
	OperationBlur       OperationType = "blur"
	OperationSharpen    OperationType = "sharpen"
	OperationGrayscale  OperationType = "grayscale"
	OperationSepia      OperationType = "sepia"
	OperationRotate     OperationType = "rotate"
	OperationFlip       OperationType = "flip"
	OperationBrightness OperationType = "brightness"
	OperationContrast   OperationType = "contrast"
	OperationSaturation OperationType = "saturation"
	OperationWatermark  OperationType = "watermark"
)

// Operation represents a single image processing operation
type Operation struct {
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Operation  OperationType          `json:"operation"`
}

// Job represents an image processing job
type Job struct {
	Operations     []Operation `json:"operations" db:"-"`
	StartedAt      *time.Time  `json:"started_at,omitempty" db:"started_at"`
	CompletedAt    *time.Time  `json:"completed_at,omitempty" db:"completed_at"`
	ProcessingTime *int64      `json:"processing_time_ms,omitempty" db:"processing_time_ms"`
	OriginalKey    string      `json:"original_key" db:"original_key"`
	ProcessedKey   string      `json:"processed_key,omitempty" db:"processed_key"`
	OriginalName   string      `json:"original_name" db:"original_name"`
	ContentType    string      `json:"content_type" db:"content_type"`
	OperationsJSON string      `json:"-" db:"operations"`
	Error          string      `json:"error,omitempty" db:"error"`
	WorkerID       string      `json:"worker_id,omitempty" db:"worker_id"`
	ID             uuid.UUID   `json:"id" db:"id"`
	CreatedAt      time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at" db:"updated_at"`
	FileSize       int64       `json:"file_size" db:"file_size"`
	Status         JobStatus   `json:"status" db:"status"`
	Progress       int         `json:"progress" db:"progress"`
}

// NewJob creates a new job with the given parameters
func NewJob(originalKey, originalName, contentType string, fileSize int64, operations []Operation) *Job {
	now := time.Now()
	return &Job{
		ID:           uuid.New(),
		Status:       JobStatusPending,
		OriginalKey:  originalKey,
		OriginalName: originalName,
		ContentType:  contentType,
		FileSize:     fileSize,
		Operations:   operations,
		Progress:     0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// MarshalOperations serializes operations to JSON for database storage
func (j *Job) MarshalOperations() error {
	data, err := json.Marshal(j.Operations)
	if err != nil {
		return err
	}
	j.OperationsJSON = string(data)
	return nil
}

// UnmarshalOperations deserializes operations from JSON
func (j *Job) UnmarshalOperations() error {
	if j.OperationsJSON == "" {
		j.Operations = []Operation{}
		return nil
	}
	return json.Unmarshal([]byte(j.OperationsJSON), &j.Operations)
}

// JobMessage represents a job message in the queue
type JobMessage struct {
	Operations []Operation `json:"operations"`
	JobID      uuid.UUID   `json:"job_id"`
}

// CreateJobRequest represents the request to create a new job
type CreateJobRequest struct {
	Operations []Operation `json:"operations"`
}

// JobListResponse represents a paginated list of jobs
type JobListResponse struct {
	Jobs       []*Job `json:"jobs"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalPages int    `json:"total_pages"`
}

// QueueStats represents queue statistics
type QueueStats struct {
	StreamLength    int64 `json:"stream_length"`
	PendingMessages int64 `json:"pending_messages"`
	ConsumerCount   int64 `json:"consumer_count"`
}
