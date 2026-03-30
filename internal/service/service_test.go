package service

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shaoyanji/bountystash/internal/packets"
)

func TestCanonicalJSONDeterministic(t *testing.T) {
	packet := packets.NormalizedPacket{
		Title:              "Deterministic Hashing",
		Kind:               packets.KindBounty,
		Scope:              []string{"a", "b"},
		Deliverables:       []string{"patch"},
		AcceptanceCriteria: []string{"tests pass"},
		RewardModel:        "fixed",
		Visibility:         packets.VisibilityDraft,
	}

	first, err := canonicalJSON(packet)
	if err != nil {
		t.Fatalf("canonicalJSON first: %v", err)
	}
	second, err := canonicalJSON(packet)
	if err != nil {
		t.Fatalf("canonicalJSON second: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("canonical JSON changed between calls:\n%s\n%s", first, second)
	}
}

func TestQuotientProjectionExcludesTitle(t *testing.T) {
	left := packets.NormalizedPacket{
		Title:              "One",
		Kind:               packets.KindRFQ,
		Scope:              []string{"scope"},
		Deliverables:       []string{"deliver"},
		AcceptanceCriteria: []string{"accept"},
		RewardModel:        "quote",
		Visibility:         packets.VisibilityPublic,
	}
	right := left
	right.Title = "Two"

	leftProjection, err := quotientProjectionJSON(left)
	if err != nil {
		t.Fatalf("left projection: %v", err)
	}
	rightProjection, err := quotientProjectionJSON(right)
	if err != nil {
		t.Fatalf("right projection: %v", err)
	}

	if string(leftProjection) != string(rightProjection) {
		t.Fatalf("expected same projection when title differs:\n%s\n%s", leftProjection, rightProjection)
	}
}

func TestServiceCreateWorkValidationErrors(t *testing.T) {
	db, recorder := newMockDB(t)
	defer func() {
		_ = db.Close()
	}()
	recorder.ExpectExec("INSERT INTO backend_events")
	recorder.ExpectExec("INSERT INTO backend_events")

	svc := NewService(db)
	_, validation, err := svc.CreateWork(context.Background(), packets.DraftInput{
		Title:      "",
		Kind:       "invalid",
		Visibility: "wat",
	})
	if err != nil {
		t.Fatalf("CreateWork returned error: %v", err)
	}
	if validation.Empty() {
		t.Fatalf("expected validation errors")
	}

	if err := recorder.ExpectationsWereMet(); err != nil {
		t.Fatalf("recorder expectations: %v", err)
	}
}

func TestServiceCreateWorkRecordsEvents(t *testing.T) {
	db, recorder := newMockDB(t)
	defer func() {
		_ = db.Close()
	}()
	recorder.ExpectExec("INSERT INTO backend_events")
	recorder.ExpectExec("INSERT INTO backend_events")
	recorder.ExpectBegin()
	createdAt := time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)
	recorder.ExpectQuery("INSERT INTO work_items").WillReturnRows(
		[]string{"created_at"},
		[]any{createdAt},
	)
	recorder.ExpectExec("INSERT INTO work_versions")
	recorder.ExpectExec("INSERT INTO backend_events")
	recorder.ExpectExec("INSERT INTO backend_events")
	recorder.ExpectCommit()

	svc := NewService(db)
	detail, validation, err := svc.CreateWork(context.Background(), packets.DraftInput{
		Title:              "title",
		Kind:               "bounty",
		Scope:              "scope",
		Deliverables:       "deliver",
		AcceptanceCriteria: "accept",
		RewardModel:        "fixed",
		Visibility:         "private",
	})
	if err != nil {
		t.Fatalf("CreateWork returned error: %v", err)
	}
	if !validation.Empty() {
		t.Fatalf("expected no validation errors")
	}
	if detail.ID == "" || detail.Version != 1 || detail.Status != "open" {
		t.Fatalf("unexpected detail: %+v", detail)
	}
	if detail.CreatedAt != createdAt {
		t.Fatalf("timestamp mismatch: got %v want %v", detail.CreatedAt, createdAt)
	}

	if err := recorder.ExpectationsWereMet(); err != nil {
		t.Fatalf("recorder expectations: %v", err)
	}
}

