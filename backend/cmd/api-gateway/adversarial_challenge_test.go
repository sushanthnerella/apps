package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/purohit-platform/backend/internal/models"
)

// 1. Stress test: Concurrent race between Priest Completion and Admin Cancellation on POOJA_STARTED
func TestAdversarialConcurrentCompletionVsAdminCancellation(t *testing.T) {
	router, storage, testSecret := setupBookingTestGateway(t)

	// Run multiple trials to stress-test race conditions and thread scheduling
	trials := 5
	for trial := 0; trial < trials; trial++ {
		devPhone := fmt.Sprintf("+9197771%05d", trial*2)
		prPhone := fmt.Sprintf("+9197771%05d", trial*2+1)
		bID, otp, _, prToken, _ := createFSMTestBooking(t, router, storage, devPhone, prPhone)
		adminToken := issueAdminTestToken(t, storage, fmt.Sprintf("admin-%d", trial), testSecret)

		// Advance booking to POOJA_STARTED
		for _, st := range []string{"ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED"} {
			payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bID, st)
			req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+prToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("trial %d: failed advancing to %s: %s", trial, st, w.Body.String())
			}
		}

		// Concurrently invoke Priest Completion and Admin Cancellation
		var wg sync.WaitGroup
		wg.Add(2)

		var completeCode, cancelCode int32
		var completeBody, cancelBody string

		startGate := make(chan struct{})

		// Goroutine 1: Priest calls /complete
		go func() {
			defer wg.Done()
			<-startGate
			compPayload := fmt.Sprintf(`{"completion_otp":"%s","dakshina_amount":3000.0}`, otp)
			req := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/complete", bytes.NewBufferString(compPayload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+prToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			atomic.StoreInt32(&completeCode, int32(w.Code))
			completeBody = w.Body.String()
		}()

		// Goroutine 2: Admin calls /admin-cancel
		go func() {
			defer wg.Done()
			<-startGate
			cancelPayload := `{"reason":"Emergency cancellation during pooja"}`
			req := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/admin-cancel", bytes.NewBufferString(cancelPayload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+adminToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			atomic.StoreInt32(&cancelCode, int32(w.Code))
			cancelBody = w.Body.String()
		}()

		// Release both goroutines simultaneously
		close(startGate)
		wg.Wait()

		cCode := atomic.LoadInt32(&completeCode)
		aCode := atomic.LoadInt32(&cancelCode)

		// Verification: Exactly one operation must succeed (200 OK) and the other must be rejected (409 Conflict)
		if (cCode == http.StatusOK && aCode == http.StatusConflict) ||
			(cCode == http.StatusConflict && aCode == http.StatusOK) {
			// Expected serialization outcome
		} else {
			t.Fatalf("trial %d: invalid race outcome: completeCode=%d (%s), cancelCode=%d (%s)",
				trial, cCode, completeBody, aCode, cancelBody)
		}

		// Verify database row state integrity
		details, err := storage.GetBookingDetails(bID)
		if err != nil {
			t.Fatalf("trial %d: failed to fetch details: %v", trial, err)
		}
		if cCode == http.StatusOK {
			if details.Status != "COMPLETED" {
				t.Fatalf("trial %d: expected COMPLETED status, got %s", trial, details.Status)
			}
		} else {
			if details.Status != "CANCELLED" {
				t.Fatalf("trial %d: expected CANCELLED status, got %s", trial, details.Status)
			}
		}
	}
}

