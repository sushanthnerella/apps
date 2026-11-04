package cancellation

import (
	"fmt"
	"math"

	"github.com/purohit-platform/backend/internal/models"
)

// CalculateRefundQuote calculates refund and fees according to Swiggy/Rapido policy
func CalculateRefundQuote(bookingID, status string, totalAmount, dakshinaAmount float64) models.CancellationQuote {
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

	return quote
}
