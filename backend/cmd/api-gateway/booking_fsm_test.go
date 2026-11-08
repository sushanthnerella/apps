package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/models"
)

func issueAdminTestToken(t *testing.T, userID, secret string) string {
	now := time.Now()
	claims := auth.CustomClaims{
		UserID:         userID,
		Phone:          "+919999900000",
		Role:           "admin",
		FullName:       "Admin User",
		SessionVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-1 * time.Minute)),
			Issuer:    auth.ExpectedJWTIssuer,
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed signing admin test token: %v", err)
	}
	return tokenString
}

// Helper to create a test booking with priest pre-assigned
func createFSMTestBooking(t *testing.T, router http.Handler, storage interface{
	RegisterDevotee(req models.DevoteeRegistrationRequest) (string, error)
	RegisterPriest(req models.PriestRegistrationRequest) (string, error)
	GetCatalog(city, cat, lang string) ([]map[string]interface{}, error)
}, devPhone, prPhone string) (string, string, string, string, string) {
	s := storage.(interface {
		RegisterDevotee(req models.DevoteeRegistrationRequest) (string, error)
		RegisterPriest(req models.PriestRegistrationRequest) (string, error)
		GetCatalog(city, cat, lang string) ([]map[string]interface{}, error)
		CreateBooking(req models.BookingCreateRequest) (string, string, error)
	})

	devID, err := s.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone: devPhone, FullName: "Devotee FSM", Password: "Password123!", ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("failed registering devotee: %v", err)
	}
	devToken := issueTestToken(t, devID, "devotee", 1)

	prID, err := s.RegisterPriest(models.PriestRegistrationRequest{
		Phone: prPhone, FullName: "Priest FSM", Password: "Password123!",
		Sampradaya: "Smartha", City: "Hyderabad", ConsentAgreed: true,
		VedaShakha: "Rigveda", ExperienceYears: 10,
	})
	if err != nil {
		t.Fatalf("failed registering priest: %v", err)
	}
	prToken := issueTestToken(t, prID, "priest", 1)

	catalog, err := s.GetCatalog("", "", "en")
	if err != nil || len(catalog) == 0 {
		t.Fatalf("failed fetching catalog: %v", err)
	}
	poojaID := catalog[0]["id"].(string)

	bookingID, otp, err := s.CreateBooking(models.BookingCreateRequest{
		UserID:       devID,
		PriestID:     prID,
		PoojaID:      poojaID,
		EventAddress: "Banjara Hills, Hyderabad",
		TotalAmount:  3000.0,
		BookingType:  "SCHEDULED",
	})
	if err != nil {
		t.Fatalf("failed creating booking: %v", err)
	}

	return bookingID, otp, devToken, prToken, prID
}

// 1. Test canonical linear sequence
func TestHTTPCanonicalBookingLifecycle(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	bookingID, otp, _, prToken, _ := createFSMTestBooking(t, router, storage, "+919876540001", "+919876540002")

	// ACCEPTED -> IN_TRANSIT -> ARRIVED -> POOJA_STARTED
	for _, st := range []string{"ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED"} {
		payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bookingID, st)
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+prToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for transition to %s, got %d: %s", st, w.Code, w.Body.String())
		}
	}

	// Completion with valid devotee OTP
	compPayload := fmt.Sprintf(`{"completion_otp":"%s","dakshina_amount":3000.0}`, otp)
	reqComp := httptest.NewRequest("POST", "/api/v1/bookings/"+bookingID+"/complete", bytes.NewBufferString(compPayload))
	reqComp.Header.Set("Content-Type", "application/json")
	reqComp.Header.Set("Authorization", "Bearer "+prToken)
	wComp := httptest.NewRecorder()
	router.ServeHTTP(wComp, reqComp)

	if wComp.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for completion, got %d: %s", wComp.Code, wComp.Body.String())
	}

	details, err := storage.GetBookingDetails(bookingID)
	if err != nil {
		t.Fatalf("failed to fetch details: %v", err)
	}
	if details.Status != "COMPLETED" {
		t.Fatalf("expected status 'COMPLETED', got %s", details.Status)
	}
}

