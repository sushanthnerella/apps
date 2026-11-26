package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/purohit-platform/backend/internal/lifecycle"
	"github.com/purohit-platform/backend/internal/models"
)

var (
	ErrEventNotOpen         = errors.New("event is not accepting this action")
	ErrEventBookingClosed   = errors.New("event bookings are closed")
	ErrParticipationState   = errors.New("participation is not in a state that allows this")
	ErrNotApprovedForEvent  = errors.New("priest is not approved for this event")
	ErrPriestUnavailable    = errors.New("priest is not available at this ghat right now")
	ErrServiceUnavailable   = errors.New("service is not offered by this priest")
	ErrInvalidSchedule      = errors.New("scheduled time is outside the allowed window")
	ErrRequestExpired       = errors.New("request expired")
	ErrPriestBusy           = errors.New("priest already has a ritual in progress")
	ErrBookingCodeRequired  = errors.New("booking code required to start the ritual")
	errWrongBookingCode     = errors.New("wrong booking code")
	ErrParticipationMissing = errors.New("participation not found")
)

// Location older than this drops a priest out of "available now" (PRD v2 LOC-004).
const staleLocationAfter = 15 * time.Minute

const (
	immediateRequestTTL = 5 * time.Minute
	scheduledRequestTTL = 2 * time.Hour
	minScheduleLead     = 30 * time.Minute
)

// ---------- Ritual templates (CMS) ----------

func (s *Storage) ListRituals(ctx context.Context, eventID string, activeOnly bool) ([]models.EventRitual, error) {
	q := `SELECT id, event_id, name, name_te, name_hi, duration_minutes, sort_order, is_active FROM event_rituals WHERE event_id = $1::uuid`
	if activeOnly {
		q += ` AND is_active = TRUE`
	}
	rows, err := s.DB.QueryContext(ctx, q+` ORDER BY sort_order, name`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.EventRitual{}
	for rows.Next() {
		var r models.EventRitual
		if err := rows.Scan(&r.ID, &r.EventID, &r.Name, &r.NameTe, &r.NameHi, &r.DurationMinutes, &r.SortOrder, &r.IsActive); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func validRitual(r models.UpsertRitualRequest) bool {
	return strings.TrimSpace(r.Name) != "" && r.DurationMinutes >= 5 && r.DurationMinutes <= 480
}

func (s *Storage) UpsertRitual(ctx context.Context, eventID, ritualID string, req models.UpsertRitualRequest, actorID string) (*models.EventRitual, error) {
	if !validRitual(req) {
		return nil, ErrInvalidEventInput
	}
	active := req.IsActive == nil || *req.IsActive
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var r models.EventRitual
	cols := `id, event_id, name, name_te, name_hi, duration_minutes, sort_order, is_active`
	args := []any{eventID, strings.TrimSpace(req.Name), strings.TrimSpace(req.NameTe), strings.TrimSpace(req.NameHi), req.DurationMinutes, req.SortOrder, active}
	var row *sql.Row
	action := "ritual.create"
	if ritualID == "" {
		row = tx.QueryRowContext(ctx, `INSERT INTO event_rituals (event_id, name, name_te, name_hi, duration_minutes, sort_order, is_active)
			VALUES ($1::uuid, $2, $3, $4, $5, $6, $7) RETURNING `+cols, args...)
	} else {
		action = "ritual.update"
		row = tx.QueryRowContext(ctx, `UPDATE event_rituals SET name=$2, name_te=$3, name_hi=$4, duration_minutes=$5, sort_order=$6, is_active=$7
			WHERE event_id=$1::uuid AND id=$8::uuid RETURNING `+cols, append(args, ritualID)...)
	}
	err = row.Scan(&r.ID, &r.EventID, &r.Name, &r.NameTe, &r.NameHi, &r.DurationMinutes, &r.SortOrder, &r.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return nil, ErrEventNotFound
		}
		return nil, mapEventDBError(err)
	}
	if err := writeAudit(ctx, tx, actorID, "admin", action, "ritual", r.ID, nil, r); err != nil {
		return nil, err
	}
	return &r, tx.Commit()
}

// ---------- Participation (priest) ----------

func newCardCode() (string, error) {
	b := make([]byte, 15)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "PV-" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)), nil
}

