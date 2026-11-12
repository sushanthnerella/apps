package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/purohit-platform/backend/internal/cancellation"
	"github.com/purohit-platform/backend/internal/lifecycle"
	"github.com/purohit-platform/backend/internal/models"
)

var (
	// ErrTerminalState indicates a booking is already in a terminal state (COMPLETED or CANCELLED).
	ErrTerminalState = lifecycle.ErrTerminalState

	// ErrUnauthorizedCancellation indicates the caller role is not permitted to cancel in the current state.
	ErrUnauthorizedCancellation = lifecycle.ErrUnauthorizedCancellation

	// ErrBookingNotFound indicates the requested booking was not found.
	ErrBookingNotFound = lifecycle.ErrBookingNotFound
)

// GetCancellationQuote evaluates dynamic refund according to Swiggy/Rapido policy
// while honest about the free pilot model (no false payment gateway / instant bank refund claims).
func (s *Storage) GetCancellationQuote(bookingID string) (*models.CancellationQuote, error) {
	var status string
	var totalAmountMinor, dakshinaAmountMinor int64
	var totalAmount, dakshinaAmount float64

	query := `SELECT status, 
                     COALESCE(total_amount_minor, ROUND(total_amount * 100)), 
                     COALESCE(dakshina_amount_minor, ROUND(COALESCE(dakshina_amount, total_amount) * 100)),
                     total_amount, COALESCE(dakshina_amount, total_amount) 
              FROM bookings WHERE id = $1`
	err := s.DB.QueryRow(query, bookingID).Scan(&status, &totalAmountMinor, &dakshinaAmountMinor, &totalAmount, &dakshinaAmount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, fmt.Errorf("booking not found: %w", err)
	}

	if lifecycle.IsTerminal(lifecycle.BookingStatus(status)) {
		return nil, fmt.Errorf("%w: current status is '%s'", ErrTerminalState, status)
	}

	quote := cancellation.CalculateRefundQuoteMinor(bookingID, status, totalAmountMinor, dakshinaAmountMinor)
	return &quote, nil
}

// CancelBookingWithRefund processes booking cancellation atomically:
// 1. Begins database transaction with Read Committed isolation.
// 2. Locks booking row with SELECT ... FOR UPDATE.
// 3. Rejects cancellations on terminal states (COMPLETED, CANCELLED) with ErrTerminalState.
// 4. Enforces role-based cancellation permissions (devotee only in REQUESTED, ACCEPTED, IN_TRANSIT).
// 5. Calculates dynamic refund/compensation quote preserving pilot financial data.
// 6. Updates booking state, cancellation reason, cancelled_by, and refund amount.
// 7. Atomically releases any linked priest availability slot in the same transaction.
func (s *Storage) CancelBookingWithRefund(req models.CancelBookingRequest) (*models.CancellationQuote, error) {
	if req.BookingID == "" {
		return nil, fmt.Errorf("booking_id is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Begin atomic database transaction
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("failed to begin cancellation transaction: %w", err)
	}
	defer tx.Rollback()

	// Step 1: Pessimistic Row Lock (SELECT ... FOR UPDATE)
	var currentStatus string
	var currentUserID string
	var currentPriestID sql.NullString
	var totalAmountMinor, dakshinaAmountMinor int64
	var totalAmount float64
	var dakshinaAmount float64

	lockQuery := `
		SELECT status, user_id::text, COALESCE(priest_id::text, ''), 
		       COALESCE(total_amount_minor, ROUND(total_amount * 100)),
		       COALESCE(dakshina_amount_minor, ROUND(COALESCE(dakshina_amount, total_amount) * 100)),
		       total_amount, COALESCE(dakshina_amount, total_amount)
		FROM bookings
		WHERE id = $1
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, lockQuery, req.BookingID).Scan(
		&currentStatus,
		&currentUserID,
		&currentPriestID,
		&totalAmountMinor,
		&dakshinaAmountMinor,
		&totalAmount,
		&dakshinaAmount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, fmt.Errorf("failed to acquire booking lock: %w", err)
	}

	// Step 2: Terminal State Protection
	if currentStatus == "COMPLETED" || currentStatus == "CANCELLED" {
		return nil, fmt.Errorf("%w: booking is already '%s'", ErrTerminalState, currentStatus)
	}

	// Step 3: Role Authorization Validation
	normalizedRole := lifecycle.NormalizeRole(req.CancelledByRole)
	switch normalizedRole {
	case lifecycle.RoleDevotee:
		// Devotees permitted only in REQUESTED, DISPATCHED, ACCEPTED, IN_TRANSIT
		if currentStatus != "REQUESTED" && currentStatus != "DISPATCHED" && currentStatus != "ACCEPTED" && currentStatus != "IN_TRANSIT" {
			return nil, fmt.Errorf("%w: devotees cannot cancel booking once priest has arrived (current status: '%s'); contact admin for support", ErrUnauthorizedCancellation, currentStatus)
		}

	case lifecycle.RolePriest:
		// Priests permitted only in REQUESTED, DISPATCHED, ACCEPTED
		if currentStatus != "REQUESTED" && currentStatus != "DISPATCHED" && currentStatus != "ACCEPTED" {
			return nil, fmt.Errorf("%w: priests cannot cancel booking once in-transit or arrived (current status: '%s')", ErrUnauthorizedCancellation, currentStatus)
		}

	case lifecycle.RoleAdmin:
		// Admin emergency override permitted on any active non-terminal state
		if strings.TrimSpace(req.ReasonText) == "" && strings.TrimSpace(req.ReasonCode) == "" {
			return nil, fmt.Errorf("mandatory cancellation reason must be provided for admin emergency override")
		}

	case lifecycle.RoleSystem:
		// System transitions permitted

	default:
		return nil, fmt.Errorf("%w: unrecognized cancellation role '%s'", ErrUnauthorizedCancellation, req.CancelledByRole)
	}

	// Step 4: Calculate Dynamic Quote & Retain Pilot Financial Math (Exact minor integer arithmetic)
	quote := cancellation.CalculateRefundQuoteMinor(req.BookingID, currentStatus, totalAmountMinor, dakshinaAmountMinor)

	escrowStatus := "REFUNDED_TO_DEVOTEE"
	if quote.PriestCompensationMinor > 0 {
		escrowStatus = "PARTIALLY_REFUNDED"
	}

	reasonRecord := req.ReasonText
	if reasonRecord == "" {
		reasonRecord = req.ReasonCode
	}
	cancelledByRecord := string(normalizedRole)

	// Step 5: Update Booking Record Atomically
	updateBookingQuery := `
		UPDATE bookings 
		SET status = 'CANCELLED', 
		    escrow_status = $2, 
		    cancellation_reason = $3, 
		    cancelled_by = $4, 
		    refund_amount = $5
		WHERE id = $1
	`
	res, err := tx.ExecContext(ctx, updateBookingQuery, req.BookingID, escrowStatus, reasonRecord, cancelledByRecord, quote.RefundAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to update booking to cancelled: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return nil, fmt.Errorf("booking update failed: concurrent modification or row deleted")
	}

	// Step 6: Atomic Slot Release
	releaseSlotQuery := `
		UPDATE priest_availability_slots 
		SET is_booked = false, booked_booking_id = NULL 
		WHERE booked_booking_id = $1::uuid
	`
	_, err = tx.ExecContext(ctx, releaseSlotQuery, req.BookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to atomically release linked availability slot: %w", err)
	}

	// Step 7: Commit Database Transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit cancellation transaction: %w", err)
	}

	return &quote, nil
}