// 2. Test skipped transitions return HTTP 409 Conflict
func TestHTTPSkippedTransitionsReturn409Conflict(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	bookingID, otp, _, prToken, _ := createFSMTestBooking(t, router, storage, "+919876540011", "+919876540012")

	// Booking is currently REQUESTED. Skipping directly to IN_TRANSIT should return 409 Conflict.
	skippedToTransit := fmt.Sprintf(`{"booking_id":"%s","status":"IN_TRANSIT"}`, bookingID)
	req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(skippedToTransit))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+prToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for skipped transition REQUESTED -> IN_TRANSIT, got %d: %s", w.Code, w.Body.String())
	}

	// Skipping to ARRIVED should return 409 Conflict
	skippedToArrived := fmt.Sprintf(`{"booking_id":"%s","status":"ARRIVED"}`, bookingID)
	reqArr := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(skippedToArrived))
	reqArr.Header.Set("Content-Type", "application/json")
	reqArr.Header.Set("Authorization", "Bearer "+prToken)
	wArr := httptest.NewRecorder()
	router.ServeHTTP(wArr, reqArr)

	if wArr.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for skipped transition REQUESTED -> ARRIVED, got %d: %s", wArr.Code, wArr.Body.String())
	}

	// Skipping directly to complete from REQUESTED should return 409 Conflict
	compPayload := fmt.Sprintf(`{"completion_otp":"%s"}`, otp)
	reqComp := httptest.NewRequest("POST", "/api/v1/bookings/"+bookingID+"/complete", bytes.NewBufferString(compPayload))
	reqComp.Header.Set("Content-Type", "application/json")
	reqComp.Header.Set("Authorization", "Bearer "+prToken)
	wComp := httptest.NewRecorder()
	router.ServeHTTP(wComp, reqComp)

	if wComp.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for completing unstarted booking, got %d: %s", wComp.Code, wComp.Body.String())
	}
}

// 3. Test backward transitions return HTTP 409 Conflict
func TestHTTPBackwardTransitionsReturn409Conflict(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	bookingID, _, _, prToken, _ := createFSMTestBooking(t, router, storage, "+919876540021", "+919876540022")

	// Advance to ACCEPTED, then IN_TRANSIT
	for _, st := range []string{"ACCEPTED", "IN_TRANSIT"} {
		payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bookingID, st)
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+prToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("failed transition to %s: %s", st, w.Body.String())
		}
	}

	// Attempt backward transition: IN_TRANSIT -> ACCEPTED -> 409 Conflict
	backwardPayload := fmt.Sprintf(`{"booking_id":"%s","status":"ACCEPTED"}`, bookingID)
	reqBack := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(backwardPayload))
	reqBack.Header.Set("Content-Type", "application/json")
	reqBack.Header.Set("Authorization", "Bearer "+prToken)
	wBack := httptest.NewRecorder()
	router.ServeHTTP(wBack, reqBack)

	if wBack.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for backward transition IN_TRANSIT -> ACCEPTED, got %d: %s", wBack.Code, wBack.Body.String())
	}
}

// 4. Test repeated transitions return HTTP 409 Conflict
func TestHTTPRepeatedTransitionsReturn409Conflict(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	bookingID, _, _, prToken, _ := createFSMTestBooking(t, router, storage, "+919876540031", "+919876540032")

	// Advance to ACCEPTED
	payload := fmt.Sprintf(`{"booking_id":"%s","status":"ACCEPTED"}`, bookingID)
	req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+prToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("failed transition to ACCEPTED: %s", w.Body.String())
	}

	// Attempt repeated transition: ACCEPTED -> ACCEPTED -> 409 Conflict
	reqRepeat := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
	reqRepeat.Header.Set("Content-Type", "application/json")
	reqRepeat.Header.Set("Authorization", "Bearer "+prToken)
	wRepeat := httptest.NewRecorder()
	router.ServeHTTP(wRepeat, reqRepeat)

	if wRepeat.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for repeated transition ACCEPTED -> ACCEPTED, got %d: %s", wRepeat.Code, wRepeat.Body.String())
	}
}

// 5. Test devotee cannot trigger priest transitions (returns HTTP 403 Forbidden)
func TestHTTPDevoteeCannotPerformPriestTransitions(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	bookingID, otp, devToken, _, _ := createFSMTestBooking(t, router, storage, "+919876540041", "+919876540042")

	for _, st := range []string{"ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED"} {
		payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bookingID, st)
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+devToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for devotee performing %s, got %d: %s", st, w.Code, w.Body.String())
		}
	}

	// Devotee attempting completion -> 403 Forbidden
	compPayload := fmt.Sprintf(`{"completion_otp":"%s"}`, otp)
	reqComp := httptest.NewRequest("POST", "/api/v1/bookings/"+bookingID+"/complete", bytes.NewBufferString(compPayload))
	reqComp.Header.Set("Content-Type", "application/json")
	reqComp.Header.Set("Authorization", "Bearer "+devToken)
	wComp := httptest.NewRecorder()
	router.ServeHTTP(wComp, reqComp)

	if wComp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for devotee completion, got %d: %s", wComp.Code, wComp.Body.String())
	}
}

