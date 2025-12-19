package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJobStatus_Constants(t *testing.T) {
	tests := []struct {
		status JobStatus
		want   string
	}{
		{JobStatusPending, "pending"},
		{JobStatusQueued, "queued"},
		{JobStatusProcessing, "processing"},
		{JobStatusCompleted, "completed"},
		{JobStatusFailed, "failed"},
		{JobStatusCancelled, "canceled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("JobStatus = %q, want %q", tt.status, tt.want)
		}
	}
}

func TestOperationType_Constants(t *testing.T) {
	tests := []struct {
		opType OperationType
		want   string
	}{
		{OperationResize, "resize"},
		{OperationThumbnail, "thumbnail"},
		{OperationBlur, "blur"},
		{OperationSharpen, "sharpen"},
		{OperationGrayscale, "grayscale"},
		{OperationSepia, "sepia"},
		{OperationRotate, "rotate"},
		{OperationFlip, "flip"},
		{OperationBrightness, "brightness"},
		{OperationContrast, "contrast"},
		{OperationSaturation, "saturation"},
		{OperationWatermark, "watermark"},
	}

	for _, tt := range tests {
		if string(tt.opType) != tt.want {
			t.Errorf("OperationType = %q, want %q", tt.opType, tt.want)
		}
	}
}

func TestNewJob(t *testing.T) {
	originalKey := "original/test/image.jpg"
	originalName := "image.jpg"
	contentType := "image/jpeg"
	fileSize := int64(12345)
	operations := []Operation{
		{Operation: OperationResize, Parameters: map[string]interface{}{"width": 100, "height": 100}},
		{Operation: OperationBlur, Parameters: map[string]interface{}{"sigma": 2.0}},
	}

	before := time.Now()
	job := NewJob(originalKey, originalName, contentType, fileSize, operations)
	after := time.Now()

	// Verify ID is set and valid
	if job.ID == uuid.Nil {
		t.Error("Job ID should not be nil")
	}

	// Verify status is pending
	if job.Status != JobStatusPending {
		t.Errorf("Status = %q, want %q", job.Status, JobStatusPending)
	}

	// Verify fields are set correctly
	if job.OriginalKey != originalKey {
		t.Errorf("OriginalKey = %q, want %q", job.OriginalKey, originalKey)
	}
	if job.OriginalName != originalName {
		t.Errorf("OriginalName = %q, want %q", job.OriginalName, originalName)
	}
	if job.ContentType != contentType {
		t.Errorf("ContentType = %q, want %q", job.ContentType, contentType)
	}
	if job.FileSize != fileSize {
		t.Errorf("FileSize = %d, want %d", job.FileSize, fileSize)
	}
	if len(job.Operations) != len(operations) {
		t.Errorf("len(Operations) = %d, want %d", len(job.Operations), len(operations))
	}
	if job.Progress != 0 {
		t.Errorf("Progress = %d, want 0", job.Progress)
	}

	// Verify timestamps are within expected range
	if job.CreatedAt.Before(before) || job.CreatedAt.After(after) {
		t.Errorf("CreatedAt = %v, should be between %v and %v", job.CreatedAt, before, after)
	}
	if job.UpdatedAt.Before(before) || job.UpdatedAt.After(after) {
		t.Errorf("UpdatedAt = %v, should be between %v and %v", job.UpdatedAt, before, after)
	}
}

func TestNewJob_EmptyOperations(t *testing.T) {
	job := NewJob("key", "name", "image/png", 100, nil)

	if job.Operations != nil {
		t.Errorf("Operations = %v, want nil", job.Operations)
	}
}

func TestJob_MarshalOperations(t *testing.T) {
	job := &Job{
		Operations: []Operation{
			{Operation: OperationResize, Parameters: map[string]interface{}{"width": 100}},
			{Operation: OperationGrayscale},
		},
	}

	err := job.MarshalOperations()
	if err != nil {
		t.Fatalf("MarshalOperations() error = %v", err)
	}

	if job.OperationsJSON == "" {
		t.Error("OperationsJSON should not be empty after marshaling")
	}

	// Verify the JSON is valid
	var ops []Operation
	if err := json.Unmarshal([]byte(job.OperationsJSON), &ops); err != nil {
		t.Errorf("OperationsJSON is not valid JSON: %v", err)
	}

	if len(ops) != 2 {
		t.Errorf("Unmarshaled %d operations, want 2", len(ops))
	}
}

