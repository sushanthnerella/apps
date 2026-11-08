package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/purohit-platform/backend/internal/models"
)

// TestAdversarialGarbageStatusStrings verifies that any unrecognized or garbage status
// strings are strictly rejected and cannot transition the booking state.
func TestAdversarialGarbageStatusStrings(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	bID, _, _, prToken, _ := createFSMTestBooking(t, router, storage, "+919777700001", "+919777700002")

	garbageInputs := []string{
		"GARBAGE_STATUS",
		"UNKNOWN_STATUS",
		"INVALID_STATE",
		"FOOBAR_123",
		"",
		"   ",
		"\t\n",
		"COMPLETED; DROP TABLE bookings;",
		"' OR 1=1 --",
		"<script>alert(1)</script>",
		"12345",
		strings.Repeat("A", 500),
		"null",
		"UNDEFINED",
		"STATUS_DOES_NOT_EXIST",
	}

	for _, garbage := range garbageInputs {
		t.Run(fmt.Sprintf("Garbage status '%s'", garbage), func(t *testing.T) {
			payload := fmt.Sprintf(`{"booking_id":"%s","status":%q}`, bID, garbage)
			req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+prToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Must be rejected with 409 Conflict (invalid transition) or 400 Bad Request; NEVER 200 OK
			if w.Code == http.StatusOK {
				t.Fatalf("CRITICAL DEFECT: garbage status %q accepted with 200 OK! Body: %s", garbage, w.Body.String())
			}
			if w.Code != http.StatusConflict && w.Code != http.StatusBadRequest {
				t.Fatalf("expected 409 Conflict or 400 Bad Request for garbage status %q, got %d: %s", garbage, w.Code, w.Body.String())
			}

			// Ensure booking state remains REQUESTED
			details, err := storage.GetBookingDetails(bID)
			if err != nil {
				t.Fatalf("failed fetching booking details: %v", err)
			}
			if details.Status != "REQUESTED" {
				t.Fatalf("CRITICAL DEFECT: booking status was corrupted to %q by garbage input %q", details.Status, garbage)
			}
		})
	}
}

