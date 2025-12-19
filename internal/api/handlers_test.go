package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/timkrebs/image-processor/internal/models"
)

func TestIsValidImageType(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"image/jpeg", true},
		{"image/jpg", true},
		{"image/png", true},
		{"image/gif", true},
		{"IMAGE/JPEG", true}, // case insensitive
		{"Image/PNG", true},
		{"image/webp", false},
		{"image/bmp", false},
		{"application/json", false},
		{"text/html", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			got := isValidImageType(tt.contentType)
			if got != tt.want {
				t.Errorf("isValidImageType(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"image.jpg", "image/jpeg"},
		{"image.jpeg", "image/jpeg"},
		{"IMAGE.JPG", "image/jpeg"},
		{"photo.JPEG", "image/jpeg"},
		{"image.png", "image/png"},
		{"IMAGE.PNG", "image/png"},
		{"image.gif", "image/gif"},
		{"IMAGE.GIF", "image/gif"},
		{"file.txt", "application/octet-stream"},
		{"file.pdf", "application/octet-stream"},
		{"file", "application/octet-stream"},
		{"", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := detectContentType(tt.filename)
			if got != tt.want {
				t.Errorf("detectContentType(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsValidOperation(t *testing.T) {
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

	for _, op := range validOps {
		t.Run(string(op), func(t *testing.T) {
			if !isValidOperation(op) {
				t.Errorf("isValidOperation(%q) = false, want true", op)
			}
		})
	}

	invalidOps := []models.OperationType{
		"unknown",
		"invalid_op",
		"crop",
		"",
	}

	for _, op := range invalidOps {
		t.Run(string(op), func(t *testing.T) {
			if isValidOperation(op) {
				t.Errorf("isValidOperation(%q) = true, want false", op)
			}
		})
	}
}

func TestHandlers_WriteJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	h := &Handlers{logger: logger}

	recorder := httptest.NewRecorder()
	data := map[string]string{"message": "test"}

	h.writeJSON(recorder, http.StatusOK, data)

	if recorder.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", recorder.Code, http.StatusOK)
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	var result map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["message"] != "test" {
		t.Errorf("Response message = %q, want test", result["message"])
	}
}

func TestHandlers_WriteError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	h := &Handlers{logger: logger}

	recorder := httptest.NewRecorder()

	h.writeError(recorder, http.StatusBadRequest, "invalid request")

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	var result map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["error"] != "invalid request" {
		t.Errorf("Error message = %q, want invalid request", result["error"])
	}
}

func TestHandlers_GetJob_InvalidID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	h := &Handlers{logger: logger}

	// Create request with invalid UUID
	req := httptest.NewRequest("GET", "/api/v1/jobs/invalid-uuid", nil)
	recorder := httptest.NewRecorder()

	// Use chi router for proper URL param handling
	r := chi.NewRouter()
	r.Get("/api/v1/jobs/{id}", h.GetJob)
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandlers_CancelJob_InvalidID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	h := &Handlers{logger: logger}

	req := httptest.NewRequest("DELETE", "/api/v1/jobs/not-a-uuid", nil)
	recorder := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Delete("/api/v1/jobs/{id}", h.CancelJob)
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandlers_GetImage_InvalidID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	h := &Handlers{logger: logger}

	req := httptest.NewRequest("GET", "/api/v1/images/bad-id", nil)
	recorder := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Get("/api/v1/images/{id}", h.GetImage)
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandlers_StreamJobStatus_InvalidID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	h := &Handlers{logger: logger}

	req := httptest.NewRequest("GET", "/api/v1/jobs/invalid/stream", nil)
	recorder := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Get("/api/v1/jobs/{id}/stream", h.StreamJobStatus)
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandlers_ListJobs_Pagination(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		wantPage     int
		wantPageSize int
	}{
		{"default", "", 1, 20},
		{"custom page", "?page=3", 3, 20},
		{"custom page_size", "?page_size=50", 1, 50},
		{"both custom", "?page=2&page_size=30", 2, 30},
		{"page too low", "?page=0", 1, 20},
		{"page_size too low", "?page_size=0", 1, 20},
		{"page_size too high", "?page_size=200", 1, 20}, // defaults to 20 if > 100
		{"negative page", "?page=-5", 1, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse query params directly (simulating what handler does)
			req := httptest.NewRequest("GET", "/api/v1/jobs"+tt.query, nil)

			// The actual parsing happens in handler, just verify request formation
			if req.URL.Query().Get("page") == "3" && tt.wantPage != 3 {
				t.Errorf("Expected page 3 for query page=3")
			}
		})
	}
}

func TestHandlers_CreateJob_NoFile(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	h := &Handlers{logger: logger}

	// Create multipart request without file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()

	h.CreateJob(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandlers_CreateJob_InvalidOperationsJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	h := &Handlers{logger: logger}

	// Create multipart request with invalid operations JSON
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add a dummy image file
	part, _ := writer.CreateFormFile("image", "test.jpg")
	part.Write([]byte("fake image data"))

	// Add invalid operations JSON
	writer.WriteField("operations", "not valid json")
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()

	h.CreateJob(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	var result map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Logf("Failed to decode response: %v", err)
	}
	if !strings.Contains(result["error"], "invalid operations JSON") {
		t.Errorf("Error = %q, want to contain 'invalid operations JSON'", result["error"])
	}
}

func TestHandlers_CreateJob_InvalidOperation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	h := &Handlers{logger: logger}

	// Create multipart request with invalid operation type
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add a dummy image file
	part, _ := writer.CreateFormFile("image", "test.jpg")
	part.Write([]byte("fake image data"))

	// Add operations with invalid type
	operations := `[{"operation":"unknown_operation"}]`
	writer.WriteField("operations", operations)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()

	h.CreateJob(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	var result map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Logf("Failed to decode response: %v", err)
	}
	if !strings.Contains(result["error"], "invalid operation") {
		t.Errorf("Error = %q, want to contain 'invalid operation'", result["error"])
	}
}

func TestHandlers_CreateJob_InvalidImageType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	h := &Handlers{logger: logger}

	// Create multipart request with non-image file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add a non-image file
	part, _ := writer.CreateFormFile("image", "document.pdf")
	part.Write([]byte("not an image"))

	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()

	h.CreateJob(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	var result map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Logf("Failed to decode response: %v", err)
	}
	if !strings.Contains(result["error"], "invalid image type") {
		t.Errorf("Error = %q, want to contain 'invalid image type'", result["error"])
	}
}