func TestJob_MarshalOperations_Empty(t *testing.T) {
	job := &Job{
		Operations: []Operation{},
	}

	err := job.MarshalOperations()
	if err != nil {
		t.Fatalf("MarshalOperations() error = %v", err)
	}

	if job.OperationsJSON != "[]" {
		t.Errorf("OperationsJSON = %q, want []", job.OperationsJSON)
	}
}

func TestJob_MarshalOperations_Nil(t *testing.T) {
	job := &Job{
		Operations: nil,
	}

	err := job.MarshalOperations()
	if err != nil {
		t.Fatalf("MarshalOperations() error = %v", err)
	}

	if job.OperationsJSON != "null" {
		t.Errorf("OperationsJSON = %q, want null", job.OperationsJSON)
	}
}

func TestJob_UnmarshalOperations(t *testing.T) {
	job := &Job{
		OperationsJSON: `[{"operation":"resize","parameters":{"width":200,"height":150}},{"operation":"blur","parameters":{"sigma":1.5}}]`,
	}

	err := job.UnmarshalOperations()
	if err != nil {
		t.Fatalf("UnmarshalOperations() error = %v", err)
	}

	if len(job.Operations) != 2 {
		t.Fatalf("len(Operations) = %d, want 2", len(job.Operations))
	}

	// Verify first operation
	if job.Operations[0].Operation != OperationResize {
		t.Errorf("Operations[0].Operation = %q, want resize", job.Operations[0].Operation)
	}
	if job.Operations[0].Parameters["width"] != float64(200) {
		t.Errorf("Operations[0].Parameters[width] = %v, want 200", job.Operations[0].Parameters["width"])
	}

	// Verify second operation
	if job.Operations[1].Operation != OperationBlur {
		t.Errorf("Operations[1].Operation = %q, want blur", job.Operations[1].Operation)
	}
}

func TestJob_UnmarshalOperations_Empty(t *testing.T) {
	job := &Job{
		OperationsJSON: "",
	}

	err := job.UnmarshalOperations()
	if err != nil {
		t.Fatalf("UnmarshalOperations() error = %v", err)
	}

	if job.Operations == nil {
		t.Error("Operations should be empty slice, not nil")
	}
	if len(job.Operations) != 0 {
		t.Errorf("len(Operations) = %d, want 0", len(job.Operations))
	}
}

func TestJob_UnmarshalOperations_EmptyArray(t *testing.T) {
	job := &Job{
		OperationsJSON: "[]",
	}

	err := job.UnmarshalOperations()
	if err != nil {
		t.Fatalf("UnmarshalOperations() error = %v", err)
	}

	if len(job.Operations) != 0 {
		t.Errorf("len(Operations) = %d, want 0", len(job.Operations))
	}
}

func TestJob_UnmarshalOperations_InvalidJSON(t *testing.T) {
	job := &Job{
		OperationsJSON: "invalid json",
	}

	err := job.UnmarshalOperations()
	if err == nil {
		t.Error("UnmarshalOperations() should return error for invalid JSON")
	}
}

func TestJob_MarshalUnmarshal_RoundTrip(t *testing.T) {
	original := &Job{
		Operations: []Operation{
			{Operation: OperationResize, Parameters: map[string]interface{}{"width": 100, "height": 200}},
			{Operation: OperationSharpen, Parameters: map[string]interface{}{"sigma": 0.5}},
			{Operation: OperationGrayscale},
		},
	}

	// Marshal
	err := original.MarshalOperations()
	if err != nil {
		t.Fatalf("MarshalOperations() error = %v", err)
	}

	// Create new job with same JSON
	restored := &Job{
		OperationsJSON: original.OperationsJSON,
	}

	// Unmarshal
	err = restored.UnmarshalOperations()
	if err != nil {
		t.Fatalf("UnmarshalOperations() error = %v", err)
	}

	// Verify
	if len(restored.Operations) != len(original.Operations) {
		t.Fatalf("len(Operations) = %d, want %d", len(restored.Operations), len(original.Operations))
	}

	for i, op := range restored.Operations {
		if op.Operation != original.Operations[i].Operation {
			t.Errorf("Operations[%d].Operation = %q, want %q", i, op.Operation, original.Operations[i].Operation)
		}
	}
}

