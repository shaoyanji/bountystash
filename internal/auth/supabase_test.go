package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("https://example.supabase.co/auth/v1/jwks")
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.jwksURL != "https://example.supabase.co/auth/v1/jwks" {
		t.Fatalf("jwksURL = %q, want %q", c.jwksURL, "https://example.supabase.co/auth/v1/jwks")
	}
}

func TestVerifyToken_EmptyToken(t *testing.T) {
	c := NewClient("")
	_, err := c.VerifyToken(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestVerifyToken_InvalidFormat(t *testing.T) {
	c := NewClient("")
	_, err := c.VerifyToken(context.Background(), "not-a-jwt")
	if err == nil {
		t.Fatal("expected error for invalid token format")
	}
}

func TestVerifyToken_ValidToken(t *testing.T) {
	c := NewClient("")

	// Create a valid JWT structure: header.payload.signature
	// Payload is base64url encoded JSON
	payload := map[string]interface{}{
		"sub": "user123",
		"app_metadata": map[string]interface{}{
			"role": "reviewer",
		},
		"user_metadata": map[string]interface{}{
			"email": "test@example.com",
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." + payloadB64 + ".signature"

	user, err := c.VerifyToken(context.Background(), token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "user123" {
		t.Errorf("ID = %q, want %q", user.ID, "user123")
	}
	if user.Role != "reviewer" {
		t.Errorf("Role = %q, want %q", user.Role, "reviewer")
	}
	if user.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "test@example.com")
	}
}

func TestVerifyToken_DefaultRole(t *testing.T) {
	c := NewClient("")

	// Token without role in app_metadata
	payload := map[string]interface{}{
		"sub": "user456",
	}
	payloadBytes, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := "header." + payloadB64 + ".signature"

	user, err := c.VerifyToken(context.Background(), token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Role != "user" {
		t.Errorf("Role = %q, want %q", user.Role, "user")
	}
}

func TestIsReviewer(t *testing.T) {
	tests := []struct {
		name string
		user *User
		want bool
	}{
		{"nil user", nil, false},
		{"regular user", &User{Role: "user"}, false},
		{"reviewer", &User{Role: "reviewer"}, true},
		{"admin", &User{Role: "admin"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.user.IsReviewer()
			if got != tt.want {
				t.Errorf("IsReviewer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserContextKey(t *testing.T) {
	// Verify the context key is properly defined
	key := UserContextKey
	if string(key) != "user" {
		t.Errorf("UserContextKey = %q, want %q", string(key), "user")
	}
}
