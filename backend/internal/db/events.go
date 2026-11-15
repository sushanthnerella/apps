package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/lib/pq"
	"github.com/purohit-platform/backend/internal/models"
)

var (
	ErrEventNotFound          = errors.New("event not found")
	ErrInvalidEventTransition = errors.New("invalid event status transition")
	ErrLiveEventConflict      = errors.New("another event is already live")
	ErrInvalidEventInput      = errors.New("invalid event input")
	ErrGhatNotFound           = errors.New("ghat not found")
	ErrDuplicateEventData     = errors.New("duplicate event slug or ghat name")
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

const eventColumns = `id, slug, name, name_te, name_hi, river, status, booking_enabled,
	card_issuance_enabled, govt_cards_enabled, starts_at, ends_at, config_version, updated_at`

type rowScanner interface{ Scan(dest ...any) error }

func scanEvent(row rowScanner) (*models.Event, error) {
	var e models.Event
	err := row.Scan(&e.ID, &e.Slug, &e.Name, &e.NameTe, &e.NameHi, &e.River, &e.Status,
		&e.BookingEnabled, &e.CardIssuanceEnabled, &e.GovtCardsEnabled,
		&e.StartsAt, &e.EndsAt, &e.ConfigVersion, &e.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, err
	}
	e.Ghats = []models.Ghat{}
	return &e, nil
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func loadGhats(ctx context.Context, q queryer, eventID string, activeOnly bool) ([]models.Ghat, error) {
	query := `SELECT id, event_id, name, name_te, name_hi, latitude, longitude, sort_order, is_active
		FROM ghats WHERE event_id = $1::uuid`
	if activeOnly {
		query += ` AND is_active = TRUE`
	}
	query += ` ORDER BY sort_order, name`
	rows, err := q.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("load ghats: %w", err)
	}
	defer rows.Close()
	ghats := []models.Ghat{}
	for rows.Next() {
		var g models.Ghat
		if err := rows.Scan(&g.ID, &g.EventID, &g.Name, &g.NameTe, &g.NameHi, &g.Latitude, &g.Longitude, &g.SortOrder, &g.IsActive); err != nil {
			return nil, fmt.Errorf("scan ghat: %w", err)
		}
		ghats = append(ghats, g)
	}
	return ghats, rows.Err()
}

// GetLiveEventConfig returns the ACTIVE or PAUSED event with its active ghats, or nil when no event is live.
func (s *Storage) GetLiveEventConfig(ctx context.Context) (*models.Event, error) {
	e, err := scanEvent(s.DB.QueryRowContext(ctx, `SELECT `+eventColumns+` FROM events WHERE status IN ('ACTIVE','PAUSED')`))
	if errors.Is(err, ErrEventNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if e.Ghats, err = loadGhats(ctx, s.DB, e.ID, true); err != nil {
		return nil, err
	}
	return e, nil
}

// ListEvents returns every event with all of its ghats, newest first.
func (s *Storage) ListEvents(ctx context.Context) ([]models.Event, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+eventColumns+` FROM events ORDER BY starts_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	var events []models.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		events = append(events, *e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// ponytail: one ghat query per event; fine for a handful of events per year.
	for i := range events {
		if events[i].Ghats, err = loadGhats(ctx, s.DB, events[i].ID, false); err != nil {
			return nil, err
		}
	}
	if events == nil {
		events = []models.Event{}
	}
	return events, nil
}

func (s *Storage) GetEvent(ctx context.Context, id string) (*models.Event, error) {
	e, err := scanEvent(s.DB.QueryRowContext(ctx, `SELECT `+eventColumns+` FROM events WHERE id = $1::uuid`, id))
	if err != nil {
		return nil, err
	}
	if e.Ghats, err = loadGhats(ctx, s.DB, e.ID, false); err != nil {
		return nil, err
	}
	return e, nil
}

func writeAudit(ctx context.Context, tx *sql.Tx, actorID, actorRole, action, entityType, entityID string, before, after any) error {
	var b, a []byte
	if before != nil {
		b, _ = json.Marshal(before)
	}
	if after != nil {
		a, _ = json.Marshal(after)
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_log (actor_id, actor_role, action, entity_type, entity_id, before_state, after_state)
		VALUES (NULLIF($1, '')::uuid, $2, $3, $4, $5, $6, $7)`, actorID, actorRole, action, entityType, entityID, nullJSON(b), nullJSON(a))
	if err != nil {
		return fmt.Errorf("write audit: %w", err)
	}
	return nil
}

func nullJSON(b []byte) any {
	if b == nil {
		return nil
	}
	return string(b)
}

func mapEventDBError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		if strings.Contains(pqErr.Constraint, "single_live") {
			return ErrLiveEventConflict
		}
		return ErrDuplicateEventData
	}
	return err
}

