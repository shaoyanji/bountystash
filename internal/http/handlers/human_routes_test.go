package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/shaoyanji/bountystash/internal/packets"
)

func TestHandleHomeBrowserGetsHTML(t *testing.T) {
	h, err := NewDraftHandler(stubService{})
	if err != nil {
		t.Fatalf("new draft handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	rec := httptest.NewRecorder()

	h.HandleHome(rec, req)

	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want html", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<form method=\"post\" action=\"/draft\"") {
		t.Fatalf("expected HTML intake form, got body: %s", body)
	}
}

func TestHandleHomeNonBrowserDefaultsToMarkdown(t *testing.T) {
	h, err := NewDraftHandler(stubService{})
	if err != nil {
		t.Fatalf("new draft handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "curl/8.0.1")
	rec := httptest.NewRecorder()

	h.HandleHome(rec, req)

	if got := rec.Header().Get("Content-Type"); got != "text/markdown; charset=utf-8" {
		t.Fatalf("content-type = %q, want markdown", got)
	}
	body := rec.Body.String()
	for _, needle := range []string{"# Bountystash", "## Useful routes", "## Useful API endpoints", "## Format overrides", "Discovery: /.well-known/bountystash-manifest"} {
		if !strings.Contains(body, needle) {
			t.Fatalf("body missing %q:\n%s", needle, body)
		}
	}
}

func TestHandleHomeTextOverride(t *testing.T) {
	h, err := NewDraftHandler(stubService{})
	if err != nil {
		t.Fatalf("new draft handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?format=text", nil)
	rec := httptest.NewRecorder()

	h.HandleHome(rec, req)

	if got := rec.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("content-type = %q, want plain text", got)
	}
	body := rec.Body.String()
	if strings.Contains(body, "# Bountystash") {
		t.Fatalf("plain text body unexpectedly used markdown heading:\n%s", body)
	}
	if !strings.Contains(body, "Bountystash\n===========") {
		t.Fatalf("plain text body missing title underline:\n%s", body)
	}
}

func TestHandleExampleShowNonBrowserDefaultsToMarkdown(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/examples/auth-loop", nil)
	req.Header.Set("User-Agent", "curl/8.0.1")
	req = withRouteParam(req, "slug", "auth-loop")
	rec := httptest.NewRecorder()

	HandleExampleShow(rec, req)

	if got := rec.Header().Get("Content-Type"); got != "text/markdown; charset=utf-8" {
		t.Fatalf("content-type = %q, want markdown", got)
	}
	body := rec.Body.String()
	for _, needle := range []string{"# Fix recurring auth session loop in dashboard", "## Example", "slug: auth-loop", "## Packet fields", "## Next step"} {
		if !strings.Contains(body, needle) {
			t.Fatalf("body missing %q:\n%s", needle, body)
		}
	}
}

func TestHandleManifestDefaultsToMarkdown(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/.well-known/bountystash-manifest", nil)
	rec := httptest.NewRecorder()

	HandleManifest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/markdown; charset=utf-8" {
		t.Fatalf("content-type = %q, want markdown", got)
	}
	body := rec.Body.String()
	for _, needle := range []string{"# Bountystash Manifest", "## Useful human routes", "## Useful API routes", "## Format overrides", "## Agent notes", "Prefer this manifest over broad scraping."} {
		if !strings.Contains(body, needle) {
			t.Fatalf("body missing %q:\n%s", needle, body)
		}
	}
}

func TestHandleManifestTextOverride(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/.well-known/bountystash-manifest?format=text", nil)
	rec := httptest.NewRecorder()

	HandleManifest(rec, req)

	if got := rec.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("content-type = %q, want plain text", got)
	}
	body := rec.Body.String()
	if strings.Contains(body, "# Bountystash Manifest") {
		t.Fatalf("plain text manifest unexpectedly used markdown heading:\n%s", body)
	}
	if !strings.Contains(body, "Bountystash Manifest\n====================") {
		t.Fatalf("plain text manifest missing title underline:\n%s", body)
	}
}

func TestHandleManifestDoesNotRequireDBState(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/.well-known/bountystash-manifest?format=md", nil)
	rec := httptest.NewRecorder()

	HandleManifest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "/api/draft") {
		t.Fatalf("manifest missing expected static API route:\n%s", rec.Body.String())
	}
}

