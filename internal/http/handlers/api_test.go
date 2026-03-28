package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIHealthz(t *testing.T) {
	h := NewAPIHandler(nil, nil)
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
	h := NewAPIHandler(nil, nil)
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
	h := NewAPIHandler(&DraftHandler{}, nil)
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
	h := NewAPIHandler(&DraftHandler{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/work/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	h.HandleWorkShow(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
