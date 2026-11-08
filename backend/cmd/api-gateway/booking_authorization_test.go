package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/models"
)

func setupBookingTestGateway(t *testing.T) (*mux.Router, *db.Storage, string) {
	testSecret := "purohit_platform_test_jwt_secret_key_32bytes_long!"
	_ = auth.InitJWT(testSecret)

	storage, _, _ := db.CreateIsolatedAuthTestDBForTest(t)

	router := mux.NewRouter()
	registerBookingRoutes(router, storage)
	registerChatRoutes(router, storage)

	return router, storage, testSecret
}

func issueTestToken(t *testing.T, userID, role string, sessionVersion int) string {
	token, err := auth.GenerateToken(userID, "+919999900000", role, "Test User", sessionVersion)
	if err != nil {
		t.Fatalf("failed generating test token: %v", err)
	}
	return token
}

func TestUnauthenticatedBookingEndpointsRejectedWith401(t *testing.T) {
	router, _, _ := setupBookingTestGateway(t)

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"POST", "/api/v1/bookings", `{"pooja_id":"test"}`},
		{"GET", "/api/v1/bookings", ""},
		{"GET", "/api/v1/priests/test-priest-id/bookings", ""},
		{"GET", "/api/v1/bookings/test-booking-id", ""},
		{"GET", "/api/v1/bookings/test-booking-id/otp", ""},
		{"POST", "/api/v1/bookings/status", `{"booking_id":"test","status":"ACCEPTED"}`},
		{"POST", "/api/v1/bookings/test-booking-id/complete", `{"completion_otp":"1234"}`},
		{"GET", "/api/v1/bookings/test-booking-id/materials", ""},
		{"POST", "/api/v1/bookings/test-booking-id/materials", `{"materials":[]}`},
		{"GET", "/api/v1/bookings/test-booking-id/cancel-quote", ""},
		{"POST", "/api/v1/bookings/cancel", `{"booking_id":"test","reason_code":"DEVOTEE_CANCELLED"}`},
		{"POST", "/api/v1/bookings/refer", `{"booking_id":"test","referred_priest_id":"colleague"}`},
		{"POST", "/api/v1/chat/rooms", `{"booking_id":"test"}`},
		{"GET", "/api/v1/chat/room-by-booking?booking_id=test", ""},
		{"GET", "/api/v1/chat/messages?room_id=test", ""},
		{"POST", "/api/v1/chat/messages", `{"room_id":"test","content":"hi"}`},
		{"GET", "/ws/chat?roomId=test", ""},
	}

	for _, ep := range endpoints {
		t.Run(fmt.Sprintf("%s %s missing token", ep.method, ep.path), func(t *testing.T) {
			var bodyBuf *bytes.Buffer
			if ep.body != "" {
				bodyBuf = bytes.NewBufferString(ep.body)
			} else {
				bodyBuf = bytes.NewBuffer(nil)
			}
			req := httptest.NewRequest(ep.method, ep.path, bodyBuf)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 Unauthorized, got %d: %s", w.Code, w.Body.String())
			}
		})

		t.Run(fmt.Sprintf("%s %s invalid token", ep.method, ep.path), func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer invalid-garbage-token")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 Unauthorized for invalid token, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestRevokedSessionRejectedWith401(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)

	// Register devotee (session_version = 1)
	devoteeID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876500101",
		FullName:      "Session Revoke Devotee",
		Password:      "TestPass123!",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("failed registering devotee: %v", err)
	}

	// Issue token with session_version = 1
	tokenV1 := issueTestToken(t, devoteeID, "devotee", 1)

	// Invalidate session by incrementing session_version in database
	err = storage.RevokeUserSessions(devoteeID, "devotee")
	if err != nil {
		t.Fatalf("failed revoking user sessions: %v", err)
	}

	// Any booking request using tokenV1 must be rejected with 401
	req := httptest.NewRequest("GET", "/api/v1/bookings", nil)
	req.Header.Set("Authorization", "Bearer "+tokenV1)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for revoked session, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "expired or revoked") {
		t.Fatalf("expected 'expired or revoked' message, got: %s", w.Body.String())
	}
}

