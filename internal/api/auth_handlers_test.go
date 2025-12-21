package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/timkrebs/image-processor/internal/database"
	"github.com/timkrebs/image-processor/internal/models"
)

func setupAuthTest(t *testing.T) (*AuthHandlers, *database.DB, *SessionStore) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	db, err := database.New(dbURL, 5)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	sessionStore := NewSessionStore(24 * time.Hour)
	handlers := NewAuthHandlers(db, sessionStore, logger)

	return handlers, db, sessionStore
}

func TestAuthHandlers_Register(t *testing.T) {
	handlers, db, _ := setupAuthTest(t)
	defer db.Close()

	tests := []struct {
		name           string
		payload        map[string]string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful registration",
			payload: map[string]string{
				"email":    "test@example.com",
				"password": "password123",
				"username": "testuser",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid email",
			payload: map[string]string{
				"email":    "invalid-email",
				"password": "password123",
				"username": "testuser",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid email address",
		},
		{
			name: "password too short",
			payload: map[string]string{
				"email":    "test2@example.com",
				"password": "short",
				"username": "testuser2",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "password must be at least 8 characters",
		},
		{
			name: "invalid username",
			payload: map[string]string{
				"email":    "test3@example.com",
				"password": "password123",
				"username": "ab",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid username (3-100 characters, alphanumeric and _ only)",
		},
		{
			name: "duplicate email",
			payload: map[string]string{
				"email":    "test@example.com",
				"password": "password123",
				"username": "testuser_new",
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "email already registered",
		},
		{
			name: "missing fields",
			payload: map[string]string{
				"email": "test4@example.com",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handlers.Register(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var resp map[string]string
				json.NewDecoder(w.Body).Decode(&resp)
				if resp["error"] != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, resp["error"])
				}
			}

			if w.Code == http.StatusCreated {
				var resp map[string]interface{}
				json.NewDecoder(w.Body).Decode(&resp)
				if resp["user"] == nil {
					t.Error("expected user in response")
				}

				cookies := w.Result().Cookies()
				found := false
				for _, cookie := range cookies {
					if cookie.Name == "session_id" {
						found = true
						if cookie.Value == "" {
							t.Error("session_id cookie should not be empty")
						}
						if !cookie.HttpOnly {
							t.Error("session_id cookie should be HttpOnly")
						}
						break
					}
				}
				if !found {
					t.Error("expected session_id cookie")
				}
			}
		})
	}
}

func TestAuthHandlers_Login(t *testing.T) {
	handlers, db, _ := setupAuthTest(t)
	defer db.Close()

	testEmail := "login_test@example.com"
	testPassword := "password123"

	_, err := db.CreateUser(context.Background(), &models.CreateUserRequest{
		Email:    testEmail,
		Password: testPassword,
		Username: "loginuser",
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	tests := []struct {
		name           string
		payload        map[string]string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful login",
			payload: map[string]string{
				"email":    testEmail,
				"password": testPassword,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "wrong password",
			payload: map[string]string{
				"email":    testEmail,
				"password": "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid email or password",
		},
		{
			name: "nonexistent email",
			payload: map[string]string{
				"email":    "nonexistent@example.com",
				"password": "password123",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid email or password",
		},
		{
			name:           "invalid request body",
			payload:        map[string]string{},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid email or password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handlers.Login(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var resp map[string]string
				json.NewDecoder(w.Body).Decode(&resp)
				if resp["error"] != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, resp["error"])
				}
			}

			if w.Code == http.StatusOK {
				var resp map[string]interface{}
				json.NewDecoder(w.Body).Decode(&resp)
				if resp["user"] == nil {
					t.Error("expected user in response")
				}

				cookies := w.Result().Cookies()
				found := false
				for _, cookie := range cookies {
					if cookie.Name == "session_id" {
						found = true
						break
					}
				}
				if !found {
					t.Error("expected session_id cookie")
				}
			}
		})
	}
}

func TestAuthHandlers_Logout(t *testing.T) {
	handlers, db, sessionStore := setupAuthTest(t)
	defer db.Close()

	userID := uuid.New()
	user := &models.User{
		ID:       userID,
		Email:    "test@example.com",
		Username: "testuser",
	}
	sessionID, _ := sessionStore.Create(user)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	})
	w := httptest.NewRecorder()

	handlers.Logout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	cookies := w.Result().Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == "session_id" {
			found = true
			if cookie.MaxAge != -1 {
				t.Error("session_id cookie should be deleted")
			}
			break
		}
	}
	if !found {
		t.Error("expected session_id cookie to be cleared")
	}

	_, exists := sessionStore.Get(sessionID)
	if exists {
		t.Error("session should be deleted from store")
	}
}

func TestAuthHandlers_GetCurrentUser(t *testing.T) {
	handlers, db, sessionStore := setupAuthTest(t)
	defer db.Close()

	user, err := db.CreateUser(context.Background(), &models.CreateUserRequest{
		Email:    "current_user@example.com",
		Password: "password123",
		Username: "currentuser",
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	sessionID, _ := sessionStore.Create(user)

	tests := []struct {
		name           string
		sessionID      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "authenticated request",
			sessionID:      sessionID,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthenticated request",
			sessionID:      "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "not authenticated",
		},
		{
			name:           "invalid session",
			sessionID:      "invalid-session-id",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "not authenticated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/auth/me", http.NoBody)
			if tt.sessionID != "" {
				req.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: tt.sessionID,
				})
			}

			ctx := req.Context()
			if tt.sessionID == sessionID {
				sess, _ := sessionStore.Get(tt.sessionID)
				if sess != nil {
					ctx = context.WithValue(ctx, sessionContextKey, sess)
				}
			}
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handlers.GetCurrentUser(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var resp map[string]string
				json.NewDecoder(w.Body).Decode(&resp)
				if resp["error"] != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, resp["error"])
				}
			}

			if w.Code == http.StatusOK {
				var resp map[string]interface{}
				json.NewDecoder(w.Body).Decode(&resp)
				if resp["user"] == nil {
					t.Error("expected user in response")
				}
				userMap := resp["user"].(map[string]interface{})
				if userMap["email"] != user.Email {
					t.Errorf("expected email %s, got %s", user.Email, userMap["email"])
				}
			}
		})
	}
}
