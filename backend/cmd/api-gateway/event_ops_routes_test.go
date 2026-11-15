package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/lifecycle"
	"github.com/purohit-platform/backend/internal/models"
)

func TestEventGhatBookingEndToEnd(t *testing.T) {
	_, storage, _ := setupBookingTestGateway(t)
	admin, err := storage.CreateAdminUser("ops-admin@example.test", "TestPass123!")
	if err != nil {
		t.Fatal(err)
	}
	priestID, err := storage.RegisterPriest(models.PriestRegistrationRequest{Phone: "+919876503001", FullName: "Ghat Priest", Sampradaya: "Smartha", Password: "TestPass123!", ConsentAgreed: true})
	if err != nil {
		t.Fatal(err)
	}
	devoteeID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{Phone: "+919876503002", FullName: "Ghat Devotee", Password: "TestPass123!", ConsentAgreed: true})
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	registerEventRoutes(router, storage, &recordingPush{})
	registerEventOpsRoutes(router, storage, &recordingPush{})
	adminTok := issueTestToken(t, admin.ID, "admin", 1)
	priestTok := issueTestToken(t, priestID, "priest", 1)
	devoteeTok := issueTestToken(t, devoteeID, "devotee", 1)

	key := 0
	call := func(method, path, token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		key++
		req.Header.Set("Idempotency-Key", fmt.Sprintf("test-key-%d-%d", key, len(body)))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	must := func(w *httptest.ResponseRecorder, code int, what string) []byte {
		t.Helper()
		if w.Code != code {
			t.Fatalf("%s: want %d got %d %s", what, code, w.Code, w.Body.String())
		}
		return w.Body.Bytes()
	}

	// Admin sets up a live event with one ghat and one ritual.
	var ev models.Event
	_ = json.Unmarshal(must(call("POST", "/api/v1/admin/events", adminTok,
		`{"slug":"godavari-test","name":"Godavari Pushkaralu","starts_at":"2020-01-01T00:00:00Z","ends_at":"2099-01-01T00:00:00Z"}`), 201, "create event"), &ev)
	base := "/api/v1/admin/events/" + ev.ID
	var ghat models.Ghat
	_ = json.Unmarshal(must(call("POST", base+"/ghats", adminTok, `{"name":"Pushkar Ghat","latitude":17.0,"longitude":81.8}`), 201, "ghat"), &ghat)
	var ritual models.EventRitual
	_ = json.Unmarshal(must(call("POST", base+"/rituals", adminTok, `{"name":"Pinda pradanam","duration_minutes":45}`), 200, "ritual"), &ritual)
	must(call("POST", base+"/rituals", adminTok, `{"name":"Too short","duration_minutes":1}`), 400, "ritual validation")
	must(call("PATCH", base, adminTok, `{"status":"SCHEDULED"}`), 200, "schedule")

	// Priest applies; cannot go on duty until approved.
	ev1 := "/api/v1/events/" + ev.ID
	must(call("POST", ev1+"/participation/me", priestTok, ``), 200, "apply")
	must(call("POST", ev1+"/participation/me", priestTok, ``), 409, "apply twice")
	must(call("POST", ev1+"/participation/me", devoteeTok, ``), 403, "devotee cannot apply")
	must(call("PUT", ev1+"/services/me", priestTok, fmt.Sprintf(`{"services":[{"ritual_id":"%s","price_minor":110000}]}`, ritual.ID)), 200, "prices")
	must(call("PUT", ev1+"/services/me", priestTok, `{"services":[{"ritual_id":"not-a-ritual","price_minor":100}]}`), 400, "unknown ritual")
	must(call("PATCH", base, adminTok, `{"status":"ACTIVE","booking_enabled":true,"card_issuance_enabled":true}`), 200, "activate")
	must(call("PUT", ev1+"/duty/me", priestTok, fmt.Sprintf(`{"ghat_id":"%s"}`, ghat.ID)), 403, "duty before approval")

	must(call("PUT", base+"/participations/"+priestID, adminTok, `{"status":"REJECTED"}`), 400, "reject needs reason")
	must(call("PUT", base+"/participations/"+priestID, adminTok, `{"status":"APPROVED"}`), 200, "approve")

	// Approved priest gets a Pooja Vahini card whose public page shows status only.
	var part models.Participation
	_ = json.Unmarshal(must(call("GET", ev1+"/participation/me", priestTok, ``), 200, "participation"), &part)
	if part.Status != "APPROVED" || part.Card == nil || part.Card.Issuer != "POOJA_VAHINI" || len(part.Services) != 1 {
		t.Fatalf("participation: %+v", part)
	}
	cardBody := must(call("GET", "/api/v1/cards/"+part.Card.CardCode, "", ``), 200, "public card")
	if bytes.Contains(cardBody, []byte("phone")) || bytes.Contains(cardBody, []byte("latitude")) || !bytes.Contains(cardBody, []byte(`"status":"ACTIVE"`)) {
		t.Fatalf("public card leaks or wrong: %s", cardBody)
	}

	// On duty but no fresh location: not listed.
	must(call("PUT", ev1+"/duty/me", priestTok, fmt.Sprintf(`{"ghat_id":"%s"}`, ghat.ID)), 200, "duty")
	priestsPath := fmt.Sprintf("%s/ghats/%s/priests", ev1, ghat.ID)
	if body := must(call("GET", priestsPath, devoteeTok, ``), 200, "list"); string(bytes.TrimSpace(body)) != "[]" {
		t.Fatalf("priest without location must not be listed: %s", body)
	}
	if _, err := storage.DB.Exec(`UPDATE priests SET latitude=17.0, longitude=81.8, location_confirmed=TRUE, location_captured_at=NOW() WHERE id=$1::uuid`, priestID); err != nil {
		t.Fatal(err)
	}
	var listed []models.GhatPriest
	_ = json.Unmarshal(must(call("GET", priestsPath, devoteeTok, ``), 200, "list"), &listed)
	if len(listed) != 1 || listed[0].Services[0].PriceMinor != 110000 {
		t.Fatalf("listed: %+v", listed)
	}
	if b := must(call("GET", priestsPath, devoteeTok, ``), 200, "list"); bytes.Contains(b, []byte("latitude")) || bytes.Contains(b, []byte("phone")) {
		t.Fatalf("discovery leaks location or phone: %s", b)
	}

	// Devotee books now; price comes from the server, not the client.
	bookBody := fmt.Sprintf(`{"priest_id":"%s","ghat_id":"%s","service_id":"%s","when":"NOW","family_size":4}`, priestID, ghat.ID, listed[0].Services[0].ID)
	var booking models.EventBookingResponse
	_ = json.Unmarshal(must(call("POST", ev1+"/bookings", devoteeTok, bookBody), 201, "book"), &booking)
	if booking.TotalAmountMinor != 110000 || len(booking.BookingCode) != 4 {
		t.Fatalf("booking: %+v", booking)
	}
	must(call("POST", ev1+"/bookings", priestTok, bookBody), 403, "priest cannot book")

	// Second request while first is live; then the priest can only accept one.
	var second models.EventBookingResponse
	_ = json.Unmarshal(must(call("POST", ev1+"/bookings", devoteeTok, bookBody), 201, "book 2"), &second)
	accept := func(id string) error {
		_, err := storage.UpdateBookingStatus(models.BookingStatusUpdateRequest{BookingID: id, Status: "ACCEPTED", PriestID: priestID, CallerRole: "priest", CallerID: priestID})
		return err
	}
	if err := accept(booking.BookingID); err != nil {
		t.Fatalf("accept: %v", err)
	}
	if err := accept(second.BookingID); !errors.Is(err, db.ErrPriestBusy) {
		t.Fatalf("second accept must be busy, got %v", err)
	}

	// Booking code is required before the ritual starts.
	step := func(status, code string) error {
		_, err := storage.UpdateBookingStatus(models.BookingStatusUpdateRequest{BookingID: booking.BookingID, Status: status, PriestID: priestID, CallerRole: "priest", CallerID: priestID, CompletionOTP: code})
		return err
	}
	for _, s := range []string{"IN_TRANSIT", "ARRIVED"} {
		if err := step(s, ""); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	if err := step("POOJA_STARTED", ""); !errors.Is(err, db.ErrBookingCodeRequired) {
		t.Fatalf("missing code: %v", err)
	}
	wrong := "0000"
	if booking.BookingCode == wrong {
		wrong = "1111"
	}
	if err := step("POOJA_STARTED", wrong); !errors.Is(err, lifecycle.ErrInvalidOTP) {
		t.Fatalf("wrong code: %v", err)
	}
	if err := step("POOJA_STARTED", booking.BookingCode); err != nil {
		t.Fatalf("right code: %v", err)
	}
	if err := step("COMPLETED", booking.BookingCode); err != nil {
		t.Fatalf("complete: %v", err)
	}

	// Unanswered requests expire and can no longer be accepted.
	if _, err := storage.DB.Exec(`UPDATE bookings SET request_expires_at = NOW() - INTERVAL '1 minute' WHERE id = $1::uuid`, second.BookingID); err != nil {
		t.Fatal(err)
	}
	if err := accept(second.BookingID); !errors.Is(err, db.ErrRequestExpired) {
		t.Fatalf("expired accept: %v", err)
	}
	if n, err := storage.ExpireEventRequests(t.Context()); err != nil || n != 1 {
		t.Fatalf("expiry sweep: %d %v", n, err)
	}

	// CMS turns booking off: new requests are refused, history stays.
	must(call("PATCH", base, adminTok, `{"booking_enabled":false}`), 200, "booking off")
	must(call("POST", ev1+"/bookings", devoteeTok, bookBody), 409, "booking while off")
	details, err := storage.GetBookingDetails(booking.BookingID)
	if err != nil || details.Status != "COMPLETED" || details.PoojaTitle != "Pinda pradanam" {
		t.Fatalf("history after toggle: %+v %v", details, err)
	}

	// Suspension removes the priest from duty and suspends the card.
	must(call("PUT", base+"/participations/"+priestID, adminTok, `{"status":"SUSPENDED","reason":"Complaint under review"}`), 200, "suspend")
	_ = json.Unmarshal(must(call("GET", ev1+"/participation/me", priestTok, ``), 200, "participation"), &part)
	if part.Duty != nil || part.Card.Status != "SUSPENDED" || part.ReviewerReason == "" {
		t.Fatalf("after suspend: %+v", part)
	}
}