// 2. Stress test: Concurrent race between Devotee Cancellation and Priest Arrived transition in IN_TRANSIT
func TestAdversarialConcurrentDevoteeCancelVsPriestArrived(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)

	trials := 5
	for trial := 0; trial < trials; trial++ {
		devPhone := fmt.Sprintf("+9197772%05d", trial*2)
		prPhone := fmt.Sprintf("+9197772%05d", trial*2+1)
		bID, _, devToken, prToken, _ := createFSMTestBooking(t, router, storage, devPhone, prPhone)

		// Advance booking to IN_TRANSIT
		for _, st := range []string{"ACCEPTED", "IN_TRANSIT"} {
			payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bID, st)
			req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+prToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("trial %d: failed advancing to %s: %s", trial, st, w.Body.String())
			}
		}

		var wg sync.WaitGroup
		wg.Add(2)

		var cancelCode, arriveCode int32
		var cancelBody, arriveBody string

		startGate := make(chan struct{})

		// Goroutine 1: Devotee calls /cancel
		go func() {
			defer wg.Done()
			<-startGate
			cancelPayload := fmt.Sprintf(`{"booking_id":"%s","reason_code":"EMERGENCY"}`, bID)
			req := httptest.NewRequest("POST", "/api/v1/bookings/cancel", bytes.NewBufferString(cancelPayload))
			req.Header.Set("Idempotency-Key", newIdemKey())
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+devToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			atomic.StoreInt32(&cancelCode, int32(w.Code))
			cancelBody = w.Body.String()
		}()

		// Goroutine 2: Priest calls /status -> ARRIVED
		go func() {
			defer wg.Done()
			<-startGate
			payload := fmt.Sprintf(`{"booking_id":"%s","status":"ARRIVED"}`, bID)
			req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+prToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			atomic.StoreInt32(&arriveCode, int32(w.Code))
			arriveBody = w.Body.String()
		}()

		close(startGate)
		wg.Wait()

		cCode := atomic.LoadInt32(&cancelCode)
		aCode := atomic.LoadInt32(&arriveCode)

		// Either:
		// 1) Cancel won: cancel=200, arrive=409 (since status is CANCELLED)
		// 2) Arrive won: arrive=200, cancel=403 (since in ARRIVED devotee cannot cancel)
		if cCode == http.StatusOK && aCode == http.StatusConflict {
			// Cancel succeeded first, arrive failed on cancelled booking
		} else if aCode == http.StatusOK && cCode == http.StatusForbidden {
			// Arrive succeeded first, devotee cancel in ARRIVED was forbidden with 403
		} else {
			t.Fatalf("trial %d: unexpected concurrent result: cancelCode=%d (%s), arriveCode=%d (%s)",
				trial, cCode, cancelBody, aCode, arriveBody)
		}
	}
}

// 3. Challenge: Attempt to cancel an already CANCELLED booking (must strictly return HTTP 409 Conflict)
func TestAdversarialCancelAlreadyCancelledBookingReturns409(t *testing.T) {
	router, storage, testSecret := setupBookingTestGateway(t)
	bID, _, devToken, _, _ := createFSMTestBooking(t, router, storage, "+919777300001", "+919777300002")
	adminToken := issueAdminTestToken(t, storage, "admin-immutability", testSecret)

	// Step 1: Devotee cancels booking in REQUESTED -> 200 OK
	cancelPayload := fmt.Sprintf(`{"booking_id":"%s","reason_code":"DEVOTEE_CANCEL"}`, bID)
	req1 := httptest.NewRequest("POST", "/api/v1/bookings/cancel", bytes.NewBufferString(cancelPayload))
	req1.Header.Set("Idempotency-Key", newIdemKey())
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+devToken)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first cancel should succeed with 200, got %d: %s", w1.Code, w1.Body.String())
	}

	// Step 2: Devotee attempts second cancel via /cancel -> must return 409 Conflict
	req2 := httptest.NewRequest("POST", "/api/v1/bookings/cancel", bytes.NewBufferString(cancelPayload))
	req2.Header.Set("Idempotency-Key", newIdemKey())
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+devToken)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on repeated cancel via /cancel, got %d: %s", w2.Code, w2.Body.String())
	}

	// Step 3: Devotee attempts cancel via /status -> must return 409 Conflict
	statusPayload := fmt.Sprintf(`{"booking_id":"%s","status":"CANCELLED"}`, bID)
	req3 := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(statusPayload))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Authorization", "Bearer "+devToken)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	if w3.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on cancel via /status for already cancelled booking, got %d: %s", w3.Code, w3.Body.String())
	}

	// Step 4: Admin attempts /admin-cancel on already CANCELLED booking -> must return 409 Conflict
	adminCancelPayload := `{"reason":"Admin attempting cancel on already cancelled booking"}`
	req4 := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/admin-cancel", bytes.NewBufferString(adminCancelPayload))
	req4.Header.Set("Content-Type", "application/json")
	req4.Header.Set("Authorization", "Bearer "+adminToken)
	w4 := httptest.NewRecorder()
	router.ServeHTTP(w4, req4)
	if w4.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on admin-cancel of already cancelled booking, got %d: %s", w4.Code, w4.Body.String())
	}
}

