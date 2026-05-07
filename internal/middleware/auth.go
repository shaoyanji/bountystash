// Package middleware provides HTTP middleware components.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/shaoyanji/bountystash/internal/auth"
)

// AuthMiddleware creates a middleware that verifies Supabase JWT tokens
// and injects user information into the request context.
type AuthMiddleware struct {
	authClient *auth.Client
}

// NewAuthMiddleware creates a new authentication middleware.
func NewAuthMiddleware(client *auth.Client) *AuthMiddleware {
	return &AuthMiddleware{
		authClient: client,
	}
}

// Handler wraps an http.Handler with authentication logic.
func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := m.extractUser(r)
		
		// Always add user to context (may be nil for unauthenticated requests)
		ctx := context.WithValue(r.Context(), auth.UserContextKey, user)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// RequireAuth wraps a handler to require authentication.
// Returns 401 if no valid token is present.
func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// RequireReviewer wraps a handler to require reviewer role.
// Returns 403 if user is not a reviewer.
func (m *AuthMiddleware) RequireReviewer(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if !user.IsReviewer() {
			http.Error(w, "Forbidden: reviewer access required", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// extractUser extracts and validates the user from the Authorization header.
func (m *AuthMiddleware) extractUser(r *http.Request) *auth.User {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil
	}

	token := parts[1]
	user, err := m.authClient.VerifyToken(r.Context(), token)
	if err != nil {
		return nil
	}

	return user
}

// GetUser retrieves the authenticated user from the request context.
func GetUser(r *http.Request) *auth.User {
	user, ok := r.Context().Value(auth.UserContextKey).(*auth.User)
	if !ok {
		return nil
	}
	return user
}

// IsReviewer checks if the current user has reviewer role.
func IsReviewer(r *http.Request) bool {
	user := GetUser(r)
	return user != nil && user.IsReviewer()
}
