package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/shaoyanji/bountystash/internal/packets"
)

type APIHandler struct {
	drafts *DraftHandler
	review *ReviewHandler
}

type apiError struct {
	Error            string                   `json:"error"`
	ValidationErrors packets.ValidationErrors `json:"validation_errors,omitempty"`
}

type exampleSummary struct {
	Slug       string             `json:"slug"`
	Title      string             `json:"title"`
	Kind       packets.Kind       `json:"kind"`
	Visibility packets.Visibility `json:"visibility"`
}

type reviewResponse struct {
	Standard []ReviewRow `json:"standard"`
	Private  []ReviewRow `json:"private"`
}

type draftCreateRequest struct {
	Title              string `json:"title"`
	Kind               string `json:"kind"`
	Scope              string `json:"scope"`
	Deliverables       string `json:"deliverables"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	RewardModel        string `json:"reward_model"`
	Visibility         string `json:"visibility"`
}

func NewAPIHandler(drafts *DraftHandler, review *ReviewHandler) *APIHandler {
	return &APIHandler{
		drafts: drafts,
		review: review,
	}
}

func (h *APIHandler) HandleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *APIHandler) HandleExamplesList(w http.ResponseWriter, _ *http.Request) {
	examples := SeededExamples()
	out := make([]exampleSummary, 0, len(examples))
	for _, ex := range examples {
		out = append(out, exampleSummary{
			Slug:       ex.Slug,
			Title:      ex.Packet.Title,
			Kind:       ex.Packet.Kind,
			Visibility: ex.Packet.Visibility,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"examples": out})
}

func (h *APIHandler) HandleExampleShow(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(chi.URLParam(r, "slug"))
	packet, ok := seededExampleBySlug(slug)
	if !ok {
		writeJSON(w, http.StatusNotFound, apiError{Error: "example not found"})
		return
	}
	writeJSON(w, http.StatusOK, Example{
		Slug:   slug,
		Packet: packet,
	})
}

func (h *APIHandler) HandleReview(w http.ResponseWriter, r *http.Request) {
	queue, err := h.review.FetchQueue(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "load review queue"})
		return
	}
	writeJSON(w, http.StatusOK, reviewResponse{
		Standard: queue.Standard,
		Private:  queue.Private,
	})
}

func (h *APIHandler) HandleWorkShow(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !isUUID(id) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "work item not found"})
		return
	}

	work, err := h.drafts.FetchCurrentPacket(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, apiError{Error: "work item not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "load work item"})
		return
	}
	writeJSON(w, http.StatusOK, work)
}

func (h *APIHandler) HandleWorkList(w http.ResponseWriter, r *http.Request) {
	limit := 25
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "invalid limit"})
			return
		}
		limit = parsed
	}

	work, err := h.drafts.FetchRecentWork(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "load work list"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": work})
}

func (h *APIHandler) HandleDraftCreate(w http.ResponseWriter, r *http.Request) {
	var req draftCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "invalid JSON body"})
		return
	}

	result, validationErrs, err := h.drafts.CreateDraft(r.Context(), packets.DraftInput{
		Title:              req.Title,
		Kind:               req.Kind,
		Scope:              req.Scope,
		Deliverables:       req.Deliverables,
		AcceptanceCriteria: req.AcceptanceCriteria,
		RewardModel:        req.RewardModel,
		Visibility:         req.Visibility,
	})
	if !validationErrs.Empty() {
		writeJSON(w, http.StatusBadRequest, apiError{
			Error:            "validation failed",
			ValidationErrors: validationErrs,
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "failed to persist draft"})
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