// 4. Challenge: Attempt to cancel an already COMPLETED booking (must strictly return HTTP 409 Conflict)
func TestAdversarialCancelAlreadyCompletedBookingReturns409(t *testing.T) {
	router, storage, testSecret := setupBookingTestGateway(t)
	bID, otp, devToken, prToken, _ := createFSMTestBooking(t, router, storage, "+919777400001", "+919777400002")
	adminToken := issueAdminTestToken(t, storage, "admin-comp-immutability", testSecret)

	// Advance through all stages to COMPLETED
	for _, st := range []string{"ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED"} {
		payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bID, st)
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+prToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("failed advance to %s: %s", st, w.Body.String())
		}
	}
	compPayload := fmt.Sprintf(`{"completion_otp":"%s","dakshina_amount":3000.0}`, otp)
	reqComp := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/complete", bytes.NewBufferString(compPayload))
	reqComp.Header.Set("Content-Type", "application/json")
	reqComp.Header.Set("Authorization", "Bearer "+prToken)
	wComp := httptest.NewRecorder()
	router.ServeHTTP(wComp, reqComp)
	if wComp.Code != http.StatusOK {
		t.Fatalf("failed completion: %s", wComp.Body.String())
	}

	// Devotee attempts cancel via /cancel -> must return 409 Conflict
	cancelPayload := fmt.Sprintf(`{"booking_id":"%s","reason_code":"DEVOTEE_MISTAKE"}`, bID)
	reqDevCancel := httptest.NewRequest("POST", "/api/v1/bookings/cancel", bytes.NewBufferString(cancelPayload))
	reqDevCancel.Header.Set("Idempotency-Key", newIdemKey())
	reqDevCancel.Header.Set("Content-Type", "application/json")
	reqDevCancel.Header.Set("Authorization", "Bearer "+devToken)
	wDevCancel := httptest.NewRecorder()
	router.ServeHTTP(wDevCancel, reqDevCancel)
	if wDevCancel.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for devotee cancelling completed booking, got %d: %s", wDevCancel.Code, wDevCancel.Body.String())
	}

	// Devotee attempts cancel via /status -> must return 409 Conflict
	statusPayload := fmt.Sprintf(`{"booking_id":"%s","status":"CANCELLED"}`, bID)
	reqStatusCancel := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(statusPayload))
	reqStatusCancel.Header.Set("Content-Type", "application/json")
	reqStatusCancel.Header.Set("Authorization", "Bearer "+devToken)
	wStatusCancel := httptest.NewRecorder()
	router.ServeHTTP(wStatusCancel, reqStatusCancel)
	if wStatusCancel.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for devotee status=CANCELLED on completed booking, got %d: %s", wStatusCancel.Code, wStatusCancel.Body.String())
	}

	// Admin attempts /admin-cancel -> must return 409 Conflict
	adminCancelPayload := `{"reason":"Admin attempting to cancel completed booking"}`
	reqAdminCancel := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/admin-cancel", bytes.NewBufferString(adminCancelPayload))
	reqAdminCancel.Header.Set("Content-Type", "application/json")
	reqAdminCancel.Header.Set("Authorization", "Bearer "+adminToken)
	wAdminCancel := httptest.NewRecorder()
	router.ServeHTTP(wAdminCancel, reqAdminCancel)
	if wAdminCancel.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for admin cancelling completed booking, got %d: %s", wAdminCancel.Code, wAdminCancel.Body.String())
	}
}