func TestRoleEnforcementAcrossEndpoints(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)

	// Create Devotee
	devoteeID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876500201",
		FullName:      "Role Devotee",
		Password:      "TestPass123!",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("failed registering devotee: %v", err)
	}
	devoteeToken := issueTestToken(t, devoteeID, "devotee", 1)

	// Create Priest
	priestID, err := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone:            "+919876500202",
		FullName:         "Role Priest",
		Password:         "TestPass123!",
		Sampradaya:       "Smartha",
		VedaShakha:       "Rigveda",
		ExperienceYears:  10,
		City:             "Hyderabad",
		ConsentAgreed:    true,
	})
	if err != nil {
		t.Fatalf("failed registering priest: %v", err)
	}
	priestToken := issueTestToken(t, priestID, "priest", 1)

	// 1. Priest attempts to create a booking -> 403 Forbidden
	t.Run("Priest cannot create booking", func(t *testing.T) {
		createPayload := `{"total_amount":1500.0}`
		req := httptest.NewRequest("POST", "/api/v1/bookings", bytes.NewBufferString(createPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+priestToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "only devotees can create bookings") {
			t.Fatalf("expected 'only devotees can create bookings', got: %s", w.Body.String())
		}
	})

	// 2. Devotee attempts to access priest bookings query route -> 403 Forbidden
	t.Run("Devotee cannot access priest bookings list", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/priests/"+priestID+"/bookings", nil)
		req.Header.Set("Authorization", "Bearer "+devoteeToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "only priests can access this endpoint") {
			t.Fatalf("expected 'only priests can access this endpoint', got: %s", w.Body.String())
		}
	})

	// 3. Devotee creates a legitimate booking
	createReq := models.BookingCreateRequest{
		UserID:            devoteeID,
		PriestID:          priestID,
		TotalAmount:       2500.0,
		EventAddress:      "Test Temple Road, Hyderabad",
		ScheduledDate:     "2026-12-01",
		SlotTime:          "09:00 AM",
		DevoteeGotram:     "Kashyapa",
		DevoteeNakshatram: "Rohini",
	}
	bookingID, _, err := storage.CreateBooking(createReq)
	if err != nil {
		t.Fatalf("failed creating test booking: %v", err)
	}

	// 4. Priest attempts to retrieve completion OTP -> 403 Forbidden
	t.Run("Priest cannot retrieve completion OTP", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/bookings/"+bookingID+"/otp", nil)
		req.Header.Set("Authorization", "Bearer "+priestToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "priests cannot retrieve the completion OTP") {
			t.Fatalf("expected 'priests cannot retrieve the completion OTP', got: %s", w.Body.String())
		}
	})

	// 5. Devotee attempts to complete booking directly -> 403 Forbidden
	t.Run("Devotee cannot complete booking", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/bookings/"+bookingID+"/complete", bytes.NewBufferString(`{"completion_otp":"1234"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+devoteeToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 6. Devotee attempts to refer booking -> 403 Forbidden
	t.Run("Devotee cannot refer booking", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/bookings/refer", bytes.NewBufferString(`{"booking_id":"`+bookingID+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+devoteeToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestCrossAccountAndUnassignedPriestRejection(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)

	// Devotee 1 & Devotee 2
	d1ID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone: "+919876500301", FullName: "Devotee One", Password: "Pass123!", ConsentAgreed: true,
	})
	d1Token := issueTestToken(t, d1ID, "devotee", 1)

	d2ID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone: "+919876500302", FullName: "Devotee Two", Password: "Pass123!", ConsentAgreed: true,
	})
	d2Token := issueTestToken(t, d2ID, "devotee", 1)

	// Priest 1 & Priest 2
	p1ID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876500303", FullName: "Priest One", Password: "Pass123!", Sampradaya: "Smartha", ExperienceYears: 5, City: "Hyderabad", ConsentAgreed: true,
	})
	p1Token := issueTestToken(t, p1ID, "priest", 1)

	p2ID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876500304", FullName: "Priest Two", Password: "Pass123!", Sampradaya: "Vaishnava", ExperienceYears: 8, City: "Hyderabad", ConsentAgreed: true,
	})
	p2Token := issueTestToken(t, p2ID, "priest", 1)

	// Devotee 1 creates Booking B1 assigned to Priest 1
	b1ID, _, err := storage.CreateBooking(models.BookingCreateRequest{
		UserID:       d1ID,
		PriestID:     p1ID,
		TotalAmount:  3000.0,
		EventAddress: "Devotee One Home, Hyderabad",
	})
	if err != nil {
		t.Fatalf("failed creating booking B1: %v", err)
	}

	// 0. Legitimate Devotee 1 and Priest 1 can view booking details
	t.Run("Owner Devotee 1 can access booking details", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/bookings/"+b1ID, nil)
		req.Header.Set("Authorization", "Bearer "+d1Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Assigned Priest 1 can access booking details", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/bookings/"+b1ID, nil)
		req.Header.Set("Authorization", "Bearer "+p1Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 1. Devotee 2 attempts to view Devotee 1's booking details -> 403 Forbidden
	t.Run("Cross-devotee booking details rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/bookings/"+b1ID, nil)
		req.Header.Set("Authorization", "Bearer "+d2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "you are not the owner of this booking") {
			t.Fatalf("expected owner error, got: %s", w.Body.String())
		}
	})

	// 2. Devotee 2 attempts to retrieve Devotee 1's OTP -> 403 Forbidden
	t.Run("Cross-devotee OTP retrieval rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/bookings/"+b1ID+"/otp", nil)
		req.Header.Set("Authorization", "Bearer "+d2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 3. Devotee 2 attempts to query bookings for Devotee 1 (?user_id=d1ID) -> 403 Forbidden
	t.Run("Cross-devotee list query rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/bookings?user_id="+d1ID, nil)
		req.Header.Set("Authorization", "Bearer "+d2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 4. Priest 2 (unassigned) attempts to view Priest 1's booking details -> 403 Forbidden
	t.Run("Unassigned priest viewing booking details rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/bookings/"+b1ID, nil)
		req.Header.Set("Authorization", "Bearer "+p2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "you are not assigned to this booking") {
			t.Fatalf("expected unassigned error, got: %s", w.Body.String())
		}
	})

	// 5. Priest 2 (unassigned) attempts to update status -> 403 Forbidden
	t.Run("Unassigned priest status update rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(`{"booking_id":"`+b1ID+`","status":"IN_TRANSIT"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 6. Priest 2 (unassigned) attempts to complete booking -> 403 Forbidden
	t.Run("Unassigned priest completion rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/bookings/"+b1ID+"/complete", bytes.NewBufferString(`{"completion_otp":"1234"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 7. Priest 2 (unassigned) attempts to refer booking -> 403 Forbidden
	t.Run("Unassigned priest refer rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/bookings/refer", bytes.NewBufferString(`{"booking_id":"`+b1ID+`","referred_priest_id":"`+p2ID+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 8. Priest 2 attempts to query Priest 1's endpoint -> 403 Forbidden
	t.Run("Cross-priest list query rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/priests/"+p1ID+"/bookings", nil)
		req.Header.Set("Authorization", "Bearer "+p2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 9. Priest 2 attempts to pass ?priest_id=p1ID to /api/v1/bookings -> 403 Forbidden
	t.Run("Cross-priest parameter query rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/bookings?priest_id="+p1ID, nil)
		req.Header.Set("Authorization", "Bearer "+p2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestAsymmetricCompletionOTPPrivacy(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)

	// Devotee and Priest
	devoteeID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone: "+919876500401", FullName: "OTP Devotee", Password: "Pass123!", ConsentAgreed: true,
	})
	devoteeToken := issueTestToken(t, devoteeID, "devotee", 1)

	priestID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876500402", FullName: "OTP Priest", Password: "Pass123!", Sampradaya: "Smartha", ExperienceYears: 7, City: "Hyderabad", ConsentAgreed: true,
	})
	priestToken := issueTestToken(t, priestID, "priest", 1)

	// Create Booking
	bookingID, secretOTP, err := storage.CreateBooking(models.BookingCreateRequest{
		UserID:       devoteeID,
		PriestID:     priestID,
		TotalAmount:  3500.0,
		EventAddress: "Privacy Test House, Hyderabad",
	})
	if err != nil {
		t.Fatalf("failed creating booking: %v", err)
	}

	// 1. Verify that GET /api/v1/bookings/{id} does NOT leak completion_otp for devotee
	t.Run("Devotee booking details omits completion_otp", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/bookings/"+bookingID, nil)
		req.Header.Set("Authorization", "Bearer "+devoteeToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), secretOTP) || strings.Contains(w.Body.String(), "completion_otp") {
			t.Fatalf("completion_otp was leaked in GET /api/v1/bookings/{id}: %s", w.Body.String())
		}
	})

	// 2. Verify that GET /api/v1/bookings/{id} does NOT leak completion_otp for priest
	t.Run("Priest booking details omits completion_otp", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/bookings/"+bookingID, nil)
		req.Header.Set("Authorization", "Bearer "+priestToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), secretOTP) || strings.Contains(w.Body.String(), "completion_otp") {
			t.Fatalf("completion_otp was leaked in GET /api/v1/bookings/{id} for priest: %s", w.Body.String())
		}
	})

	// 3. Verify that shared list queries omit completion_otp
	t.Run("Shared bookings list omits completion_otp for devotee", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/bookings", nil)
		req.Header.Set("Authorization", "Bearer "+devoteeToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), secretOTP) || strings.Contains(w.Body.String(), "completion_otp") {
			t.Fatalf("completion_otp was leaked in GET /api/v1/bookings list: %s", w.Body.String())
		}
	})

	t.Run("Shared bookings list omits completion_otp for priest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/bookings", nil)
		req.Header.Set("Authorization", "Bearer "+priestToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), secretOTP) || strings.Contains(w.Body.String(), "completion_otp") {
			t.Fatalf("completion_otp was leaked in GET /api/v1/bookings list for priest: %s", w.Body.String())
		}
	})

	// 4. Verify that GET /api/v1/priests/{id}/bookings omits completion_otp
	t.Run("Priest bookings endpoint omits completion_otp", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/priests/"+priestID+"/bookings", nil)
		req.Header.Set("Authorization", "Bearer "+priestToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), secretOTP) || strings.Contains(w.Body.String(), "completion_otp") {
			t.Fatalf("completion_otp was leaked in GET /api/v1/priests/{id}/bookings: %s", w.Body.String())
		}
	})

	// 5. Exclusively provide completion OTP via protected route to authenticated devotee owner
	t.Run("Devotee owner retrieves completion OTP via protected route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/bookings/"+bookingID+"/otp", nil)
		req.Header.Set("Authorization", "Bearer "+devoteeToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var otpRes models.BookingOTPResponse
		if err := json.Unmarshal(w.Body.Bytes(), &otpRes); err != nil {
			t.Fatalf("failed parsing OTP response: %v", err)
		}
		if otpRes.CompletionOTP != secretOTP {
			t.Fatalf("expected OTP %s, got %s", secretOTP, otpRes.CompletionOTP)
		}
		if otpRes.BookingID != bookingID {
			t.Fatalf("expected booking_id %s, got %s", bookingID, otpRes.BookingID)
		}
	})
}

