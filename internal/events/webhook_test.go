package events

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestNewDispatcher(t *testing.T) {
	// Test with empty webhook URL (should not start workers that do anything useful)
	d := NewDispatcher("", 10, 2)
	if d == nil {
		t.Fatal("NewDispatcher returned nil")
	}
	if d.webhookURL != "" {
		t.Errorf("webhookURL = %q, want empty", d.webhookURL)
	}
	if d.queue == nil {
		t.Error("queue is nil")
	}
	if d.shutdown == nil {
		t.Error("shutdown channel is nil")
	}
	d.Shutdown()
}

func TestNewDispatcher_WithURL(t *testing.T) {
	d := NewDispatcher("https://example.com/webhook", 5, 1)
	if d.webhookURL != "https://example.com/webhook" {
		t.Errorf("webhookURL = %q, want %q", d.webhookURL, "https://example.com/webhook")
	}
	d.Shutdown()
}

func TestDispatcher_Enqueue(t *testing.T) {
	d := NewDispatcher("", 10, 1)
	defer d.Shutdown()

	event := Event{
		Type: "test_event",
		Payload: map[string]interface{}{
			"key": "value",
		},
	}

	// Enqueue should not block
	d.Enqueue(event)

	// Give worker time to process
	time.Sleep(50 * time.Millisecond)
}

func TestDispatcher_Enqueue_FullQueue(t *testing.T) {
	// Create dispatcher with very small queue
	d := NewDispatcher("", 1, 1)

	// Fill the queue
	d.Enqueue(Event{Type: "event1"})
	d.Enqueue(Event{Type: "event2"}) // This should be dropped, not block

	// If we get here, the non-blocking behavior works
	time.Sleep(50 * time.Millisecond)
	d.Shutdown()
}

func TestDispatcher_Shutdown_DrainsQueue(t *testing.T) {
	var receivedEvents []Event
	var mu sync.Mutex

	// Create a test server to receive webhooks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err == nil {
			mu.Lock()
			receivedEvents = append(receivedEvents, event)
			mu.Unlock()
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher(server.URL, 100, 1)

	// Enqueue multiple events
	for i := 0; i < 5; i++ {
		d.Enqueue(Event{Type: "test_event"})
	}

	// Shutdown should drain the queue
	d.Shutdown()

	// Check that events were processed
	mu.Lock()
	count := len(receivedEvents)
	mu.Unlock()

	if count != 5 {
		t.Errorf("processed %d events, want 5", count)
	}
}

func TestEventString(t *testing.T) {
	e := Event{
		Type:      "work_item_created",
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	s := e.String()
	if s == "" {
		t.Error("String() returned empty string")
	}
}
