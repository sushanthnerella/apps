package cancellation

import (
	"testing"
)

func TestRefundCalculationRequestedState(t *testing.T) {
	quote := CalculateRefundQuote("booking_001", "REQUESTED", 5000.0, 5000.0)
	if !quote.IsFreeCancellation {
		t.Fatalf("Expected free cancellation for REQUESTED status")
	}
	if quote.RefundAmount != 5000.0 {
		t.Fatalf("Expected full 5000.0 refund, got %f", quote.RefundAmount)
	}
	if quote.CancellationFee != 0.0 {
		t.Fatalf("Expected 0.0 cancellation fee, got %f", quote.CancellationFee)
	}
}

func TestRefundCalculationInTransitState(t *testing.T) {
	quote := CalculateRefundQuote("booking_002", "IN_TRANSIT", 5000.0, 5000.0)
	if quote.IsFreeCancellation {
		t.Fatalf("Expected non-free cancellation for IN_TRANSIT status")
	}
	if quote.TravelFee != 150.0 {
		t.Fatalf("Expected 150.0 travel fee for priest, got %f", quote.TravelFee)
	}
	if quote.RefundAmount != 4850.0 {
		t.Fatalf("Expected 4850.0 devotee refund (5000 - 150), got %f", quote.RefundAmount)
	}
	if quote.PriestCompensation != 150.0 {
		t.Fatalf("Expected 150.0 priest compensation, got %f", quote.PriestCompensation)
	}
}

func TestRefundCalculationArrivedState(t *testing.T) {
	quote := CalculateRefundQuote("booking_003", "ARRIVED", 4000.0, 4000.0)
	expectedPriestShare := 1000.0 // 25% of 4000
	expectedRefund := 3000.0      // 75% of 4000

	if quote.PriestCompensation != expectedPriestShare {
		t.Fatalf("Expected %f priest compensation, got %f", expectedPriestShare, quote.PriestCompensation)
	}
	if quote.RefundAmount != expectedRefund {
		t.Fatalf("Expected %f devotee refund, got %f", expectedRefund, quote.RefundAmount)
	}
}

func TestRefundCalculationExactIntegerMinorUnits(t *testing.T) {
	// Prompt requirement: cancellation quote on 3716.00 INR with the Rs 150 fee returns exact integers
	totalAmountMinor := int64(371600) // 3716.00 INR
	quote := CalculateRefundQuoteMinor("booking_minor_001", "IN_TRANSIT", totalAmountMinor, totalAmountMinor)

	if quote.IsFreeCancellation {
		t.Fatalf("Expected non-free cancellation for IN_TRANSIT status")
	}
	if quote.CancellationFeeMinor != 15000 {
		t.Fatalf("Expected exactly 15000 paise cancellation fee, got %d", quote.CancellationFeeMinor)
	}
	if quote.TravelFeeMinor != 15000 {
		t.Fatalf("Expected exactly 15000 paise travel fee, got %d", quote.TravelFeeMinor)
	}
	if quote.PriestCompensationMinor != 15000 {
		t.Fatalf("Expected exactly 15000 paise priest compensation, got %d", quote.PriestCompensationMinor)
	}
	expectedRefundMinor := int64(356600) // 371600 - 15000 = 356600 paise
	if quote.RefundAmountMinor != expectedRefundMinor {
		t.Fatalf("Expected exactly %d paise refund, got %d", expectedRefundMinor, quote.RefundAmountMinor)
	}
	if quote.RefundAmount != 3566.0 {
		t.Fatalf("Expected 3566.0 float alias, got %f", quote.RefundAmount)
	}
}
