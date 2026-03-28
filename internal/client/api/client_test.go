package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/shaoyanji/bountystash/internal/packets"
)

func TestCreateDraftValidationErrorDecoding(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != "/api/draft" {
					t.Fatalf("path = %s, want /api/draft", req.URL.Path)
				}
				body := `{"error":"validation failed","validation_errors":{"title":"Title is required."}}`
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(body)),
					Request:    req,
				}, nil
			}),
		},
	}

	_, err := client.CreateDraft(context.Background(), packets.DraftInput{
		Kind:       "bounty",
		Visibility: "draft",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", apiErr.StatusCode, http.StatusBadRequest)
	}
	if apiErr.ValidationErrors["title"] == "" {
		t.Fatalf("missing decoded validation errors: %+v", apiErr.ValidationErrors)
	}
}

func TestHealthSuccess(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != "/api/healthz" {
					t.Fatalf("path = %s, want /api/healthz", req.URL.Path)
				}
				body, _ := json.Marshal(map[string]string{"status": "ok"})
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(string(body))),
					Request:    req,
				}, nil
			}),
		},
	}

	status, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("health error: %v", err)
	}
	if status != "ok" {
		t.Fatalf("status = %q, want ok", status)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
