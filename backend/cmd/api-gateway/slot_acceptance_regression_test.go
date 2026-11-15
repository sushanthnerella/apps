package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// An already reserved afternoon slot must not collide with an unrelated morning booking.
func TestAcceptUsesBoundAvailabilitySlot(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	booking, _, _, priestToken, priest := createFSMTestBooking(t, router, storage, "+919876541201", "+919876541202")
	otherBooking, _, _, _, _ := createFSMTestBooking(t, router, storage, "+919876541203", "+919876541204")
	_, err := storage.DB.Exec(`UPDATE bookings SET scheduled_date = '2026-11-20', slot_time = '14:00 - 15:00' WHERE id = $1::uuid`, booking)
	if err != nil {
		t.Fatal(err)
	}
	_, err = storage.DB.Exec(`INSERT INTO priest_availability_slots (priest_id, slot_date, start_time, end_time, is_booked, booked_booking_id) VALUES ($1::uuid, '2026-11-20', '09:00', '11:00', true, $2::uuid), ($1::uuid, '2026-11-20', '14:00', '15:00', true, $3::uuid)`, priest, otherBooking, booking)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(fmt.Sprintf(`{"booking_id":"%s","status":"ACCEPTED"}`, booking)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+priestToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("accepting reserved afternoon slot: HTTP %d: %s", w.Code, w.Body.String())
	}
	var count int
	if err := storage.DB.QueryRow(`SELECT count(*) FROM priest_availability_slots WHERE booked_booking_id = $1::uuid`, booking).Scan(&count); err != nil || count != 1 {
		t.Fatalf("expected exactly one bound slot, got %d: %v", count, err)
	}
}