// 5. Challenge: Cancellation authorization - Devotee cannot cancel in ARRIVED or POOJA_STARTED (returns HTTP 403 Forbidden)
func TestAdversarialCancellationAuthorizationDevoteeForbidden(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	bID, _, devToken, prToken, _ := createFSMTestBooking(t, router, storage, "+919777500001", "+919777500002")

	// Advance to ARRIVED
	for _, st := range []string{"ACCEPTED", "IN_TRANSIT", "ARRIVED"} {
		payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bID, st)
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+prToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("failed advance to %s: %s", st, w.Body.String())
		}
	}

	// Devotee cancel in ARRIVED via /cancel -> 403 Forbidden
	cancelPayload := fmt.Sprintf(`{"booking_id":"%s","reason_code":"TOO_LATE"}`, bID)
	reqArr1 := httptest.NewRequest("POST", "/api/v1/bookings/cancel", bytes.NewBufferString(cancelPayload))
	reqArr1.Header.Set("Idempotency-Key", newIdemKey())
	reqArr1.Header.Set("Content-Type", "application/json")
	reqArr1.Header.Set("Authorization", "Bearer "+devToken)
	wArr1 := httptest.NewRecorder()
	router.ServeHTTP(wArr1, reqArr1)
	if wArr1.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for devotee cancel in ARRIVED via /cancel, got %d: %s", wArr1.Code, wArr1.Body.String())
	}

	// Devotee cancel in ARRIVED via /status -> 403 Forbidden
	statusPayload := fmt.Sprintf(`{"booking_id":"%s","status":"CANCELLED"}`, bID)
	reqArr2 := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(statusPayload))
	reqArr2.Header.Set("Content-Type", "application/json")
	reqArr2.Header.Set("Authorization", "Bearer "+devToken)
	wArr2 := httptest.NewRecorder()
	router.ServeHTTP(wArr2, reqArr2)
	if wArr2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for devotee cancel in ARRIVED via /status, got %d: %s", wArr2.Code, wArr2.Body.String())
	}

	// Advance to POOJA_STARTED
	reqAdv := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(fmt.Sprintf(`{"booking_id":"%s","status":"POOJA_STARTED"}`, bID)))
	reqAdv.Header.Set("Content-Type", "application/json")
	reqAdv.Header.Set("Authorization", "Bearer "+prToken)
	wAdv := httptest.NewRecorder()
	router.ServeHTTP(wAdv, reqAdv)
	if wAdv.Code != http.StatusOK {
		t.Fatalf("failed advance to POOJA_STARTED: %s", wAdv.Body.String())
	}

	// Devotee cancel in POOJA_STARTED via /cancel -> 403 Forbidden
	reqPooja1 := httptest.NewRequest("POST", "/api/v1/bookings/cancel", bytes.NewBufferString(cancelPayload))
	reqPooja1.Header.Set("Idempotency-Key", newIdemKey())
	reqPooja1.Header.Set("Content-Type", "application/json")
	reqPooja1.Header.Set("Authorization", "Bearer "+devToken)
	wPooja1 := httptest.NewRecorder()
	router.ServeHTTP(wPooja1, reqPooja1)
	if wPooja1.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for devotee cancel in POOJA_STARTED via /cancel, got %d: %s", wPooja1.Code, wPooja1.Body.String())
	}

	// Devotee cancel in POOJA_STARTED via /status -> 403 Forbidden
	reqPooja2 := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(statusPayload))
	reqPooja2.Header.Set("Content-Type", "application/json")
	reqPooja2.Header.Set("Authorization", "Bearer "+devToken)
	wPooja2 := httptest.NewRecorder()
	router.ServeHTTP(wPooja2, reqPooja2)
	if wPooja2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for devotee cancel in POOJA_STARTED via /status, got %d: %s", wPooja2.Code, wPooja2.Body.String())
	}
}

