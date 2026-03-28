package handlers

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shaoyanji/bountystash/internal/http/represent"
	"github.com/shaoyanji/bountystash/internal/packets"
)

type humanDocument struct {
	Title    string
	Summary  string
	Fields   []humanField
	Lists    []humanList
	Sections []humanSection
}

type humanSection struct {
	Heading    string
	Paragraphs []string
	Fields     []humanField
	Lists      []humanList
}

type humanField struct {
	Label string
	Value string
}

type humanList struct {
	Title string
	Items []string
}

func humanRouteRepresentation(r *http.Request) represent.Determination {
	return represent.Determine(r, represent.Options{Contract: represent.ContractHuman})
}

func writeHumanDocument(w http.ResponseWriter, status int, rep represent.Representation, doc humanDocument) {
	if rep == represent.RepresentationHTML {
		rep = represent.RepresentationMarkdown
	}

	w.Header().Set("Content-Type", rep.ContentType())
	w.WriteHeader(status)
	_, _ = io.WriteString(w, renderHumanDocument(doc, rep))
}

func renderHumanDocument(doc humanDocument, rep represent.Representation) string {
	if rep == represent.RepresentationPlainText {
		return renderPlainTextDocument(doc)
	}
	return renderMarkdownDocument(doc)
}

func renderMarkdownDocument(doc humanDocument) string {
	lines := make([]string, 0, 32)
	lines = append(lines, "# "+doc.Title)

	if doc.Summary != "" {
		lines = append(lines, "", doc.Summary)
	}

	lines = appendMarkdownFields(lines, doc.Fields)
	lines = appendMarkdownLists(lines, doc.Lists)
	for _, section := range doc.Sections {
		lines = append(lines, "")
		if section.Heading != "" {
			lines = append(lines, "## "+section.Heading)
			lines = append(lines, "")
		}
		for _, paragraph := range section.Paragraphs {
			lines = append(lines, paragraph)
		}
		lines = appendMarkdownFields(lines, section.Fields)
		lines = appendMarkdownLists(lines, section.Lists)
	}

	return strings.TrimSpace(strings.Join(lines, "\n")) + "\n"
}

func appendMarkdownFields(lines []string, fields []humanField) []string {
	for _, field := range fields {
		lines = append(lines, field.Label+": "+field.Value)
	}
	return lines
}

func appendMarkdownLists(lines []string, lists []humanList) []string {
	for _, list := range lists {
		if list.Title != "" {
			lines = append(lines, list.Title+":")
		}
		if len(list.Items) == 0 {
			lines = append(lines, "- None.")
			continue
		}
		for _, item := range list.Items {
			lines = append(lines, "- "+item)
		}
	}
	return lines
}

func renderPlainTextDocument(doc humanDocument) string {
	lines := make([]string, 0, 32)
	lines = append(lines, doc.Title, strings.Repeat("=", len(doc.Title)))

	if doc.Summary != "" {
		lines = append(lines, "", doc.Summary)
	}

	lines = appendPlainTextFields(lines, doc.Fields)
	lines = appendPlainTextLists(lines, doc.Lists)
	for _, section := range doc.Sections {
		lines = append(lines, "")
		if section.Heading != "" {
			lines = append(lines, section.Heading, strings.Repeat("-", len(section.Heading)))
		}
		for _, paragraph := range section.Paragraphs {
			lines = append(lines, paragraph)
		}
		lines = appendPlainTextFields(lines, section.Fields)
		lines = appendPlainTextLists(lines, section.Lists)
	}

	return strings.TrimSpace(strings.Join(lines, "\n")) + "\n"
}

func appendPlainTextFields(lines []string, fields []humanField) []string {
	for _, field := range fields {
		lines = append(lines, field.Label+": "+field.Value)
	}
	return lines
}

func appendPlainTextLists(lines []string, lists []humanList) []string {
	for _, list := range lists {
		if list.Title != "" {
			lines = append(lines, list.Title+":")
		}
		if len(list.Items) == 0 {
			lines = append(lines, "- None.")
			continue
		}
		for _, item := range list.Items {
			lines = append(lines, "- "+item)
		}
	}
	return lines
}

