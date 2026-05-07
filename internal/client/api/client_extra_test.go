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

func TestNew_ValidURL(t *testing.T) {
	client, err := New("https://example.com", 5*time.Second)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client == nil {
		t.Fatal("New() returned nil")
	}
	if client.baseURL != "https://example.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://example.com")
	}
}

func TestNew_EmptyURL(t *testing.T) {
	_, err := New("", 5*time.Second)
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestNew_InvalidURL(t *testing.T) {
	_, err := New("://invalid", 5*time.Second)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestNew_NoScheme(t *testing.T) {
	_, err := New("example.com", 5*time.Second)
	if err == nil {
		t.Fatal("expected error for URL without scheme")
	}
}

func TestNew_DefaultTimeout(t *testing.T) {
	client, err := New("https://example.com", 0)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client.http.Timeout != 8*time.Second {
		t.Errorf("timeout = %v, want %v", client.http.Timeout, 8*time.Second)
	}
}

func TestListExamples_Success(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				body, _ := json.Marshal(map[string]interface{}{
					"examples": []map[string]interface{}{
						{"slug": "test", "title": "Test", "kind": "bounty", "visibility": "public"},
					},
				})
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(string(body))),
					Request:    req,
				}, nil
			}),
		},
	}

	examples, err := client.ListExamples(context.Background())
	if err != nil {
		t.Fatalf("ListExamples() error = %v", err)
	}
	if len(examples) != 1 {
		t.Fatalf("len(examples) = %d, want 1", len(examples))
	}
	if examples[0].Slug != "test" {
		t.Errorf("slug = %q, want %q", examples[0].Slug, "test")
	}
}

func TestGetExample_Success(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				body, _ := json.Marshal(map[string]interface{}{
					"slug":  "test",
					"title": "Test Example",
				})
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(string(body))),
					Request:    req,
				}, nil
			}),
		},
	}

	example, err := client.GetExample(context.Background(), "test")
	if err != nil {
		t.Fatalf("GetExample() error = %v", err)
	}
	if example.Slug != "test" {
		t.Errorf("slug = %q, want %q", example.Slug, "test")
	}
}

func TestListReview_Success(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				body, _ := json.Marshal(map[string]interface{}{
					"standard": []map[string]interface{}{},
					"private":  []map[string]interface{}{},
				})
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(string(body))),
					Request:    req,
				}, nil
			}),
		},
	}

	queue, err := client.ListReview(context.Background())
	if err != nil {
		t.Fatalf("ListReview() error = %v", err)
	}
	if queue.Standard == nil || queue.Private == nil {
		t.Fatal("queue should have non-nil slices")
	}
}

func TestListWork_Success(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				body, _ := json.Marshal(map[string]interface{}{
					"items": []map[string]interface{}{
						{"id": "123", "title": "Test Work"},
					},
				})
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(string(body))),
					Request:    req,
				}, nil
			}),
		},
	}

	items, err := client.ListWork(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListWork() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
}

func TestListWork_DefaultLimit(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if !strings.Contains(req.URL.String(), "limit=25") {
					t.Errorf("expected default limit=25 in URL, got %s", req.URL.String())
				}
				body, _ := json.Marshal(map[string]interface{}{
					"items": []map[string]interface{}{},
				})
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(string(body))),
					Request:    req,
				}, nil
			}),
		},
	}

	_, err := client.ListWork(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListWork() error = %v", err)
	}
}

func TestGetWork_Success(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				body, _ := json.Marshal(map[string]interface{}{
					"id":    "123",
					"title": "Test Work",
				})
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(string(body))),
					Request:    req,
				}, nil
			}),
		},
	}

	work, err := client.GetWork(context.Background(), "123")
	if err != nil {
		t.Fatalf("GetWork() error = %v", err)
	}
	if work.ID != "123" {
		t.Errorf("id = %q, want %q", work.ID, "123")
	}
}

func TestCreateDraft_Success(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				body, _ := json.Marshal(map[string]interface{}{
					"id":  "new-123",
					"url": "/work/new-123",
				})
				return &http.Response{
					StatusCode: http.StatusCreated,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(string(body))),
					Request:    req,
				}, nil
			}),
		},
	}

	result, err := client.CreateDraft(context.Background(), packets.DraftInput{
		Title: "Test Draft",
		Kind:  "bounty",
	})
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	if result.ID != "new-123" {
		t.Errorf("id = %q, want %q", result.ID, "new-123")
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		apiError *APIError
		want     string
	}{
		{
			name:     "with validation errors",
			apiError: &APIError{StatusCode: 400, Message: "validation failed", ValidationErrors: packets.ValidationErrors{"title": "required"}},
			want:     "validation failed (status 400)",
		},
		{
			name:     "without validation errors",
			apiError: &APIError{StatusCode: 500, Message: "internal error"},
			want:     "internal error (status 500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.apiError.Error()
			if !strings.Contains(got, tt.want) {
				t.Errorf("Error() = %q, want contains %q", got, tt.want)
			}
		})
	}
}