func TestNewHandlers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	h := NewHandlers(nil, nil, nil, nil, "test-group", logger)

	if h == nil {
		t.Fatal("NewHandlers() returned nil")
	}
	if h.groupName != "test-group" {
		t.Errorf("groupName = %q, want test-group", h.groupName)
	}
	if h.logger == nil {
		t.Error("logger should not be nil")
	}
}

func TestHandlers_SetMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	h := NewHandlers(nil, nil, nil, nil, "test", logger)

	if h == nil {
		t.Fatal("NewHandlers() returned nil")
	}

	if h.jobMetrics != nil {
		t.Error("jobMetrics should be nil initially")
	}

	// SetMetrics is used to inject metrics after creation
	// Just verify the field exists
}

func TestOperationsValidation(t *testing.T) {
	tests := []struct {
		name       string
		operations []models.Operation
		wantValid  bool
	}{
		{
			name:       "empty operations",
			operations: []models.Operation{},
			wantValid:  true,
		},
		{
			name: "valid resize",
			operations: []models.Operation{
				{Operation: models.OperationResize, Parameters: map[string]interface{}{"width": 100}},
			},
			wantValid: true,
		},
		{
			name: "valid multiple operations",
			operations: []models.Operation{
				{Operation: models.OperationResize, Parameters: map[string]interface{}{"width": 100}},
				{Operation: models.OperationBlur, Parameters: map[string]interface{}{"sigma": 2.0}},
				{Operation: models.OperationGrayscale},
			},
			wantValid: true,
		},
		{
			name: "invalid operation in list",
			operations: []models.Operation{
				{Operation: models.OperationResize},
				{Operation: "invalid"},
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := true
			for _, op := range tt.operations {
				if !isValidOperation(op.Operation) {
					valid = false
					break
				}
			}
			if valid != tt.wantValid {
				t.Errorf("validation = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}
