package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/timkrebs/image-processor/internal/database"
	"github.com/timkrebs/image-processor/internal/metrics"
	"github.com/timkrebs/image-processor/internal/models"
	"github.com/timkrebs/image-processor/internal/queue"
	"github.com/timkrebs/image-processor/internal/storage"
)

// Handlers holds all HTTP handlers
type Handlers struct {
	jobRepo    *database.JobRepository
	storage    *storage.Storage
	producer   *queue.Producer
	logger     *slog.Logger
	db         *database.DB
	jobMetrics *metrics.JobMetrics
	groupName  string
}

// NewHandlers creates a new handlers instance
func NewHandlers(
	jobRepo *database.JobRepository,
	storage *storage.Storage,
	producer *queue.Producer,
	db *database.DB,
	groupName string,
	logger *slog.Logger,
) *Handlers {
	return &Handlers{
		jobRepo:   jobRepo,
		storage:   storage,
		producer:  producer,
		db:        db,
		groupName: groupName,
		logger:    logger,
	}
}

// SetMetrics injects metrics collectors into handlers
func (h *Handlers) SetMetrics(jobMetrics *metrics.JobMetrics) {
	h.jobMetrics = jobMetrics
}

// writeJSON writes a JSON response
func (h *Handlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON response", "error", err)
	}
}

// writeError writes an error response
func (h *Handlers) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

// CreateJob handles POST /api/v1/jobs
func (h *Handlers) CreateJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user session (optional for backward compatibility)
	session, ok := GetSession(ctx)
	var userID uuid.UUID
	if ok && session != nil {
		userID = session.UserID
	} else {
		// Use system user for unauthenticated requests
		userID = uuid.MustParse("00000000-0000-0000-0000-000000000000")
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(50 << 20); err != nil { // 50MB max
		h.writeError(w, http.StatusBadRequest, "failed to parse form: "+err.Error())
		return
	}

	// Get the uploaded file
	file, header, err := r.FormFile("image")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "image file is required")
		return
	}
	defer file.Close()

	// Validate content type
	contentType := header.Header.Get("Content-Type")
	if !isValidImageType(contentType) {
		// Try to detect from extension
		contentType = detectContentType(header.Filename)
		if !isValidImageType(contentType) {
			h.writeError(w, http.StatusBadRequest, "invalid image type, must be JPEG, PNG, or GIF")
			return
		}
	}

	// Parse operations
	operationsJSON := r.FormValue("operations")
	var operations []models.Operation
	if operationsJSON != "" {
		if err := json.Unmarshal([]byte(operationsJSON), &operations); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid operations JSON: "+err.Error())
			return
		}
	}

	// Validate operations
	for _, op := range operations {
		if !isValidOperation(op.Operation) {
			h.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid operation: %s", op.Operation))
			return
		}
	}

	// Default operation if none provided
	if len(operations) == 0 {
		operations = []models.Operation{
			{Operation: models.OperationThumbnail, Parameters: map[string]interface{}{"size": 150}},
		}
	}

	// Generate storage key with user isolation
	id := uuid.New()
	originalKey := fmt.Sprintf("users/%s/original/%s/%s", userID.String(), id.String(), header.Filename)

	// Upload to storage
	if err := h.storage.Upload(ctx, originalKey, file, header.Size, contentType); err != nil {
		h.logger.Error("failed to upload file", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to upload file")
		return
	}

	// Create job
	job := models.NewJob(originalKey, header.Filename, contentType, header.Size, operations)
	job.ID = id
	job.UserID = userID

	if err := h.jobRepo.Create(ctx, job); err != nil {
		h.logger.Error("failed to create job", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	// Update status to queued
	if err := h.jobRepo.UpdateStatus(ctx, job.ID, models.JobStatusQueued); err != nil {
		h.logger.Error("failed to update job status", "error", err)
	}
	job.Status = models.JobStatusQueued

	// Enqueue the job
	msg := &models.JobMessage{
		JobID:      job.ID,
		Operations: operations,
	}
	if err := h.producer.Enqueue(ctx, msg); err != nil {
		h.logger.Error("failed to enqueue job", "error", err)
		// Update status back to pending on queue failure
		if updateErr := h.jobRepo.UpdateStatus(ctx, job.ID, models.JobStatusPending); updateErr != nil {
			h.logger.Error("failed to update job status", "error", updateErr)
		}
		h.writeError(w, http.StatusInternalServerError, "failed to queue job")
		return
	}

	h.logger.Info("job created", "job_id", job.ID, "operations", len(operations))
	h.writeJSON(w, http.StatusCreated, job)
}

// GetJob handles GET /api/v1/jobs/{id}
func (h *Handlers) GetJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	job, err := h.jobRepo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get job", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get job")
		return
	}

	if job == nil {
		h.writeError(w, http.StatusNotFound, "job not found")
		return
	}

	h.writeJSON(w, http.StatusOK, job)
}