func TestDecodeResponse_NonJSONResponse(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"text/plain"}},
					Body:       io.NopCloser(strings.NewReader("not json")),
					Request:    req,
				}, nil
			}),
		},
	}

	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("expected error for non-JSON response")
	}
}

func TestGet_RequestError(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("connection refused")
			}),
		},
	}

	_, err := client.ListExamples(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPost_RequestError(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("connection refused")
			}),
		},
	}

	_, err := client.CreateDraft(context.Background(), packets.DraftInput{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// Cyclomatic complexity: test decodeResponse branches

func TestGetExample_ErrorResponse(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"error":"not found"}`)),
					Request:    req,
				}, nil
			}),
		},
	}

	_, err := client.GetExample(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestListReview_ErrorResponse(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"error":"internal"}`)),
					Request:    req,
				}, nil
			}),
		},
	}

	_, err := client.ListReview(context.Background())
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestGetWork_ErrorResponse(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Header:     http.Header{"Content-Type": []string{"text/plain"}},
					Body:       io.NopCloser(strings.NewReader("not found")),
					Request:    req,
				}, nil
			}),
		},
	}

	_, err := client.GetWork(context.Background(), "nope")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestListWork_ErrorResponse(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"error":"bad gateway"}`)),
					Request:    req,
				}, nil
			}),
		},
	}

	_, err := client.ListWork(context.Background(), 5)
	if err == nil {
		t.Fatal("expected error for 502")
	}
}

// decodeResponse: plain text error body (no JSON "error" field)

func TestDecodeResponse_PlainTextErrorBody(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Header:     http.Header{"Content-Type": []string{"text/plain"}},
					Body:       io.NopCloser(strings.NewReader("the thing you asked for does not exist")),
					Request:    req,
				}, nil
			}),
		},
	}

	_, err := client.GetExample(context.Background(), "gone")
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Message != "the thing you asked for does not exist" {
		t.Errorf("Message = %q, want body text", apiErr.Message)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusNotFound)
	}
}

// decodeResponse: completely empty error body

func TestDecodeResponse_EmptyErrorBody(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Header:     http.Header{"Content-Type": []string{"text/plain"}},
					Body:       io.NopCloser(strings.NewReader("")),
					Request:    req,
				}, nil
			}),
		},
	}

	_, err := client.GetWork(context.Background(), "empty")
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Message != "request failed" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "request failed")
	}
}

// decodeResponse: error response with JSON but no "error" key

func TestDecodeResponse_JSONWithoutErrorKey(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"details":"something went sideways"}`)),
					Request:    req,
				}, nil
			}),
		},
	}

	_, err := client.ListReview(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Message != `{"details":"something went sideways"}` {
		t.Errorf("Message = %q, want raw JSON body", apiErr.Message)
	}
}

// HTTP 200 with decodeResponse where out is nil (should not error)

func TestDecodeResponse_NilOutOnSuccess(t *testing.T) {
	client := &HTTPClient{
		baseURL: "http://example.test",
		http: &http.Client{
			Timeout: 2 * time.Second,
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
					Request:    req,
				}, nil
			}),
		},
	}
	// We can test this via the internal get method by calling Health
	// which provides an out pointer. But to test nil out, we call get directly.
	err := client.get(context.Background(), "/api/healthz", nil)
	if err != nil {
		t.Fatalf("get with nil out should not error: %v", err)
	}
}

// New: trailing slash removal

func TestNew_StripTrailingSlash(t *testing.T) {
	client, err := New("https://example.com/", 5*time.Second)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client.baseURL != "https://example.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://example.com")
	}
}

// New: error for URL with missing host

func TestNew_MissingHost(t *testing.T) {
	_, err := New("https://", 5*time.Second)
	if err == nil {
		t.Fatal("expected error for URL without host")
	}
}

// Cyclomatic complexity: APIError with validation errors vs without

func TestAPIError_WithValidationErrors(t *testing.T) {
	err := &APIError{
		StatusCode: 422,
		Message:    "unprocessable",
		ValidationErrors: packets.ValidationErrors{
			"title": "Title is required",
			"kind":  "Kind must be one of: bounty, rfq, rfp, private_security",
		},
	}
	msg := err.Error()
	if msg != "unprocessable (status 422)" {
		t.Errorf("Error() = %q, want %q", msg, "unprocessable (status 422)")
	}
}

func TestAPIError_WithoutValidationErrors(t *testing.T) {
	err := &APIError{
		StatusCode: 500,
		Message:    "boom",
	}
	msg := err.Error()
	if msg != "boom (status 500)" {
		t.Errorf("Error() = %q, want %q", msg, "boom (status 500)")
	}
}