func TestLegitimateBookingLifecycleAndCompletion(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)

	// Devotee and Priest
	devoteeID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone: "+919876500501", FullName: "Flow Devotee", Password: "Pass123!", ConsentAgreed: true,
	})
	devoteeToken := issueTestToken(t, devoteeID, "devotee", 1)

	priestID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876500502", FullName: "Flow Priest", Password: "Pass123!", Sampradaya: "Smartha", ExperienceYears: 12, City: "Hyderabad", ConsentAgreed: true,
	})
	priestToken := issueTestToken(t, priestID, "priest", 1)

	// 1. Devotee creates booking
	createPayload := fmt.Sprintf(`{
		"priest_id": "%s",
		"total_amount": 4500.0,
		"event_address": "Flat 302, Vedic Residency, Hyderabad",
		"scheduled_date": "2026-11-20",
		"slot_time": "10:00 AM",
		"devotee_gotram": "Bharadwaja",
		"devotee_nakshatram": "Pushya"
	}`, priestID)
	reqCreate := httptest.NewRequest("POST", "/api/v1/bookings", bytes.NewBufferString(createPayload))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("Authorization", "Bearer "+devoteeToken)
	wCreate := httptest.NewRecorder()
	router.ServeHTTP(wCreate, reqCreate)

	if wCreate.Code != http.StatusOK {
		t.Fatalf("booking creation failed, code %d: %s", wCreate.Code, wCreate.Body.String())
	}
	var createdRes map[string]interface{}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &createdRes)
	bookingID := createdRes["booking_id"].(string)
	initialOTP := createdRes["completion_otp"].(string)

	if len(initialOTP) != 4 {
		t.Fatalf("expected 4-digit OTP, got %s", initialOTP)
	}

	// 2. Devotee verifies OTP matches protected endpoint
	reqOTP := httptest.NewRequest("GET", "/api/v1/bookings/"+bookingID+"/otp", nil)
	reqOTP.Header.Set("Authorization", "Bearer "+devoteeToken)
	wOTP := httptest.NewRecorder()
	router.ServeHTTP(wOTP, reqOTP)

	if wOTP.Code != http.StatusOK {
		t.Fatalf("OTP fetch failed, code %d: %s", wOTP.Code, wOTP.Body.String())
	}
	var otpRes models.BookingOTPResponse
	_ = json.Unmarshal(wOTP.Body.Bytes(), &otpRes)
	if otpRes.CompletionOTP != initialOTP {
		t.Fatalf("expected OTP %s, got %s", initialOTP, otpRes.CompletionOTP)
	}

	// 3. Priest transitions status: ACCEPTED -> IN_TRANSIT -> ARRIVED -> POOJA_STARTED
	for _, st := range []string{"ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED"} {
		statusPayload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bookingID, st)
		reqSt := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(statusPayload))
		reqSt.Header.Set("Content-Type", "application/json")
		reqSt.Header.Set("Authorization", "Bearer "+priestToken)
		wSt := httptest.NewRecorder()
		router.ServeHTTP(wSt, reqSt)

		if wSt.Code != http.StatusOK {
			t.Fatalf("failed transition to %s, code %d: %s", st, wSt.Code, wSt.Body.String())
		}
	}

	// 4. Priest attempts completion with incorrect OTP -> 409 Conflict
	badOTPPayload := fmt.Sprintf(`{"booking_id":"%s","status":"COMPLETED","completion_otp":"9999"}`, bookingID)
	reqBad := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(badOTPPayload))
	reqBad.Header.Set("Content-Type", "application/json")
	reqBad.Header.Set("Authorization", "Bearer "+priestToken)
	wBad := httptest.NewRecorder()
	router.ServeHTTP(wBad, reqBad)

	if wBad.Code != http.StatusConflict && wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 409 Conflict (or 400) for incorrect OTP, got %d: %s", wBad.Code, wBad.Body.String())
	}
	if !strings.Contains(wBad.Body.String(), "invalid completion OTP") {
		t.Fatalf("expected 'invalid completion OTP', got: %s", wBad.Body.String())
	}
	// Verify that error message never leaked the secret OTP
	if strings.Contains(wBad.Body.String(), initialOTP) {
		t.Fatalf("error message leaked the expected OTP: %s", wBad.Body.String())
	}

	// 5. Priest completes booking with correct OTP via POST /api/v1/bookings/{id}/complete
	completePayload := fmt.Sprintf(`{"completion_otp":"%s","dakshina_amount":4500.0}`, initialOTP)
	reqComplete := httptest.NewRequest("POST", "/api/v1/bookings/"+bookingID+"/complete", bytes.NewBufferString(completePayload))
	reqComplete.Header.Set("Content-Type", "application/json")
	reqComplete.Header.Set("Authorization", "Bearer "+priestToken)
	wComplete := httptest.NewRecorder()
	router.ServeHTTP(wComplete, reqComplete)

	if wComplete.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid completion, got %d: %s", wComplete.Code, wComplete.Body.String())
	}

	var completeRes map[string]interface{}
	_ = json.Unmarshal(wComplete.Body.Bytes(), &completeRes)
	if completeRes["escrow_status"] != "RELEASED_TO_PRIEST" {
		t.Fatalf("expected escrow_status 'RELEASED_TO_PRIEST', got %v", completeRes["escrow_status"])
	}

	// 6. Verify final booking status in database
	details, err := storage.GetBookingDetails(bookingID)
	if err != nil {
		t.Fatalf("failed fetching details: %v", err)
	}
	if details.Status != "COMPLETED" {
		t.Fatalf("expected status 'COMPLETED', got %s", details.Status)
	}
	if details.EscrowStatus != "RELEASED_TO_PRIEST" {
		t.Fatalf("expected escrow_status 'RELEASED_TO_PRIEST', got %s", details.EscrowStatus)
	}
}

