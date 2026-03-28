package api

import (
	"context"
	"encoding/json"
	"errors"
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

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err type = %T, want wrapped *APIError", err)
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

func TestHealthInvalidJSONResponse(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				body := `{"status":"ok"`
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(body)),
					Request:    req,
				}, nil
			}),
		},
	}

	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid JSON response") {
		t.Fatalf("expected invalid JSON response error, got %q", err.Error())
	}
}

func TestHealthRequestErrorWrapped(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("dial failure")
			}),
		},
	}

	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "request GET /api/healthz") {
		t.Fatalf("expected wrapped request context, got %q", err.Error())
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