// TestAdversarialRoleConfusionAndSpoofing tests role confusion, spoofing in JSON payloads,
// and cross-role violations.
func TestAdversarialRoleConfusionAndSpoofing(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	bID, otp, devToken, prToken, prID := createFSMTestBooking(t, router, storage, "+919777800001", "+919777800002")

	// 1. Devotee attempting priest transitions (ACCEPTED, IN_TRANSIT, ARRIVED, POOJA_STARTED, COMPLETED)
	t.Run("Devotee cannot perform priest transitions", func(t *testing.T) {
		for _, targetStatus := range []string{"ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED", "COMPLETED"} {
			payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bID, targetStatus)
			req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+devToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Fatalf("CRITICAL DEFECT: devotee transition to %s returned %d instead of 403 Forbidden: %s", targetStatus, w.Code, w.Body.String())
			}
		}
	})

	// 2. Devotee payload spoofing: sending caller_role="priest" and priest_id in body
	t.Run("Devotee cannot spoof priest role in payload body", func(t *testing.T) {
		spoofedPayload := fmt.Sprintf(`{
			"booking_id":"%s",
			"status":"ACCEPTED",
			"caller_role":"priest",
			"caller_id":"%s",
			"priest_id":"%s"
		}`, bID, prID, prID)

		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(spoofedPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+devToken) // Still authenticated as devotee
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("CRITICAL DEFECT: role spoofing in body succeeded! Got %d instead of 403 Forbidden: %s", w.Code, w.Body.String())
		}
	})

	// 3. Devotee calling /complete endpoint
	t.Run("Devotee calling complete endpoint returns 403 Forbidden", func(t *testing.T) {
		compPayload := fmt.Sprintf(`{"completion_otp":"%s"}`, otp)
		req := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/complete", bytes.NewBufferString(compPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+devToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("CRITICAL DEFECT: devotee calling /complete returned %d instead of 403 Forbidden: %s", w.Code, w.Body.String())
		}
	})

	// 4. Priest calling /otp endpoint (devotee-only asymmetric privacy)
	t.Run("Priest calling /otp endpoint returns 403 Forbidden", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/bookings/"+bID+"/otp", nil)
		req.Header.Set("Authorization", "Bearer "+prToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("CRITICAL DEFECT: priest calling /otp returned %d instead of 403 Forbidden: %s", w.Code, w.Body.String())
		}
	})

	// 5. Unassigned Priest attempting transitions
	t.Run("Unassigned priest cannot accept or advance booking", func(t *testing.T) {
		p2ID, _ := storage.RegisterPriest(models.PriestRegistrationRequest{
			Phone: "+919777800099", FullName: "Priest Imposter", Password: "Password123!",
			Sampradaya: "Smartha", City: "Hyderabad", ConsentAgreed: true,
			VedaShakha: "Rigveda", ExperienceYears: 5,
		})
		p2Token := issueTestToken(t, p2ID, "priest", 1)

		payload := fmt.Sprintf(`{"booking_id":"%s","status":"ACCEPTED"}`, bID)
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("CRITICAL DEFECT: unassigned priest got %d instead of 403 Forbidden: %s", w.Code, w.Body.String())
		}
	})

	// 6. Non-owner Devotee attempting cancellation
	t.Run("Non-owner devotee cannot cancel booking", func(t *testing.T) {
		d2ID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
			Phone: "+919777800088", FullName: "Devotee Stranger", Password: "Password123!", ConsentAgreed: true,
		})
		d2Token := issueTestToken(t, d2ID, "devotee", 1)

		cancelPayload := fmt.Sprintf(`{"booking_id":"%s","reason_code":"ATTACK"}`, bID)
		req := httptest.NewRequest("POST", "/api/v1/bookings/cancel", bytes.NewBufferString(cancelPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+d2Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("CRITICAL DEFECT: non-owner devotee got %d instead of 403 Forbidden: %s", w.Code, w.Body.String())
		}
	})
}