func TestServiceReviewQueueRecordsEvent(t *testing.T) {
	db, recorder := newMockDB(t)
	defer func() {
		_ = db.Close()
	}()
	recorder.ExpectQuery("SELECT wi.id").WillReturnRows(
		[]string{"id", "kind", "visibility", "status", "created_at", "packet"},
		[]any{"first", "bounty", "public", "open", time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC), []byte(`{"title":"one"}`)},
		[]any{"second", "private_security", "private", "review", time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC), []byte(`{"title":"two"}`)},
	)
	recorder.ExpectExec("INSERT INTO backend_events")

	svc := NewService(db)
	queue, err := svc.ReviewQueue(context.Background())
	if err != nil {
		t.Fatalf("ReviewQueue error: %v", err)
	}
	if len(queue.Standard) != 1 || len(queue.Private) != 1 {
		t.Fatalf("unexpected queue: %+v", queue)
	}

	if err := recorder.ExpectationsWereMet(); err != nil {
		t.Fatalf("recorder expectations: %v", err)
	}
}

func TestServiceWorkHistoryReturnsEvents(t *testing.T) {
	db, recorder := newMockDB(t)
	defer func() {
		_ = db.Close()
	}()
	recorder.ExpectQuery("SELECT id").WillReturnRows(
		[]string{"id", "event_type", "work_item_id", "work_version_id", "payload", "created_at"},
		[]any{
			"evt-history-1",
			"intake_received",
			"work-1",
			nil,
			[]byte(`{"title":"test"}`),
			time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC),
		},
		[]any{
			"evt-history-2",
			"work_version_persisted",
			"work-1",
			"version-1",
			[]byte(`{"version_number":1}`),
			time.Date(2026, 3, 30, 11, 0, 0, 0, time.UTC),
		},
	)

	svc := NewService(db)
	events, err := svc.WorkHistory(context.Background(), "work-1")
	if err != nil {
		t.Fatalf("WorkHistory error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events len = %d, want 2", len(events))
	}
	if events[0].ID != "evt-history-1" || events[1].ID != "evt-history-2" {
		t.Fatalf("unexpected events: %+v", events)
	}

	if err := recorder.ExpectationsWereMet(); err != nil {
		t.Fatalf("recorder expectations: %v", err)
	}
}

func TestServiceWorkHistoryEmpty(t *testing.T) {
	db, recorder := newMockDB(t)
	defer func() {
		_ = db.Close()
	}()
	recorder.ExpectQuery("SELECT id").WillReturnRows(
		[]string{"id", "event_type", "work_item_id", "work_version_id", "payload", "created_at"},
	)

	svc := NewService(db)
	events, err := svc.WorkHistory(context.Background(), "work-empty")
	if err != nil {
		t.Fatalf("WorkHistory error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected empty events, got %d", len(events))
	}

	if err := recorder.ExpectationsWereMet(); err != nil {
		t.Fatalf("recorder expectations: %v", err)
	}
}

func newMockDB(t *testing.T) (*sql.DB, *mockRecorder) {
	recorder := newMockRecorder(t)
	db, err := sql.Open(mockDriverName, recorder.name)
	if err != nil {
		t.Fatalf("open mock db: %v", err)
	}
	return db, recorder
}

const mockDriverName = "bountystashmock"

func init() {
	sql.Register(mockDriverName, &mockDriver{})
}

type mockDriver struct{}

func (m *mockDriver) Open(name string) (driver.Conn, error) {
	registryMu.Lock()
	recorder, ok := recorderRegistry[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("unknown recorder %s", name)
	}
	return &mockConn{recorder: recorder}, nil
}

type mockConn struct {
	recorder *mockRecorder
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return c.prepareStmt(query)
}

func (c *mockConn) Close() error {
	return nil
}

func (c *mockConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *mockConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if _, err := c.recorder.consume("begin", ""); err != nil {
		return nil, err
	}
	return &mockTx{conn: c}, nil
}

func (c *mockConn) prepareStmt(query string) (driver.Stmt, error) {
	return &mockStmt{conn: c, query: query}, nil
}

func (c *mockConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return c.recorder.exec(query)
}

func (c *mockConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.recorder.query(query)
}

type mockTx struct {
	conn *mockConn
}

func (m *mockTx) Commit() error {
	_, err := m.conn.recorder.consume("commit", "")
	return err
}

func (m *mockTx) Rollback() error {
	return nil
}

type mockStmt struct {
	conn  *mockConn
	query string
}

func (s *mockStmt) Close() error {
	return nil
}

