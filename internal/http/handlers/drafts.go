package handlers

import (
	"context"
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/shaoyanji/bountystash/internal/http/represent"
	"github.com/shaoyanji/bountystash/internal/packets"
	"github.com/shaoyanji/bountystash/internal/service"
	"github.com/shaoyanji/bountystash/internal/views"
)

type DraftHandler struct {
	svc              service.WorkService
	homeTemplate     *template.Template
	workShowTemplate *template.Template
	historyTemplate  *template.Template
}

type DraftCreateResult struct {
	ID           string                   `json:"id"`
	Status       string                   `json:"status"`
	Version      int                      `json:"version"`
	ExactHash    string                   `json:"exact_hash"`
	QuotientHash string                   `json:"quotient_hash"`
	Packet       packets.NormalizedPacket `json:"packet"`
}

type WorkDetail = service.WorkDetail

type WorkListRow struct {
	ID         string             `json:"id"`
	Title      string             `json:"title"`
	Kind       packets.Kind       `json:"kind"`
	Visibility packets.Visibility `json:"visibility"`
	Status     string             `json:"status"`
	CreatedAt  time.Time          `json:"created_at"`
}

type homeData struct {
	Input  packets.DraftInput
	Errors map[string]string
}

func NewDraftHandler(svc service.WorkService) (*DraftHandler, error) {
	if svc == nil {
		return nil, errors.New("service is required")
	}

	homeTemplate, err := views.Parse("home.tmpl")
	if err != nil {
		return nil, err
	}

	workShowTemplate, err := views.Parse("work_show.tmpl", "work_packet.tmpl")
	if err != nil {
		return nil, err
	}

	historyTemplate, err := views.Parse("history.tmpl")
	if err != nil {
		return nil, err
	}

	return &DraftHandler{
		svc:              svc,
		homeTemplate:     homeTemplate,
		workShowTemplate: workShowTemplate,
		historyTemplate:  historyTemplate,
	}, nil
}

func (h *DraftHandler) HandleHome(w http.ResponseWriter, r *http.Request) {
	det := humanRouteRepresentation(r)
	if det.Representation != represent.RepresentationHTML {
		writeHumanDocument(w, http.StatusOK, det.Representation, landingDocument())
		return
	}

	h.renderHome(w, homeData{}, http.StatusOK)
}

func (h *DraftHandler) renderHome(w http.ResponseWriter, data homeData, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.homeTemplate.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "template render error", http.StatusInternalServerError)
	}
}

func (h *DraftHandler) HandleDraftPost(w http.ResponseWriter, r *http.Request) {
	det := humanRouteRepresentation(r)
	if err := r.ParseForm(); err != nil {
		if det.Representation == represent.RepresentationHTML {
			http.Error(w, "parse form", http.StatusBadRequest)
			return
		}
		writeHumanDocument(w, http.StatusBadRequest, det.Representation, errorDocument("Validation failed", r.URL.Path, http.StatusBadRequest, []string{"parse form"}))
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

	result, validationErrors, err := h.CreateDraft(r.Context(), input)
	if !validationErrors.Empty() {
		if det.Representation != represent.RepresentationHTML {
			writeHumanDocument(w, http.StatusBadRequest, det.Representation, validationErrorsDocument(r.URL.Path, http.StatusBadRequest, validationErrors))
			return
		}
		h.renderHome(w, homeData{
			Input:  input,
			Errors: validationErrors,
		}, http.StatusBadRequest)
		return
	}
	if err != nil {
		if det.Representation == represent.RepresentationHTML {
			http.Error(w, "failed to persist draft", http.StatusInternalServerError)
			return
		}
		writeHumanDocument(w, http.StatusInternalServerError, det.Representation, errorDocument("Draft creation failed", r.URL.Path, http.StatusInternalServerError, []string{"failed to persist draft"}))
		return
	}

	http.Redirect(w, r, "/work/"+result.ID, http.StatusSeeOther)
}

func (h *DraftHandler) HandleWorkShow(w http.ResponseWriter, r *http.Request) {
	det := humanRouteRepresentation(r)
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !isUUID(id) {
		if det.Representation == represent.RepresentationHTML {
			http.NotFound(w, r)
			return
		}
		writeHumanDocument(w, http.StatusNotFound, det.Representation, errorDocument("Not found", r.URL.Path, http.StatusNotFound, []string{"work item not found"}))
		return
	}

	data, err := h.FetchCurrentPacket(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			if det.Representation == represent.RepresentationHTML {
				http.NotFound(w, r)
				return
			}
			writeHumanDocument(w, http.StatusNotFound, det.Representation, errorDocument("Not found", r.URL.Path, http.StatusNotFound, []string{"work item not found"}))
			return
		}
		if det.Representation == represent.RepresentationHTML {
			http.Error(w, "load work item", http.StatusInternalServerError)
			return
		}
		writeHumanDocument(w, http.StatusInternalServerError, det.Representation, errorDocument("Backend error", r.URL.Path, http.StatusInternalServerError, []string{"load work item"}))
		return
	}

	if det.Representation != represent.RepresentationHTML {
		writeHumanDocument(w, http.StatusOK, det.Representation, workDetailDocument(data))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.workShowTemplate.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "template render error", http.StatusInternalServerError)
	}
}