// GetParticipation returns the priest's state for an event. An approved priest gets a card
// lazily once the CMS card-issuance switch is on.
func (s *Storage) GetParticipation(ctx context.Context, eventID, priestID string) (*models.Participation, error) {
	ev, err := s.GetEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	p := &models.Participation{EventID: eventID, Status: "NOT_APPLIED"}
	var submitted time.Time
	err = s.DB.QueryRowContext(ctx, `SELECT status, reviewer_reason, submitted_at FROM event_participations WHERE event_id=$1::uuid AND priest_id=$2::uuid`,
		eventID, priestID).Scan(&p.Status, &p.ReviewerReason, &submitted)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		p.SubmittedAt = &submitted
	}
	if p.Status == models.ParticipationApproved && ev.CardIssuanceEnabled && (ev.Status == models.EventActive || ev.Status == models.EventPaused || ev.Status == models.EventScheduled) {
		code, err := newCardCode()
		if err != nil {
			return nil, err
		}
		if _, err := s.DB.ExecContext(ctx, `INSERT INTO event_credentials (card_code, event_id, priest_id) VALUES ($1, $2::uuid, $3::uuid)
			ON CONFLICT (event_id, priest_id) DO NOTHING`, code, eventID, priestID); err != nil {
			return nil, err
		}
	}
	var card models.EventCard
	err = s.DB.QueryRowContext(ctx, `SELECT card_code, issuer, status, issued_at FROM event_credentials WHERE event_id=$1::uuid AND priest_id=$2::uuid`,
		eventID, priestID).Scan(&card.CardCode, &card.Issuer, &card.Status, &card.IssuedAt)
	if err == nil {
		card.EventName, card.StartsAt, card.EndsAt = ev.Name, ev.StartsAt, ev.EndsAt
		p.Card = &card
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if p.Rituals, err = s.ListRituals(ctx, eventID, true); err != nil {
		return nil, err
	}
	if p.Services, err = s.priestServices(ctx, eventID, []string{priestID}); err != nil {
		return nil, err
	}
	var d models.DutyState
	err = s.DB.QueryRowContext(ctx, `SELECT ghat_id::text, duty_status, started_at FROM event_duty WHERE event_id=$1::uuid AND priest_id=$2::uuid`,
		eventID, priestID).Scan(&d.GhatID, &d.DutyStatus, &d.StartedAt)
	if err == nil {
		p.Duty = &d
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return p, nil
}

// SubmitParticipation applies (or re-applies after "changes requested").
func (s *Storage) SubmitParticipation(ctx context.Context, eventID, priestID string) error {
	var status string
	if err := s.DB.QueryRowContext(ctx, `SELECT status FROM events WHERE id=$1::uuid`, eventID).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEventNotFound
		}
		return err
	}
	if status != models.EventScheduled && status != models.EventActive && status != models.EventPaused {
		return ErrEventNotOpen
	}
	res, err := s.DB.ExecContext(ctx, `INSERT INTO event_participations (event_id, priest_id) VALUES ($1::uuid, $2::uuid)
		ON CONFLICT (event_id, priest_id) DO UPDATE SET status='SUBMITTED', reviewer_reason='', submitted_at=CURRENT_TIMESTAMP
		WHERE event_participations.status = 'CHANGES_REQUESTED'`, eventID, priestID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrParticipationState
	}
	return nil
}

