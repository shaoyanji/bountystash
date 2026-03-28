package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shaoyanji/bountystash/internal/http/handlers"
	"github.com/shaoyanji/bountystash/internal/packets"
)

type Client interface {
	Health(ctx context.Context) (string, error)
	ListExamples(ctx context.Context) ([]ExampleSummary, error)
	GetExample(ctx context.Context, slug string) (handlers.Example, error)
	ListReview(ctx context.Context) (ReviewQueue, error)
	ListWork(ctx context.Context, limit int) ([]handlers.WorkListRow, error)
	GetWork(ctx context.Context, id string) (handlers.WorkDetail, error)
	CreateDraft(ctx context.Context, input packets.DraftInput) (handlers.DraftCreateResult, error)
}

type HTTPClient struct {
	baseURL string
	http    *http.Client
}

type APIError struct {
	StatusCode       int
	Message          string
	ValidationErrors packets.ValidationErrors
}

func (e *APIError) Error() string {
	if len(e.ValidationErrors) > 0 {
		return fmt.Sprintf("%s (status %d)", e.Message, e.StatusCode)
	}
	return fmt.Sprintf("%s (status %d)", e.Message, e.StatusCode)
}

type ExampleSummary struct {
	Slug       string             `json:"slug"`
	Title      string             `json:"title"`
	Kind       packets.Kind       `json:"kind"`
	Visibility packets.Visibility `json:"visibility"`
}

type ReviewQueue struct {
	Standard []handlers.ReviewRow `json:"standard"`
	Private  []handlers.ReviewRow `json:"private"`
}

type draftCreateRequest struct {
	Title              string `json:"title"`
	Kind               string `json:"kind"`
	Scope              string `json:"scope"`
	Deliverables       string `json:"deliverables"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	RewardModel        string `json:"reward_model"`
	Visibility         string `json:"visibility"`
}

func New(baseURL string, timeout time.Duration) (*HTTPClient, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("base URL must include scheme and host")
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}

	return &HTTPClient{
		baseURL: parsed.String(),
		http: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *HTTPClient) Health(ctx context.Context) (string, error) {
	var out struct {
		Status string `json:"status"`
	}
	if err := c.get(ctx, "/api/healthz", &out); err != nil {
		return "", err
	}
	return out.Status, nil
}

func (c *HTTPClient) ListExamples(ctx context.Context) ([]ExampleSummary, error) {
	var out struct {
		Examples []ExampleSummary `json:"examples"`
	}
	if err := c.get(ctx, "/api/examples", &out); err != nil {
		return nil, err
	}
	return out.Examples, nil
}

func (c *HTTPClient) GetExample(ctx context.Context, slug string) (handlers.Example, error) {
	var out handlers.Example
	if err := c.get(ctx, "/api/examples/"+slug, &out); err != nil {
		return handlers.Example{}, err
	}
	return out, nil
}

func (c *HTTPClient) ListReview(ctx context.Context) (ReviewQueue, error) {
	var out ReviewQueue
	if err := c.get(ctx, "/api/review", &out); err != nil {
		return ReviewQueue{}, err
	}
	return out, nil
}

func (c *HTTPClient) ListWork(ctx context.Context, limit int) ([]handlers.WorkListRow, error) {
	if limit <= 0 {
		limit = 25
	}

	var out struct {
		Items []handlers.WorkListRow `json:"items"`
	}
	if err := c.get(ctx, fmt.Sprintf("/api/work?limit=%d", limit), &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *HTTPClient) GetWork(ctx context.Context, id string) (handlers.WorkDetail, error) {
	var out handlers.WorkDetail
	if err := c.get(ctx, "/api/work/"+id, &out); err != nil {
		return handlers.WorkDetail{}, err
	}
	return out, nil
}

func (c *HTTPClient) CreateDraft(ctx context.Context, input packets.DraftInput) (handlers.DraftCreateResult, error) {
	var out handlers.DraftCreateResult
	req := draftCreateRequest{
		Title:              input.Title,
		Kind:               input.Kind,
		Scope:              input.Scope,
		Deliverables:       input.Deliverables,
		AcceptanceCriteria: input.AcceptanceCriteria,
		RewardModel:        input.RewardModel,
		Visibility:         input.Visibility,
	}

	if err := c.post(ctx, "/api/draft", req, &out); err != nil {
		return handlers.DraftCreateResult{}, err
	}
	return out, nil
}

func (c *HTTPClient) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request %s: %w", path, err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if err := decodeResponse(resp, out); err != nil {
		return fmt.Errorf("decode GET %s: %w", path, err)
	}
	return nil
}

func (c *HTTPClient) post(ctx context.Context, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode request body %s: %w", path, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request POST %s: %w", path, err)
	}
	defer resp.Body.Close()
	if err := decodeResponse(resp, out); err != nil {
		return fmt.Errorf("decode POST %s: %w", path, err)
	}
	return nil
}

func decodeResponse(resp *http.Response, out any) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var payload struct {
			Error            string                   `json:"error"`
			ValidationErrors packets.ValidationErrors `json:"validation_errors"`
		}
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &payload)
		msg := strings.TrimSpace(payload.Error)
		if msg == "" {
			msg = strings.TrimSpace(string(body))
		}
		if msg == "" {
			msg = "request failed"
		}
		return &APIError{
			StatusCode:       resp.StatusCode,
			Message:          msg,
			ValidationErrors: payload.ValidationErrors,
		}
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("invalid JSON response: %w", err)
	}
	return nil
}
