package middleware

import (
	"context"
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

// Cyclomatic complexity tests

func TestRequireAuth_Authenticated(t *testing.T) {
	c := auth.NewClient("")
	m := NewAuthMiddleware(c)

	called := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := m.RequireAuth(testHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Inject an authenticated user into context
	user := &auth.User{ID: "user-1", Role: "user"}
	ctx := context.WithValue(req.Context(), auth.UserContextKey, user)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !called {
		t.Error("handler was not called")
	}
}

func TestRequireReviewer_Unauthenticated(t *testing.T) {
	c := auth.NewClient("")
	m := NewAuthMiddleware(c)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := m.RequireReviewer(testHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireReviewer_NonReviewerUser(t *testing.T) {
	c := auth.NewClient("")
	m := NewAuthMiddleware(c)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := m.RequireReviewer(testHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Inject a non-reviewer user (role = "user")
	user := &auth.User{ID: "user-1", Role: "user"}
	ctx := context.WithValue(req.Context(), auth.UserContextKey, user)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRequireReviewer_ReviewerAccess(t *testing.T) {
	c := auth.NewClient("")
	m := NewAuthMiddleware(c)

	called := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := m.RequireReviewer(testHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Inject a reviewer user
	user := &auth.User{ID: "reviewer-1", Role: "reviewer"}
	ctx := context.WithValue(req.Context(), auth.UserContextKey, user)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !called {
		t.Error("handler was not called")
	}
}

func TestExtractUser_NonBearerAuthHeader(t *testing.T) {
	c := auth.NewClient("https://example.supabase.co/auth/v1/jwks")
	m := NewAuthMiddleware(c)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	user := m.extractUser(req)
	if user != nil {
		t.Error("expected nil user for non-Bearer auth")
	}
}

func TestExtractUser_MalformedAuthHeader(t *testing.T) {
	c := auth.NewClient("https://example.supabase.co/auth/v1/jwks")
	m := NewAuthMiddleware(c)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "justoneword")
	user := m.extractUser(req)
	if user != nil {
		t.Error("expected nil user for malformed auth header")
	}
}

func TestHandler_InjectsNilUserForNoAuth(t *testing.T) {
	c := auth.NewClient("https://example.supabase.co/auth/v1/jwks")
	m := NewAuthMiddleware(c)

	var capturedUser *auth.User
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = GetUser(r)
		w.WriteHeader(http.StatusOK)
	})

	handler := m.Handler(testHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if capturedUser != nil {
		t.Error("expected nil user when no auth header present")
	}
}

func TestIsReviewer_NonReviewerUser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	user := &auth.User{ID: "user-1", Role: "user"}
	ctx := context.WithValue(req.Context(), auth.UserContextKey, user)
	req = req.WithContext(ctx)

	if IsReviewer(req) {
		t.Error("expected false for user with role=user")
	}
}

func TestIsReviewer_ReviewerUser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	user := &auth.User{ID: "reviewer-1", Role: "reviewer"}
	ctx := context.WithValue(req.Context(), auth.UserContextKey, user)
	req = req.WithContext(ctx)

	if !IsReviewer(req) {
		t.Error("expected true for user with role=reviewer")
	}
}

func TestGetUser_WithUserInContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	user := &auth.User{ID: "user-1", Role: "user"}
	ctx := context.WithValue(req.Context(), auth.UserContextKey, user)
	req = req.WithContext(ctx)

	got := GetUser(req)
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.ID != "user-1" {
		t.Errorf("ID = %q, want %q", got.ID, "user-1")
	}
}

func TestGetUser_WrongContextType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Put a string in the context instead of *auth.User
	ctx := context.WithValue(req.Context(), auth.UserContextKey, "not-a-user")
	req = req.WithContext(ctx)

	got := GetUser(req)
	if got != nil {
		t.Error("expected nil when context value is wrong type")
	}
}