// 7. Unassigned priest cannot accept or reject a booking merely by knowing its ID
func TestUnassignedPriestCannotAcceptOrRejectBookingMerelyKnowingID(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)

	// Devotee creates an unassigned booking (priest_id is empty)
	devoteeID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone: "+919876599001", FullName: "Devotee Unassigned", Password: "Pass123!", ConsentAgreed: true,
	})
	catalog, _ := storage.GetCatalog("", "", "en")
	poojaID := catalog[0]["id"].(string)

	bookingID, _, err := storage.CreateBooking(models.BookingCreateRequest{
		UserID:       devoteeID,
		PoojaID:      poojaID,
		EventAddress: "Plot 10, Banjara Hills, Hyderabad",
		TotalAmount:  2500.0,
		BookingType:  "INSTANT_DISPATCH",
		PriestID:     "", // Unassigned
	})
	if err != nil {
		t.Fatalf("failed creating unassigned booking: %v", err)
	}

	// Unassigned priest attempts to interact with booking
	priestID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876599002", FullName: "Priest Unassigned", Password: "Pass123!", Sampradaya: "Smartha", City: "Hyderabad", ConsentAgreed: true,
		VedaShakha: "Rigveda", ExperienceYears: 8,
	})
	pToken := issueTestToken(t, priestID, "priest", 1)

	// 1. Attempt to ACCEPT -> 403 Forbidden
	t.Run("Unassigned priest accept rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(
			fmt.Sprintf(`{"booking_id":"%s","status":"ACCEPTED"}`, bookingID),
		))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+pToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for unassigned priest accept, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "you are not assigned to this booking") {
			t.Fatalf("expected assignment error message, got: %s", w.Body.String())
		}
	})

	// 2. Attempt to REJECT -> 403 Forbidden
	t.Run("Unassigned priest reject rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(
			fmt.Sprintf(`{"booking_id":"%s","status":"REJECTED"}`, bookingID),
		))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+pToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for unassigned priest reject, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 3. Attempt to transition to IN_TRANSIT -> 403 Forbidden
	t.Run("Unassigned priest transition rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(
			fmt.Sprintf(`{"booking_id":"%s","status":"IN_TRANSIT"}`, bookingID),
		))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+pToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for unassigned priest transition, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 4. Attempt to complete -> 403 Forbidden
	t.Run("Unassigned priest complete rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/bookings/"+bookingID+"/complete", bytes.NewBufferString(
			`{"completion_otp":"1234"}`,
		))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+pToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for unassigned priest complete, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 5. Verify database booking state remains unmodified
	_, assignedPID, status, _, err := storage.GetBookingOwner(bookingID)
	if err != nil {
		t.Fatalf("failed fetching booking owner: %v", err)
	}
	if assignedPID != "" {
		t.Fatalf("booking priest_id should remain unassigned, got: %s", assignedPID)
	}
	if status != "REQUESTED" {
		t.Fatalf("booking status should remain REQUESTED, got: %s", status)
	}
}

