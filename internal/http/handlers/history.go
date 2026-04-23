package handlers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shaoyanji/bountystash/internal/service"
)

type historyEntry struct {
	Timestamp   time.Time
	EventType   string
	DisplayType string
	Summary     string
	Details     []string
	WorkItemID  string
}

type historyPageData struct {
	NavActive string
	WorkID    string
	WorkTitle string
	Entries   []historyEntry
}

func historyDocument(workID string, entries []historyEntry) humanDocument {
	doc := humanDocument{
		Title:   "Work history",
		Summary: fmt.Sprintf("Durable events for work item %s", workID),
	}

	if len(entries) == 0 {
		doc.Sections = append(doc.Sections, humanSection{
			Heading: "Timeline",
			Paragraphs: []string{
				"No recorded events exist for this work item yet.",
			},
		})
		return doc
	}

	for _, entry := range entries {
		line := fmt.Sprintf("%s — %s", entry.Timestamp.Format(time.RFC3339), entry.DisplayType)
		if entry.Summary != "" {
			line = fmt.Sprintf("%s: %s", line, entry.Summary)
		}
		section := humanSection{
			Heading: line,
		}
		if len(entry.Details) > 0 {
			section.Lists = append(section.Lists, humanList{Items: entry.Details})
		}
		doc.Sections = append(doc.Sections, section)
	}

	return doc
}

func summarizeHistoryEvents(events []service.Event) []historyEntry {
	entries := make([]historyEntry, 0, len(events))
	for _, evt := range events {
		if entry, ok := summarizeHistoryEvent(evt); ok {
			entries = append(entries, entry)
		}
	}
	return entries
}

func summarizeHistoryEvent(evt service.Event) (historyEntry, bool) {
	var payload map[string]any
	if len(evt.Payload) > 0 {
		_ = json.Unmarshal(evt.Payload, &payload)
	}

	var entry historyEntry
	entry.Timestamp = evt.CreatedAt
	entry.EventType = evt.EventType
	entry.DisplayType = humanReadableEventType(evt.EventType)
	entry.WorkItemID = evt.WorkItemID

	switch evt.EventType {
	case "intake_received":
		entry.Summary = "Intake accepted"
		entry.Details = append(entry.Details, payloadSummary(payload, "title", "Title"))
		entry.Details = append(entry.Details, payloadSummary(payload, "kind", "Kind"))
		entry.Details = append(entry.Details, payloadSummary(payload, "visibility", "Visibility"))
	case "intake_validation_failed":
		entry.Summary = "Validation failed"
		if details := validationDetails(payload["errors"]); len(details) > 0 {
			entry.Details = append(entry.Details, details...)
		} else {
			entry.Details = append(entry.Details, "Errors recorded but not provided.")
		}
	case "packet_normalized":
		entry.Summary = "Packet normalized"
		entry.Details = append(entry.Details, payloadSummary(payload, "reward_model", "Reward model"))
	case "work_item_created":
		entry.Summary = "Work item created"
		entry.Details = append(entry.Details, payloadSummary(payload, "status", "Status"))
		entry.Details = append(entry.Details, payloadSummary(payload, "visibility", "Visibility"))
	case "work_version_persisted":
		if num, ok := payload["version_number"]; ok {
			entry.Summary = fmt.Sprintf("Version %v persisted", num)
		} else {
			entry.Summary = "Version persisted"
		}
		if hash := payloadString(payload, "exact_hash"); hash != "" {
			entry.Details = append(entry.Details, fmt.Sprintf("Exact hash: %s", hash))
		}
		if hash := payloadString(payload, "quotient_hash"); hash != "" {
			entry.Details = append(entry.Details, fmt.Sprintf("Quotient hash: %s", hash))
		}
	default:
		return historyEntry{}, false
	}

	cleanDetails := make([]string, 0, len(entry.Details))
	for _, detail := range entry.Details {
		if strings.TrimSpace(detail) != "" {
			cleanDetails = append(cleanDetails, detail)
		}
	}
	entry.Details = cleanDetails

	return entry, true
}

func payloadSummary(payload map[string]any, key, label string) string {
	if v, ok := payload[key]; ok {
		if s := fmt.Sprint(v); s != "" {
			return fmt.Sprintf("%s: %s", label, s)
		}
	}
	return ""
}

