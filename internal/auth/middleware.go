package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// Middleware creates authentication middleware that validates JWT or API key
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If auth is disabled, pass through
		if !s.enabled {
			next.ServeHTTP(w, r)
			return
		}

		var user *AuthenticatedUser

		// Try JWT token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := s.ValidateJWT(tokenString)
			if err != nil {
				sendError(w, http.StatusUnauthorized, err.Error())
				return
			}
			user = &AuthenticatedUser{
				Username: claims.Username,
				Role:     claims.Role,
				AuthType: "jwt",
			}
		}

		// Try API key from X-API-Key header
		if user == nil {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey != "" {
				key, err := s.ValidateAPIKey(apiKey)
				if err != nil {
					sendError(w, http.StatusUnauthorized, err.Error())
					return
				}
				user = &AuthenticatedUser{
					Username: key.Name,
					Role:     key.Role,
					AuthType: "apikey",
				}
			}
		}

		// No auth provided
		if user == nil {
			sendError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		// Store user in context and continue
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalMiddleware creates middleware that validates auth if provided, but doesn't require it
func (s *Service) OptionalMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If auth is disabled, pass through
		if !s.enabled {
			next.ServeHTTP(w, r)
			return
		}

		var user *AuthenticatedUser

		// Try JWT token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := s.ValidateJWT(tokenString)
			if err == nil {
				user = &AuthenticatedUser{
					Username: claims.Username,
					Role:     claims.Role,
					AuthType: "jwt",
				}
			}
		}

		// Try API key from X-API-Key header
		if user == nil {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey != "" {
				key, err := s.ValidateAPIKey(apiKey)
				if err == nil {
					user = &AuthenticatedUser{
						Username: key.Name,
						Role:     key.Role,
						AuthType: "apikey",
					}
				}
			}
		}

		// Store user in context if found
		if user != nil {
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// RequireRole creates middleware that checks if user has required role
func RequireRole(roles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				sendError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			// Check if user has any of the required roles
			for _, role := range roles {
				if user.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			sendError(w, http.StatusForbidden, "insufficient permissions")
		})
	}
}

// RequireAdmin creates middleware that requires admin role
func RequireAdmin() func(http.Handler) http.Handler {
	return RequireRole(RoleAdmin)
}

// RequireAuth creates middleware that requires authentication (any role)
func RequireAuth() func(http.Handler) http.Handler {
	return RequireRole(RoleAdmin, RoleViewer)
}

// GetUserFromContext retrieves the authenticated user from context
func GetUserFromContext(ctx context.Context) *AuthenticatedUser {
	user, ok := ctx.Value(UserContextKey).(*AuthenticatedUser)
	if !ok {
		return nil
	}
	return user
}

// IsAuthenticated checks if the request has an authenticated user
func IsAuthenticated(ctx context.Context) bool {
	return GetUserFromContext(ctx) != nil
}

// IsAdmin checks if the authenticated user is an admin
func IsAdmin(ctx context.Context) bool {
	user := GetUserFromContext(ctx)
	return user != nil && user.Role == RoleAdmin
}

// sendError sends a JSON error response
func sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