func TestJob_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	job := &Job{
		ID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Status:       JobStatusCompleted,
		OriginalKey:  "original/test.jpg",
		ProcessedKey: "processed/test.jpg",
		OriginalName: "test.jpg",
		ContentType:  "image/jpeg",
		FileSize:     12345,
		Operations: []Operation{
			{Operation: OperationResize, Parameters: map[string]interface{}{"width": 100}},
		},
		Progress:  100,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Marshal to JSON
	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Verify the JSON contains expected fields
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if result["id"] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("JSON id = %v, want UUID string", result["id"])
	}
	if result["status"] != "completed" {
		t.Errorf("JSON status = %v, want completed", result["status"])
	}

	// OperationsJSON should not be in the output (json:"-" tag)
	if _, exists := result["operations_json"]; exists {
		t.Error("OperationsJSON should not be in JSON output")
	}
}

func TestJobMessage_JSON(t *testing.T) {
	msg := &JobMessage{
		JobID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Operations: []Operation{
			{Operation: OperationBlur, Parameters: map[string]interface{}{"sigma": 2.0}},
		},
	}

	// Marshal
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal
	var restored JobMessage
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if restored.JobID != msg.JobID {
		t.Errorf("JobID = %v, want %v", restored.JobID, msg.JobID)
	}
	if len(restored.Operations) != 1 {
		t.Errorf("len(Operations) = %d, want 1", len(restored.Operations))
	}
}

func TestCreateJobRequest_JSON(t *testing.T) {
	req := &CreateJobRequest{
		Operations: []Operation{
			{Operation: OperationThumbnail, Parameters: map[string]interface{}{"size": 150}},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var restored CreateJobRequest
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(restored.Operations) != 1 {
		t.Errorf("len(Operations) = %d, want 1", len(restored.Operations))
	}
	if restored.Operations[0].Operation != OperationThumbnail {
		t.Errorf("Operation = %q, want thumbnail", restored.Operations[0].Operation)
	}
}

func TestJobListResponse_JSON(t *testing.T) {
	now := time.Now()
	resp := &JobListResponse{
		Jobs: []*Job{
			{ID: uuid.New(), Status: JobStatusPending, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), Status: JobStatusCompleted, CreatedAt: now, UpdatedAt: now},
		},
		Total:      100,
		Page:       2,
		PageSize:   10,
		TotalPages: 10,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var restored JobListResponse
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(restored.Jobs) != 2 {
		t.Errorf("len(Jobs) = %d, want 2", len(restored.Jobs))
	}
	if restored.Total != 100 {
		t.Errorf("Total = %d, want 100", restored.Total)
	}
	if restored.Page != 2 {
		t.Errorf("Page = %d, want 2", restored.Page)
	}
	if restored.PageSize != 10 {
		t.Errorf("PageSize = %d, want 10", restored.PageSize)
	}
	if restored.TotalPages != 10 {
		t.Errorf("TotalPages = %d, want 10", restored.TotalPages)
	}
}

func TestQueueStats_JSON(t *testing.T) {
	stats := &QueueStats{
		StreamLength:    1000,
		PendingMessages: 50,
		ConsumerCount:   4,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var restored QueueStats
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if restored.StreamLength != 1000 {
		t.Errorf("StreamLength = %d, want 1000", restored.StreamLength)
	}
	if restored.PendingMessages != 50 {
		t.Errorf("PendingMessages = %d, want 50", restored.PendingMessages)
	}
	if restored.ConsumerCount != 4 {
		t.Errorf("ConsumerCount = %d, want 4", restored.ConsumerCount)
	}
}

func TestOperation_Parameters(t *testing.T) {
	op := Operation{
		Operation: OperationResize,
		Parameters: map[string]interface{}{
			"width":  float64(800),
			"height": float64(600),
			"fit":    "contain",
		},
	}

	// Test JSON round-trip
	data, err := json.Marshal(op)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var restored Operation
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if restored.Operation != OperationResize {
		t.Errorf("Operation = %q, want resize", restored.Operation)
	}
	if restored.Parameters["width"] != float64(800) {
		t.Errorf("Parameters[width] = %v, want 800", restored.Parameters["width"])
	}
	if restored.Parameters["fit"] != "contain" {
		t.Errorf("Parameters[fit] = %v, want contain", restored.Parameters["fit"])
	}
}

func TestOperation_NoParameters(t *testing.T) {
	op := Operation{
		Operation: OperationGrayscale,
	}

	data, err := json.Marshal(op)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Verify parameters field is omitted
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if _, exists := result["parameters"]; exists {
		t.Error("Parameters should be omitted when nil/empty")
	}
}