func (h *DraftHandler) HandleWorkHistory(w http.ResponseWriter, r *http.Request) {
	det := humanRouteRepresentation(r)
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !isUUID(id) {
		if det.Representation == represent.RepresentationHTML {
			http.NotFound(w, r)
			return
		}
		writeHumanDocument(w, http.StatusNotFound, det.Representation, errorDocument("Not found", r.URL.Path, http.StatusNotFound, []string{"work item not found"}))
		return
	}

	work, err := h.FetchCurrentPacket(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			if det.Representation == represent.RepresentationHTML {
				http.NotFound(w, r)
				return
			}
			writeHumanDocument(w, http.StatusNotFound, det.Representation, errorDocument("Not found", r.URL.Path, http.StatusNotFound, []string{"work item not found"}))
			return
		}
		if det.Representation == represent.RepresentationHTML {
			http.Error(w, "load work history", http.StatusInternalServerError)
			return
		}
		writeHumanDocument(w, http.StatusInternalServerError, det.Representation, errorDocument("Backend error", r.URL.Path, http.StatusInternalServerError, []string{"load work history"}))
		return
	}

	events, err := h.svc.WorkHistory(r.Context(), id)
	if err != nil {
		if det.Representation == represent.RepresentationHTML {
			http.Error(w, "load work history", http.StatusInternalServerError)
			return
		}
		writeHumanDocument(w, http.StatusInternalServerError, det.Representation, errorDocument("Backend error", r.URL.Path, http.StatusInternalServerError, []string{"load work history"}))
		return
	}

	historyEntries := summarizeHistoryEvents(events)

	if det.Representation != represent.RepresentationHTML {
		writeHumanDocument(w, http.StatusOK, det.Representation, historyDocument(work.ID, historyEntries))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.historyTemplate.ExecuteTemplate(w, "layout", historyPageData{
		WorkID:    work.ID,
		WorkTitle: work.Packet.Title,
		Entries:   historyEntries,
	}); err != nil {
		http.Error(w, "template render error", http.StatusInternalServerError)
	}
}

func (h *DraftHandler) CreateDraft(ctx context.Context, input packets.DraftInput) (DraftCreateResult, packets.ValidationErrors, error) {
	if h.svc == nil {
		return DraftCreateResult{}, packets.ValidationErrors{}, errors.New("service is not initialized")
	}

	detail, validation, err := h.svc.CreateWork(ctx, input)
	if !validation.Empty() {
		return DraftCreateResult{}, validation, nil
	}
	if err != nil {
		return DraftCreateResult{}, packets.ValidationErrors{}, err
	}

	return DraftCreateResult{
		ID:           detail.ID,
		Status:       detail.Status,
		Version:      detail.Version,
		ExactHash:    detail.ExactHash,
		QuotientHash: detail.QuotientHash,
		Packet:       detail.Packet,
	}, packets.ValidationErrors{}, nil
}

func (h *DraftHandler) FetchCurrentPacket(ctx context.Context, workID string) (WorkDetail, error) {
	if h.svc == nil {
		return WorkDetail{}, errors.New("service is not initialized")
	}
	return h.svc.GetWork(ctx, workID)
}

func (h *DraftHandler) FetchRecentWork(ctx context.Context, limit int) ([]WorkListRow, error) {
	if h.svc == nil {
		return nil, errors.New("service is not initialized")
	}

	summaries, err := h.svc.ListRecentWork(ctx, limit)
	if err != nil {
		return nil, err
	}

	out := make([]WorkListRow, 0, len(summaries))
	for _, summary := range summaries {
		out = append(out, WorkListRow{
			ID:         summary.ID,
			Title:      summary.Title,
			Kind:       summary.Kind,
			Visibility: summary.Visibility,
			Status:     summary.Status,
			CreatedAt:  summary.CreatedAt,
		})
	}
	return out, nil
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func isUUID(v string) bool {
	return uuidPattern.MatchString(strings.ToLower(v))
}