func (s *mockStmt) NumInput() int {
	return -1
}

func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.conn.recorder.exec(s.query)
}

func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.conn.recorder.query(s.query)
}

func (s *mockStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	return s.conn.recorder.exec(s.query)
}

func (s *mockStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	return s.conn.recorder.query(s.query)
}

type mockRecorder struct {
	t            *testing.T
	name         string
	expectations []*expectation
	idx          int
}

func newMockRecorder(t *testing.T) *mockRecorder {
	id := fmt.Sprintf("recorder-%d", atomic.AddUint64(&recorderID, 1))
	rec := &mockRecorder{
		t:    t,
		name: id,
	}
	registryMu.Lock()
	recorderRegistry[id] = rec
	registryMu.Unlock()
	return rec
}

var (
	recorderRegistry = map[string]*mockRecorder{}
	registryMu       sync.Mutex
	recorderID       uint64
)

func (r *mockRecorder) ExpectExec(query string) *execExpectation {
	exp := &expectation{kind: "exec", query: query}
	r.expectations = append(r.expectations, exp)
	return &execExpectation{exp: exp}
}

func (r *mockRecorder) ExpectQuery(query string) *queryExpectation {
	exp := &expectation{kind: "query", query: query}
	r.expectations = append(r.expectations, exp)
	return &queryExpectation{exp: exp}
}

func (r *mockRecorder) ExpectBegin() {
	r.expectations = append(r.expectations, &expectation{kind: "begin"})
}

func (r *mockRecorder) ExpectCommit() {
	r.expectations = append(r.expectations, &expectation{kind: "commit"})
}

func (r *mockRecorder) ExpectationsWereMet() error {
	if r.idx != len(r.expectations) {
		return fmt.Errorf("expectations out of sync: idx=%d total=%d", r.idx, len(r.expectations))
	}
	return nil
}

func (r *mockRecorder) exec(query string) (driver.Result, error) {
	exp, err := r.consume("exec", query)
	if err != nil {
		return nil, err
	}
	if exp.result == nil {
		return mockResult{rowsAffected: 1}, nil
	}
	return exp.result, nil
}

func (r *mockRecorder) query(query string) (driver.Rows, error) {
	exp, err := r.consume("query", query)
	if err != nil {
		return nil, err
	}
	if exp.rows == nil {
		return &mockRows{}, nil
	}
	return exp.rows, nil
}

func (r *mockRecorder) consume(kind, query string) (*expectation, error) {
	if r.idx >= len(r.expectations) {
		return nil, fmt.Errorf("unexpected %s, no expectation left", kind)
	}
	exp := r.expectations[r.idx]
	if exp.kind != kind {
		return nil, fmt.Errorf("expected %s, got %s", exp.kind, kind)
	}
	if exp.query != "" && !strings.Contains(strings.ToLower(query), strings.ToLower(exp.query)) {
		return nil, fmt.Errorf("query %q does not match expectation %q", query, exp.query)
	}
	r.idx++
	return exp, nil
}

type expectation struct {
	kind   string
	query  string
	rows   *mockRows
	result driver.Result
}

type execExpectation struct {
	exp *expectation
}

func (e *execExpectation) WillReturnResult(lastInsertID, rowsAffected int64) {
	e.exp.result = mockResult{lastInsertID: lastInsertID, rowsAffected: rowsAffected}
}

type queryExpectation struct {
	exp *expectation
}

func (q *queryExpectation) WillReturnRows(columns []string, rows ...[]any) {
	values := make([][]driver.Value, len(rows))
	for i, row := range rows {
		values[i] = make([]driver.Value, len(row))
		for j, cell := range row {
			values[i][j] = driver.Value(cell)
		}
	}
	q.exp.rows = &mockRows{columns: columns, rows: values}
}

type mockRows struct {
	columns []string
	rows    [][]driver.Value
	idx     int
}

func (m *mockRows) Columns() []string {
	return m.columns
}

func (m *mockRows) Close() error {
	return nil
}

func (m *mockRows) Next(dest []driver.Value) error {
	if m.idx >= len(m.rows) {
		return io.EOF
	}
	copy(dest, m.rows[m.idx])
	m.idx++
	return nil
}

type mockResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (m mockResult) LastInsertId() (int64, error) {
	return m.lastInsertID, nil
}

func (m mockResult) RowsAffected() (int64, error) {
	return m.rowsAffected, nil
}