// TestAdversarialSkippedIntermediateStatesExhaustive tests every single illegal
// transition jump across all states, verifying that HTTP 409 Conflict is returned.
func TestAdversarialSkippedIntermediateStatesExhaustive(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)

	// Subtest A: Illegal jumps from REQUESTED
	t.Run("Jumps from REQUESTED", func(t *testing.T) {
		bID, otp, _, prToken, _ := createFSMTestBooking(t, router, storage, "+919777900001", "+919777900002")

		for _, illegalTarget := range []string{"IN_TRANSIT", "ARRIVED", "POOJA_STARTED", "COMPLETED"} {
			payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bID, illegalTarget)
			req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+prToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusConflict {
				t.Fatalf("REQUESTED -> %s returned %d instead of 409 Conflict: %s", illegalTarget, w.Code, w.Body.String())
			}
		}

		// Also check direct /complete endpoint call
		compPayload := fmt.Sprintf(`{"completion_otp":"%s"}`, otp)
		reqComp := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/complete", bytes.NewBufferString(compPayload))
		reqComp.Header.Set("Content-Type", "application/json")
		reqComp.Header.Set("Authorization", "Bearer "+prToken)
		wComp := httptest.NewRecorder()
		router.ServeHTTP(wComp, reqComp)
		if wComp.Code != http.StatusConflict {
			t.Fatalf("Direct /complete from REQUESTED returned %d instead of 409 Conflict: %s", wComp.Code, wComp.Body.String())
		}
	})

	// Subtest B: Illegal jumps from ACCEPTED
	t.Run("Jumps from ACCEPTED", func(t *testing.T) {
		bID, otp, _, prToken, _ := createFSMTestBooking(t, router, storage, "+919777900011", "+919777900012")

		// Advance to ACCEPTED
		reqAcc := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(fmt.Sprintf(`{"booking_id":"%s","status":"ACCEPTED"}`, bID)))
		reqAcc.Header.Set("Content-Type", "application/json")
		reqAcc.Header.Set("Authorization", "Bearer "+prToken)
		wAcc := httptest.NewRecorder()
		router.ServeHTTP(wAcc, reqAcc)
		if wAcc.Code != http.StatusOK {
			t.Fatalf("failed moving to ACCEPTED: %s", wAcc.Body.String())
		}

		for _, illegalTarget := range []string{"ARRIVED", "POOJA_STARTED", "COMPLETED"} {
			payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bID, illegalTarget)
			req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+prToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusConflict {
				t.Fatalf("ACCEPTED -> %s returned %d instead of 409 Conflict: %s", illegalTarget, w.Code, w.Body.String())
			}
		}

		// Also check /complete
		reqComp := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/complete", bytes.NewBufferString(fmt.Sprintf(`{"completion_otp":"%s"}`, otp)))
		reqComp.Header.Set("Content-Type", "application/json")
		reqComp.Header.Set("Authorization", "Bearer "+prToken)
		wComp := httptest.NewRecorder()
		router.ServeHTTP(wComp, reqComp)
		if wComp.Code != http.StatusConflict {
			t.Fatalf("Direct /complete from ACCEPTED returned %d instead of 409 Conflict: %s", wComp.Code, wComp.Body.String())
		}
	})

	// Subtest C: Illegal jumps from IN_TRANSIT
	t.Run("Jumps from IN_TRANSIT", func(t *testing.T) {
		bID, otp, _, prToken, _ := createFSMTestBooking(t, router, storage, "+919777900021", "+919777900022")

		for _, st := range []string{"ACCEPTED", "IN_TRANSIT"} {
			req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bID, st)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+prToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("failed moving to %s: %s", st, w.Body.String())
			}
		}

		for _, illegalTarget := range []string{"POOJA_STARTED", "COMPLETED"} {
			payload := fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bID, illegalTarget)
			req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+prToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusConflict {
				t.Fatalf("IN_TRANSIT -> %s returned %d instead of 409 Conflict: %s", illegalTarget, w.Code, w.Body.String())
			}
		}

		reqComp := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/complete", bytes.NewBufferString(fmt.Sprintf(`{"completion_otp":"%s"}`, otp)))
		reqComp.Header.Set("Content-Type", "application/json")
		reqComp.Header.Set("Authorization", "Bearer "+prToken)
		wComp := httptest.NewRecorder()
		router.ServeHTTP(wComp, reqComp)
		if wComp.Code != http.StatusConflict {
			t.Fatalf("Direct /complete from IN_TRANSIT returned %d instead of 409 Conflict: %s", wComp.Code, wComp.Body.String())
		}
	})

	// Subtest D: Illegal jumps from ARRIVED
	t.Run("Jumps from ARRIVED", func(t *testing.T) {
		bID, otp, _, prToken, _ := createFSMTestBooking(t, router, storage, "+919777900031", "+919777900032")

		for _, st := range []string{"ACCEPTED", "IN_TRANSIT", "ARRIVED"} {
			req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bID, st)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+prToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("failed moving to %s: %s", st, w.Body.String())
			}
		}

		// ARRIVED -> COMPLETED (skipping POOJA_STARTED)
		payload := fmt.Sprintf(`{"booking_id":"%s","status":"COMPLETED","completion_otp":"%s"}`, bID, otp)
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+prToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("ARRIVED -> COMPLETED via /status returned %d instead of 409 Conflict: %s", w.Code, w.Body.String())
		}

		reqComp := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/complete", bytes.NewBufferString(fmt.Sprintf(`{"completion_otp":"%s"}`, otp)))
		reqComp.Header.Set("Content-Type", "application/json")
		reqComp.Header.Set("Authorization", "Bearer "+prToken)
		wComp := httptest.NewRecorder()
		router.ServeHTTP(wComp, reqComp)
		if wComp.Code != http.StatusConflict {
			t.Fatalf("Direct /complete from ARRIVED returned %d instead of 409 Conflict: %s", wComp.Code, wComp.Body.String())
		}
	})
}