func (s *Storage) ListParticipations(ctx context.Context, eventID string) ([]models.ParticipationRow, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT ep.priest_id::text, COALESCE(p.full_name,''), COALESCE(p.sampradaya,''), COALESCE(p.experience_years,0),
		ep.status, ep.reviewer_reason, ep.submitted_at, ep.reviewed_at,
		(SELECT count(*) FROM event_priest_services s WHERE s.priest_id = ep.priest_id AND s.event_id = ep.event_id AND s.is_active)
		FROM event_participations ep JOIN priests p ON p.id = ep.priest_id
		WHERE ep.event_id = $1::uuid ORDER BY (ep.status = 'SUBMITTED') DESC, ep.submitted_at`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.ParticipationRow{}
	for rows.Next() {
		var r models.ParticipationRow
		if err := rows.Scan(&r.PriestID, &r.PriestName, &r.Sampradaya, &r.ExperienceYears, &r.Status, &r.ReviewerReason, &r.SubmittedAt, &r.ReviewedAt, &r.ServicesCount); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ReviewParticipation is the CMS decision. Suspension also suspends the card and ends duty.
func (s *Storage) ReviewParticipation(ctx context.Context, eventID, priestID string, req models.ReviewParticipationRequest, actorID string) error {
	reason := strings.TrimSpace(req.Reason)
	switch req.Status {
	case models.ParticipationApproved:
	case models.ParticipationChangesRequested, models.ParticipationRejected, models.ParticipationSuspended:
		if reason == "" {
			return ErrInvalidEventInput
		}
	default:
		return ErrInvalidEventInput
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var before string
	err = tx.QueryRowContext(ctx, `SELECT status FROM event_participations WHERE event_id=$1::uuid AND priest_id=$2::uuid FOR UPDATE`, eventID, priestID).Scan(&before)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrParticipationMissing
	}
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE event_participations SET status=$3, reviewer_reason=$4, reviewed_at=CURRENT_TIMESTAMP, reviewed_by=NULLIF($5,'')::uuid
		WHERE event_id=$1::uuid AND priest_id=$2::uuid`, eventID, priestID, req.Status, reason, actorID); err != nil {
		return err
	}
	cardStatus := "SUSPENDED"
	if req.Status == models.ParticipationApproved {
		cardStatus = "ACTIVE"
	} else if req.Status == models.ParticipationRejected {
		cardStatus = "REVOKED"
	}
	if _, err := tx.ExecContext(ctx, `UPDATE event_credentials SET status=$3, updated_at=CURRENT_TIMESTAMP WHERE event_id=$1::uuid AND priest_id=$2::uuid`,
		eventID, priestID, cardStatus); err != nil {
		return err
	}
	if req.Status != models.ParticipationApproved {
		if _, err := tx.ExecContext(ctx, `DELETE FROM event_duty WHERE event_id=$1::uuid AND priest_id=$2::uuid`, eventID, priestID); err != nil {
			return err
		}
	}
	if err := writeAudit(ctx, tx, actorID, "admin", "participation.review", "participation", eventID+":"+priestID,
		map[string]string{"status": before}, map[string]string{"status": req.Status, "reason": reason}); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------- Priest services ----------

func (s *Storage) priestServices(ctx context.Context, eventID string, priestIDs []string) ([]models.PriestService, error) {
	m, err := s.priestServicesByPriest(ctx, eventID, priestIDs)
	if err != nil {
		return nil, err
	}
	out := []models.PriestService{}
	for _, id := range priestIDs {
		out = append(out, m[id]...)
	}
	return out, nil
}