// 6. Test unauthorized priest rejected with HTTP 403 Forbidden
func TestHTTPUnauthorizedPriestRejectedWith403(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	bookingID, _, _, _, _ := createFSMTestBooking(t, router, storage, "+919876540051", "+919876540052")

	// Register a second priest not assigned to this booking
	p2ID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876540053", FullName: "Priest Imposter", Password: "Pass123!",
		Sampradaya: "Smartha", City: "Hyderabad", ConsentAgreed: true,
		VedaShakha: "Rigveda", ExperienceYears: 5,
	})
	p2Token := issueTestToken(t, p2ID, "priest", 1)

	payload := fmt.Sprintf(`{"booking_id":"%s","status":"ACCEPTED"}`, bookingID)
	req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p2Token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for unassigned priest accept, got %d: %s", w.Code, w.Body.String())
	}
}

// 7. Test invalid OTP on completion returns HTTP 409 Conflict
func TestHTTPCompletionBadOTPReturns409Conflict(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	bookingID, realOTP, _, prToken, _ := createFSMTestBooking(t, router, storage, "+919876540061", "+919876540062")

	// Advance to POOJA_STARTED
	for _, st := range []string{"ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED"} {
		payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bookingID, st)
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+prToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("failed transition to %s: %s", st, w.Body.String())
		}
	}

	// Bad OTP -> 409 Conflict
	badCompPayload := `{"completion_otp":"0000","dakshina_amount":3000.0}`
	reqBad := httptest.NewRequest("POST", "/api/v1/bookings/"+bookingID+"/complete", bytes.NewBufferString(badCompPayload))
	reqBad.Header.Set("Content-Type", "application/json")
	reqBad.Header.Set("Authorization", "Bearer "+prToken)
	wBad := httptest.NewRecorder()
	router.ServeHTTP(wBad, reqBad)

	if wBad.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for bad OTP, got %d: %s", wBad.Code, wBad.Body.String())
	}
	if strings.Contains(wBad.Body.String(), realOTP) {
		t.Fatalf("security violation: response leaked secret OTP: %s", wBad.Body.String())
	}

	// Correct OTP -> 200 OK
	goodCompPayload := fmt.Sprintf(`{"completion_otp":"%s","dakshina_amount":3000.0}`, realOTP)
	reqGood := httptest.NewRequest("POST", "/api/v1/bookings/"+bookingID+"/complete", bytes.NewBufferString(goodCompPayload))
	reqGood.Header.Set("Content-Type", "application/json")
	reqGood.Header.Set("Authorization", "Bearer "+prToken)
	wGood := httptest.NewRecorder()
	router.ServeHTTP(wGood, reqGood)

	if wGood.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for correct OTP completion, got %d: %s", wGood.Code, wGood.Body.String())
	}
}

// 8. Test cancellation of completed booking returns HTTP 409 Conflict
func TestHTTPCancellationOfCompletedBookingReturns409Conflict(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	bookingID, otp, devToken, prToken, _ := createFSMTestBooking(t, router, storage, "+919876540071", "+919876540072")

	// Advance through completion
	for _, st := range []string{"ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED"} {
		payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bookingID, st)
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+prToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("failed transition to %s: %s", st, w.Body.String())
		}
	}
	compPayload := fmt.Sprintf(`{"completion_otp":"%s"}`, otp)
	reqComp := httptest.NewRequest("POST", "/api/v1/bookings/"+bookingID+"/complete", bytes.NewBufferString(compPayload))
	reqComp.Header.Set("Content-Type", "application/json")
	reqComp.Header.Set("Authorization", "Bearer "+prToken)
	wComp := httptest.NewRecorder()
	router.ServeHTTP(wComp, reqComp)
	if wComp.Code != http.StatusOK {
		t.Fatalf("failed completion: %s", wComp.Body.String())
	}

	// Devotee attempts cancellation on COMPLETED booking -> 409 Conflict
	cancelPayload := fmt.Sprintf(`{"booking_id":"%s","reason_code":"DEVOTEE_MISTAKE"}`, bookingID)
	reqCancel := httptest.NewRequest("POST", "/api/v1/bookings/cancel", bytes.NewBufferString(cancelPayload))
	reqCancel.Header.Set("Content-Type", "application/json")
	reqCancel.Header.Set("Authorization", "Bearer "+devToken)
	wCancel := httptest.NewRecorder()
	router.ServeHTTP(wCancel, reqCancel)

	if wCancel.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for canceling completed booking, got %d: %s", wCancel.Code, wCancel.Body.String())
	}
}

