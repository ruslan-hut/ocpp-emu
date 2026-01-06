package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Handler handles auth-related HTTP endpoints
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler creates a new auth handler
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// HandleLogin handles POST /api/auth/login
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		sendError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	token, expiresAt, err := h.service.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		switch err {
		case ErrInvalidCredentials:
			sendError(w, http.StatusUnauthorized, "invalid username or password")
		case ErrUserDisabled:
			sendError(w, http.StatusForbidden, "user account is disabled")
		default:
			h.logger.Error("login error", "error", err)
			sendError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	// Get user to include role in response
	user, _ := h.service.GetUser(req.Username)

	resp := LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		User: UserResponse{
			Username: req.Username,
			Role:     string(user.Role),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleLogout handles POST /api/auth/logout
// This is mostly for logging purposes as JWT tokens are stateless
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user := GetUserFromContext(r.Context())
	if user != nil {
		h.logger.Info("user logged out", "username", user.Username)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleMe handles GET /api/auth/me - returns current user info
func (h *Handler) HandleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user := GetUserFromContext(r.Context())
	if user == nil {
		sendError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	resp := UserResponse{
		Username: user.Username,
		Role:     string(user.Role),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleRefresh handles POST /api/auth/refresh - refresh JWT token
func (h *Handler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user := GetUserFromContext(r.Context())
	if user == nil {
		sendError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	// Only JWT auth can be refreshed
	if user.AuthType != "jwt" {
		sendError(w, http.StatusBadRequest, "only JWT tokens can be refreshed")
		return
	}

	// Verify user still exists and is enabled
	configUser, ok := h.service.GetUser(user.Username)
	if !ok || !configUser.Enabled {
		sendError(w, http.StatusForbidden, "user account is disabled")
		return
	}

	// Generate new token directly
	token, expiresAt := h.service.generateToken(user.Username, configUser.Role)

	resp := LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		User: UserResponse{
			Username: user.Username,
			Role:     string(configUser.Role),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
