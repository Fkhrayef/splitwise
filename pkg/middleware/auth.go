package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/fkhayef/splitwise/pkg/response"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// UserIDKey is the context key for the authenticated user ID
	UserIDKey ContextKey = "user_id"
)

// AuthMiddleware is a placeholder for JWT authentication
// TODO: Implement proper JWT validation
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Unauthorized(w, "Authorization header required")
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(w, "Invalid authorization header format")
			return
		}

		token := parts[1]

		// TODO: Validate JWT token and extract user ID
		// For now, we'll use a placeholder user ID
		// In production, decode the JWT and extract the user_id claim
		userID := validateToken(token)
		if userID == 0 {
			response.Unauthorized(w, "Invalid or expired token")
			return
		}

		// Add user ID to context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validateToken is a placeholder for JWT validation
// TODO: Implement proper JWT validation
func validateToken(token string) int64 {
	// Placeholder: In production, decode and validate the JWT
	// Return 0 if invalid, otherwise return the user ID
	if token == "" {
		return 0
	}
	// For development, accept any non-empty token and return a test user ID
	return 1
}

// TestUserMiddleware allows setting user ID via X-Test-User-ID header (DEV ONLY)
// This makes it easy to test as different users without real auth
func TestUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDStr := r.Header.Get("X-Test-User-ID")
		if userIDStr != "" {
			if userID, err := strconv.ParseInt(userIDStr, 10, 64); err == nil && userID > 0 {
				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		// Default to user 1 if no header provided
		ctx := context.WithValue(r.Context(), UserIDKey, int64(1))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts the user ID from the request context
func GetUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(UserIDKey).(int64)
	return userID, ok
}