// 9. Test pilot cancellation policy: Devotee allowed in REQUESTED, ACCEPTED, IN_TRANSIT; forbidden in ARRIVED/POOJA_STARTED
func TestHTTPPilotCancellationPolicy(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)

	// Devotee cancels in REQUESTED -> 200 OK
	t.Run("Cancel in REQUESTED succeeds", func(t *testing.T) {
		bID, _, devToken, _, _ := createFSMTestBooking(t, router, storage, "+919876540081", "+919876540082")
		cancelPayload := fmt.Sprintf(`{"booking_id":"%s","reason_code":"DEVOTEE_CHANGE_MIND"}`, bID)
		req := httptest.NewRequest("POST", "/api/v1/bookings/cancel", bytes.NewBufferString(cancelPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+devToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for cancel in REQUESTED, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Devotee cancels in IN_TRANSIT -> 200 OK (with travel fee deducted)
	t.Run("Cancel in IN_TRANSIT succeeds with travel fee", func(t *testing.T) {
		bID, _, devToken, prToken, _ := createFSMTestBooking(t, router, storage, "+919876540083", "+919876540084")
		for _, st := range []string{"ACCEPTED", "IN_TRANSIT"} {
			payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bID, st)
			req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+prToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("failed transition to %s: %s", st, w.Body.String())
			}
		}

		cancelPayload := fmt.Sprintf(`{"booking_id":"%s","reason_code":"EMERGENCY"}`, bID)
		req := httptest.NewRequest("POST", "/api/v1/bookings/cancel", bytes.NewBufferString(cancelPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+devToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for cancel in IN_TRANSIT, got %d: %s", w.Code, w.Body.String())
		}
		var res map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if res["priest_compensation"] != 150.0 {
			t.Fatalf("expected 150.0 priest compensation, got %v", res["priest_compensation"])
		}
	})

	// Devotee cancels in ARRIVED -> 403 Forbidden
	t.Run("Cancel in ARRIVED rejected with 403", func(t *testing.T) {
		bID, _, devToken, prToken, _ := createFSMTestBooking(t, router, storage, "+919876540085", "+919876540086")
		for _, st := range []string{"ACCEPTED", "IN_TRANSIT", "ARRIVED"} {
			payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bID, st)
			req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+prToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("failed transition to %s: %s", st, w.Body.String())
			}
		}

		cancelPayload := fmt.Sprintf(`{"booking_id":"%s","reason_code":"CHANGE_MIND"}`, bID)
		req := httptest.NewRequest("POST", "/api/v1/bookings/cancel", bytes.NewBufferString(cancelPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+devToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for cancel in ARRIVED, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// 10. Test admin cancellation endpoint POST /api/v1/bookings/{id}/admin-cancel
func TestHTTPAdminCancelEndpoint(t *testing.T) {
	router, storage, testSecret := setupBookingTestGateway(t)
	bID, _, devToken, prToken, _ := createFSMTestBooking(t, router, storage, "+919876540091", "+919876540092")

	// Advance to ARRIVED
	for _, st := range []string{"ACCEPTED", "IN_TRANSIT", "ARRIVED"} {
		payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bID, st)
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+prToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("failed transition to %s: %s", st, w.Body.String())
		}
	}

	adminToken := issueAdminTestToken(t, "admin-user-001", testSecret)

	// Non-admin calling /admin-cancel -> 403 Forbidden
	reqNonAdmin := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/admin-cancel", bytes.NewBufferString(`{"reason":"emergency"}`))
	reqNonAdmin.Header.Set("Content-Type", "application/json")
	reqNonAdmin.Header.Set("Authorization", "Bearer "+devToken)
	wNonAdmin := httptest.NewRecorder()
	router.ServeHTTP(wNonAdmin, reqNonAdmin)
	if wNonAdmin.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-admin calling /admin-cancel, got %d", wNonAdmin.Code)
	}

	// Admin calling with empty reason -> 400 Bad Request
	reqEmptyReason := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/admin-cancel", bytes.NewBufferString(`{"reason":"   "}`))
	reqEmptyReason.Header.Set("Content-Type", "application/json")
	reqEmptyReason.Header.Set("Authorization", "Bearer "+adminToken)
	wEmptyReason := httptest.NewRecorder()
	router.ServeHTTP(wEmptyReason, reqEmptyReason)
	if wEmptyReason.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for empty reason, got %d", wEmptyReason.Code)
	}

	// Admin calling with valid reason -> 200 OK
	validPayload := `{"reason":"Devotee family dispute requiring immediate ritual cancellation"}`
	reqAdmin := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/admin-cancel", bytes.NewBufferString(validPayload))
	reqAdmin.Header.Set("Content-Type", "application/json")
	reqAdmin.Header.Set("Authorization", "Bearer "+adminToken)
	wAdmin := httptest.NewRecorder()
	router.ServeHTTP(wAdmin, reqAdmin)

	if wAdmin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for admin cancellation, got %d: %s", wAdmin.Code, wAdmin.Body.String())
	}

	var res map[string]interface{}
	_ = json.Unmarshal(wAdmin.Body.Bytes(), &res)
	if res["status"] != "CANCELLED" || res["cancelled_by"] != "admin" {
		t.Fatalf("expected status 'CANCELLED' and cancelled_by 'admin', got: %v", res)
	}
}

