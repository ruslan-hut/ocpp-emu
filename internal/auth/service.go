package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserDisabled       = errors.New("user is disabled")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrInvalidAPIKey      = errors.New("invalid API key")
	ErrAPIKeyDisabled     = errors.New("API key is disabled")
	ErrAPIKeyExpired      = errors.New("API key has expired")
)

// Config holds auth configuration
type Config struct {
	Enabled   bool          `mapstructure:"enabled"`
	JWTSecret string        `mapstructure:"jwt_secret"`
	JWTExpiry time.Duration `mapstructure:"jwt_expiry"`
	Users     []User        `mapstructure:"users"`
	APIKeys   []APIKey      `mapstructure:"api_keys"`
}

// Service handles authentication logic
type Service struct {
	users     map[string]*User   // username -> user
	apiKeys   map[string]*APIKey // key hash -> api key
	jwtSecret []byte
	jwtExpiry time.Duration
	logger    *slog.Logger
	enabled   bool
}

// NewService creates a new auth service from config
func NewService(cfg *Config, logger *slog.Logger) *Service {
	s := &Service{
		users:     make(map[string]*User),
		apiKeys:   make(map[string]*APIKey),
		jwtSecret: []byte(cfg.JWTSecret),
		jwtExpiry: cfg.JWTExpiry,
		logger:    logger,
		enabled:   cfg.Enabled,
	}

	// Default JWT expiry to 24 hours
	if s.jwtExpiry == 0 {
		s.jwtExpiry = 24 * time.Hour
	}

	// Index users by username
	for i := range cfg.Users {
		user := &cfg.Users[i]
		s.users[user.Username] = user
		logger.Info("loaded user", "username", user.Username, "role", user.Role, "enabled", user.Enabled)
	}

	// Index API keys by hash
	for i := range cfg.APIKeys {
		key := &cfg.APIKeys[i]
		s.apiKeys[key.KeyHash] = key
		logger.Info("loaded API key", "name", key.Name, "role", key.Role, "enabled", key.Enabled)
	}

	return s
}

// IsEnabled returns whether auth is enabled
func (s *Service) IsEnabled() bool {
	return s.enabled
}

// AuthenticateUser validates username/password and returns JWT token
func (s *Service) AuthenticateUser(username, password string) (string, time.Time, error) {
	user, ok := s.users[username]
	if !ok {
		s.logger.Warn("login attempt for unknown user", "username", username)
		return "", time.Time{}, ErrInvalidCredentials
	}

	if !user.Enabled {
		s.logger.Warn("login attempt for disabled user", "username", username)
		return "", time.Time{}, ErrUserDisabled
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.logger.Warn("invalid password", "username", username)
		return "", time.Time{}, ErrInvalidCredentials
	}

	// Generate JWT token
	expiresAt := time.Now().Add(s.jwtExpiry)
	claims := &Claims{
		Username: username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ocpp-emu",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		s.logger.Error("failed to sign JWT", "error", err)
		return "", time.Time{}, err
	}

	s.logger.Info("user logged in", "username", username, "role", user.Role)
	return tokenString, expiresAt, nil
}

// ValidateJWT validates JWT token and returns claims
func (s *Service) ValidateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateAPIKey validates API key and returns associated key info
func (s *Service) ValidateAPIKey(apiKey string) (*APIKey, error) {
	// Hash the provided key
	keyHash := HashAPIKey(apiKey)

	key, ok := s.apiKeys[keyHash]
	if !ok {
		s.logger.Warn("invalid API key attempt")
		return nil, ErrInvalidAPIKey
	}

	if !key.Enabled {
		s.logger.Warn("disabled API key attempt", "name", key.Name)
		return nil, ErrAPIKeyDisabled
	}

	if key.IsExpired() {
		s.logger.Warn("expired API key attempt", "name", key.Name)
		return nil, ErrAPIKeyExpired
	}

	return key, nil
}

// GetUser returns user info by username
func (s *Service) GetUser(username string) (*User, bool) {
	user, ok := s.users[username]
	return user, ok
}

// generateToken generates a JWT token for a user (internal use for refresh)
func (s *Service) generateToken(username string, role Role) (string, time.Time) {
	expiresAt := time.Now().Add(s.jwtExpiry)
	claims := &Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ocpp-emu",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(s.jwtSecret)
	return tokenString, expiresAt
}

// GeneratePasswordHash generates bcrypt hash for a password
func GeneratePasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// HashAPIKey generates SHA-256 hash for an API key
func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return "sha256:" + hex.EncodeToString(hash[:])
}

// GenerateAPIKeyHash generates a SHA-256 hash prefixed with "sha256:"
// This is used for storing in config
func GenerateAPIKeyHash(apiKey string) string {
	return HashAPIKey(apiKey)
}

// ParseAPIKeyHash extracts the hash from a "sha256:..." string
func ParseAPIKeyHash(hashStr string) (string, bool) {
	if !strings.HasPrefix(hashStr, "sha256:") {
		return "", false
	}
	return hashStr, true
}
