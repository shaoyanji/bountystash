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

	"bountystash/internal/packets"
	"bountystash/internal/views"
	"github.com/go-chi/chi/v5"
)

type DraftHandler struct {
	db               *sql.DB
	workShowTemplate *template.Template
}

type workShowData struct {
	ID     string
	Packet packets.NormalizedPacket
}

func NewDraftHandler(db *sql.DB) (*DraftHandler, error) {
	workShowTemplate, err := template.ParseFS(views.FS, "work_show.tmpl")
	if err != nil {
		return nil, err
	}

	return &DraftHandler{
		db:               db,
		workShowTemplate: workShowTemplate,
	}, nil
}

// RegisterDraftRoutes wires only the draft preview endpoint for milestone intake flow.
func RegisterDraftRoutes(mux *http.ServeMux) {
	_ = mux
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

	normalized := normalizeDraft(input)
	workID, err := h.insertDraft(r.Context(), normalized)
	if err != nil {
		http.Error(w, "failed to persist draft", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/work/"+workID, http.StatusSeeOther)
}

func (h *DraftHandler) HandleWorkShow(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !isUUID(id) {
		http.NotFound(w, r)
		return
	}

	packet, err := h.fetchCurrentPacket(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "load work item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.workShowTemplate.Execute(w, workShowData{
		ID:     id,
		Packet: packet,
	}); err != nil {
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

func (h *DraftHandler) fetchCurrentPacket(ctx context.Context, workID string) (packets.NormalizedPacket, error) {
	var (
		kind       string
		visibility string
		packetJSON []byte
	)

	err := h.db.QueryRowContext(ctx, `
		SELECT wi.kind, wi.visibility, wv.packet
		FROM work_items AS wi
		JOIN work_versions AS wv
		  ON wv.work_item_id = wi.id
		 AND wv.id = wi.current_version_id
		WHERE wi.id = $1
	`, workID).Scan(&kind, &visibility, &packetJSON)
	if err != nil {
		return packets.NormalizedPacket{}, err
	}

	var packet packets.NormalizedPacket
	if err := json.Unmarshal(packetJSON, &packet); err != nil {
		return packets.NormalizedPacket{}, err
	}
	packet.Kind = packets.Kind(kind)
	packet.Visibility = packets.Visibility(visibility)

	return packet, nil
}

func canonicalJSON(packet packets.NormalizedPacket) ([]byte, error) {
	return json.Marshal(packet)
}

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
	normalizedRaw := strings.ReplaceAll(raw, `\n`, "\n")
	rows := strings.Split(normalizedRaw, "\n")
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
