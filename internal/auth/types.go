package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Role represents user authorization level
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleViewer Role = "viewer"
)

// IsValid checks if the role is a valid role
func (r Role) IsValid() bool {
	return r == RoleAdmin || r == RoleViewer
}

// CanWrite returns true if the role has write permissions
func (r Role) CanWrite() bool {
	return r == RoleAdmin
}

// User represents an authenticated user from config
type User struct {
	Username     string `mapstructure:"username"`
	PasswordHash string `mapstructure:"password_hash"`
	Role         Role   `mapstructure:"role"`
	Enabled      bool   `mapstructure:"enabled"`
}

// APIKey represents a programmatic API key from config
type APIKey struct {
	Name      string     `mapstructure:"name"`
	KeyHash   string     `mapstructure:"key_hash"`
	Role      Role       `mapstructure:"role"`
	Enabled   bool       `mapstructure:"enabled"`
	ExpiresAt *time.Time `mapstructure:"expires_at,omitempty"`
}

// IsExpired checks if the API key has expired
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// Claims represents JWT claims
type Claims struct {
	Username string `json:"username"`
	Role     Role   `json:"role"`
	jwt.RegisteredClaims
}

// AuthenticatedUser represents the authenticated user in request context
type AuthenticatedUser struct {
	Username string
	Role     Role
	AuthType string // "jwt" or "apikey"
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token     string       `json:"token"`
	ExpiresAt string       `json:"expiresAt"`
	User      UserResponse `json:"user"`
}

// UserResponse represents user info in responses
type UserResponse struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// contextKey is the type for context keys
type contextKey string

// UserContextKey is the context key for the authenticated user
const UserContextKey contextKey = "auth_user"
