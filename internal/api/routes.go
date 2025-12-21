package api

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/timkrebs/image-processor/internal/database"
	"github.com/timkrebs/image-processor/internal/metrics"
)

// NewRouter creates a new HTTP router with all routes configured
func NewRouter(handlers *Handlers, httpMetrics *metrics.HTTPMetrics, maxUploadSize int64, db *database.DB, sessionStore *SessionStore, logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Create auth handlers
	authHandlers := NewAuthHandlers(db, sessionStore, logger)

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(StructuredLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(CORS)
	r.Use(MaxUploadSize(maxUploadSize))
	r.Use(MetricsMiddleware(httpMetrics))
	r.Use(OptionalAuth(sessionStore))

	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Health check
		r.Get("/health", handlers.Health)

		// Authentication
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandlers.Register)
			r.Post("/login", authHandlers.Login)
			r.Post("/logout", authHandlers.Logout)
			r.With(AuthRequired(sessionStore)).Get("/me", authHandlers.GetCurrentUser)
		})

		// Jobs
		r.Route("/jobs", func(r chi.Router) {
			r.Post("/", handlers.CreateJob)
			r.Get("/", handlers.ListJobs)
			r.Get("/{id}", handlers.GetJob)
			r.Get("/{id}/stream", handlers.StreamJobStatus)
			r.Delete("/{id}", handlers.CancelJob)
		})

		// Images
		r.Get("/images/{id}", handlers.GetImage)

		// Stats
		r.Get("/stats/queue", handlers.GetQueueStats)
	})

	return r
}