// TestAdversarialCompletionOTPMatrix attacks the OTP completion endpoint and status endpoint
// with missing, empty, whitespace, malformed, and incorrect OTPs, ensuring 409 Conflict is returned
// and the expected OTP is NEVER leaked in the HTTP response.
func TestAdversarialCompletionOTPMatrix(t *testing.T) {
	router, storage, _ := setupBookingTestGateway(t)
	bID, realOTP, _, prToken, _ := createFSMTestBooking(t, router, storage, "+919777900041", "+919777900042")

	// Advance sequentially to POOJA_STARTED
	for _, st := range []string{"ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED"} {
		req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBufferString(fmt.Sprintf(`{"booking_id":"%s","status":"%s"}`, bID, st)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+prToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("failed moving to %s: %s", st, w.Body.String())
		}
	}

	adversarialOTPs := []struct {
		name    string
		payload string
	}{
		{"Missing OTP field completely", `{"dakshina_amount":3000.0}`},
		{"Empty string OTP", `{"completion_otp":""}`},
		{"Whitespace spaces only", `{"completion_otp":"   "}`},
		{"Whitespace tabs and newlines", `{"completion_otp":"\t\r\n"}`},
		{"Wrong 4-digit numeric OTP", `{"completion_otp":"0000"}`},
		{"Wrong 6-digit numeric OTP", `{"completion_otp":"123456"}`},
		{"Partial prefix of real OTP", fmt.Sprintf(`{"completion_otp":"%s"}`, realOTP[:len(realOTP)/2])},
		{"Alpha characters OTP", `{"completion_otp":"ABCDEF"}`},
		{"SQL injection probe in OTP", `{"completion_otp":"' OR '1'='1"}`},
		{"XSS payload in OTP", `{"completion_otp":"<script>alert(1)</script>"}`},
		{"Overly long buffer OTP", fmt.Sprintf(`{"completion_otp":"%s"}`, strings.Repeat("9", 5000))},
		{"Null literal in OTP", `{"completion_otp":null}`},
	}

	for _, tc := range adversarialOTPs {
		t.Run(tc.name+" via /complete", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/complete", bytes.NewBufferString(tc.payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+prToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusConflict {
				t.Fatalf("expected 409 Conflict for %s, got %d: %s", tc.name, w.Code, w.Body.String())
			}

			// Invariant: Response must NEVER leak realOTP
			if strings.Contains(w.Body.String(), realOTP) {
				t.Fatalf("CRITICAL SECURITY LEAK: response body leaked real OTP %s! Body: %s", realOTP, w.Body.String())
			}
		})

		t.Run(tc.name+" via /status", func(t *testing.T) {
			var m map[string]interface{}
			_ = json.Unmarshal([]byte(tc.payload), &m)
			m["booking_id"] = bID
			m["status"] = "COMPLETED"
			bodyBytes, _ := json.Marshal(m)

			req := httptest.NewRequest("POST", "/api/v1/bookings/status", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+prToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusConflict {
				t.Fatalf("expected 409 Conflict for %s via /status, got %d: %s", tc.name, w.Code, w.Body.String())
			}

			// Invariant: Response must NEVER leak realOTP
			if strings.Contains(w.Body.String(), realOTP) {
				t.Fatalf("CRITICAL SECURITY LEAK: response body leaked real OTP %s! Body: %s", realOTP, w.Body.String())
			}
		})
	}

	// Finally, verify that supplying the EXACT genuine OTP succeeds
	t.Run("Genuine OTP succeeds", func(t *testing.T) {
		validPayload := fmt.Sprintf(`{"completion_otp":"%s","dakshina_amount":3000.0}`, realOTP)
		req := httptest.NewRequest("POST", "/api/v1/bookings/"+bID+"/complete", bytes.NewBufferString(validPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+prToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for valid completion OTP, got %d: %s", w.Code, w.Body.String())
		}

		// Ensure state is now COMPLETED
		details, err := storage.GetBookingDetails(bID)
		if err != nil {
			t.Fatalf("failed fetching details: %v", err)
		}
		if details.Status != "COMPLETED" {
			t.Fatalf("expected status COMPLETED, got %s", details.Status)
		}
	})
}
