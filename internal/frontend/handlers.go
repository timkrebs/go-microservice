package frontend

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/timkrebs/image-processor/internal/models"
)

// Handlers holds frontend HTTP handlers
type Handlers struct {
	templates *Templates
	logger    *slog.Logger
	client    *http.Client
	apiURL    string
}

// NewHandlers creates new frontend handlers
func NewHandlers(apiURL string, logger *slog.Logger) *Handlers {
	return &Handlers{
		templates: NewTemplates(),
		apiURL:    apiURL,
		logger:    logger,
		client:    &http.Client{},
	}
}

// Home renders the home page
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	// Fetch queue stats from API
	var stats *models.QueueStats
	resp, err := h.client.Get(h.apiURL + "/api/v1/stats/queue")
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		stats = &models.QueueStats{}
		if err := json.NewDecoder(resp.Body).Decode(stats); err != nil {
			stats = nil // Reset on decode error
		}
	}

	data := PageData{
		Title:  "Dashboard",
		Active: "home",
		Content: HomeData{
			Stats: stats,
		},
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.Render(w, "home", data); err != nil {
		h.logger.Error("failed to render home", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Upload renders the upload page
func (h *Handlers) Upload(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title:   "Upload",
		Active:  "upload",
		Content: nil,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.Render(w, "upload", data); err != nil {
		h.logger.Error("failed to render upload", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Jobs renders the jobs list page
func (h *Handlers) Jobs(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	// Fetch jobs from API
	resp, err := h.client.Get(h.apiURL + "/api/v1/jobs?page=" + strconv.Itoa(page) + "&page_size=12")
	if err != nil {
		h.logger.Error("failed to fetch jobs", "error", err)
		http.Error(w, "Failed to fetch jobs", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var jobsResp models.JobListResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobsResp); err != nil {
		h.logger.Error("failed to decode jobs", "error", err)
		http.Error(w, "Failed to decode jobs", http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:  "Jobs",
		Active: "jobs",
		Content: JobsData{
			Jobs:       jobsResp.Jobs,
			Page:       jobsResp.Page,
			TotalPages: jobsResp.TotalPages,
			Total:      jobsResp.Total,
		},
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.Render(w, "jobs", data); err != nil {
		h.logger.Error("failed to render jobs", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// JobDetail renders a single job detail page
func (h *Handlers) JobDetail(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")

	// Fetch job from API
	resp, err := h.client.Get(h.apiURL + "/api/v1/jobs/" + jobID)
	if err != nil {
		h.logger.Error("failed to fetch job", "error", err)
		http.Error(w, "Failed to fetch job", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	var job models.Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		h.logger.Error("failed to decode job", "error", err)
		http.Error(w, "Failed to decode job", http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:  "Job Details",
		Active: "jobs",
		Content: JobDetailData{
			Job: &job,
		},
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.Render(w, "job-detail", data); err != nil {
		h.logger.Error("failed to render job detail", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ProxyAPI proxies requests to the API service
func (h *Handlers) ProxyAPI(w http.ResponseWriter, r *http.Request) {
	// Create proxy request
	proxyURL := h.apiURL + r.URL.Path
	if r.URL.RawQuery != "" {
		proxyURL += "?" + r.URL.RawQuery
	}

	proxyReq, err := http.NewRequest(r.Method, proxyURL, r.Body)
	if err != nil {
		h.logger.Error("failed to create proxy request", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Execute request
	resp, err := h.client.Do(proxyReq)
	if err != nil {
		h.logger.Error("failed to proxy request", "error", err)
		http.Error(w, "API unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		// Response headers already sent, can't return error to client
		h.logger.Error("failed to copy response body", "error", err)
	}
}
