// Package auth provides Supabase JWT authentication utilities.
package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// User represents an authenticated user with their role.
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Role     string `json:"role"` // "reviewer" or "user"
	RawToken string `json:"-"`
}

// Client handles Supabase JWT verification.
// For production use, integrate with Supabase JWKS for proper signature verification.
type Client struct {
	jwksURL string
}

// NewClient creates a new Supabase auth client.
// The jwksURL is typically "https://<project-ref>.supabase.co/auth/v1/jwks".
func NewClient(jwksURL string) *Client {
	return &Client{
		jwksURL: jwksURL,
	}
}

// VerifyToken parses and validates a JWT token, returning the user info.
// Note: This implementation decodes the token payload but does not verify the signature.
// For production, use a proper JWT library with JWKS verification.
func (c *Client) VerifyToken(ctx context.Context, tokenString string) (*User, error) {
	if tokenString == "" {
		return nil, errors.New("missing token")
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	var rawClaims map[string]interface{}
	if err := json.Unmarshal(payload, &rawClaims); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	user := &User{RawToken: tokenString}
	if sub, ok := rawClaims["sub"].(string); ok {
		user.ID = sub
	}
	if appMeta, ok := rawClaims["app_metadata"].(map[string]interface{}); ok {
		if role, ok := appMeta["role"].(string); ok {
			user.Role = role
		}
	}
	if userMeta, ok := rawClaims["user_metadata"].(map[string]interface{}); ok {
		if email, ok := userMeta["email"].(string); ok {
			user.Email = email
		}
	}

	// Default role if not set
	if user.Role == "" {
		user.Role = "user"
	}

	return user, nil
}

// IsReviewer returns true if the user has the reviewer role.
func (u *User) IsReviewer() bool {
	return u != nil && u.Role == "reviewer"
}

// ContextKey is the type for context keys.
type ContextKey string

const (
	// UserContextKey is the key used to store user info in request context.
	UserContextKey ContextKey = "user"
)
