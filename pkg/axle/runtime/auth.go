package runtime

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const UserIDKey contextKey = "axle_user_id"

// AuthMiddleware extracts user identity from the request and injects it into context.
// Supports Bearer token (JWT parsing stub) and X-User-ID header for development.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := extractUserID(r)
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractUserID(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		// Stub: in production, parse JWT and extract sub claim
		if token == "test-token" {
			return "test-user"
		}
		return "anon"
	}

	if uid := r.Header.Get("X-User-ID"); uid != "" {
		return uid
	}

	return "anon"
}

// GetUserID extracts the user ID from request context.
func GetUserID(ctx context.Context) string {
	if uid, ok := ctx.Value(UserIDKey).(string); ok {
		return uid
	}
	return "anon"
}
