package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shaoyanji/bountystash/internal/auth"
)

func TestNewAuthMiddleware(t *testing.T) {
	c := auth.NewClient("https://example.supabase.co/auth/v1/jwks")
	m := NewAuthMiddleware(c)
	if m == nil {
		t.Fatal("NewAuthMiddleware returned nil")
	}
	if m.authClient == nil {
		t.Error("authClient is nil")
	}
}

func TestHandler_Unauthenticated(t *testing.T) {
	c := auth.NewClient("")
	m := NewAuthMiddleware(c)

	// Create a test handler that checks if user is nil
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = GetUser(r) // Check but don't store
		w.WriteHeader(http.StatusOK)
	})

	handler := m.Handler(testHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandler_WithToken(t *testing.T) {
	c := auth.NewClient("")
	m := NewAuthMiddleware(c)

	// Create a token with proper structure
	// This is a simplified test - the middleware will try to verify the token
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = GetUser(r) // Check user but don't store
		w.WriteHeader(http.StatusOK)
	})

	handler := m.Handler(testHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// The user might be nil since the token is invalid
	// But the handler should still call the next handler
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRequireAuth_Unauthenticated(t *testing.T) {
	c := auth.NewClient("")
	m := NewAuthMiddleware(c)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := m.RequireAuth(testHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireReviewer_NonReviewer(t *testing.T) {
	c := auth.NewClient("")
	m := NewAuthMiddleware(c)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := m.RequireReviewer(testHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No user in context, should return 401
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestGetUser(t *testing.T) {
	// Test with no user in context
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	user := GetUser(req)
	if user != nil {
		t.Error("expected nil user when no user in context")
	}
}

func TestIsReviewer(t *testing.T) {
	// Test with no user in context
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if IsReviewer(req) {
		t.Error("expected false when no user in context")
	}
}
