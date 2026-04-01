package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shaoyanji/bountystash/internal/service"
)

func TestSummarizeHistoryEventIntakeReceived(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"title":  "Test bounty",
		"kind":   "bounty",
		"source": "service",
	})

	evt := service.Event{
		ID:        "evt-1",
		EventType: "intake_received",
		CreatedAt: time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC),
		Payload:   payload,
	}

	entry, ok := summarizeHistoryEvent(evt)
	if !ok {
		t.Fatalf("expected entry to be summarized")
	}
	if entry.Summary != "Intake accepted" {
		t.Fatalf("summary = %q, want %q", entry.Summary, "Intake accepted")
	}
	if entry.DisplayType != "intake received" {
		t.Fatalf("displayType = %q, want %q", entry.DisplayType, "intake received")
	}
}

func TestSummarizeHistoryEventWorkVersionPersisted(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"version_number": 1,
		"exact_hash":     "abc123def456",
		"quotient_hash":  "xyz789",
	})

	evt := service.Event{
		ID:            "evt-2",
		EventType:     "work_version_persisted",
		WorkItemID:    "work-1",
		WorkVersionID: ptrString("version-1"),
		CreatedAt:     time.Date(2026, 3, 30, 11, 0, 0, 0, time.UTC),
		Payload:       payload,
	}

	entry, ok := summarizeHistoryEvent(evt)
	if !ok {
		t.Fatalf("expected entry to be summarized")
	}
	if entry.Summary != "Version 1 persisted" {
		t.Fatalf("summary = %q, want %q", entry.Summary, "Version 1 persisted")
	}
}

func TestSummarizeHistoryEventUnknown(t *testing.T) {
	evt := service.Event{
		ID:        "evt-unknown",
		EventType: "unknown_event",
		CreatedAt: time.Now(),
	}

	_, ok := summarizeHistoryEvent(evt)
	if ok {
		t.Fatalf("expected unknown event to return false")
	}
}

func TestSummarizeSystemHistoryEventIntakeReceived(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"title":  "System test",
		"kind":   "rfq",
		"source": "service",
	})

	evt := service.Event{
		ID:        "evt-sys-1",
		EventType: "intake_received",
		CreatedAt: time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC),
		Payload:   payload,
	}

	entry, ok := summarizeSystemHistoryEvent(evt)
	if !ok {
		t.Fatalf("expected entry to be summarized")
	}
	if entry.Summary != "Intake accepted" {
		t.Fatalf("summary = %q, want %q", entry.Summary, "Intake accepted")
	}
}

func TestSummarizeSystemHistoryEventWorkItemCreated(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"kind":       "bounty",
		"visibility": "public",
		"source":     "service",
	})

	evt := service.Event{
		ID:         "evt-sys-2",
		EventType:  "work_item_created",
		WorkItemID: "work-new-1",
		CreatedAt:  time.Date(2026, 3, 30, 11, 0, 0, 0, time.UTC),
		Payload:    payload,
	}

	entry, ok := summarizeSystemHistoryEvent(evt)
	if !ok {
		t.Fatalf("expected entry to be summarized")
	}
	if entry.Summary != "Work item created" {
		t.Fatalf("summary = %q, want %q", entry.Summary, "Work item created")
	}
	foundID := false
	for _, d := range entry.Details {
		if d == "ID: work-new-1" {
			foundID = true
		}
	}
	if !foundID {
		t.Fatalf("expected work item ID in details")
	}
}

func TestSummarizeSystemHistoryEventReviewQueueRead(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"standard_count": 5,
		"private_count":  2,
		"source":         "service",
	})

	evt := service.Event{
		ID:        "evt-sys-3",
		EventType: "review_queue_read",
		CreatedAt: time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC),
		Payload:   payload,
	}

	entry, ok := summarizeSystemHistoryEvent(evt)
	if !ok {
		t.Fatalf("expected entry to be summarized")
	}
	if entry.Summary != "Review queue accessed" {
		t.Fatalf("summary = %q, want %q", entry.Summary, "Review queue accessed")
	}
}

func TestSystemHistoryDocumentEmpty(t *testing.T) {
	doc := systemHistoryDocument([]historyEntry{})
	if doc.Title != "Recent activity" {
		t.Fatalf("title = %q, want %q", doc.Title, "Recent activity")
	}
	if len(doc.Sections) != 1 {
		t.Fatalf("expected 1 section for empty document")
	}
}

func TestSystemHistoryDocumentWithEntries(t *testing.T) {
	entries := []historyEntry{
		{
			Timestamp:   time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC),
			DisplayType: "intake received",
			Summary:     "Intake accepted",
			Details:     []string{"Title: Test"},
		},
	}

	doc := systemHistoryDocument(entries)
	if len(doc.Sections) != 1 {
		t.Fatalf("expected 1 section")
	}
}

func ptrString(s string) *string {
	return &s
}