func (s *Storage) priestServicesByPriest(ctx context.Context, eventID string, priestIDs []string) (map[string][]models.PriestService, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT s.priest_id::text, s.id, r.id, r.name, r.name_te, r.name_hi, r.duration_minutes, s.price_minor
		FROM event_priest_services s JOIN event_rituals r ON r.id = s.ritual_id
		WHERE s.event_id=$1::uuid AND s.priest_id::text = ANY($2) AND s.is_active AND r.is_active
		ORDER BY r.sort_order, r.name`, eventID, pq.Array(priestIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]models.PriestService{}
	for rows.Next() {
		var pid string
		var ps models.PriestService
		if err := rows.Scan(&pid, &ps.ID, &ps.RitualID, &ps.RitualName, &ps.RitualNameTe, &ps.RitualNameHi, &ps.DurationMinutes, &ps.PriceMinor); err != nil {
			return nil, err
		}
		out[pid] = append(out[pid], ps)
	}
	return out, rows.Err()
}

// SetPriestServices replaces the priest's price list. Existing bookings keep their frozen snapshot.
func (s *Storage) SetPriestServices(ctx context.Context, eventID, priestID string, req models.SetServicesRequest) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM event_participations WHERE event_id=$1::uuid AND priest_id=$2::uuid`, eventID, priestID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && (status == models.ParticipationRejected || status == models.ParticipationSuspended)) {
		return ErrParticipationState
	}
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE event_priest_services SET is_active=FALSE, updated_at=CURRENT_TIMESTAMP WHERE event_id=$1::uuid AND priest_id=$2::uuid`, eventID, priestID); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, svc := range req.Services {
		if seen[svc.RitualID] || svc.PriceMinor <= 0 || svc.PriceMinor > 10000000 {
			return ErrInvalidEventInput
		}
		seen[svc.RitualID] = true
		res, err := tx.ExecContext(ctx, `INSERT INTO event_priest_services (event_id, priest_id, ritual_id, price_minor)
			SELECT $1::uuid, $2::uuid, r.id, $4 FROM event_rituals r WHERE r.id::text = $3 AND r.event_id = $1::uuid AND r.is_active
			ON CONFLICT (priest_id, ritual_id) DO UPDATE SET price_minor = EXCLUDED.price_minor, is_active = TRUE, updated_at = CURRENT_TIMESTAMP`,
			eventID, priestID, svc.RitualID, svc.PriceMinor)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrInvalidEventInput
		}
	}
	return tx.Commit()
}

// ---------- Duty at a ghat ----------

// SetDuty starts or updates duty. It also marks the priest online so the existing location ingest accepts points.
func (s *Storage) SetDuty(ctx context.Context, eventID, priestID string, req models.DutyRequest) (*models.DutyState, error) {
	if req.DutyStatus == "" {
		req.DutyStatus = "AVAILABLE"
	}
	if req.DutyStatus != "AVAILABLE" && req.DutyStatus != "PERFORMING" && req.DutyStatus != "BREAK" {
		return nil, ErrInvalidEventInput
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var evStatus, partStatus string
	err = tx.QueryRowContext(ctx, `SELECT e.status, COALESCE(ep.status,'') FROM events e
		LEFT JOIN event_participations ep ON ep.event_id = e.id AND ep.priest_id = $2::uuid WHERE e.id = $1::uuid`, eventID, priestID).Scan(&evStatus, &partStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, err
	}
	if evStatus != models.EventActive && evStatus != models.EventPaused {
		return nil, ErrEventNotOpen
	}
	if partStatus != models.ParticipationApproved {
		return nil, ErrNotApprovedForEvent
	}
	var ok bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM ghats WHERE id::text=$1 AND event_id=$2::uuid AND is_active)`, req.GhatID, eventID).Scan(&ok); err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrGhatNotFound
	}
	var d models.DutyState
	err = tx.QueryRowContext(ctx, `INSERT INTO event_duty (event_id, priest_id, ghat_id, duty_status) VALUES ($1::uuid, $2::uuid, $3::uuid, $4)
		ON CONFLICT (event_id, priest_id) DO UPDATE SET ghat_id = EXCLUDED.ghat_id, duty_status = EXCLUDED.duty_status, updated_at = CURRENT_TIMESTAMP
		RETURNING ghat_id::text, duty_status, started_at`, eventID, priestID, req.GhatID, req.DutyStatus).Scan(&d.GhatID, &d.DutyStatus, &d.StartedAt)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE priests SET is_online = TRUE WHERE id = $1::uuid`, priestID); err != nil {
		return nil, err
	}
	return &d, tx.Commit()
}

// EndDuty stops event duty. Active bookings are never touched (PRD v2 DUT-002).
func (s *Storage) EndDuty(ctx context.Context, eventID, priestID string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM event_duty WHERE event_id=$1::uuid AND priest_id=$2::uuid`, eventID, priestID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE priests SET is_online = FALSE WHERE id = $1::uuid`, priestID); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------- Discovery ----------

const availablePriestFilter = `d.event_id = $1::uuid AND d.duty_status = 'AVAILABLE'
	AND ep.status = 'APPROVED' AND p.is_online = TRUE
	AND p.location_captured_at IS NOT NULL AND p.location_captured_at > NOW() - make_interval(secs => $2)`

// ListGhatPriests returns approved, on-duty, recently located priests at a ghat. Never returns coordinates.
func (s *Storage) ListGhatPriests(ctx context.Context, eventID, ghatID string) ([]models.GhatPriest, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT p.id::text, COALESCE(p.full_name,''), COALESCE(p.rating_avg,0), COALESCE(p.total_completed_poojas,0),
		d.duty_status, EXTRACT(EPOCH FROM (NOW() - p.location_captured_at))::bigint
		FROM event_duty d
		JOIN event_participations ep ON ep.event_id = d.event_id AND ep.priest_id = d.priest_id
		JOIN priests p ON p.id = d.priest_id
		WHERE `+availablePriestFilter+` AND d.ghat_id = $3::uuid
		ORDER BY p.location_captured_at DESC LIMIT 100`, eventID, staleLocationAfter.Seconds(), ghatID)
	if err != nil {
		return nil, err
	}
	var out []models.GhatPriest
	var ids []string
	for rows.Next() {
		var g models.GhatPriest
		if err := rows.Scan(&g.PriestID, &g.FullName, &g.RatingAvg, &g.CompletedPoojas, &g.DutyStatus, &g.LocationAgeSec); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, g)
		ids = append(ids, g.PriestID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	services, err := s.priestServicesByPriest(ctx, eventID, ids)
	if err != nil {
		return nil, err
	}
	result := []models.GhatPriest{}
	for _, g := range out {
		if g.Services = services[g.PriestID]; len(g.Services) > 0 { // a priest with no prices cannot be booked
			result = append(result, g)
		}
	}
	return result, nil
}

