package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/purohit-platform/backend/internal/models"
)

// TestHTTPAtomicSlotBindingAtCreation verifies:
// 1. Creating a booking with slot_id binds the slot atomically (is_booked=true, booked_booking_id=bookingID)
// 2. A second booking attempt with the same slot_id returns HTTP 409 Conflict (double-booking prevented)
func TestHTTPAtomicSlotBindingAtCreation(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	_, _, devToken1, prToken, prID := createFSMTestBooking(t, router, storage, "+919876540201", "+919876540202")

	// Devotee 2 registration
	dev2ID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone: "+919876540203", FullName: "Devotee Two", Password: "Password123!", ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("failed registering devotee 2: %v", err)
	}
	devToken2 := issueTestToken(t, dev2ID, "devotee", 1)

	// Priest publishes a specific slot
	slotID, err := storage.CreatePriestSlot(models.CreateSlotRequest{
		PriestID:  prID,
		SlotDate:  "2026-11-20",
		StartTime: "09:00",
		EndTime:   "11:00",
	})
	if err != nil {
		t.Fatalf("failed creating slot: %v", err)
	}

	catalog, err := storage.GetCatalog("", "", "en")
	if err != nil || len(catalog) == 0 {
		t.Fatalf("failed fetching catalog: %v", err)
	}
	poojaID := catalog[0]["id"].(string)

	// Devotee 1 creates booking with slot_id
	createReq1 := models.BookingCreateRequest{
		PriestID:     prID,
		PoojaID:      poojaID,
		SlotID:       slotID,
		EventAddress: "Plot 10, Banjara Hills, Hyderabad",
		TotalAmount:  3500.0,
		BookingType:  "SCHEDULED",
	}
	body1, _ := json.Marshal(createReq1)
	req1 := httptest.NewRequest("POST", "/api/v1/bookings", bytes.NewBuffer(body1))
	req1.Header.Set("Idempotency-Key", newIdemKey())
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+devToken1)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("devotee 1 booking with slot failed: code=%d body=%s", w1.Code, w1.Body.String())
	}

	var resp1 map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &resp1)
	bID1 := resp1["booking_id"].(string)

	// Verify slot is booked and linked to booking 1
	var isBooked bool
	var bookedBookingID string
	err = storage.DB.QueryRow(`SELECT is_booked, COALESCE(booked_booking_id::text, '') FROM priest_availability_slots WHERE id = $1::uuid`, slotID).Scan(&isBooked, &bookedBookingID)
	if err != nil {
		t.Fatalf("failed checking slot state: %v", err)
	}
	if !isBooked || bookedBookingID != bID1 {
		t.Fatalf("expected slot to be booked by %s, got is_booked=%v, booked_id=%s", bID1, isBooked, bookedBookingID)
	}

	// Devotee 2 attempts to book the same slot -> must receive HTTP 409 Conflict
	createReq2 := models.BookingCreateRequest{
		PriestID:     prID,
		PoojaID:      poojaID,
		SlotID:       slotID,
		EventAddress: "Plot 20, Jubilee Hills, Hyderabad",
		TotalAmount:  3500.0,
		BookingType:  "SCHEDULED",
	}
	body2, _ := json.Marshal(createReq2)
	req2 := httptest.NewRequest("POST", "/api/v1/bookings", bytes.NewBuffer(body2))
	req2.Header.Set("Idempotency-Key", newIdemKey())
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+devToken2)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("expected HTTP 409 Conflict on double booking, got code=%d body=%s", w2.Code, w2.Body.String())
	}
	t.Logf("✅ Double booking correctly rejected with HTTP 409 Conflict: %s", w2.Body.String())

	// Cancelling booking 1 must atomically release the slot
	cancelPayload := fmt.Sprintf(`{"booking_id":"%s","reason_code":"CHANGE_PLANS"}`, bID1)
	reqCancel := httptest.NewRequest("POST", "/api/v1/bookings/cancel", bytes.NewBufferString(cancelPayload))
	reqCancel.Header.Set("Idempotency-Key", newIdemKey())
	reqCancel.Header.Set("Content-Type", "application/json")
	reqCancel.Header.Set("Authorization", "Bearer "+devToken1)
	wCancel := httptest.NewRecorder()
	router.ServeHTTP(wCancel, reqCancel)

	if wCancel.Code != http.StatusOK {
		t.Fatalf("failed cancellation: %s", wCancel.Body.String())
	}

	// Verify slot is freed
	err = storage.DB.QueryRow(`SELECT is_booked, COALESCE(booked_booking_id::text, '') FROM priest_availability_slots WHERE id = $1::uuid`, slotID).Scan(&isBooked, &bookedBookingID)
	if err != nil {
		t.Fatalf("failed checking slot after cancellation: %v", err)
	}
	if isBooked || bookedBookingID != "" {
		t.Fatalf("expected slot to be free after cancellation, got is_booked=%v, booked_id=%s", isBooked, bookedBookingID)
	}

	// Now Devotee 2 can book the released slot successfully
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("POST", "/api/v1/bookings", bytes.NewBuffer(body2))
	req3.Header.Set("Idempotency-Key", newIdemKey())
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Authorization", "Bearer "+devToken2)
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("devotee 2 booking of freed slot failed: code=%d body=%s", w3.Code, w3.Body.String())
	}
	_ = prToken
}