// CreateEvent stores a new DRAFT event with booking and issuance off.
func (s *Storage) CreateEvent(ctx context.Context, req models.CreateEventRequest, actorID string) (*models.Event, error) {
	req.Slug = strings.TrimSpace(strings.ToLower(req.Slug))
	req.Name = strings.TrimSpace(req.Name)
	if !slugPattern.MatchString(req.Slug) || len(req.Slug) > 80 || req.Name == "" || req.StartsAt.IsZero() || !req.EndsAt.After(req.StartsAt) {
		return nil, ErrInvalidEventInput
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	e, err := scanEvent(tx.QueryRowContext(ctx, `INSERT INTO events (slug, name, name_te, name_hi, river, starts_at, ends_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+eventColumns,
		req.Slug, req.Name, strings.TrimSpace(req.NameTe), strings.TrimSpace(req.NameHi), strings.TrimSpace(req.River), req.StartsAt, req.EndsAt))
	if err != nil {
		return nil, mapEventDBError(err)
	}
	if err := writeAudit(ctx, tx, actorID, "admin", "event.create", "event", e.ID, nil, e); err != nil {
		return nil, err
	}
	return e, tx.Commit()
}

func transitionAllowed(from, to string) bool {
	for _, s := range models.EventTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// UpdateEvent changes status and switches atomically, bumps config_version and writes an audit row.
// Leaving the live states turns booking and issuance off; it never touches existing bookings.
func (s *Storage) UpdateEvent(ctx context.Context, id string, req models.UpdateEventRequest, actorID string) (*models.Event, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	before, err := scanEvent(tx.QueryRowContext(ctx, `SELECT `+eventColumns+` FROM events WHERE id = $1::uuid FOR UPDATE`, id))
	if err != nil {
		return nil, err
	}
	next := *before
	if req.Status != nil && *req.Status != before.Status {
		if !transitionAllowed(before.Status, *req.Status) {
			return nil, ErrInvalidEventTransition
		}
		next.Status = *req.Status
	}
	if req.BookingEnabled != nil {
		next.BookingEnabled = *req.BookingEnabled
	}
	if req.CardIssuanceEnabled != nil {
		next.CardIssuanceEnabled = *req.CardIssuanceEnabled
	}
	if next.Status != models.EventActive && next.Status != models.EventPaused {
		next.BookingEnabled, next.CardIssuanceEnabled = false, false
	}
	if next.Status == before.Status && next.BookingEnabled == before.BookingEnabled && next.CardIssuanceEnabled == before.CardIssuanceEnabled {
		before.Ghats, err = loadGhats(ctx, tx, id, false)
		return before, err
	}
	after, err := scanEvent(tx.QueryRowContext(ctx, `UPDATE events SET status = $2, booking_enabled = $3, card_issuance_enabled = $4,
		config_version = config_version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid RETURNING `+eventColumns,
		id, next.Status, next.BookingEnabled, next.CardIssuanceEnabled))
	if err != nil {
		return nil, mapEventDBError(err)
	}
	if err := writeAudit(ctx, tx, actorID, "admin", "event.update", "event", id, before, after); err != nil {
		return nil, err
	}
	if after.Ghats, err = loadGhats(ctx, tx, id, false); err != nil {
		return nil, err
	}
	return after, tx.Commit()
}

func validGhat(req models.UpsertGhatRequest) bool {
	if strings.TrimSpace(req.Name) == "" || (req.Latitude == nil) != (req.Longitude == nil) {
		return false
	}
	if req.Latitude != nil && (*req.Latitude < -90 || *req.Latitude > 90 || *req.Longitude < -180 || *req.Longitude > 180) {
		return false
	}
	return true
}

func bumpEventVersion(ctx context.Context, tx *sql.Tx, eventID string) error {
	res, err := tx.ExecContext(ctx, `UPDATE events SET config_version = config_version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid`, eventID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrEventNotFound
	}
	return nil
}

// CreateGhat adds a ghat to an event; ghats are part of app config, so the event version is bumped.
func (s *Storage) CreateGhat(ctx context.Context, eventID string, req models.UpsertGhatRequest, actorID string) (*models.Ghat, error) {
	if !validGhat(req) {
		return nil, ErrInvalidEventInput
	}
	active := req.IsActive == nil || *req.IsActive
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := bumpEventVersion(ctx, tx, eventID); err != nil {
		return nil, err
	}
	var g models.Ghat
	err = tx.QueryRowContext(ctx, `INSERT INTO ghats (event_id, name, name_te, name_hi, latitude, longitude, sort_order, is_active)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, event_id, name, name_te, name_hi, latitude, longitude, sort_order, is_active`,
		eventID, strings.TrimSpace(req.Name), strings.TrimSpace(req.NameTe), strings.TrimSpace(req.NameHi), req.Latitude, req.Longitude, req.SortOrder, active).
		Scan(&g.ID, &g.EventID, &g.Name, &g.NameTe, &g.NameHi, &g.Latitude, &g.Longitude, &g.SortOrder, &g.IsActive)
	if err != nil {
		return nil, mapEventDBError(err)
	}
	if err := writeAudit(ctx, tx, actorID, "admin", "ghat.create", "ghat", g.ID, nil, g); err != nil {
		return nil, err
	}
	return &g, tx.Commit()
}

// UpdateGhat replaces a ghat's fields; deactivating hides it from apps without deleting history.
func (s *Storage) UpdateGhat(ctx context.Context, eventID, ghatID string, req models.UpsertGhatRequest, actorID string) (*models.Ghat, error) {
	if !validGhat(req) {
		return nil, ErrInvalidEventInput
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var before models.Ghat
	err = tx.QueryRowContext(ctx, `SELECT id, event_id, name, name_te, name_hi, latitude, longitude, sort_order, is_active
		FROM ghats WHERE id = $1::uuid AND event_id = $2::uuid FOR UPDATE`, ghatID, eventID).
		Scan(&before.ID, &before.EventID, &before.Name, &before.NameTe, &before.NameHi, &before.Latitude, &before.Longitude, &before.SortOrder, &before.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrGhatNotFound
	}
	if err != nil {
		return nil, err
	}
	active := before.IsActive
	if req.IsActive != nil {
		active = *req.IsActive
	}
	var g models.Ghat
	err = tx.QueryRowContext(ctx, `UPDATE ghats SET name = $2, name_te = $3, name_hi = $4, latitude = $5, longitude = $6,
		sort_order = $7, is_active = $8, updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid
		RETURNING id, event_id, name, name_te, name_hi, latitude, longitude, sort_order, is_active`,
		ghatID, strings.TrimSpace(req.Name), strings.TrimSpace(req.NameTe), strings.TrimSpace(req.NameHi), req.Latitude, req.Longitude, req.SortOrder, active).
		Scan(&g.ID, &g.EventID, &g.Name, &g.NameTe, &g.NameHi, &g.Latitude, &g.Longitude, &g.SortOrder, &g.IsActive)
	if err != nil {
		return nil, mapEventDBError(err)
	}
	if err := bumpEventVersion(ctx, tx, eventID); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, tx, actorID, "admin", "ghat.update", "ghat", g.ID, before, g); err != nil {
		return nil, err
	}
	return &g, tx.Commit()
}

// ListEventAudit returns recent audit rows for an event and its ghats.
func (s *Storage) ListEventAudit(ctx context.Context, eventID string, limit int) ([]models.AuditEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT a.id, a.actor_id::text, a.actor_role, a.action, a.entity_type, a.entity_id,
		a.before_state, a.after_state, a.created_at
		FROM audit_log a
		WHERE (a.entity_type = 'event' AND a.entity_id = $1)
		   OR (a.entity_type = 'ghat' AND a.entity_id IN (SELECT id::text FROM ghats WHERE event_id = $1::uuid))
		ORDER BY a.created_at DESC, a.id DESC LIMIT $2`, eventID, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	defer rows.Close()
	entries := []models.AuditEntry{}
	for rows.Next() {
		var e models.AuditEntry
		if err := rows.Scan(&e.ID, &e.ActorID, &e.ActorRole, &e.Action, &e.EntityType, &e.EntityID, &e.BeforeState, &e.AfterState, &e.CreatedAt); err != nil {
			return nil, err
		}
		if e.BeforeState != nil {
			e.Before = json.RawMessage(e.BeforeState)
		}
		if e.AfterState != nil {
			e.After = json.RawMessage(e.AfterState)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
