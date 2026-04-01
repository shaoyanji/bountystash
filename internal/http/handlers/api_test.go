package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/shaoyanji/bountystash/internal/service"
)

func TestAPIHealthz(t *testing.T) {
	h := NewAPIHandler(stubService{})
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	rec := httptest.NewRecorder()

	h.HandleHealthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("status = %q, want ok", payload["status"])
	}
}

func TestAPIExamplesStableShape(t *testing.T) {
	h := NewAPIHandler(stubService{})
	req := httptest.NewRequest(http.MethodGet, "/api/examples", nil)
	rec := httptest.NewRecorder()

	h.HandleExamplesList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload struct {
		Examples []struct {
			Slug       string `json:"slug"`
			Title      string `json:"title"`
			Kind       string `json:"kind"`
			Visibility string `json:"visibility"`
		} `json:"examples"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Examples) != 3 {
		t.Fatalf("examples len = %d, want 3", len(payload.Examples))
	}
	if payload.Examples[0].Slug == "" || payload.Examples[0].Title == "" {
		t.Fatalf("first example missing required fields: %+v", payload.Examples[0])
	}
}

func TestAPIDraftCreateValidationErrors(t *testing.T) {
	h := NewAPIHandler(validationStubService())
	reqBody := `{"title":"","kind":"bad_kind","visibility":"wat"}`
	req := httptest.NewRequest(http.MethodPost, "/api/draft", bytes.NewBufferString(reqBody))
	rec := httptest.NewRecorder()

	h.HandleDraftCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var payload struct {
		Error            string            `json:"error"`
		ValidationErrors map[string]string `json:"validation_errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error != "validation failed" {
		t.Fatalf("error = %q, want validation failed", payload.Error)
	}
	if payload.ValidationErrors["title"] == "" || payload.ValidationErrors["kind"] == "" || payload.ValidationErrors["visibility"] == "" {
		t.Fatalf("missing expected validation fields: %+v", payload.ValidationErrors)
	}
}

func TestAPIWorkShowRejectsInvalidID(t *testing.T) {
	h := NewAPIHandler(stubService{})
	req := httptest.NewRequest(http.MethodGet, "/api/work/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	h.HandleWorkShow(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAPIWorkHistoryReturnsEvents(t *testing.T) {
	const workID = "00000000-0000-1000-8000-000000000001"
	svc := stubService{
		get: func(ctx context.Context, id string) (service.WorkDetail, error) {
			return service.WorkDetail{ID: workID}, nil
		},
		history: func(ctx context.Context, id string) ([]service.Event, error) {
			return []service.Event{
				{
					ID:         "evt1",
					EventType:  "intake_received",
					WorkItemID: workID,
					CreatedAt:  time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC),
					Payload:    json.RawMessage(`{"title":"history test"}`),
				},
			}, nil
		},
	}
	h := NewAPIHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/work/"+workID+"/history", nil)
	req = withAPIParam(req, "id", workID)
	rec := httptest.NewRecorder()

	h.HandleWorkHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload historyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.WorkItemID != workID || len(payload.Events) != 1 {
		t.Fatalf("unexpected history payload: %+v", payload)
	}
}

func withAPIParam(req *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestAPIRecentEventsReturnsEvents(t *testing.T) {
	svc := stubService{
		recentEvents: func(ctx context.Context, limit int) ([]service.Event, error) {
			return []service.Event{
				{
					ID:         "evt-recent-1",
					EventType:  "work_version_persisted",
					WorkItemID: "work-1",
					CreatedAt:  time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC),
					Payload:    json.RawMessage(`{"version_number":1}`),
				},
				{
					ID:         "evt-recent-2",
					EventType:  "intake_received",
					WorkItemID: "work-2",
					CreatedAt:  time.Date(2026, 3, 30, 11, 0, 0, 0, time.UTC),
					Payload:    json.RawMessage(`{"title":"test"}`),
				},
			}, nil
		},
	}
	h := NewAPIHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/events/recent", nil)
	rec := httptest.NewRecorder()

	h.HandleRecentEvents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload recentEventsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Events) != 2 {
		t.Fatalf("events len = %d, want 2", len(payload.Events))
	}
	if payload.Limit != 2 {
		t.Fatalf("limit = %d, want 2", payload.Limit)
	}
}

func TestAPIRecentEventsEmpty(t *testing.T) {
	svc := stubService{
		recentEvents: func(ctx context.Context, limit int) ([]service.Event, error) {
			return []service.Event{}, nil
		},
	}
	h := NewAPIHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/events/recent", nil)
	rec := httptest.NewRecorder()

	h.HandleRecentEvents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload recentEventsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Events) != 0 {
		t.Fatalf("events len = %d, want 0", len(payload.Events))
	}
}

func TestAPIRecentEventsWithLimit(t *testing.T) {
	svc := stubService{
		recentEvents: func(ctx context.Context, limit int) ([]service.Event, error) {
			return []service.Event{}, nil
		},
	}
	h := NewAPIHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/events/recent?limit=10", nil)
	rec := httptest.NewRecorder()

	h.HandleRecentEvents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAPIRecentEventsInvalidLimit(t *testing.T) {
	svc := stubService{}
	h := NewAPIHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/events/recent?limit=invalid", nil)
	rec := httptest.NewRecorder()

	h.HandleRecentEvents(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