// 11. Test atomic slot release on cancellation
func TestHTTPAtomicSlotReleaseOnCancellation(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	bID, _, devToken, prToken, prID := createFSMTestBooking(t, router, storage, "+919876540101", "+919876540102")

	// Priest accepts booking (this creates/locks the slot)
	payload := fmt.Sprintf(`{"booking_id":"%s","status":"ACCEPTED"}`, bID)
	req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+prToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("failed accept: %s", w.Body.String())
	}

	// Verify slot is booked in DB
	var isBooked bool
	var bookedBookingID string
	err := storage.DB.QueryRow(`SELECT is_booked, COALESCE(booked_booking_id::text, '') FROM priest_availability_slots WHERE priest_id = $1::uuid AND booked_booking_id = $2::uuid`, prID, bID).Scan(&isBooked, &bookedBookingID)
	if err != nil {
		t.Fatalf("failed checking slot: %v", err)
	}
	if !isBooked || bookedBookingID != bID {
		t.Fatalf("expected slot to be booked by booking %s, got is_booked=%v, booking_id=%s", bID, isBooked, bookedBookingID)
	}

	// Devotee cancels booking
	cancelPayload := fmt.Sprintf(`{"booking_id":"%s","reason_code":"CHANGE_PLANS"}`, bID)
	reqCancel := httptest.NewRequest("POST", "/api/v1/bookings/cancel", bytes.NewBufferString(cancelPayload))
	reqCancel.Header.Set("Content-Type", "application/json")
	reqCancel.Header.Set("Authorization", "Bearer "+devToken)
	wCancel := httptest.NewRecorder()
	router.ServeHTTP(wCancel, reqCancel)
	if wCancel.Code != http.StatusOK {
		t.Fatalf("failed cancel: %s", wCancel.Body.String())
	}

	// Verify slot is now atomically freed
	err = storage.DB.QueryRow(`SELECT is_booked, COALESCE(booked_booking_id::text, '') FROM priest_availability_slots WHERE priest_id = $1::uuid AND id = (SELECT id FROM priest_availability_slots WHERE priest_id = $1::uuid LIMIT 1)`, prID).Scan(&isBooked, &bookedBookingID)
	if err != nil {
		t.Fatalf("failed checking slot after cancellation: %v", err)
	}
	if isBooked || bookedBookingID != "" {
		t.Fatalf("expected slot to be unbooked and null booking_id, got is_booked=%v, booking_id=%s", isBooked, bookedBookingID)
	}
}

// 12. Test concurrent completion race condition: exactly 1 succeeds, remaining fail with 409 Conflict
func TestHTTPConcurrentCompletionRaceCondition(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	bID, otp, _, prToken, _ := createFSMTestBooking(t, router, storage, "+919876540111", "+919876540112")

	// Advance to POOJA_STARTED
	for _, st := range []string{"ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED"} {
		payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bID, st)
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+prToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("failed transition to %s: %s", st, w.Body.String())
		}
	}

	// Concurrently invoke /complete with valid OTP from 5 goroutines
	concurrency := 5
	var successCount int32
	var conflictCount int32
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			compPayload := fmt.Sprintf(`{"completion_otp":"%s","dakshina_amount":3000.0}`, otp)
			req := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/complete", bytes.NewBufferString(compPayload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+prToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				atomic.AddInt32(&successCount, 1)
			} else if w.Code == http.StatusConflict {
				atomic.AddInt32(&conflictCount, 1)
			}
		}()
	}
	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful completion, got %d", successCount)
	}
	if conflictCount != int32(concurrency-1) {
		t.Fatalf("expected %d 409 Conflict responses, got %d", concurrency-1, conflictCount)
	}
}