func payloadString(payload map[string]any, key string) string {
	if v, ok := payload[key]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

func validationDetails(raw any) []string {
	details := make([]string, 0)
	errMap, ok := raw.(map[string]any)
	if !ok {
		if str, ok := raw.(string); ok && str != "" {
			return []string{str}
		}
		return nil
	}
	keys := make([]string, 0, len(errMap))
	for k := range errMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		details = append(details, fmt.Sprintf("%s: %v", k, errMap[k]))
	}
	return details
}

func humanReadableEventType(eventType string) string {
	return strings.ReplaceAll(eventType, "_", " ")
}

func summarizeSystemHistoryEvents(events []service.Event) []historyEntry {
	entries := make([]historyEntry, 0, len(events))
	for _, evt := range events {
		if entry, ok := summarizeSystemHistoryEvent(evt); ok {
			entries = append(entries, entry)
		}
	}
	return entries
}

func summarizeSystemHistoryEvent(evt service.Event) (historyEntry, bool) {
	var payload map[string]any
	if len(evt.Payload) > 0 {
		_ = json.Unmarshal(evt.Payload, &payload)
	}

	var entry historyEntry
	entry.Timestamp = evt.CreatedAt
	entry.EventType = evt.EventType
	entry.DisplayType = humanReadableEventType(evt.EventType)
	entry.WorkItemID = evt.WorkItemID

	switch evt.EventType {
	case "intake_received":
		entry.Summary = "Intake accepted"
		if title := payloadSummary(payload, "title", ""); title != "" {
			entry.Details = append(entry.Details, title)
		}
		entry.Details = append(entry.Details, payloadSummary(payload, "kind", "Kind"))
	case "intake_validation_failed":
		entry.Summary = "Validation failed"
		if details := validationDetails(payload["errors"]); len(details) > 0 {
			entry.Details = append(entry.Details, details[:min(len(details), 3)]...)
		}
	case "packet_normalized":
		entry.Summary = "Packet normalized"
		entry.Details = append(entry.Details, payloadSummary(payload, "kind", "Kind"))
	case "work_item_created":
		entry.Summary = "Work item created"
		if evt.WorkItemID != "" {
			entry.Details = append(entry.Details, "ID: "+evt.WorkItemID)
		}
		entry.Details = append(entry.Details, payloadSummary(payload, "kind", "Kind"))
	case "work_version_persisted":
		if num, ok := payload["version_number"]; ok {
			entry.Summary = fmt.Sprintf("Version %v persisted", num)
		} else {
			entry.Summary = "Version persisted"
		}
		if hash := payloadString(payload, "exact_hash"); hash != "" {
			shortHash := hash
			if len(hash) > 16 {
				shortHash = hash[:16] + "..."
			}
			entry.Details = append(entry.Details, "Hash: "+shortHash)
		}
	case "work_item_read":
		entry.Summary = "Work item viewed"
		entry.Details = append(entry.Details, payloadSummary(payload, "kind", "Kind"))
	case "review_queue_read":
		entry.Summary = "Review queue accessed"
		if std, ok := payload["standard_count"]; ok {
			entry.Details = append(entry.Details, fmt.Sprintf("Standard: %v", std))
		}
		if priv, ok := payload["private_count"]; ok {
			entry.Details = append(entry.Details, fmt.Sprintf("Private: %v", priv))
		}
	default:
		entry.Summary = evt.EventType
	}

	cleanDetails := make([]string, 0, len(entry.Details))
	for _, detail := range entry.Details {
		if strings.TrimSpace(detail) != "" {
			cleanDetails = append(cleanDetails, detail)
		}
	}
	entry.Details = cleanDetails

	return entry, true
}

func systemHistoryDocument(entries []historyEntry) humanDocument {
	doc := humanDocument{
		Title:   "Recent activity",
		Summary: "System-wide ledger of recent backend events",
	}

	if len(entries) == 0 {
		doc.Sections = append(doc.Sections, humanSection{
			Heading: "Timeline",
			Paragraphs: []string{
				"No recent events found.",
			},
		})
		return doc
	}

	for _, entry := range entries {
		line := fmt.Sprintf("%s — %s", entry.Timestamp.Format(time.RFC3339), entry.DisplayType)
		if entry.Summary != "" {
			line = fmt.Sprintf("%s: %s", line, entry.Summary)
		}
		section := humanSection{
			Heading: line,
		}
		if len(entry.Details) > 0 {
			section.Lists = append(section.Lists, humanList{Items: entry.Details})
		}
		doc.Sections = append(doc.Sections, section)
	}

	return doc
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
