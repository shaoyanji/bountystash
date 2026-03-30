package handlers

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"time"

	"github.com/shaoyanji/bountystash/internal/http/represent"
	"github.com/shaoyanji/bountystash/internal/packets"
	"github.com/shaoyanji/bountystash/internal/service"
	"github.com/shaoyanji/bountystash/internal/views"
)

type ReviewHandler struct {
	svc           service.WorkService
	reviewTmpl    *template.Template
	standardLimit int
}

type ReviewRow struct {
	ID         string             `json:"id"`
	Title      string             `json:"title"`
	Kind       packets.Kind       `json:"kind"`
	Visibility packets.Visibility `json:"visibility"`
	Status     string             `json:"status"`
	CreatedAt  time.Time          `json:"created_at"`
}

type ReviewQueueData struct {
	Standard []ReviewRow `json:"standard"`
	Private  []ReviewRow `json:"private"`
}

func NewReviewHandler(svc service.WorkService) (*ReviewHandler, error) {
	if svc == nil {
		return nil, errors.New("service is required")
	}

	tmpl, err := views.Parse("review_queue.tmpl")
	if err != nil {
		return nil, err
	}

	return &ReviewHandler{
		svc:           svc,
		reviewTmpl:    tmpl,
		standardLimit: 100,
	}, nil
}

func (h *ReviewHandler) HandleQueue(w http.ResponseWriter, r *http.Request) {
	det := humanRouteRepresentation(r)
	data, err := h.FetchQueue(r.Context())
	if err != nil {
		if det.Representation == represent.RepresentationHTML {
			http.Error(w, "load review queue", http.StatusInternalServerError)
			return
		}
		writeHumanDocument(w, http.StatusInternalServerError, det.Representation, errorDocument("Backend error", r.URL.Path, http.StatusInternalServerError, []string{"load review queue"}))
		return
	}

	if det.Representation != represent.RepresentationHTML {
		writeHumanDocument(w, http.StatusOK, det.Representation, reviewQueueDocument(data))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.reviewTmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "template render error", http.StatusInternalServerError)
	}
}

func (h *ReviewHandler) FetchQueue(ctx context.Context) (ReviewQueueData, error) {
	if h.svc == nil {
		return ReviewQueueData{}, errors.New("service is not initialized")
	}

	queue, err := h.svc.ReviewQueue(ctx)
	if err != nil {
		return ReviewQueueData{}, err
	}

	return ReviewQueueData{
		Standard: toReviewRows(queue.Standard),
		Private:  toReviewRows(queue.Private),
	}, nil
}

func toReviewRows(summaries []service.WorkSummary) []ReviewRow {
	if len(summaries) == 0 {
		return nil
	}
	out := make([]ReviewRow, 0, len(summaries))
	for _, summary := range summaries {
		out = append(out, ReviewRow{
			ID:         summary.ID,
			Title:      summary.Title,
			Kind:       summary.Kind,
			Visibility: summary.Visibility,
			Status:     summary.Status,
			CreatedAt:  summary.CreatedAt,
		})
	}
	return out
}
