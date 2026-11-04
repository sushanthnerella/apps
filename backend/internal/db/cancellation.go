package db

import (
	"fmt"
	"math"

	"github.com/purohit-platform/backend/internal/models"
)

// CalculateRefund evaluates dynamic refund according to Swiggy/Rapido policy
func (s *Storage) GetCancellationQuote(bookingID string) (*models.CancellationQuote, error) {
	var status string
	var totalAmount float64
	var dakshinaAmount float64

	query := `SELECT status, total_amount, COALESCE(dakshina_amount, total_amount) FROM bookings WHERE id = $1`
	err := s.DB.QueryRow(query, bookingID).Scan(&status, &totalAmount, &dakshinaAmount)
	if err != nil {
		return nil, fmt.Errorf("booking not found: %w", err)
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
		quote.PolicyNotice = "100% Instant Refund - Free Cancellation before priest departs"

	case "IN_TRANSIT":
		quote.IsFreeCancellation = false
		quote.CancellationFee = 150.0 // Travel allowance credited to priest
		quote.TravelFee = 150.0
		quote.PriestCompensation = 150.0
		quote.RefundAmount = math.Max(0.0, totalAmount-150.0)
		quote.PolicyNotice = "Priest is in-transit. Nominal travel allowance of Rs. 150 applies; remainder refunded instantly."

	case "ARRIVED", "POOJA_STARTED":
		quote.IsFreeCancellation = false
		priestShare := math.Round(dakshinaAmount*0.25*100) / 100 // 25% retained for priest
		quote.CancellationFee = priestShare
		quote.PriestCompensation = priestShare
		quote.RefundAmount = math.Max(0.0, totalAmount-priestShare)
		quote.PolicyNotice = "Priest has arrived. 25% Dakshina retained for the priest; 75% refunded to original source."

	default:
		quote.IsFreeCancellation = false
		quote.CancellationFee = 0.0
		quote.RefundAmount = 0.0
		quote.PolicyNotice = fmt.Sprintf("Booking is in '%s' status. Cancellation not applicable.", status)
	}

	return &quote, nil
}

// CancelBookingWithRefund processes booking cancellation, records reason, and updates escrow
func (s *Storage) CancelBookingWithRefund(req models.CancelBookingRequest) (*models.CancellationQuote, error) {
	quote, err := s.GetCancellationQuote(req.BookingID)
	if err != nil {
		return nil, err
	}

	escrowStatus := "REFUNDED_TO_DEVOTEE"
	if quote.PriestCompensation > 0 {
		escrowStatus = "PARTIALLY_REFUNDED_DEVOTEE_AND_PRIEST"
	}

	query := `
		UPDATE bookings 
		SET status = 'CANCELLED', escrow_status = $2 
		WHERE id = $1
	`
	_, err = s.DB.Exec(query, req.BookingID, escrowStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel booking: %w", err)
	}

	return quote, nil
}
