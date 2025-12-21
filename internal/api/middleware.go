package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/timkrebs/image-processor/internal/metrics"
	"github.com/timkrebs/image-processor/internal/models"
)

// StructuredLogger returns a middleware that logs HTTP requests using slog
func StructuredLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				logger.Info("request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"bytes", ww.BytesWritten(),
					"duration_ms", time.Since(start).Milliseconds(),
					"remote_addr", r.RemoteAddr,
					"user_agent", r.UserAgent(),
					"request_id", middleware.GetReqID(r.Context()),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// MaxUploadSize limits the size of uploaded files
func MaxUploadSize(maxSize int64) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" || r.Method == "PUT" {
				r.Body = http.MaxBytesReader(w, r.Body, maxSize)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CORS adds CORS headers for development
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// MetricsMiddleware instruments HTTP requests with Prometheus metrics
func MetricsMiddleware(m *metrics.HTTPMetrics) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			m.RequestsInFlight.Inc()
			defer m.RequestsInFlight.Dec()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(ww.Status())
			path := r.URL.Path

			m.RequestDuration.WithLabelValues(r.Method, path, status).Observe(duration)
			m.RequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		})
	}
}

// Session represents a user session
type Session struct {
	UserID    uuid.UUID
	Email     string
	Username  string
	ExpiresAt time.Time
}

// SessionStore manages user sessions
type SessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	ttl      time.Duration
}

// NewSessionStore creates a new session store
func NewSessionStore(ttl time.Duration) *SessionStore {
	store := &SessionStore{
		sessions: make(map[string]*Session),
		ttl:      ttl,
	}

	// Start cleanup goroutine
	go store.cleanupExpired()

	return store
}

// Create creates a new session
func (s *SessionStore) Create(user *models.User) (string, error) {
	// Generate session ID
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	sessionID := base64.URLEncoding.EncodeToString(b)

	session := &Session{
		UserID:    user.ID,
		Email:     user.Email,
		Username:  user.Username,
		ExpiresAt: time.Now().Add(s.ttl),
	}

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	return sessionID, nil
}

// Get retrieves a session by ID
func (s *SessionStore) Get(sessionID string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	return session, true
}

// Delete removes a session
func (s *SessionStore) Delete(sessionID string) {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
}

// cleanupExpired removes expired sessions periodically
func (s *SessionStore) cleanupExpired() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for id, session := range s.sessions {
			if now.After(session.ExpiresAt) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

// Context keys
type contextKey string

const (
	userContextKey    contextKey = "user"
	sessionContextKey contextKey = "session"
)

// AuthRequired middleware ensures the user is authenticated
func AuthRequired(store *SessionStore) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			session, exists := store.Get(cookie.Value)
			if !exists {
				http.Error(w, `{"error":"session expired"}`, http.StatusUnauthorized)
				return
			}

			// Add session to context
			ctx := context.WithValue(r.Context(), sessionContextKey, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth middleware extracts session if present but doesn't require it
func OptionalAuth(store *SessionStore) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err == nil {
				if session, exists := store.Get(cookie.Value); exists {
					ctx := context.WithValue(r.Context(), sessionContextKey, session)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetSession retrieves the session from context
func GetSession(ctx context.Context) (*Session, bool) {
	session, ok := ctx.Value(sessionContextKey).(*Session)
	return session, ok
}
