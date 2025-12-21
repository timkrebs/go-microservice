package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/timkrebs/image-processor/internal/database"
	"github.com/timkrebs/image-processor/internal/models"
)

// AuthHandlers handles authentication-related HTTP requests
type AuthHandlers struct {
	db           *database.DB
	sessionStore *SessionStore
	logger       *slog.Logger
}

// NewAuthHandlers creates a new authentication handlers instance
func NewAuthHandlers(db *database.DB, sessionStore *SessionStore, logger *slog.Logger) *AuthHandlers {
	return &AuthHandlers{
		db:           db,
		sessionStore: sessionStore,
		logger:       logger,
	}
}

// writeJSON writes a JSON response
func (h *AuthHandlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON response", "error", err)
	}
}

// writeError writes an error response
func (h *AuthHandlers) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

// Register handles POST /auth/register
func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create user
	user, err := h.db.CreateUser(r.Context(), &req)
	if err != nil {
		switch err {
		case database.ErrEmailAlreadyExists:
			h.writeError(w, http.StatusConflict, "email already registered")
		case database.ErrUsernameAlreadyExists:
			h.writeError(w, http.StatusConflict, "username already taken")
		default:
			h.logger.Error("failed to create user", "error", err)
			h.writeError(w, http.StatusInternalServerError, "failed to create user")
		}
		return
	}

	h.logger.Info("user registered", "user_id", user.ID, "email", user.Email)

	// Create session
	sessionID, err := h.sessionStore.Create(user)
	if err != nil {
		h.logger.Error("failed to create session", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 hours
	})

	// Update last login
	if err := h.db.UpdateLastLogin(r.Context(), user.ID); err != nil {
		h.logger.Error("failed to update last login", "error", err)
	}

	h.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"user": user,
	})
}

// Login handles POST /auth/login
func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get user by email
	user, err := h.db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if err == database.ErrUserNotFound {
			h.writeError(w, http.StatusUnauthorized, "invalid email or password")
		} else {
			h.logger.Error("failed to get user", "error", err)
			h.writeError(w, http.StatusInternalServerError, "login failed")
		}
		return
	}

	// Check password
	if passErr := user.CheckPassword(req.Password); passErr != nil {
		h.writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	h.logger.Info("user logged in", "user_id", user.ID, "email", user.Email)

	// Create session
	sessionID, err := h.sessionStore.Create(user)
	if err != nil {
		h.logger.Error("failed to create session", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 hours
	})

	// Update last login
	if err := h.db.UpdateLastLogin(r.Context(), user.ID); err != nil {
		h.logger.Error("failed to update last login", "error", err)
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

// Logout handles POST /auth/logout
func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	// Get session from cookie
	cookie, err := r.Cookie("session_id")
	if err == nil {
		// Delete session
		h.sessionStore.Delete(cookie.Value)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // Delete cookie
	})

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

// GetCurrentUser handles GET /auth/me
func (h *AuthHandlers) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	session, ok := GetSession(r.Context())
	if !ok || session == nil {
		h.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	// Get full user details from database
	user, err := h.db.GetUserByID(r.Context(), session.UserID)
	if err != nil {
		if err == database.ErrUserNotFound {
			h.writeError(w, http.StatusUnauthorized, "user not found")
		} else {
			h.logger.Error("failed to get user", "error", err)
			h.writeError(w, http.StatusInternalServerError, "failed to get user")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}
