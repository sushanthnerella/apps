package main

import (
	"encoding/json"
	"github.com/purohit-platform/backend/internal/models"
	"strings"
	"testing"
)

func TestCompletionAttemptLimitPersists(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	booking, otp, _, _, priest := createFSMTestBooking(t, router, storage, "+919876509101", "+919876509102")
	if _, err := storage.DB.Exec("UPDATE bookings SET status='POOJA_STARTED' WHERE id=$1", booking); err != nil {
		t.Fatal(err)
	}
	wrong := "0000"
	if otp == wrong {
		wrong = "0001"
	}
	req := models.BookingStatusUpdateRequest{BookingID: booking, Status: "COMPLETED", CallerRole: "priest", CallerID: priest, CompletionOTP: wrong}
	for i := 0; i < 5; i++ {
		if _, err := storage.UpdateBookingStatus(req); err == nil {
			t.Fatal("wrong OTP accepted")
		}
	}
	req.CompletionOTP = otp
	if _, err := storage.UpdateBookingStatus(req); err == nil {
		t.Fatal("attempt cap bypassed with correct OTP")
	}
	var attempts int
	var status string
	if err := storage.DB.QueryRow("SELECT completion_otp_attempts,status FROM bookings WHERE id=$1", booking).Scan(&attempts, &status); err != nil {
		t.Fatal(err)
	}
	if attempts != 5 || status != "POOJA_STARTED" {
		t.Fatalf("attempts=%d status=%s", attempts, status)
	}
}
func TestPublicPriestJSONOmitsPhone(t *testing.T) {
	bytes, err := json.Marshal(models.NearbyPriest{ID: "priest", Phone: "private-number"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bytes), "phone") || strings.Contains(string(bytes), "private-number") {
		t.Fatal("phone leaked")
	}
}
