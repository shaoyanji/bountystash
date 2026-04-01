package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shaoyanji/bountystash/internal/packets"
)

const (
	maxListLimit     = 100
	defaultListLimit = 25
	reviewQueueLimit = 100
	eventSource      = "service"
)

type WorkService interface {
	CreateWork(context.Context, packets.DraftInput) (WorkDetail, packets.ValidationErrors, error)
	GetWork(context.Context, string) (WorkDetail, error)
	ListRecentWork(context.Context, int) ([]WorkSummary, error)
	ReviewQueue(context.Context) (ReviewQueueData, error)
	WorkHistory(context.Context, string) ([]Event, error)
	RecentEvents(context.Context, int) ([]Event, error)
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
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

type WorkSummary struct {
	ID         string             `json:"id"`
	Title      string             `json:"title"`
	Kind       packets.Kind       `json:"kind"`
	Visibility packets.Visibility `json:"visibility"`
	Status     string             `json:"status"`
	CreatedAt  time.Time          `json:"created_at"`
}

type ReviewQueueData struct {
	Standard []WorkSummary `json:"standard"`
	Private  []WorkSummary `json:"private"`
}

type Event struct {
	ID            string          `json:"id"`
	EventType     string          `json:"event_type"`
	WorkItemID    string          `json:"work_item_id"`
	WorkVersionID *string         `json:"work_version_id"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     time.Time       `json:"created_at"`
}

type eventType string

const (
	eventIntakeReceived         eventType = "intake_received"
	eventIntakeValidationFailed eventType = "intake_validation_failed"
	eventPacketNormalized       eventType = "packet_normalized"
	eventWorkItemCreated        eventType = "work_item_created"
	eventWorkVersionPersisted   eventType = "work_version_persisted"
	eventWorkItemRead           eventType = "work_item_read"
	eventReviewQueueRead        eventType = "review_queue_read"
)

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// CreateWork executes the authoritative intake path: validation, normalization, persistence, and event logging.
func (s *Service) CreateWork(ctx context.Context, input packets.DraftInput) (WorkDetail, packets.ValidationErrors, error) {
	if err := s.recordEvent(ctx, nil, eventIntakeReceived, nil, nil, map[string]any{
		"title":      input.Title,
		"kind":       input.Kind,
		"visibility": input.Visibility,
		"source":     eventSource,
	}); err != nil {
		return WorkDetail{}, packets.ValidationErrors{}, err
	}

	validation := packets.ValidateDraftInput(input)
	if !validation.Empty() {
		if err := s.recordEvent(ctx, nil, eventIntakeValidationFailed, nil, nil, map[string]any{
			"errors": validation,
			"source": eventSource,
		}); err != nil {
			return WorkDetail{}, validation, err
		}
		return WorkDetail{}, validation, nil
	}

	normalized := packets.NormalizeDraft(input)
	if err := s.recordEvent(ctx, nil, eventPacketNormalized, nil, nil, map[string]any{
		"kind":                normalized.Kind,
		"visibility":          normalized.Visibility,
		"scope":               normalized.Scope,
		"deliverables":        normalized.Deliverables,
		"acceptance_criteria": normalized.AcceptanceCriteria,
		"reward_model":        normalized.RewardModel,
		"source":              eventSource,
	}); err != nil {
		return WorkDetail{}, packets.ValidationErrors{}, err
	}

	packetBytes, err := canonicalJSON(normalized)
	if err != nil {
		return WorkDetail{}, packets.ValidationErrors{}, err
	}
	exactHash := sha256Hex(packetBytes)
	quotientBytes, err := quotientProjectionJSON(normalized)
	if err != nil {
		return WorkDetail{}, packets.ValidationErrors{}, err
	}
	quotientHash := sha256Hex(quotientBytes)
	packetJSON := string(packetBytes)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return WorkDetail{}, packets.ValidationErrors{}, err
	}
	defer tx.Rollback()

	workID, err := newUUID()
	if err != nil {
		return WorkDetail{}, packets.ValidationErrors{}, err
	}
	versionID, err := newUUID()
	if err != nil {
		return WorkDetail{}, packets.ValidationErrors{}, err
	}

	var createdAt time.Time
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO work_items (id, kind, visibility, status, current_version_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`, workID, normalized.Kind, normalized.Visibility, "open", versionID).Scan(&createdAt); err != nil {
		return WorkDetail{}, packets.ValidationErrors{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO work_versions (
			id,
			work_item_id,
			version_number,
			packet,
			exact_hash,
			quotient_hash
		)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6)
	`, versionID, workID, 1, packetJSON, exactHash, quotientHash); err != nil {
		return WorkDetail{}, packets.ValidationErrors{}, err
	}

	if err := s.recordEvent(ctx, tx, eventWorkItemCreated, ptrString(workID), ptrString(versionID), map[string]any{
		"kind":       normalized.Kind,
		"visibility": normalized.Visibility,
		"status":     "open",
		"source":     eventSource,
	}); err != nil {
		return WorkDetail{}, packets.ValidationErrors{}, err
	}

	if err := s.recordEvent(ctx, tx, eventWorkVersionPersisted, ptrString(workID), ptrString(versionID), map[string]any{
		"version_number": 1,
		"exact_hash":     exactHash,
		"quotient_hash":  quotientHash,
		"source":         eventSource,
	}); err != nil {
		return WorkDetail{}, packets.ValidationErrors{}, err
	}

	if err := tx.Commit(); err != nil {
		return WorkDetail{}, packets.ValidationErrors{}, err
	}

	return WorkDetail{
		ID:           workID,
		Status:       "open",
		CreatedAt:    createdAt,
		Version:      1,
		ExactHash:    exactHash,
		QuotientHash: quotientHash,
		Packet:       normalized,
	}, packets.ValidationErrors{}, nil
}

func (s *Service) GetWork(ctx context.Context, workID string) (WorkDetail, error) {
	var (
		kind       string
		visibility string
		packetJSON []byte
		data       WorkDetail
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT wi.kind, wi.visibility, wi.status, wi.created_at, wv.version_number, wv.exact_hash, wv.quotient_hash, wv.packet
		FROM work_items AS wi
		JOIN work_versions AS wv
		  ON wv.work_item_id = wi.id
		 AND wv.id = wi.current_version_id
		WHERE wi.id = $1
	`, workID).Scan(
		&kind,
		&visibility,
		&data.Status,
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
	data.Packet = packet

	if err := s.recordEvent(ctx, nil, eventWorkItemRead, ptrString(workID), nil, map[string]any{
		"status":     data.Status,
		"version":    data.Version,
		"visibility": packet.Visibility,
		"kind":       packet.Kind,
		"source":     eventSource,
	}); err != nil {
		return WorkDetail{}, err
	}

	return data, nil
}

func (s *Service) ListRecentWork(ctx context.Context, limit int) ([]WorkSummary, error) {
	if limit <= 0 || limit > maxListLimit {
		limit = defaultListLimit
	}

	rows, err := s.db.QueryContext(ctx, `
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

	out := make([]WorkSummary, 0, limit)
	for rows.Next() {
		var (
			row       WorkSummary
			rawPacket []byte
		)
		if err := rows.Scan(&row.ID, &row.Kind, &row.Visibility, &row.Status, &row.CreatedAt, &rawPacket); err != nil {
			return nil, err
		}
		var packet packets.NormalizedPacket
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

func (s *Service) ReviewQueue(ctx context.Context) (ReviewQueueData, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT wi.id, wi.kind, wi.visibility, wi.status, wi.created_at, wv.packet
		FROM work_items AS wi
		JOIN work_versions AS wv
		  ON wv.work_item_id = wi.id
		 AND wv.id = wi.current_version_id
		WHERE wi.status IN ('open', 'review')
		ORDER BY wi.created_at DESC
		LIMIT $1
	`, reviewQueueLimit)
	if err != nil {
		return ReviewQueueData{}, err
	}
	defer rows.Close()

	data := ReviewQueueData{
		Standard: make([]WorkSummary, 0, reviewQueueLimit),
		Private:  make([]WorkSummary, 0, reviewQueueLimit),
	}
	for rows.Next() {
		var (
			row       WorkSummary
			rawPacket []byte
		)
		if err := rows.Scan(&row.ID, &row.Kind, &row.Visibility, &row.Status, &row.CreatedAt, &rawPacket); err != nil {
			return ReviewQueueData{}, err
		}
		var packet packets.NormalizedPacket
		if err := json.Unmarshal(rawPacket, &packet); err != nil {
			return ReviewQueueData{}, err
		}
		row.Title = packet.Title
		if row.Kind == packets.KindPrivateSecurity {
			data.Private = append(data.Private, row)
			continue
		}
		data.Standard = append(data.Standard, row)
	}

	if err := rows.Err(); err != nil {
		return ReviewQueueData{}, err
	}

	if err := s.recordEvent(ctx, nil, eventReviewQueueRead, nil, nil, map[string]any{
		"standard_count": len(data.Standard),
		"private_count":  len(data.Private),
		"source":         eventSource,
	}); err != nil {
		return ReviewQueueData{}, err
	}

	return data, nil
}

func (s *Service) WorkHistory(ctx context.Context, workID string) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, event_type, work_item_id, work_version_id, payload, created_at
		FROM backend_events
		WHERE work_item_id = $1
		ORDER BY created_at ASC
	`, workID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Event, 0)
	for rows.Next() {
		var (
			event        Event
			payloadBytes []byte
			versionID    sql.NullString
		)
		if err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.WorkItemID,
			&versionID,
			&payloadBytes,
			&event.CreatedAt,
		); err != nil {
			return nil, err
		}
		if versionID.Valid {
			event.WorkVersionID = ptrString(versionID.String)
		} else {
			event.WorkVersionID = nil
		}
		if payloadBytes != nil {
			event.Payload = append(json.RawMessage(nil), payloadBytes...)
		}
		out = append(out, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) RecentEvents(ctx context.Context, limit int) ([]Event, error) {
	if limit <= 0 || limit > maxListLimit {
		limit = defaultListLimit
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, event_type, work_item_id, work_version_id, payload, created_at
		FROM backend_events
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Event, 0, limit)
	for rows.Next() {
		var (
			event        Event
			payloadBytes []byte
			versionID    sql.NullString
		)
		if err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.WorkItemID,
			&versionID,
			&payloadBytes,
			&event.CreatedAt,
		); err != nil {
			return nil, err
		}
		if versionID.Valid {
			event.WorkVersionID = ptrString(versionID.String)
		} else {
			event.WorkVersionID = nil
		}
		if payloadBytes != nil {
			event.Payload = append(json.RawMessage(nil), payloadBytes...)
		}
		out = append(out, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) recordEvent(ctx context.Context, tx *sql.Tx, typ eventType, workItemID, workVersionID *string, payload map[string]any) error {
	var data []byte
	var err error
	if len(payload) > 0 {
		data, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	var executor execer = s.db
	if tx != nil {
		executor = tx
	}
	if executor == nil {
		return sql.ErrConnDone
	}

	eventID, err := newUUID()
	if err != nil {
		return err
	}

	_, err = executor.ExecContext(ctx, `
		INSERT INTO backend_events (id, event_type, work_item_id, work_version_id, payload)
		VALUES ($1, $2, $3, $4, $5)
	`, eventID, string(typ), workItemID, workVersionID, data)
	return err
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

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
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

func ptrString(v string) *string {
	return &v
}