// 6. Challenge: Slot release atomicity and re-booking verification
func TestAdversarialSlotReleaseAndRebooking(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)

	// Booking 1
	bID1, _, devToken1, prToken, prID := createFSMTestBooking(t, router, storage, "+919777600001", "+919777600002")

	// Priest accepts Booking 1 (locks the slot)
	acceptPayload1 := fmt.Sprintf(`{"booking_id":"%s","status":"ACCEPTED"}`, bID1)
	reqAcc1 := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(acceptPayload1))
	reqAcc1.Header.Set("Content-Type", "application/json")
	reqAcc1.Header.Set("Authorization", "Bearer "+prToken)
	wAcc1 := httptest.NewRecorder()
	router.ServeHTTP(wAcc1, reqAcc1)
	if wAcc1.Code != http.StatusOK {
		t.Fatalf("failed accept booking 1: %s", wAcc1.Body.String())
	}

	// Register Devotee 2 and create Booking 2 with the same priest and default slot
	devID2, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone: "+919777600003", FullName: "Devotee 2", Password: "Password123!", ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("failed registering devotee 2: %v", err)
	}
	catalog, _ := storage.GetCatalog("", "", "en")
	poojaID := catalog[0]["id"].(string)

	bID2, _, err := storage.CreateBooking(models.BookingCreateRequest{
		UserID:       devID2,
		PriestID:     prID,
		PoojaID:      poojaID,
		EventAddress: "Somajiguda, Hyderabad",
		TotalAmount:  3500.0,
		BookingType:  "SCHEDULED",
	})
	if err != nil {
		t.Fatalf("failed creating booking 2: %v", err)
	}

	// Attempt to accept Booking 2 for the same priest and slot while Booking 1 is active -> must fail with 409 Conflict
	acceptPayload2 := fmt.Sprintf(`{"booking_id":"%s","status":"ACCEPTED"}`, bID2)
	reqAcc2 := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(acceptPayload2))
	reqAcc2.Header.Set("Content-Type", "application/json")
	reqAcc2.Header.Set("Authorization", "Bearer "+prToken)
	wAcc2 := httptest.NewRecorder()
	router.ServeHTTP(wAcc2, reqAcc2)
	if wAcc2.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for accepting conflicting slot, got %d: %s", wAcc2.Code, wAcc2.Body.String())
	}

	// Now Devotee 1 cancels Booking 1
	cancelPayload1 := fmt.Sprintf(`{"booking_id":"%s","reason_code":"CHANGE_OF_PLANS"}`, bID1)
	reqCancel1 := httptest.NewRequest("POST", "/api/v1/bookings/cancel", bytes.NewBufferString(cancelPayload1))
	reqCancel1.Header.Set("Idempotency-Key", newIdemKey())
	reqCancel1.Header.Set("Content-Type", "application/json")
	reqCancel1.Header.Set("Authorization", "Bearer "+devToken1)
	wCancel1 := httptest.NewRecorder()
	router.ServeHTTP(wCancel1, reqCancel1)
	if wCancel1.Code != http.StatusOK {
		t.Fatalf("failed cancel booking 1: %s", wCancel1.Body.String())
	}

	// Verify slot is now atomically freed in database
	var isBooked bool
	var bookedBookingID string
	err = storage.DB.QueryRow(`SELECT is_booked, COALESCE(booked_booking_id::text, '') FROM priest_availability_slots WHERE priest_id = $1::uuid`, prID).Scan(&isBooked, &bookedBookingID)
	if err != nil {
		t.Fatalf("failed querying slot after cancellation: %v", err)
	}
	if isBooked || bookedBookingID != "" {
		t.Fatalf("slot should be freed: is_booked=%v, booked_booking_id=%s", isBooked, bookedBookingID)
	}

	// Priest now accepts Booking 2 -> must succeed with 200 OK!
	reqAcc2Retry := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(acceptPayload2))
	reqAcc2Retry.Header.Set("Content-Type", "application/json")
	reqAcc2Retry.Header.Set("Authorization", "Bearer "+prToken)
	wAcc2Retry := httptest.NewRecorder()
	router.ServeHTTP(wAcc2Retry, reqAcc2Retry)
	if wAcc2Retry.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for accept after slot release, got %d: %s", wAcc2Retry.Code, wAcc2Retry.Body.String())
	}

	// Verify slot is now booked by Booking 2
	err = storage.DB.QueryRow(`SELECT is_booked, COALESCE(booked_booking_id::text, '') FROM priest_availability_slots WHERE priest_id = $1::uuid`, prID).Scan(&isBooked, &bookedBookingID)
	if err != nil {
		t.Fatalf("failed querying slot after re-accept: %v", err)
	}
	if !isBooked || bookedBookingID != bID2 {
		t.Fatalf("expected slot to be booked by booking 2 (%s), got is_booked=%v, booked_booking_id=%s", bID2, isBooked, bookedBookingID)
	}
}
