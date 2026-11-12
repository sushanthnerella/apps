package cancellation

import (
	"fmt"
	"math"

	"github.com/purohit-platform/backend/internal/models"
)

// CalculateRefundQuoteMinor calculates refund and fees using exact integer arithmetic (paise)
func CalculateRefundQuoteMinor(bookingID, status string, totalAmountMinor, dakshinaAmountMinor int64) models.CancellationQuote {
	quote := models.CancellationQuote{
		BookingID:        bookingID,
		CurrentStatus:    status,
		AdvancePaidMinor: totalAmountMinor,
		Currency:         "INR",
		AdvancePaid:      float64(totalAmountMinor) / 100.0,
	}

	switch status {
	case "REQUESTED", "DISPATCHED", "ACCEPTED":
		quote.IsFreeCancellation = true
		quote.CancellationFeeMinor = 0
		quote.TravelFeeMinor = 0
		quote.PriestCompensationMinor = 0
		quote.RefundAmountMinor = totalAmountMinor
		quote.CancellationFee = 0.0
		quote.TravelFee = 0.0
		quote.PriestCompensation = 0.0
		quote.RefundAmount = float64(totalAmountMinor) / 100.0
		quote.PolicyNotice = "100% Instant Refund - Free Cancellation before priest departs"

	case "IN_TRANSIT":
		quote.IsFreeCancellation = false
		travelAllowanceMinor := int64(15000) // Rs 150.00
		quote.CancellationFeeMinor = travelAllowanceMinor
		quote.TravelFeeMinor = travelAllowanceMinor
		quote.PriestCompensationMinor = travelAllowanceMinor
		refundMinor := totalAmountMinor - travelAllowanceMinor
		if refundMinor < 0 {
			refundMinor = 0
		}
		quote.RefundAmountMinor = refundMinor
		quote.CancellationFee = float64(travelAllowanceMinor) / 100.0
		quote.TravelFee = float64(travelAllowanceMinor) / 100.0
		quote.PriestCompensation = float64(travelAllowanceMinor) / 100.0
		quote.RefundAmount = float64(refundMinor) / 100.0
		quote.PolicyNotice = "Priest is in-transit. Nominal travel allowance of Rs. 150 applies; remainder refunded instantly."

	case "ARRIVED", "POOJA_STARTED":
		quote.IsFreeCancellation = false
		priestShareMinor := (dakshinaAmountMinor * 25) / 100 // 25% retained for priest (integer arithmetic)
		quote.CancellationFeeMinor = priestShareMinor
		quote.PriestCompensationMinor = priestShareMinor
		refundMinor := totalAmountMinor - priestShareMinor
		if refundMinor < 0 {
			refundMinor = 0
		}
		quote.RefundAmountMinor = refundMinor
		quote.CancellationFee = float64(priestShareMinor) / 100.0
		quote.PriestCompensation = float64(priestShareMinor) / 100.0
		quote.RefundAmount = float64(refundMinor) / 100.0
		quote.PolicyNotice = "Priest has arrived. 25% Dakshina retained for the priest; 75% refunded to original source."

	default:
		quote.IsFreeCancellation = false
		quote.CancellationFeeMinor = 0
		quote.RefundAmountMinor = 0
		quote.CancellationFee = 0.0
		quote.RefundAmount = 0.0
		quote.PolicyNotice = fmt.Sprintf("Booking is in '%s' status. Cancellation not applicable.", status)
	}

	return quote
}

// CalculateRefundQuote calculates refund and fees according to Swiggy/Rapido policy
func CalculateRefundQuote(bookingID, status string, totalAmount, dakshinaAmount float64) models.CancellationQuote {
	return CalculateRefundQuoteMinor(
		bookingID,
		status,
		int64(math.Round(totalAmount*100)),
		int64(math.Round(dakshinaAmount*100)),
	)
}
