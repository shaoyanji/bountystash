package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/shaoyanji/bountystash/internal/http/represent"
	"github.com/shaoyanji/bountystash/internal/packets"
	"github.com/shaoyanji/bountystash/internal/views"
)

type ReviewHandler struct {
	db            *sql.DB
	reviewTmpl    *template.Template
	standardLimit int
}

type ReviewRow struct {
	ID         string
	Title      string
	Kind       packets.Kind
	Visibility packets.Visibility
	Status     string
	CreatedAt  time.Time
}

type ReviewQueueData struct {
	Standard []ReviewRow
	Private  []ReviewRow
}

func NewReviewHandler(db *sql.DB) (*ReviewHandler, error) {
	tmpl, err := views.Parse("review_queue.tmpl")
	if err != nil {
		return nil, err
	}

	return &ReviewHandler{
		db:            db,
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
	rows, err := h.fetchQueueRows(ctx)
	if err != nil {
		return ReviewQueueData{}, err
	}

	data := ReviewQueueData{
		Standard: make([]ReviewRow, 0, len(rows)),
		Private:  make([]ReviewRow, 0, len(rows)),
	}
	for _, row := range rows {
		if row.Kind == packets.KindPrivateSecurity {
			data.Private = append(data.Private, row)
			continue
		}
		data.Standard = append(data.Standard, row)
	}

	return data, nil
}

func (h *ReviewHandler) fetchQueueRows(ctx context.Context) ([]ReviewRow, error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT wi.id, wi.kind, wi.visibility, wi.status, wi.created_at, wv.packet
		FROM work_items AS wi
		JOIN work_versions AS wv
		  ON wv.work_item_id = wi.id
		 AND wv.id = wi.current_version_id
		WHERE wi.status IN ('open', 'review')
		ORDER BY wi.created_at DESC
		LIMIT $1
	`, h.standardLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ReviewRow, 0, h.standardLimit)
	for rows.Next() {
		var (
			row       ReviewRow
			packet    packets.NormalizedPacket
			rawPacket []byte
		)
		if err := rows.Scan(&row.ID, &row.Kind, &row.Visibility, &row.Status, &row.CreatedAt, &rawPacket); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rawPacket, &packet); err != nil {
			return nil, err
		}
		row.Title = packet.Title
		out = append(out, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