func (s *Storage) GhatAvailability(ctx context.Context, eventID string) ([]models.GhatAvailability, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT d.ghat_id::text, count(*) FROM event_duty d
		JOIN event_participations ep ON ep.event_id = d.event_id AND ep.priest_id = d.priest_id
		JOIN priests p ON p.id = d.priest_id
		WHERE `+availablePriestFilter+`
		  AND EXISTS (SELECT 1 FROM event_priest_services s WHERE s.priest_id = d.priest_id AND s.event_id = d.event_id AND s.is_active)
		GROUP BY d.ghat_id`, eventID, staleLocationAfter.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.GhatAvailability{}
	for rows.Next() {
		var a models.GhatAvailability
		if err := rows.Scan(&a.GhatID, &a.AvailableCount); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ---------- Event bookings ----------

// CreateEventBooking validates everything server-side and freezes the agreed terms in a snapshot.
func (s *Storage) CreateEventBooking(ctx context.Context, userID, eventID string, req models.CreateEventBookingRequest, now time.Time) (*models.EventBookingResponse, error) {
	if req.When != "NOW" && req.When != "SCHEDULED" {
		return nil, ErrInvalidEventInput
	}
	if req.FamilySize < 1 {
		req.FamilySize = 1
	}
	if req.FamilySize > 50 {
		return nil, ErrInvalidEventInput
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var evName, evStatus string
	var bookingOn bool
	var startsAt, endsAt time.Time
	err = tx.QueryRowContext(ctx, `SELECT name, status, booking_enabled, starts_at, ends_at FROM events WHERE id=$1::uuid FOR SHARE`, eventID).
		Scan(&evName, &evStatus, &bookingOn, &startsAt, &endsAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, err
	}
	if evStatus != models.EventActive || !bookingOn {
		return nil, ErrEventBookingClosed
	}

	var ghatName string
	var lat, lng sql.NullFloat64
	err = tx.QueryRowContext(ctx, `SELECT name, latitude, longitude FROM ghats WHERE id::text=$1 AND event_id=$2::uuid AND is_active`, req.GhatID, eventID).Scan(&ghatName, &lat, &lng)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrGhatNotFound
	}
	if err != nil {
		return nil, err
	}

	var ritualName string
	var duration int
	var price int64
	var priestName string
	err = tx.QueryRowContext(ctx, `SELECT r.name, r.duration_minutes, s.price_minor, COALESCE(p.full_name,'')
		FROM event_priest_services s
		JOIN event_rituals r ON r.id = s.ritual_id
		JOIN event_participations ep ON ep.event_id = s.event_id AND ep.priest_id = s.priest_id
		JOIN priests p ON p.id = s.priest_id
		WHERE s.id::text = $1 AND s.priest_id::text = $2 AND s.event_id = $3::uuid AND s.is_active AND r.is_active AND ep.status = 'APPROVED'`,
		req.ServiceID, req.PriestID, eventID).Scan(&ritualName, &duration, &price, &priestName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrServiceUnavailable
	}
	if err != nil {
		return nil, err
	}

	bookingType := models.BookingTypeEventNow
	expires := now.Add(immediateRequestTTL)
	scheduled := now
	if req.When == "NOW" {
		var available bool
		err = tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM event_duty d
			JOIN event_participations ep ON ep.event_id = d.event_id AND ep.priest_id = d.priest_id
			JOIN priests p ON p.id = d.priest_id
			WHERE `+availablePriestFilter+` AND d.ghat_id::text = $3 AND d.priest_id::text = $4)`,
			eventID, staleLocationAfter.Seconds(), req.GhatID, req.PriestID).Scan(&available)
		if err != nil {
			return nil, err
		}
		if !available {
			return nil, ErrPriestUnavailable
		}
		// The priest app shows no new alerts while a ritual is under way or due within
		// the hour (see priest app dispatch poll), so a "Now" request would just sit
		// until expiry. Refuse it up front. An accepted ritual scheduled later does not count.
		var busy bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM bookings WHERE priest_id::text = $1
			AND (status IN ('IN_TRANSIT','ARRIVED','POOJA_STARTED')
				OR (status = 'ACCEPTED' AND (scheduled_at IS NULL OR scheduled_at < now() + make_interval(mins => $2 + 60)))))`,
			req.PriestID, duration).Scan(&busy); err != nil {
			return nil, err
		}
		if busy {
			return nil, ErrPriestBusy
		}
	} else {
		if req.ScheduledAt == nil || req.ScheduledAt.Before(now.Add(minScheduleLead)) || req.ScheduledAt.Before(startsAt) || req.ScheduledAt.After(endsAt) {
			return nil, ErrInvalidSchedule
		}
		bookingType = models.BookingTypeEventScheduled
		scheduled = *req.ScheduledAt
		expires = now.Add(scheduledRequestTTL)
		if cutoff := scheduled.Add(-minScheduleLead); cutoff.Before(expires) {
			expires = cutoff
		}
		// Refuse up front when the priest already has an accepted ritual at that time,
		// instead of letting the request wait until it expires.
		var clash bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM bookings WHERE priest_id::text = $1
			AND event_id IS NOT NULL AND status IN ('ACCEPTED','IN_TRANSIT','ARRIVED','POOJA_STARTED')
			AND scheduled_at < $2::timestamptz + make_interval(mins => $3)
			AND $2::timestamptz < scheduled_at + make_interval(mins => (agreement_snapshot->>'duration_minutes')::int))`,
			req.PriestID, scheduled, duration).Scan(&clash); err != nil {
			return nil, err
		}
		if clash {
			return nil, ErrPriestBusy
		}
	}

	codeN, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return nil, err
	}
	code := fmt.Sprintf("%04d", codeN.Int64())
	snapshot, _ := json.Marshal(map[string]any{
		"event": evName, "ghat": ghatName, "ritual": ritualName, "duration_minutes": duration,
		"price_minor": price, "currency": "INR", "priest": priestName, "family_size": req.FamilySize,
		"when": req.When, "scheduled_at": scheduled.UTC().Format(time.RFC3339), "payment": "PAY_AT_GHAT",
	})
	ist := time.FixedZone("IST", 5*3600+1800)
	var bookingID string
	err = tx.QueryRowContext(ctx, `INSERT INTO bookings (user_id, priest_id, booking_type, status, escrow_status,
			total_amount, total_amount_minor, completion_otp, event_address, latitude, longitude,
			scheduled_date, slot_time, sankalpam_purpose,
			event_id, ghat_id, event_service_id, agreement_snapshot, request_expires_at, scheduled_at, scheduled_muhurtham)
		VALUES ($1::uuid, $2::uuid, $3, 'REQUESTED', 'PAY_AT_GHAT', $4, $5, $6, $7, $8, $9, $10, $11, $12,
			$13::uuid, $14::uuid, $15::uuid, $16, $17, $18, $18) RETURNING id::text`,
		userID, req.PriestID, bookingType, float64(price)/100, price, code, ghatName, lat.Float64, lng.Float64,
		scheduled.In(ist).Format("02 Jan 2006"), scheduled.In(ist).Format("03:04 PM"), ritualName,
		eventID, req.GhatID, req.ServiceID, string(snapshot), expires, scheduled).Scan(&bookingID)
	if err != nil {
		return nil, err
	}
	roomID, err := s.GetOrCreateChatRoomTx(ctx, tx, bookingID, userID, req.PriestID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &models.EventBookingResponse{
		BookingID: bookingID, Status: "REQUESTED", BookingCode: code, TotalAmountMinor: price,
		RequestExpiresAt: expires, ChatRoomID: roomID,
	}, nil
}

