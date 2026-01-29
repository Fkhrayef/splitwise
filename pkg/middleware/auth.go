package middleware

import (
	"context"
	"net/http"
	"strconv"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// UserIDKey is the context key for the authenticated user ID
	UserIDKey ContextKey = "user_id"
)

// TestUserMiddleware allows setting user ID via X-Test-User-ID header
// This makes it easy to test as different users
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
