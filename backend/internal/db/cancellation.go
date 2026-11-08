package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

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
	var totalAmount float64
	var dakshinaAmount float64

	query := `SELECT status, total_amount, COALESCE(dakshina_amount, total_amount) FROM bookings WHERE id = $1`
	err := s.DB.QueryRow(query, bookingID).Scan(&status, &totalAmount, &dakshinaAmount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, fmt.Errorf("booking not found: %w", err)
	}

	if lifecycle.IsTerminal(lifecycle.BookingStatus(status)) {
		return nil, fmt.Errorf("%w: current status is '%s'", ErrTerminalState, status)
	}

	quote := models.CancellationQuote{
		BookingID:     bookingID,
		CurrentStatus: status,
		AdvancePaid:   totalAmount,
	}

	switch status {
	case "REQUESTED", "DISPATCHED", "ACCEPTED":
		quote.IsFreeCancellation = true
		quote.CancellationFee = 0.0
		quote.TravelFee = 0.0
		quote.PriestCompensation = 0.0
		quote.RefundAmount = totalAmount
		quote.PolicyNotice = "Free Pilot: 100% Free Cancellation before priest departs (no charges incurred)"

	case "IN_TRANSIT":
		quote.IsFreeCancellation = false
		quote.CancellationFee = 150.0 // Travel allowance credited to priest
		quote.TravelFee = 150.0
		quote.PriestCompensation = 150.0
		quote.RefundAmount = math.Max(0.0, totalAmount-150.0)
		quote.PolicyNotice = "Free Pilot: Priest is in-transit. Nominal travel allowance Rs. 150 accounted for pilot records."

	case "ARRIVED", "POOJA_STARTED":
		quote.IsFreeCancellation = false
		priestShare := math.Round(dakshinaAmount*0.25*100) / 100 // 25% retained for priest
		quote.CancellationFee = priestShare
		quote.PriestCompensation = priestShare
		quote.RefundAmount = math.Max(0.0, totalAmount-priestShare)
		quote.PolicyNotice = "Free Pilot: Priest has arrived. 25% nominal Dakshina accounted for priest compensation."

	default:
		quote.IsFreeCancellation = false
		quote.CancellationFee = 0.0
		quote.RefundAmount = 0.0
		quote.PolicyNotice = fmt.Sprintf("Booking is in '%s' status. Cancellation not applicable.", status)
	}

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
	var totalAmount float64
	var dakshinaAmount float64

	lockQuery := `
		SELECT status, user_id::text, COALESCE(priest_id::text, ''), total_amount, COALESCE(dakshina_amount, total_amount)
		FROM bookings
		WHERE id = $1
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, lockQuery, req.BookingID).Scan(
		&currentStatus,
		&currentUserID,
		&currentPriestID,
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

	// Step 4: Calculate Dynamic Quote & Retain Pilot Financial Math
	quote := models.CancellationQuote{
		BookingID:     req.BookingID,
		CurrentStatus: currentStatus,
		AdvancePaid:   totalAmount,
	}

	switch currentStatus {
	case "REQUESTED", "DISPATCHED", "ACCEPTED":
		quote.IsFreeCancellation = true
		quote.CancellationFee = 0.0
		quote.TravelFee = 0.0
		quote.PriestCompensation = 0.0
		quote.RefundAmount = totalAmount
		quote.PolicyNotice = "Free Pilot: 100% Free Cancellation before priest departs (no charges incurred)"

	case "IN_TRANSIT":
		quote.IsFreeCancellation = false
		quote.CancellationFee = 150.0
		quote.TravelFee = 150.0
		quote.PriestCompensation = 150.0
		quote.RefundAmount = math.Max(0.0, totalAmount-150.0)
		quote.PolicyNotice = "Free Pilot: Priest is in-transit. Nominal travel allowance Rs. 150 accounted for pilot records."

	case "ARRIVED", "POOJA_STARTED":
		quote.IsFreeCancellation = false
		priestShare := math.Round(dakshinaAmount*0.25*100) / 100
		quote.CancellationFee = priestShare
		quote.PriestCompensation = priestShare
		quote.RefundAmount = math.Max(0.0, totalAmount-priestShare)
		quote.PolicyNotice = "Free Pilot: Priest has arrived. 25% nominal Dakshina accounted for priest compensation."

	default:
		quote.IsFreeCancellation = false
		quote.CancellationFee = 0.0
		quote.RefundAmount = 0.0
		quote.PolicyNotice = fmt.Sprintf("Booking is in '%s' status. Cancellation not applicable.", currentStatus)
	}

	escrowStatus := "REFUNDED_TO_DEVOTEE"
	if quote.PriestCompensation > 0 {
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