// NormalRequestTTL is how long a home-visit request waits for the priest before it expires.
const NormalRequestTTL = 3 * time.Minute

// ExpireEventRequests cancels requests nobody accepted in time: event requests by
// request_expires_at, home-visit requests after NormalRequestTTL. Slots held by
// expired home-visit requests are released. Returns rows changed.
func (s *Storage) ExpireEventRequests(ctx context.Context) (int64, error) {
	res, err := s.DB.QueryContext(ctx, `WITH expired AS (
			UPDATE bookings SET status='CANCELLED', cancellation_reason='request_expired', cancelled_by='system'
			WHERE status='REQUESTED' AND (
				(event_id IS NOT NULL AND request_expires_at < NOW())
				OR (event_id IS NULL AND COALESCE(request_expires_at, created_at + make_interval(secs => $1)) < NOW()))
			RETURNING id
		), released AS (
			UPDATE priest_availability_slots SET is_booked = false, booked_booking_id = NULL
			WHERE booked_booking_id IN (SELECT id FROM expired)
		)
		SELECT COUNT(*) FROM expired`, NormalRequestTTL.Seconds())
	if err != nil {
		return 0, err
	}
	defer res.Close()
	var n int64
	if res.Next() {
		if err := res.Scan(&n); err != nil {
			return 0, err
		}
	}
	return n, res.Err()
}

