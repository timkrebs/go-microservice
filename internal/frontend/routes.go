package frontend

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates the frontend HTTP router
func NewRouter(handlers *Handlers, staticDir string, logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	// Static files
	fileServer := http.FileServer(http.Dir(staticDir))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Frontend pages
	r.Get("/", handlers.Home)
	r.Get("/upload", handlers.Upload)
	r.Get("/jobs", handlers.Jobs)
	r.Get("/jobs/{id}", handlers.JobDetail)

	// API proxy (so frontend can make API calls to same origin)
	r.HandleFunc("/api/*", handlers.ProxyAPI)

	return r
}
