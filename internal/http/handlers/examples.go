package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/shaoyanji/bountystash/internal/http/represent"
	"github.com/shaoyanji/bountystash/internal/packets"
	"github.com/shaoyanji/bountystash/internal/views"
)

var exampleShowTemplate = template.Must(views.Parse("examples_show.tmpl", "work_packet.tmpl"))

type exampleShowData struct {
	NavActive string
	Slug      string
	Packet    packets.NormalizedPacket
}

type Example struct {
	Slug   string                   `json:"slug"`
	Packet packets.NormalizedPacket `json:"packet"`
}

// HandleExampleShow renders one seeded example packet by slug.
func HandleExampleShow(w http.ResponseWriter, r *http.Request) {
	det := humanRouteRepresentation(r)
	slug := strings.TrimSpace(chi.URLParam(r, "slug"))
	example, ok := seededExampleBySlug(slug)
	if !ok {
		if det.Representation == represent.RepresentationHTML {
			http.NotFound(w, r)
			return
		}
		writeHumanDocument(w, http.StatusNotFound, det.Representation, errorDocument("Not found", r.URL.Path, http.StatusNotFound, []string{"example not found"}))
		return
	}

	data := exampleShowData{
		NavActive: "examples",
		Slug:      slug,
		Packet:    example,
	}
	if det.Representation != represent.RepresentationHTML {
		writeHumanDocument(w, http.StatusOK, det.Representation, exampleDocument(Example{
			Slug:   slug,
			Packet: example,
		}))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := exampleShowTemplate.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "template render error", http.StatusInternalServerError)
	}
}

func SeededExamples() []Example {
	return []Example{
		{
			Slug: "auth-loop",
			Packet: packets.NormalizedPacket{
				Title: "Fix recurring auth session loop in dashboard",
				Kind:  packets.KindBounty,
				Scope: []string{
					"Reproduce session refresh loop on stale tokens",
					"Patch middleware refresh flow",
					"Add regression checks for login redirect cycle",
				},
				Deliverables: []string{
					"Code patch with root-cause note",
					"Test coverage for auth refresh path",
				},
				AcceptanceCriteria: []string{
					"User remains authenticated after token refresh",
					"No infinite redirect between login and dashboard",
				},
				RewardModel: "Fixed bounty: USD 1,500",
				Visibility:  packets.VisibilityPublic,
			},
		},
		{
			Slug: "webhook-rfq",
			Packet: packets.NormalizedPacket{
				Title: "RFQ: webhook reliability hardening package",
				Kind:  packets.KindRFQ,
				Scope: []string{
					"Delivery retry policy with backoff",
					"Signature verification baseline",
					"Failure replay process",
				},
				Deliverables: []string{
					"Firm quote with timeline",
					"Implementation plan and assumptions",
				},
				AcceptanceCriteria: []string{
					"Quoted scope covers retry, signature, and replay",
					"Includes rollout and validation steps",
				},
				RewardModel: "Quoted engagement",
				Visibility:  packets.VisibilityPublic,
			},
		},
		{
			Slug: "pipeline-rfp",
			Packet: packets.NormalizedPacket{
				Title: "RFP: modernize CI pipeline for multi-service releases",
				Kind:  packets.KindRFP,
				Scope: []string{
					"Current-state CI assessment",
					"Proposed target architecture",
					"Migration approach with risk controls",
				},
				Deliverables: []string{
					"Formal proposal document",
					"Milestone plan and staffing model",
				},
				AcceptanceCriteria: []string{
					"Proposal includes implementation and ownership boundaries",
					"Risk, rollback, and delivery timeline are explicit",
				},
				RewardModel: "Proposal-based selection",
				Visibility:  packets.VisibilityPublic,
			},
		},
	}
}

func seededExampleBySlug(slug string) (packets.NormalizedPacket, bool) {
	for _, example := range SeededExamples() {
		if example.Slug == slug {
			return example.Packet, true
		}
	}
	return packets.NormalizedPacket{}, false
}