// 8. Devotee status request cannot set or change priest_id; devotee cannot perform priest transitions
func TestDevoteeStatusRequestCannotSetOrChangePriestID(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)

	devoteeID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone: "+919876599011", FullName: "Devotee Owner", Password: "Pass123!", ConsentAgreed: true,
	})
	dToken := issueTestToken(t, devoteeID, "devotee", 1)

	priest1ID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876599012", FullName: "Priest One", Password: "Pass123!", Sampradaya: "Smartha", City: "Hyderabad", ConsentAgreed: true,
		VedaShakha: "Rigveda", ExperienceYears: 10,
	})
	priest2ID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876599013", FullName: "Priest Two", Password: "Pass123!", Sampradaya: "Vaishnava", City: "Hyderabad", ConsentAgreed: true,
		VedaShakha: "Yajurveda", ExperienceYears: 5,
	})

	catalog, _ := storage.GetCatalog("", "", "en")
	poojaID := catalog[0]["id"].(string)

	bookingID, _, _ := storage.CreateBooking(models.BookingCreateRequest{
		UserID:       devoteeID,
		PriestID:     priest1ID,
		PoojaID:      poojaID,
		EventAddress: "Hyderabad",
		TotalAmount:  3000.0,
	})

	// 1. Devotee attempts priest transition (ACCEPTED) -> 403 Forbidden
	t.Run("Devotee cannot transition to ACCEPTED", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(
			fmt.Sprintf(`{"booking_id":"%s","status":"ACCEPTED"}`, bookingID),
		))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+dToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "devotees can only cancel bookings") {
			t.Fatalf("expected devotee transition restriction error, got: %s", w.Body.String())
		}
	})

	// 2. Devotee attempts priest transition (COMPLETED) -> 403 Forbidden
	t.Run("Devotee cannot transition to COMPLETED", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(
			fmt.Sprintf(`{"booking_id":"%s","status":"COMPLETED"}`, bookingID),
		))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+dToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 3. Devotee attempts to inject priest_id in status request -> ignored and forbidden to reassign
	t.Run("Devotee status request cannot change priest_id", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(
			fmt.Sprintf(`{"booking_id":"%s","status":"CANCELLED","priest_id":"%s"}`, bookingID, priest2ID),
		))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+dToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for cancellation, got %d: %s", w.Code, w.Body.String())
		}

		// Verify database: priest_id MUST still be priest1ID
		_, assignedPID, status, _, err := storage.GetBookingOwner(bookingID)
		if err != nil {
			t.Fatalf("failed fetching owner: %v", err)
		}
		if assignedPID != priest1ID {
			t.Fatalf("devotee payload modified priest_id! Expected %s, got %s", priest1ID, assignedPID)
		}
		if status != "CANCELLED" {
			t.Fatalf("expected status CANCELLED, got: %s", status)
		}
	})
}

