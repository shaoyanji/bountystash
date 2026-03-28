package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/shaoyanji/bountystash/internal/packets"
	"github.com/shaoyanji/bountystash/internal/views"
)

type DraftHandler struct {
	db               *sql.DB
	homeTemplate     *template.Template
	workShowTemplate *template.Template
}

type DraftCreateResult struct {
	ID           string                   `json:"id"`
	Status       string                   `json:"status"`
	Version      int                      `json:"version"`
	ExactHash    string                   `json:"exact_hash"`
	QuotientHash string                   `json:"quotient_hash"`
	Packet       packets.NormalizedPacket `json:"packet"`
}

type WorkDetail struct {
	ID           string                   `json:"id"`
	Status       string                   `json:"status"`
	CreatedAt    time.Time                `json:"created_at"`
	Version      int                      `json:"version"`
	ExactHash    string                   `json:"exact_hash"`
	QuotientHash string                   `json:"quotient_hash"`
	Packet       packets.NormalizedPacket `json:"packet"`
}

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

func NewDraftHandler(db *sql.DB) (*DraftHandler, error) {
	homeTemplate, err := views.Parse("home.tmpl")
	if err != nil {
		return nil, err
	}

	workShowTemplate, err := views.Parse("work_show.tmpl", "work_packet.tmpl")
	if err != nil {
		return nil, err
	}

	return &DraftHandler{
		db:               db,
		homeTemplate:     homeTemplate,
		workShowTemplate: workShowTemplate,
	}, nil
}

func (h *DraftHandler) HandleHome(w http.ResponseWriter, r *http.Request) {
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
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form", http.StatusBadRequest)
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
		h.renderHome(w, homeData{
			Input:  input,
			Errors: validationErrors,
		}, http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "failed to persist draft", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/work/"+result.ID, http.StatusSeeOther)
}

func (h *DraftHandler) HandleWorkShow(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !isUUID(id) {
		http.NotFound(w, r)
		return
	}

	data, err := h.FetchCurrentPacket(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "load work item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.workShowTemplate.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "template render error", http.StatusInternalServerError)
	}
}

func (h *DraftHandler) insertDraft(ctx context.Context, normalized packets.NormalizedPacket) (string, error) {
	workID, err := newUUID()
	if err != nil {
		return "", err
	}
	versionID, err := newUUID()
	if err != nil {
		return "", err
	}

	packetBytes, err := canonicalJSON(normalized)
	if err != nil {
		return "", err
	}

	exactHash := sha256Hex(packetBytes)
	quotientBytes, err := quotientProjectionJSON(normalized)
	if err != nil {
		return "", err
	}
	quotientHash := sha256Hex(quotientBytes)
	packetJSON := string(packetBytes)

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO work_items (id, kind, visibility, status, current_version_id)
		VALUES ($1, $2, $3, $4, $5)
	`, workID, normalized.Kind, normalized.Visibility, "open", versionID)
	if err != nil {
		return "", err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO work_versions (
			id,
			work_item_id,
			version_number,
			packet,
			exact_hash,
			quotient_hash
		)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6)
	`, versionID, workID, 1, packetJSON, exactHash, quotientHash)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return workID, nil
}

func (h *DraftHandler) CreateDraft(ctx context.Context, input packets.DraftInput) (DraftCreateResult, packets.ValidationErrors, error) {
	validationErrors := packets.ValidateDraftInput(input)
	if !validationErrors.Empty() {
		return DraftCreateResult{}, validationErrors, nil
	}

	normalized := packets.NormalizeDraft(input)
	workID, err := h.insertDraft(ctx, normalized)
	if err != nil {
		return DraftCreateResult{}, packets.ValidationErrors{}, err
	}

	work, err := h.FetchCurrentPacket(ctx, workID)
	if err != nil {
		return DraftCreateResult{}, packets.ValidationErrors{}, err
	}

	return DraftCreateResult{
		ID:           work.ID,
		Status:       work.Status,
		Version:      work.Version,
		ExactHash:    work.ExactHash,
		QuotientHash: work.QuotientHash,
		Packet:       work.Packet,
	}, packets.ValidationErrors{}, nil
}

func (h *DraftHandler) FetchCurrentPacket(ctx context.Context, workID string) (WorkDetail, error) {
	var (
		kind         string
		visibility   string
		packetJSON   []byte
		data         WorkDetail
		packetStatus string
	)

	err := h.db.QueryRowContext(ctx, `
		SELECT wi.kind, wi.visibility, wi.status, wi.created_at, wv.version_number, wv.exact_hash, wv.quotient_hash, wv.packet
		FROM work_items AS wi
		JOIN work_versions AS wv
		  ON wv.work_item_id = wi.id
		 AND wv.id = wi.current_version_id
		WHERE wi.id = $1
	`, workID).Scan(
		&kind,
		&visibility,
		&packetStatus,
		&data.CreatedAt,
		&data.Version,
		&data.ExactHash,
		&data.QuotientHash,
		&packetJSON,
	)
	if err != nil {
		return WorkDetail{}, err
	}

	var packet packets.NormalizedPacket
	if err := json.Unmarshal(packetJSON, &packet); err != nil {
		return WorkDetail{}, err
	}
	packet.Kind = packets.Kind(kind)
	packet.Visibility = packets.Visibility(visibility)

	data.ID = workID
	data.Status = packetStatus
	data.Packet = packet
	return data, nil
}

func (h *DraftHandler) FetchRecentWork(ctx context.Context, limit int) ([]WorkListRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	rows, err := h.db.QueryContext(ctx, `
		SELECT wi.id, wi.kind, wi.visibility, wi.status, wi.created_at, wv.packet
		FROM work_items AS wi
		JOIN work_versions AS wv
		  ON wv.work_item_id = wi.id
		 AND wv.id = wi.current_version_id
		ORDER BY wi.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]WorkListRow, 0, limit)
	for rows.Next() {
		var (
			row       WorkListRow
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

// canonicalJSON is hash material for exact packet provenance.
// The packet excludes runtime fields so identical input normalizes to identical bytes.
func canonicalJSON(packet packets.NormalizedPacket) ([]byte, error) {
	return json.Marshal(packet)
}

// quotientProjectionJSON is the deliberate projection used for quotient lineage grouping.
func quotientProjectionJSON(packet packets.NormalizedPacket) ([]byte, error) {
	type quotientProjection struct {
		Kind               packets.Kind       `json:"kind"`
		Scope              []string           `json:"scope"`
		Deliverables       []string           `json:"deliverables"`
		AcceptanceCriteria []string           `json:"acceptance_criteria"`
		RewardModel        string             `json:"reward_model"`
		Visibility         packets.Visibility `json:"visibility"`
	}

	return json.Marshal(quotientProjection{
		Kind:               packet.Kind,
		Scope:              packet.Scope,
		Deliverables:       packet.Deliverables,
		AcceptanceCriteria: packet.AcceptanceCriteria,
		RewardModel:        packet.RewardModel,
		Visibility:         packet.Visibility,
	})
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func isUUID(v string) bool {
	return uuidPattern.MatchString(strings.ToLower(v))
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", errors.New("generate uuid bytes")
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	), nil
}