func landingDocument() humanDocument {
	exampleRoutes := make([]string, 0, len(SeededExamples()))
	for _, example := range SeededExamples() {
		exampleRoutes = append(exampleRoutes, fmt.Sprintf("/examples/%s - seeded example packet", example.Slug))
	}

	usefulRoutes := []string{
		"/ - landing page and browser intake form",
		"/review - reviewer queue with private security separated",
		"/work/{id} - persisted work detail when you already have a work item ID",
	}
	usefulRoutes = append(usefulRoutes, exampleRoutes...)

	return humanDocument{
		Title:   "Bountystash",
		Summary: "Thin server-rendered intake app for turning messy technical asks into deterministic, immutable work packets with persisted provenance hashes.",
		Sections: []humanSection{
			{
				Heading: "Current capabilities",
				Lists: []humanList{
					{
						Items: []string{
							"Normalize intake into stable packet fields for bounty, rfq, rfp, and private_security work.",
							"Persist immutable work versions with exact and quotient hashes.",
							"Expose a minimal review queue and additive JSON API routes.",
						},
					},
				},
			},
			{
				Heading: "Useful routes",
				Lists: []humanList{
					{Items: usefulRoutes},
				},
			},
			{
				Heading: "Useful API endpoints",
				Paragraphs: []string{
					"/api/* routes stay JSON for machine use; non-API routes default to readable output for non-browser clients.",
				},
				Lists: []humanList{
					{
						Items: []string{
							"/api/examples - seeded examples as JSON",
							"/api/examples/{slug} - one seeded example as JSON",
							"/api/review - review queue as JSON",
							"/api/work - recent work items as JSON",
							"/api/work/{id} - one persisted work item as JSON",
							"/api/draft - create a draft from JSON",
						},
					},
				},
			},
			{
				Heading: "Format overrides",
				Lists: []humanList{
					{
						Items: []string{
							"?format=html - force browser HTML on a human-facing route",
							"?format=md - force markdown output on a human-facing route",
							"?format=text - force plain text output on a human-facing route",
						},
					},
				},
			},
			{
				Heading: "Next step",
				Paragraphs: []string{
					"Start with /examples/auth-loop for a safe sample, or move to /api/examples when you need structured JSON immediately.",
				},
			},
		},
	}
}

func workDetailDocument(work WorkDetail) humanDocument {
	return humanDocument{
		Title:   work.Packet.Title,
		Summary: "Immutable work item detail for the current persisted packet version.",
		Sections: []humanSection{
			{
				Heading: "Work item",
				Fields: []humanField{
					{Label: "work item id", Value: work.ID},
					{Label: "status", Value: fallbackValue(work.Status, "not available")},
					{Label: "created", Value: formatHumanTime(work.CreatedAt)},
					{Label: "version", Value: strconv.Itoa(work.Version)},
					{Label: "kind", Value: string(work.Packet.Kind)},
					{Label: "visibility", Value: string(work.Packet.Visibility)},
				},
			},
			{
				Heading: "Provenance",
				Fields: []humanField{
					{Label: "exact hash", Value: fallbackValue(work.ExactHash, "not available")},
					{Label: "quotient hash", Value: fallbackValue(work.QuotientHash, "not available")},
				},
			},
			{
				Heading: "Current packet",
				Fields: []humanField{
					{Label: "title", Value: fallbackValue(work.Packet.Title, "not available")},
					{Label: "reward model", Value: fallbackValue(work.Packet.RewardModel, "not specified")},
				},
				Lists: packetLists(work.Packet),
			},
			{
				Heading: "Structured JSON",
				Paragraphs: []string{
					fmt.Sprintf("Use /api/work/%s when you need the machine-readable packet and metadata as JSON.", work.ID),
				},
			},
		},
	}
}