// checkEventTransition adds event rules on top of the shared booking state machine.
func checkEventTransition(ctx context.Context, tx *sql.Tx, req models.BookingStatusUpdateRequest, priestID, bookingType, expectedCode string, attempts int, expiresAt sql.NullTime) error {
	switch req.Status {
	case "ACCEPTED":
		if expiresAt.Valid && time.Now().After(expiresAt.Time) {
			return ErrRequestExpired
		}
		if priestID != "" {
			// Serialise accepts per priest, then make sure no other ritual is live.
			if _, err := tx.ExecContext(ctx, `SELECT 1 FROM priests WHERE id = $1::uuid FOR UPDATE`, priestID); err != nil {
				return err
			}
			var busy bool
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM bookings WHERE priest_id = $1::uuid AND id <> $2::uuid
				AND event_id IS NOT NULL AND status IN ('ACCEPTED','IN_TRANSIT','ARRIVED','POOJA_STARTED')
                AND (
                    ($3 = 'EVENT_NOW' AND booking_type = 'EVENT_NOW')
                    OR EXISTS (SELECT 1 FROM bookings requested WHERE requested.id = $2::uuid
                        AND bookings.scheduled_at < requested.scheduled_at + make_interval(mins => (requested.agreement_snapshot->>'duration_minutes')::int)
                        AND requested.scheduled_at < bookings.scheduled_at + make_interval(mins => (bookings.agreement_snapshot->>'duration_minutes')::int))
                ))`,
				priestID, req.BookingID, bookingType).Scan(&busy); err != nil {
				return err
			}
			if busy {
				return ErrPriestBusy
			}
		}
	case "POOJA_STARTED":
		// PRD v2 BKG-006: the devotee's booking code is checked before the ritual starts.
		if attempts >= 5 {
			return fmt.Errorf("%w: attempt limit reached; contact support", lifecycle.ErrInvalidOTP)
		}
		if strings.TrimSpace(req.CompletionOTP) == "" {
			return ErrBookingCodeRequired
		}
		if req.CompletionOTP != expectedCode {
			if _, err := tx.ExecContext(ctx, `UPDATE bookings SET completion_otp_attempts = completion_otp_attempts + 1 WHERE id = $1`, req.BookingID); err != nil {
				return err
			}
			return errWrongBookingCode
		}
	}
	return nil
}

// GetPublicCardStatus powers the QR page. It reveals status only (PRD v2 ID-003).
func (s *Storage) GetPublicCardStatus(ctx context.Context, code string) (*models.PublicCardStatus, error) {
	var c models.PublicCardStatus
	err := s.DB.QueryRowContext(ctx, `SELECT c.status, c.issuer, COALESCE(p.full_name,''), e.name, e.starts_at, e.ends_at
		FROM event_credentials c JOIN priests p ON p.id = c.priest_id JOIN events e ON e.id = c.event_id
		WHERE c.card_code = $1`, code).Scan(&c.Status, &c.Issuer, &c.PriestName, &c.EventName, &c.ValidFrom, &c.ValidUntil)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, err
	}
	if time.Now().After(c.ValidUntil) && c.Status == "ACTIVE" {
		c.Status = "EXPIRED"
	}
	return &c, nil
}
