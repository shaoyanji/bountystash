package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"bountystash/internal/packets"
)

var draftPreviewTemplate = template.Must(template.New("draft_preview").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Draft Preview</title>
</head>
<body>
  <main>
    <h1>Draft Preview</h1>
    <p><strong>Title:</strong> {{.Title}}</p>
    <p><strong>Kind:</strong> {{.Kind}}</p>
    <p><strong>Visibility:</strong> {{.Visibility}}</p>

    <section>
      <h2>Scope</h2>
      <ul>{{range .Scope}}<li>{{.}}</li>{{else}}<li>None provided</li>{{end}}</ul>
    </section>

    <section>
      <h2>Deliverables</h2>
      <ul>{{range .Deliverables}}<li>{{.}}</li>{{else}}<li>None provided</li>{{end}}</ul>
    </section>

    <section>
      <h2>Acceptance Criteria</h2>
      <ul>{{range .AcceptanceCriteria}}<li>{{.}}</li>{{else}}<li>None provided</li>{{end}}</ul>
    </section>

    <p><strong>Reward / Quote Model:</strong> {{.RewardModel}}</p>
    <p><small>Normalized at: {{.NormalizedAt.Format "2006-01-02 15:04:05 MST"}}</small></p>
  </main>
</body>
</html>
`))

// RegisterDraftRoutes wires only the draft preview endpoint for milestone intake flow.
func RegisterDraftRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /draft", HandleDraftPreviewPost)
}

// HandleDraftPreviewPost accepts draft form input and returns a server-rendered preview.
// Persistence is intentionally deferred; this endpoint only normalizes and renders.
func HandleDraftPreviewPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("parse form: %v", err), http.StatusBadRequest)
		return
	}

	input := packets.DraftInput{
		Title:              r.FormValue("title"),
		Kind:               r.FormValue("kind"),
		Scope:              r.FormValue("scope"),
		Deliverables:       r.FormValue("deliverables"),
		AcceptanceCriteria: r.FormValue("acceptance_criteria"),
		RewardModel:        r.FormValue("reward_model"),
		Visibility:         r.FormValue("visibility"),
	}

	normalized := normalizeDraft(input)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := draftPreviewTemplate.Execute(w, normalized); err != nil {
		http.Error(w, fmt.Sprintf("render draft preview: %v", err), http.StatusInternalServerError)
	}
}

func normalizeDraft(in packets.DraftInput) packets.NormalizedPacket {
	kind := normalizeKind(in.Kind)
	visibility := normalizeVisibility(in.Visibility)

	// Safety default: private security items must not become public by default.
	if kind == packets.KindPrivateSecurity && visibility != packets.VisibilityPrivate {
		visibility = packets.VisibilityPrivate
	}

	return packets.NormalizedPacket{
		Title:              normalizeTitle(in.Title),
		Kind:               kind,
		Scope:              normalizeList(in.Scope),
		Deliverables:       normalizeList(in.Deliverables),
		AcceptanceCriteria: normalizeList(in.AcceptanceCriteria),
		RewardModel:        strings.TrimSpace(in.RewardModel),
		Visibility:         visibility,
		NormalizedAt:       time.Now().UTC(),
	}
}

func normalizeTitle(raw string) string {
	t := strings.TrimSpace(raw)
	if t == "" {
		return "Untitled Draft"
	}
	return t
}

func normalizeKind(raw string) packets.Kind {
	switch packets.Kind(strings.TrimSpace(strings.ToLower(raw))) {
	case packets.KindBounty:
		return packets.KindBounty
	case packets.KindRFQ:
		return packets.KindRFQ
	case packets.KindRFP:
		return packets.KindRFP
	case packets.KindPrivateSecurity:
		return packets.KindPrivateSecurity
	default:
		return packets.KindBounty
	}
}

func normalizeVisibility(raw string) packets.Visibility {
	switch packets.Visibility(strings.TrimSpace(strings.ToLower(raw))) {
	case packets.VisibilityDraft:
		return packets.VisibilityDraft
	case packets.VisibilityPrivate:
		return packets.VisibilityPrivate
	case packets.VisibilityPublic:
		return packets.VisibilityPublic
	case packets.VisibilityArchived:
		return packets.VisibilityArchived
	default:
		return packets.VisibilityDraft
	}
}

func normalizeList(raw string) []string {
	rows := strings.Split(raw, "\n")
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		v := strings.TrimSpace(row)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}