func exampleDocument(example Example) humanDocument {
	return humanDocument{
		Title:   example.Packet.Title,
		Summary: "Seeded example packet for deterministic shape, route discovery, and safe create/API exploration.",
		Sections: []humanSection{
			{
				Heading: "Example",
				Fields: []humanField{
					{Label: "slug", Value: example.Slug},
					{Label: "kind", Value: string(example.Packet.Kind)},
					{Label: "visibility", Value: string(example.Packet.Visibility)},
					{Label: "reward model", Value: fallbackValue(example.Packet.RewardModel, "not specified")},
				},
				Paragraphs: []string{
					"This example demonstrates the normalized packet fields Bountystash persists and exposes through the readable and JSON surfaces.",
				},
			},
			{
				Heading: "Packet fields",
				Fields: []humanField{
					{Label: "title", Value: fallbackValue(example.Packet.Title, "not available")},
				},
				Lists: packetLists(example.Packet),
			},
			{
				Heading: "Next step",
				Paragraphs: []string{
					fmt.Sprintf("Use /api/examples/%s for the structured JSON version, or use this packet shape as a guide for browser or API draft creation.", example.Slug),
				},
			},
		},
	}
}

func reviewQueueDocument(queue ReviewQueueData) humanDocument {
	return humanDocument{
		Title:   "Review queue",
		Summary: "Reviewer-facing queue for persisted work items currently in open or review status.",
		Sections: []humanSection{
			reviewQueueSection("Standard queue", queue.Standard, false, "No standard queue items."),
			reviewQueueSection("Private security queue", queue.Private, true, "No private security queue items."),
			{
				Heading: "Structured JSON",
				Paragraphs: []string{
					"Use /api/review when you need the queue split in machine-readable JSON.",
				},
			},
		},
	}
}

func errorDocument(title, route string, status int, errors []string) humanDocument {
	return humanDocument{
		Title: title,
		Fields: []humanField{
			{Label: "status", Value: strconv.Itoa(status)},
			{Label: "route", Value: route},
		},
		Lists: []humanList{
			{Title: "errors", Items: errors},
		},
	}
}

func validationErrorsDocument(route string, status int, errs packets.ValidationErrors) humanDocument {
	keys := make([]string, 0, len(errs))
	for key := range errs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s: %s", key, errs[key]))
	}

	return errorDocument("Validation failed", route, status, lines)
}

func packetLists(packet packets.NormalizedPacket) []humanList {
	return []humanList{
		{Title: "scope", Items: defaultListItems(packet.Scope, "No scope items provided.")},
		{Title: "deliverables", Items: defaultListItems(packet.Deliverables, "No deliverables provided.")},
		{Title: "acceptance criteria", Items: defaultListItems(packet.AcceptanceCriteria, "No acceptance criteria provided.")},
	}
}

func reviewRows(rows []ReviewRow, privateOnly bool) []string {
	if len(rows) == 0 {
		return nil
	}

	out := make([]string, 0, len(rows))
	for _, row := range rows {
		item := fmt.Sprintf("%s [%s | %s | created %s] -> /work/%s", row.Title, row.Visibility, row.Status, formatHumanTime(row.CreatedAt), row.ID)
		if !privateOnly {
			item = fmt.Sprintf("%s (%s)", item, row.Kind)
		}
		out = append(out, item)
	}
	return out
}

func emptyQueueParagraphs(rows []ReviewRow, empty string) []string {
	if len(rows) == 0 {
		return []string{empty}
	}
	return nil
}

func reviewQueueSection(heading string, rows []ReviewRow, privateOnly bool, empty string) humanSection {
	section := humanSection{
		Heading:    heading,
		Paragraphs: emptyQueueParagraphs(rows, empty),
	}
	if len(rows) == 0 {
		return section
	}
	section.Lists = []humanList{{Items: reviewRows(rows, privateOnly)}}
	return section
}

func defaultListItems(items []string, empty string) []string {
	if len(items) == 0 {
		return []string{empty}
	}
	return items
}

func formatHumanTime(ts time.Time) string {
	if ts.IsZero() {
		return "not available"
	}
	return ts.UTC().Format(time.RFC3339)
}

func fallbackValue(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