// TestHTTPConcurrentSlotReservationRaceCondition verifies that when multiple devotees
// attempt to book the exact same slot concurrently, exactly one succeeds (HTTP 200)
// and all others receive HTTP 409 Conflict.
func TestHTTPConcurrentSlotReservationRaceCondition(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	_, _, _, _, prID := createFSMTestBooking(t, router, storage, "+919876540301", "+919876540302")

	// Priest publishes a slot
	slotID, err := storage.CreatePriestSlot(models.CreateSlotRequest{
		PriestID:  prID,
		SlotDate:  "2026-11-21",
		StartTime: "14:00",
		EndTime:   "16:00",
	})
	if err != nil {
		t.Fatalf("failed creating slot: %v", err)
	}

	catalog, err := storage.GetCatalog("", "", "en")
	if err != nil || len(catalog) == 0 {
		t.Fatalf("failed fetching catalog: %v", err)
	}
	poojaID := catalog[0]["id"].(string)

	concurrentUsers := 6
	devTokens := make([]string, concurrentUsers)
	for i := 0; i < concurrentUsers; i++ {
		uid, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
			Phone:         fmt.Sprintf("+9198765403%02d", i+10),
			FullName:      fmt.Sprintf("Devotee Concurrent %d", i),
			Password:      "Password123!",
			ConsentAgreed: true,
		})
		if err != nil {
			t.Fatalf("failed registering devotee %d: %v", i, err)
		}
		devTokens[i] = issueTestToken(t, uid, "devotee", 1)
	}

	var wg sync.WaitGroup
	var successCount int32
	var conflictCount int32
	startGate := make(chan struct{})

	for i := 0; i < concurrentUsers; i++ {
		wg.Add(1)
		token := devTokens[i]
		go func(idx int, userToken string) {
			defer wg.Done()
			<-startGate // synchronize simultaneous start

			createReq := models.BookingCreateRequest{
				PriestID:     prID,
				PoojaID:      poojaID,
				SlotID:       slotID,
				EventAddress: fmt.Sprintf("Street %d, Hyderabad", idx),
				TotalAmount:  3000.0,
				BookingType:  "SCHEDULED",
			}
			body, _ := json.Marshal(createReq)
			req := httptest.NewRequest("POST", "/api/v1/bookings", bytes.NewBuffer(body))
			req.Header.Set("Idempotency-Key", newIdemKey())
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+userToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				atomic.AddInt32(&successCount, 1)
			} else if w.Code == http.StatusConflict {
				atomic.AddInt32(&conflictCount, 1)
			} else {
				t.Errorf("user %d got unexpected code %d: %s", idx, w.Code, w.Body.String())
			}
		}(i, token)
	}

	close(startGate) // fire all requests simultaneously
	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful booking on the slot, got %d (conflicts: %d)", successCount, conflictCount)
	}
	if conflictCount != int32(concurrentUsers-1) {
		t.Fatalf("expected %d conflicts, got %d", concurrentUsers-1, conflictCount)
	}
	t.Logf("✅ Concurrent race condition verified: exactly 1 booking succeeded, %d failed with 409 Conflict", conflictCount)
}
