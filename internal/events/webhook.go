// Package events provides webhook notification dispatch functionality.
package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Event represents a backend event to be dispatched via webhook.
type Event struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// Dispatcher handles asynchronous webhook event delivery.
type Dispatcher struct {
	webhookURL string
	queue      chan Event
	workerPool int
	client     *http.Client
	wg         sync.WaitGroup
	shutdown   chan struct{}
}

// NewDispatcher creates a new webhook dispatcher with the given configuration.
func NewDispatcher(webhookURL string, queueSize int, workerPool int) *Dispatcher {
	d := &Dispatcher{
		webhookURL: webhookURL,
		queue:      make(chan Event, queueSize),
		workerPool: workerPool,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		shutdown: make(chan struct{}),
	}

	// Start worker pool
	for i := 0; i < workerPool; i++ {
		d.wg.Add(1)
		go d.worker()
	}

	return d
}

// Enqueue adds an event to the dispatch queue (non-blocking).
func (d *Dispatcher) Enqueue(event Event) {
	event.Timestamp = time.Now().UTC()

	select {
	case d.queue <- event:
		// Successfully queued
	default:
		// Queue full, drop event (could log this)
	}
}

// worker processes events from the queue.
func (d *Dispatcher) worker() {
	defer d.wg.Done()

	for {
		select {
		case event := <-d.queue:
			d.sendEvent(event)
		case <-d.shutdown:
			// Drain remaining events before shutdown
			for {
				select {
				case event := <-d.queue:
					d.sendEvent(event)
				default:
					return
				}
			}
		}
	}
}

// sendEvent sends a single event to the webhook URL.
func (d *Dispatcher) sendEvent(event Event) {
	if d.webhookURL == "" {
		return
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(payload))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Event-Type", event.Type)

	resp, err := d.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Non-blocking: we don't retry on failure
}

// Shutdown gracefully stops the dispatcher.
func (d *Dispatcher) Shutdown() {
	close(d.shutdown)
	d.wg.Wait()
}

// String returns a string representation of the event type.
func (e Event) String() string {
	return fmt.Sprintf("Event{Type: %s, Timestamp: %s}", e.Type, e.Timestamp.Format(time.RFC3339))
}