// ListJobs handles GET /api/v1/jobs
func (h *Handlers) ListJobs(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Get user session for filtering
	session, ok := GetSession(r.Context())
	var userID uuid.UUID
	if ok && session != nil {
		userID = session.UserID
	} else {
		// Use system user for unauthenticated requests
		userID = uuid.MustParse("00000000-0000-0000-0000-000000000000")
	}

	jobs, total, err := h.jobRepo.List(r.Context(), userID, page, pageSize)
	if err != nil {
		h.logger.Error("failed to list jobs", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to list jobs")
		return
	}

	totalPages := (total + pageSize - 1) / pageSize

	response := models.JobListResponse{
		Jobs:       jobs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// CancelJob handles DELETE /api/v1/jobs/{id}
func (h *Handlers) CancelJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	if err := h.jobRepo.CancelJob(r.Context(), id); err != nil {
		h.logger.Error("failed to cancel job", "error", err)
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "canceled"})
}

// GetImage handles GET /api/v1/images/{id}
func (h *Handlers) GetImage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid image ID")
		return
	}

	job, err := h.jobRepo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get job", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get job")
		return
	}

	if job == nil {
		h.writeError(w, http.StatusNotFound, "image not found")
		return
	}

	// Determine which image to serve (processed or original)
	imageKey := job.ProcessedKey
	if imageKey == "" || r.URL.Query().Get("original") == "true" {
		imageKey = job.OriginalKey
	}

	// Download from storage
	reader, err := h.storage.Download(r.Context(), imageKey)
	if err != nil {
		h.logger.Error("failed to download image", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to download image")
		return
	}
	defer reader.Close()

	// Set content type
	w.Header().Set("Content-Type", job.ContentType)

	// Stream the image
	if _, err := io.Copy(w, reader); err != nil {
		h.logger.Error("failed to stream image", "error", err)
	}
}

// GetQueueStats handles GET /api/v1/stats/queue
func (h *Handlers) GetQueueStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.producer.GetStats(r.Context(), h.groupName)
	if err != nil {
		h.logger.Error("failed to get queue stats", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get queue stats")
		return
	}

	h.writeJSON(w, http.StatusOK, stats)
}

// StreamJobStatus handles GET /api/v1/jobs/{id}/stream
// Streams job status updates using Server-Sent Events (SSE)
func (h *Handlers) StreamJobStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	// Check if job exists
	job, err := h.jobRepo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get job", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get job")
		return
	}
	if job == nil {
		h.writeError(w, http.StatusNotFound, "job not found")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Create flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Send initial job status
	data, _ := json.Marshal(job)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	// Poll for updates while job is not terminal
	ticker := time.NewTicker(500 * time.Millisecond) // Update every 500ms
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return
		case <-ticker.C:
			// Fetch updated job
			job, err := h.jobRepo.GetByID(ctx, id)
			if err != nil {
				h.logger.Error("failed to get job during stream", "error", err)
				return
			}
			if job == nil {
				return
			}

			// Send update
			data, _ := json.Marshal(job)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

			// Stop streaming if job is terminal
			if job.Status == models.JobStatusCompleted ||
				job.Status == models.JobStatusFailed ||
				job.Status == models.JobStatusCancelled {
				return
			}
		}
	}
}

// Health handles GET /api/v1/health
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status := "healthy"
	checks := make(map[string]interface{})

	// Check database
	if err := h.db.Health(ctx); err != nil {
		status = "unhealthy"
		checks["database"] = map[string]string{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		dbStats := h.db.Stats()
		checks["database"] = map[string]interface{}{
			"status":           "healthy",
			"open_connections": dbStats.OpenConnections,
			"in_use":           dbStats.InUse,
			"idle":             dbStats.Idle,
		}
	}

	// Check storage
	if err := h.storage.Health(ctx); err != nil {
		status = "unhealthy"
		checks["storage"] = map[string]string{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		checks["storage"] = map[string]string{
			"status": "healthy",
		}
	}

	// Check Redis (via queue stats)
	if _, err := h.producer.GetStats(ctx, h.groupName); err != nil {
		status = "unhealthy"
		checks["redis"] = map[string]string{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		checks["redis"] = map[string]string{
			"status": "healthy",
		}
	}

	response := map[string]interface{}{
		"status": status,
		"checks": checks,
	}

	statusCode := http.StatusOK
	if status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	h.writeJSON(w, statusCode, response)
}

// Helper functions

func isValidImageType(contentType string) bool {
	validTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
	}
	for _, t := range validTypes {
		if strings.EqualFold(contentType, t) {
			return true
		}
	}
	return false
}

func detectContentType(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}

func isValidOperation(op models.OperationType) bool {
	validOps := []models.OperationType{
		models.OperationResize,
		models.OperationThumbnail,
		models.OperationBlur,
		models.OperationSharpen,
		models.OperationGrayscale,
		models.OperationSepia,
		models.OperationRotate,
		models.OperationFlip,
		models.OperationBrightness,
		models.OperationContrast,
		models.OperationSaturation,
		models.OperationWatermark,
	}
	for _, valid := range validOps {
		if op == valid {
			return true
		}
	}
	return false
}