// 9. Reassignment race protection: Old priest cannot transition, complete or re-refer after referral
func TestReassignmentRaceProtectionInDatabaseMutation(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)

	devoteeID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone: "+919876599021", FullName: "Devotee Race", Password: "Pass123!", ConsentAgreed: true,
	})
	p1ID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876599022", FullName: "Priest Orig", Password: "Pass123!", Sampradaya: "Smartha", City: "Hyderabad", ConsentAgreed: true,
		VedaShakha: "Rigveda", ExperienceYears: 12,
	})
	p2ID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876599023", FullName: "Priest Referred", Password: "Pass123!", Sampradaya: "Samaveda", City: "Hyderabad", ConsentAgreed: true,
		VedaShakha: "Samaveda", ExperienceYears: 7,
	})

	p1Token := issueTestToken(t, p1ID, "priest", 1)
	p2Token := issueTestToken(t, p2ID, "priest", 1)

	catalog, _ := storage.GetCatalog("", "", "en")
	poojaID := catalog[0]["id"].(string)

	bookingID, otp, _ := storage.CreateBooking(models.BookingCreateRequest{
		UserID:       devoteeID,
		PriestID:     p1ID,
		PoojaID:      poojaID,
		EventAddress: "Hyderabad",
		TotalAmount:  3500.0,
	})

	// Priest 1 accepts booking
	reqAccept := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(
		fmt.Sprintf(`{"booking_id":"%s","status":"ACCEPTED"}`, bookingID),
	))
	reqAccept.Header.Set("Content-Type", "application/json")
	reqAccept.Header.Set("Authorization", "Bearer "+p1Token)
	wAccept := httptest.NewRecorder()
	router.ServeHTTP(wAccept, reqAccept)
	if wAccept.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for accept, got %d: %s", wAccept.Code, wAccept.Body.String())
	}

	// Priest 1 refers booking to Priest 2
	referPayload := fmt.Sprintf(`{"booking_id":"%s","referring_priest_id":"%s","referred_priest_id":"%s","reason":"Health emergency"}`,
		bookingID, p1ID, p2ID)
	reqRefer := httptest.NewRequest("POST", "/api/v1/bookings/refer", bytes.NewBufferString(referPayload))
	reqRefer.Header.Set("Content-Type", "application/json")
	reqRefer.Header.Set("Authorization", "Bearer "+p1Token)
	wRefer := httptest.NewRecorder()
	router.ServeHTTP(wRefer, reqRefer)

	if wRefer.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for referral, got %d: %s", wRefer.Code, wRefer.Body.String())
	}

	// Now Priest 1 (no longer assigned) attempts status update -> must fail
	t.Run("Old priest status update rejected after referral", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(
			fmt.Sprintf(`{"booking_id":"%s","status":"IN_TRANSIT"}`, bookingID),
		))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p1Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for old priest status update, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Priest 1 attempts completion with OTP -> must fail
	t.Run("Old priest completion rejected after referral", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/bookings/"+bookingID+"/complete", bytes.NewBufferString(
			fmt.Sprintf(`{"completion_otp":"%s"}`, otp),
		))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p1Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for old priest completion, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Priest 1 attempts another referral -> must fail
	t.Run("Old priest re-referral rejected after referral", func(t *testing.T) {
		reReferPayload := fmt.Sprintf(`{"booking_id":"%s","referred_priest_id":"%s"}`, bookingID, p2ID)
		req := httptest.NewRequest("POST", "/api/v1/bookings/refer", bytes.NewBufferString(reReferPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p1Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for old priest re-referral, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Newly assigned Priest 2 can successfully update status and complete
	t.Run("New assigned priest successfully updates and completes", func(t *testing.T) {
		reqTransit := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(
			fmt.Sprintf(`{"booking_id":"%s","status":"IN_TRANSIT"}`, bookingID),
		))
		reqTransit.Header.Set("Content-Type", "application/json")
		reqTransit.Header.Set("Authorization", "Bearer "+p2Token)
		wTransit := httptest.NewRecorder()
		router.ServeHTTP(wTransit, reqTransit)

		if wTransit.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for new priest transit, got %d: %s", wTransit.Code, wTransit.Body.String())
		}

		for _, st := range []string{"ARRIVED", "POOJA_STARTED"} {
			reqSt := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(
				fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bookingID, st),
			))
			reqSt.Header.Set("Content-Type", "application/json")
			reqSt.Header.Set("Authorization", "Bearer "+p2Token)
			wSt := httptest.NewRecorder()
			router.ServeHTTP(wSt, reqSt)
			if wSt.Code != http.StatusOK {
				t.Fatalf("expected 200 OK for new priest %s, got %d: %s", st, wSt.Code, wSt.Body.String())
			}
		}

		reqComp := httptest.NewRequest("POST", "/api/v1/bookings/"+bookingID+"/complete", bytes.NewBufferString(
			fmt.Sprintf(`{"completion_otp":"%s"}`, otp),
		))
		reqComp.Header.Set("Content-Type", "application/json")
		reqComp.Header.Set("Authorization", "Bearer "+p2Token)
		wComp := httptest.NewRecorder()
		router.ServeHTTP(wComp, reqComp)

		if wComp.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for new priest completion, got %d: %s", wComp.Code, wComp.Body.String())
		}
	})
}

// 10. Concurrent referral race condition: exactly 1 succeeds, others fail atomically
func TestConcurrentReferralRaceCondition(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)

	devoteeID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone: "+919876599031", FullName: "Devotee Conc", Password: "Pass123!", ConsentAgreed: true,
	})
	p1ID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876599032", FullName: "Priest Master", Password: "Pass123!", Sampradaya: "Smartha", City: "Hyderabad", ConsentAgreed: true,
		VedaShakha: "Rigveda", ExperienceYears: 15,
	})
	p2ID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876599033", FullName: "Colleague Two", Password: "Pass123!", Sampradaya: "Yajurveda", City: "Hyderabad", ConsentAgreed: true,
		VedaShakha: "Yajurveda", ExperienceYears: 5,
	})
	p3ID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876599034", FullName: "Colleague Three", Password: "Pass123!", Sampradaya: "Samaveda", City: "Hyderabad", ConsentAgreed: true,
		VedaShakha: "Samaveda", ExperienceYears: 6,
	})

	p1Token := issueTestToken(t, p1ID, "priest", 1)

	catalog, _ := storage.GetCatalog("", "", "en")
	poojaID := catalog[0]["id"].(string)

	bookingID, _, _ := storage.CreateBooking(models.BookingCreateRequest{
		UserID:       devoteeID,
		PriestID:     p1ID,
		PoojaID:      poojaID,
		EventAddress: "Hyderabad",
		TotalAmount:  4000.0,
	})

	const concurrentAttempts = 10
	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	failureCount := 0

	for i := 0; i < concurrentAttempts; i++ {
		targetColleague := p2ID
		if i%2 == 1 {
			targetColleague = p3ID
		}
		wg.Add(1)
		go func(colleague string) {
			defer wg.Done()
			payload := fmt.Sprintf(`{"booking_id":"%s","referring_priest_id":"%s","referred_priest_id":"%s","reason":"Concurrent race test"}`,
				bookingID, p1ID, colleague)
			req := httptest.NewRequest("POST", "/api/v1/bookings/refer", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+p1Token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			mu.Lock()
			defer mu.Unlock()
			if w.Code == http.StatusOK {
				successCount++
			} else {
				failureCount++
			}
		}(targetColleague)
	}

	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 referral to succeed under concurrency, got %d (failures: %d)", successCount, failureCount)
	}
	if failureCount != concurrentAttempts-1 {
		t.Fatalf("expected %d failures, got %d", concurrentAttempts-1, failureCount)
	}

	// Verify database state: exactly 1 referral row in priest_referrals
	var referralRows int
	_ = storage.DB.QueryRow(`SELECT count(*) FROM priest_referrals WHERE booking_id = $1`, bookingID).Scan(&referralRows)
	if referralRows != 1 {
		t.Fatalf("expected exactly 1 referral record in DB, got: %d", referralRows)
	}
}

