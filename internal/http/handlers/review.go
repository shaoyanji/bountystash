package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/shaoyanji/bountystash/internal/packets"
	"github.com/shaoyanji/bountystash/internal/views"
)

type ReviewHandler struct {
	db            *sql.DB
	reviewTmpl    *template.Template
	standardLimit int
}

type reviewRow struct {
	ID         string
	Title      string
	Kind       packets.Kind
	Visibility packets.Visibility
	Status     string
	CreatedAt  time.Time
}

type reviewQueueData struct {
	Standard []reviewRow
	Private  []reviewRow
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
	rows, err := h.fetchQueueRows(r.Context())
	if err != nil {
		http.Error(w, "load review queue", http.StatusInternalServerError)
		return
	}

	data := reviewQueueData{
		Standard: make([]reviewRow, 0, len(rows)),
		Private:  make([]reviewRow, 0, len(rows)),
	}
	for _, row := range rows {
		if row.Kind == packets.KindPrivateSecurity {
			data.Private = append(data.Private, row)
			continue
		}
		data.Standard = append(data.Standard, row)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.reviewTmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "template render error", http.StatusInternalServerError)
	}
}

func (h *ReviewHandler) fetchQueueRows(ctx context.Context) ([]reviewRow, error) {
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

	out := make([]reviewRow, 0, h.standardLimit)
	for rows.Next() {
		var (
			row       reviewRow
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