func TestHandleExampleShowNonBrowserNotFoundIsReadable(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/examples/missing", nil)
	req.Header.Set("User-Agent", "curl/8.0.1")
	req = withRouteParam(req, "slug", "missing")
	rec := httptest.NewRecorder()

	HandleExampleShow(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	body := rec.Body.String()
	for _, needle := range []string{"# Not found", "status: 404", "route: /examples/missing", "errors:", "- example not found"} {
		if !strings.Contains(body, needle) {
			t.Fatalf("body missing %q:\n%s", needle, body)
		}
	}
}

func TestHandleDraftPostValidationErrorsNonBrowser(t *testing.T) {
	h, err := NewDraftHandler(validationStubService())
	if err != nil {
		t.Fatalf("new draft handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/draft", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "curl/8.0.1")
	rec := httptest.NewRecorder()

	h.HandleDraftPost(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/markdown; charset=utf-8" {
		t.Fatalf("content-type = %q, want markdown", got)
	}
	body := rec.Body.String()
	for _, needle := range []string{"# Validation failed", "status: 400", "route: /draft", "errors:", "- kind:", "- title:", "- visibility:"} {
		if !strings.Contains(body, needle) {
			t.Fatalf("body missing %q:\n%s", needle, body)
		}
	}
}

func TestHandleDraftPostValidationErrorsBrowserStillHTML(t *testing.T) {
	h, err := NewDraftHandler(validationStubService())
	if err != nil {
		t.Fatalf("new draft handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/draft?format=html", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.HandleDraftPost(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want html", got)
	}
	if !strings.Contains(rec.Body.String(), "<h2>Submission Error</h2>") {
		t.Fatalf("expected HTML validation output, got:\n%s", rec.Body.String())
	}
}

func TestAPIExamplesRemainJSONWithFormatOverride(t *testing.T) {
	h := NewAPIHandler(stubService{})
	req := httptest.NewRequest(http.MethodGet, "/api/examples?format=text", nil)
	rec := httptest.NewRecorder()

	h.HandleExamplesList(rec, req)

	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content-type = %q, want json", got)
	}
	if strings.Contains(rec.Body.String(), "# ") {
		t.Fatalf("api response unexpectedly looks human-readable:\n%s", rec.Body.String())
	}
}

func TestWorkDetailDocumentMarkdownContainsKeySections(t *testing.T) {
	body := renderHumanDocument(workDetailDocument(WorkDetail{
		ID:           "1234",
		Status:       "open",
		CreatedAt:    time.Date(2026, 3, 28, 12, 30, 0, 0, time.UTC),
		Version:      1,
		ExactHash:    "exact-hash",
		QuotientHash: "quotient-hash",
		Packet:       seededPacket(),
	}), "markdown")

	for _, needle := range []string{"# Fix recurring auth session loop in dashboard", "## Work item", "work item id: 1234", "## Provenance", "exact hash: exact-hash", "## Current packet", "scope:", "acceptance criteria:", "## Structured JSON"} {
		if !strings.Contains(body, needle) {
			t.Fatalf("body missing %q:\n%s", needle, body)
		}
	}
}

func TestReviewQueueDocumentSeparatesPrivateItems(t *testing.T) {
	body := renderHumanDocument(reviewQueueDocument(ReviewQueueData{
		Standard: []ReviewRow{
			{
				ID:         "standard-id",
				Title:      "Standard queue item",
				Kind:       "bounty",
				Visibility: "public",
				Status:     "open",
				CreatedAt:  time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC),
			},
		},
		Private: []ReviewRow{
			{
				ID:         "private-id",
				Title:      "Private queue item",
				Kind:       "private_security",
				Visibility: "private",
				Status:     "review",
				CreatedAt:  time.Date(2026, 3, 28, 11, 0, 0, 0, time.UTC),
			},
		},
	}), "markdown")

	for _, needle := range []string{"# Review queue", "## Standard queue", "/work/standard-id", "(bounty)", "## Private security queue", "/work/private-id", "## Structured JSON"} {
		if !strings.Contains(body, needle) {
			t.Fatalf("body missing %q:\n%s", needle, body)
		}
	}
}

func withRouteParam(req *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func seededPacket() packets.NormalizedPacket {
	packet, ok := seededExampleBySlug("auth-loop")
	if !ok {
		panic("missing seeded example")
	}
	return packet
}