// 11. Chat room lookup, message reads/writes, WebSocket access & sender identity spoofing prevention
func TestChatAuthorizationAndParticipantChecks(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)

	// Devotee 1 & Priest 1 (Legitimate participants)
	d1ID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone: "+919876599041", FullName: "Chat Devotee 1", Password: "Pass123!", ConsentAgreed: true,
	})
	p1ID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876599042", FullName: "Chat Priest 1", Password: "Pass123!", Sampradaya: "Smartha", City: "Hyderabad", ConsentAgreed: true,
		VedaShakha: "Rigveda", ExperienceYears: 8,
	})

	// Devotee 2 & Priest 2 (Unrelated / cross-account)
	d2ID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone: "+919876599043", FullName: "Chat Devotee 2", Password: "Pass123!", ConsentAgreed: true,
	})
	p2ID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876599044", FullName: "Chat Priest 2", Password: "Pass123!", Sampradaya: "Vaishnava", City: "Hyderabad", ConsentAgreed: true,
		VedaShakha: "Yajurveda", ExperienceYears: 4,
	})

	d1Token := issueTestToken(t, d1ID, "devotee", 1)
	p1Token := issueTestToken(t, p1ID, "priest", 1)
	d2Token := issueTestToken(t, d2ID, "devotee", 1)
	p2Token := issueTestToken(t, p2ID, "priest", 1)

	catalog, _ := storage.GetCatalog("", "", "en")
	poojaID := catalog[0]["id"].(string)

	bookingID, _, _ := storage.CreateBooking(models.BookingCreateRequest{
		UserID:       d1ID,
		PriestID:     p1ID,
		PoojaID:      poojaID,
		EventAddress: "Hyderabad",
		TotalAmount:  3000.0,
	})

	var roomID string

	// 1. D1 retrieves chat room -> 200 OK
	t.Run("Booking owner devotee retrieves room", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/chat/room-by-booking?booking_id="+bookingID, nil)
		req.Header.Set("Authorization", "Bearer "+d1Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
		var res map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		roomID = res["room_id"]
		if roomID == "" {
			t.Fatalf("expected non-empty room_id")
		}
	})

	// 2. Assigned Priest 1 retrieves chat room -> 200 OK
	t.Run("Assigned priest retrieves room", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/chat/room-by-booking?booking_id="+bookingID, nil)
		req.Header.Set("Authorization", "Bearer "+p1Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 3. Cross-account Devotee 2 attempts to get room by booking -> 403 Forbidden
	t.Run("Cross-account devotee room lookup rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/chat/room-by-booking?booking_id="+bookingID, nil)
		req.Header.Set("Authorization", "Bearer "+d2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 4. Unassigned Priest 2 attempts to get room by booking -> 403 Forbidden
	t.Run("Unassigned priest room lookup rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/chat/room-by-booking?booking_id="+bookingID, nil)
		req.Header.Set("Authorization", "Bearer "+p2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 5. Cross-account Devotee 2 attempts POST /api/v1/chat/rooms -> 403 Forbidden
	t.Run("Cross-account devotee POST chat rooms rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/chat/rooms", bytes.NewBufferString(`{"booking_id":"`+bookingID+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+d2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 6. Devotee 1 sends chat message with client-forged sender_id and sender_role
	// Server must NEVER trust client-supplied sender IDs or roles!
	t.Run("Client-forged sender ID and role overwritten by server", func(t *testing.T) {
		spoofedPayload := fmt.Sprintf(`{
			"room_id": "%s",
			"sender_id": "%s",
			"sender_role": "PRIEST",
			"content": "Devotee coordinating arrival time"
		}`, roomID, p2ID)
		req := httptest.NewRequest("POST", "/api/v1/chat/messages", bytes.NewBufferString(spoofedPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+d1Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var sentMsg models.ChatMessage
		_ = json.Unmarshal(w.Body.Bytes(), &sentMsg)
		if sentMsg.SenderID != d1ID {
			t.Fatalf("server trusted spoofed sender_id! Expected %s, got %s", d1ID, sentMsg.SenderID)
		}
		if sentMsg.SenderRole != "DEVOTEE" {
			t.Fatalf("server trusted spoofed sender_role! Expected DEVOTEE, got %s", sentMsg.SenderRole)
		}
	})

	// 7. Cross-account Devotee 2 attempts to send message to Room 1 -> 403 Forbidden
	t.Run("Cross-account devotee message post rejected", func(t *testing.T) {
		msgPayload := fmt.Sprintf(`{"room_id":"%s","content":"Intruder message"}`, roomID)
		req := httptest.NewRequest("POST", "/api/v1/chat/messages", bytes.NewBufferString(msgPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+d2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 8. Cross-account Devotee 2 attempts to read messages from Room 1 -> 403 Forbidden
	t.Run("Cross-account devotee message query rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/chat/messages?room_id="+roomID, nil)
		req.Header.Set("Authorization", "Bearer "+d2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 9. Assigned Priest 1 reads messages -> 200 OK
	t.Run("Assigned priest reads messages successfully", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/chat/messages?room_id="+roomID, nil)
		req.Header.Set("Authorization", "Bearer "+p1Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
		var messages []models.ChatMessage
		_ = json.Unmarshal(w.Body.Bytes(), &messages)
		if len(messages) == 0 {
			t.Fatalf("expected at least 1 message")
		}
	})

	// 10. WebSocket Access Authorization and Session Checks
	t.Run("WebSocket authentication and participant protection", func(t *testing.T) {
		ts := httptest.NewServer(router)
		defer ts.Close()

		wsBase := "ws" + strings.TrimPrefix(ts.URL, "http")

		// Case A: Missing token -> 401 Unauthorized
		_, respA, errA := websocket.DefaultDialer.Dial(wsBase+"/ws/chat?roomId="+roomID, nil)
		if errA == nil || (respA != nil && respA.StatusCode != http.StatusUnauthorized) {
			t.Fatalf("expected 401 Unauthorized for missing WS token, got resp: %v, err: %v", respA, errA)
		}

		// Case B: Cross-account token (Devotee 2) -> 403 Forbidden
		_, respB, errB := websocket.DefaultDialer.Dial(wsBase+"/ws/chat?roomId="+roomID+"&token="+d2Token, nil)
		if errB == nil || (respB != nil && respB.StatusCode != http.StatusForbidden) {
			t.Fatalf("expected 403 Forbidden for cross-account WS connection, got resp: %v, err: %v", respB, errB)
		}

		// Case C: Legitimate participant token (Devotee 1) -> 101 Switching Protocols
		conn, respC, errC := websocket.DefaultDialer.Dial(wsBase+"/ws/chat?roomId="+roomID+"&token="+d1Token, nil)
		if errC != nil {
			t.Fatalf("failed connecting legitimate WS participant: %v (resp: %v)", errC, respC)
		}
		defer conn.Close()

		// Send message over WS with forged sender_id and sender_role
		wsMsg := `{"content":"Hello via WebSocket","sender_id":"forged_id","sender_role":"PRIEST"}`
		if err := conn.WriteMessage(websocket.TextMessage, []byte(wsMsg)); err != nil {
			t.Fatalf("failed sending WS message: %v", err)
		}

		// Verify in DB that saved message has authenticated Devotee 1 ID and DEVOTEE role
		var savedSenderID, savedRole string
		for attempt := 0; attempt < 25; attempt++ {
			err := storage.DB.QueryRow(`
				SELECT sender_id::text, sender_role FROM chat_messages 
				WHERE room_id = $1 AND content = 'Hello via WebSocket'
			`, roomID).Scan(&savedSenderID, &savedRole)
			if err == nil && savedSenderID != "" {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}

		if savedSenderID != d1ID {
			t.Fatalf("WebSocket trusted spoofed sender_id! Expected %s, got %s", d1ID, savedSenderID)
		}
		if savedRole != "DEVOTEE" {
			t.Fatalf("WebSocket trusted spoofed sender_role! Expected DEVOTEE, got %s", savedRole)
		}
	})
}

// 12. Free pilot must not claim that money is held in escrow or released
func TestFreePilotNoEscrowOrMoneyClaim(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)

	devoteeID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone: "+919876599051", FullName: "Pilot Devotee", Password: "Pass123!", ConsentAgreed: true,
	})
	priestID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876599052", FullName: "Pilot Priest", Password: "Pass123!", Sampradaya: "Smartha", City: "Hyderabad", ConsentAgreed: true,
		VedaShakha: "Rigveda", ExperienceYears: 10,
	})

	dToken := issueTestToken(t, devoteeID, "devotee", 1)
	pToken := issueTestToken(t, priestID, "priest", 1)

	catalog, _ := storage.GetCatalog("", "", "en")
	poojaID := catalog[0]["id"].(string)

	// 1. Create Booking: Message must not claim money is held in escrow
	createPayload := fmt.Sprintf(`{
		"user_id": "%s",
		"priest_id": "%s",
		"pooja_id": "%s",
		"total_amount": 3500.0,
		"event_address": "Hyderabad"
	}`, devoteeID, priestID, poojaID)

	reqCreate := httptest.NewRequest("POST", "/api/v1/bookings", bytes.NewBufferString(createPayload))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("Authorization", "Bearer "+dToken)
	wCreate := httptest.NewRecorder()
	router.ServeHTTP(wCreate, reqCreate)

	if wCreate.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", wCreate.Code, wCreate.Body.String())
	}

	var createRes map[string]interface{}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &createRes)
	createMsg, _ := createRes["message"].(string)
	if strings.Contains(strings.ToLower(createMsg), "payment held") || strings.Contains(strings.ToLower(createMsg), "money held") {
		t.Fatalf("creation response claims money held in escrow during free pilot: %s", createMsg)
	}
	bookingID := createRes["booking_id"].(string)
	otp := createRes["completion_otp"].(string)

	// Accept booking
	reqAccept := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(
		fmt.Sprintf(`{"booking_id":"%s","status":"ACCEPTED"}`, bookingID),
	))
	reqAccept.Header.Set("Content-Type", "application/json")
	reqAccept.Header.Set("Authorization", "Bearer "+pToken)
	wAccept := httptest.NewRecorder()
	router.ServeHTTP(wAccept, reqAccept)
	if wAccept.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for accept: %s", wAccept.Body.String())
	}

	// Step through intermediate states: IN_TRANSIT -> ARRIVED -> POOJA_STARTED
	for _, st := range []string{"IN_TRANSIT", "ARRIVED", "POOJA_STARTED"} {
		reqSt := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(
			fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bookingID, st),
		))
		reqSt.Header.Set("Content-Type", "application/json")
		reqSt.Header.Set("Authorization", "Bearer "+pToken)
		wSt := httptest.NewRecorder()
		router.ServeHTTP(wSt, reqSt)
		if wSt.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for %s: %s", st, wSt.Body.String())
		}
	}

	// 2. Complete Booking: Message must not claim dakshina released to priest
	reqComp := httptest.NewRequest("POST", "/api/v1/bookings/"+bookingID+"/complete", bytes.NewBufferString(
		fmt.Sprintf(`{"completion_otp":"%s","dakshina_amount":3500.0}`, otp),
	))
	reqComp.Header.Set("Content-Type", "application/json")
	reqComp.Header.Set("Authorization", "Bearer "+pToken)
	wComp := httptest.NewRecorder()
	router.ServeHTTP(wComp, reqComp)

	if wComp.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for complete, got %d: %s", wComp.Code, wComp.Body.String())
	}

	var compRes map[string]interface{}
	_ = json.Unmarshal(wComp.Body.Bytes(), &compRes)
	compMsg, _ := compRes["message"].(string)
	if strings.Contains(strings.ToLower(compMsg), "dakshina released") || strings.Contains(strings.ToLower(compMsg), "money released") {
		t.Fatalf("completion response claims money/dakshina released during free pilot: %s", compMsg)
	}
}
