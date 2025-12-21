package models

import (
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidEmail     = errors.New("invalid email address")
	ErrInvalidUsername  = errors.New("invalid username (3-100 characters, alphanumeric and _ only)")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrInvalidPassword  = errors.New("invalid password")
)

var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,100}$`)
)

const bcryptCost = 12

// User represents a user account
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"` // Never serialize password hash
	Username     string     `json:"username"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// CreateUserRequest represents the data needed to create a new user
type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate validates the user creation request
func (r *CreateUserRequest) Validate() error {
	// Validate email
	if !emailRegex.MatchString(r.Email) {
		return ErrInvalidEmail
	}

	// Validate username
	if !usernameRegex.MatchString(r.Username) {
		return ErrInvalidUsername
	}

	// Validate password
	if len(r.Password) < 8 {
		return ErrPasswordTooShort
	}

	return nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword compares a plain text password with a hashed password
func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
}

// Sanitize returns a copy of the user with sensitive fields removed
func (u *User) Sanitize() *User {
	return &User{
		ID:          u.ID,
		Email:       u.Email,
		Username:    u.Username,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
		LastLoginAt: u.LastLoginAt,
	}
}
