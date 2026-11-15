package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/models"
	"github.com/purohit-platform/backend/internal/notify"
)

type recordingPush struct {
	mu    sync.Mutex
	calls int
}

func (r *recordingPush) SendToToken(context.Context, string, notify.NotificationMessage) error {
	return nil
}
func (r *recordingPush) SendToTokens(context.Context, []string, notify.NotificationMessage) error {
	return nil
}
func (r *recordingPush) SendDataToTopic(_ context.Context, topic string, data map[string]string) error {
	if topic == notify.EventConfigTopic && data["type"] == "event_config" {
		r.mu.Lock()
		r.calls++
		r.mu.Unlock()
	}
	return nil
}
func (r *recordingPush) count() int {
	for i := 0; i < 50; i++ { // push is sent asynchronously
		r.mu.Lock()
		n := r.calls
		r.mu.Unlock()
		if n > 0 {
			return n
		}
		time.Sleep(20 * time.Millisecond)
	}
	return 0
}

func TestEventConfigToggleLifecycle(t *testing.T) {
	_, storage, _ := setupBookingTestGateway(t)
	admin, err := storage.CreateAdminUser("events-admin@example.test", "TestPass123!")
	if err != nil {
		t.Fatal(err)
	}
	devotee, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{Phone: "+919876502001", FullName: "Event Devotee", Password: "TestPass123!", ConsentAgreed: true})
	if err != nil {
		t.Fatal(err)
	}
	push := &recordingPush{}
	router := mux.NewRouter()
	registerEventRoutes(router, storage, push)
	adminToken := issueTestToken(t, admin.ID, "admin", 1)

	call := func(method, path, token, body string, headers ...string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		for i := 0; i+1 < len(headers); i += 2 {
			req.Header.Set(headers[i], headers[i+1])
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	// No live event: apps get null and show nothing.
	w := call("GET", "/api/v1/events/config", "", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"event":null`) {
		t.Fatalf("empty config: %d %s", w.Code, w.Body.String())
	}

	// Only admins may manage events.
	if w := call("GET", "/api/v1/admin/events", "", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous admin list: %d", w.Code)
	}
	if w := call("GET", "/api/v1/admin/events", issueTestToken(t, devotee, "devotee", 1), ""); w.Code != http.StatusForbidden {
		t.Fatalf("devotee admin list: %d", w.Code)
	}

	createBody := `{"slug":"godavari-2027","name":"Godavari Pushkaralu","name_te":"గోదావరి పుష్కరాలు","river":"Godavari","starts_at":"2027-07-01T00:00:00Z","ends_at":"2027-07-12T23:59:00Z"}`
	w = call("POST", "/api/v1/admin/events", adminToken, createBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var ev models.Event
	_ = json.Unmarshal(w.Body.Bytes(), &ev)
	if ev.Status != models.EventDraft || ev.BookingEnabled {
		t.Fatalf("new event must be DRAFT with booking off: %+v", ev)
	}
	if w := call("POST", "/api/v1/admin/events", adminToken, createBody); w.Code != http.StatusConflict {
		t.Fatalf("duplicate slug: %d", w.Code)
	}
	if w := call("POST", "/api/v1/admin/events", adminToken, `{"slug":"Bad Slug","name":"x","starts_at":"2027-07-01T00:00:00Z","ends_at":"2027-07-02T00:00:00Z"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("bad slug: %d", w.Code)
	}

	// Ghats.
	w = call("POST", "/api/v1/admin/events/"+ev.ID+"/ghats", adminToken, `{"name":"Pushkar Ghat","name_te":"పుష్కర ఘాట్","latitude":17.0005,"longitude":81.8040,"sort_order":1}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("ghat create: %d %s", w.Code, w.Body.String())
	}
	var ghat models.Ghat
	_ = json.Unmarshal(w.Body.Bytes(), &ghat)
	if w := call("POST", "/api/v1/admin/events/"+ev.ID+"/ghats", adminToken, `{"name":"Half","latitude":17.0}`); w.Code != http.StatusBadRequest {
		t.Fatalf("ghat with only latitude: %d", w.Code)
	}

	// Lifecycle: DRAFT cannot jump to ACTIVE.
	if w := call("PATCH", "/api/v1/admin/events/"+ev.ID, adminToken, `{"status":"ACTIVE"}`); w.Code != http.StatusConflict {
		t.Fatalf("draft->active must be rejected: %d", w.Code)
	}
	if w := call("PATCH", "/api/v1/admin/events/"+ev.ID, adminToken, `{"status":"SCHEDULED"}`); w.Code != http.StatusOK {
		t.Fatalf("schedule: %d %s", w.Code, w.Body.String())
	}
	w = call("PATCH", "/api/v1/admin/events/"+ev.ID, adminToken, `{"status":"ACTIVE","booking_enabled":true}`)
	if w.Code != http.StatusOK {
		t.Fatalf("activate: %d %s", w.Code, w.Body.String())
	}

	// Live config reflects the toggle and carries an ETag.
	w = call("GET", "/api/v1/events/config", "", "")
	var cfg struct {
		Event *models.Event `json:"event"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &cfg)
	if cfg.Event == nil || cfg.Event.Status != models.EventActive || !cfg.Event.BookingEnabled || len(cfg.Event.Ghats) != 1 {
		t.Fatalf("live config: %s", w.Body.String())
	}
	etag := w.Header().Get("ETag")
	if w := call("GET", "/api/v1/events/config", "", "", "If-None-Match", etag); w.Code != http.StatusNotModified {
		t.Fatalf("etag revalidation: %d", w.Code)
	}

	// A second event cannot go live at the same time.
	w = call("POST", "/api/v1/admin/events", adminToken, `{"slug":"krishna-2028","name":"Krishna Pushkaralu","starts_at":"2028-08-01T00:00:00Z","ends_at":"2028-08-12T00:00:00Z"}`)
	var second models.Event
	_ = json.Unmarshal(w.Body.Bytes(), &second)
	call("PATCH", "/api/v1/admin/events/"+second.ID, adminToken, `{"status":"SCHEDULED"}`)
	if w := call("PATCH", "/api/v1/admin/events/"+second.ID, adminToken, `{"status":"ACTIVE"}`); w.Code != http.StatusConflict {
		t.Fatalf("second live event must be rejected: %d %s", w.Code, w.Body.String())
	}

	// Turning booking off bumps the version, so the old ETag no longer matches.
	if w := call("PATCH", "/api/v1/admin/events/"+ev.ID, adminToken, `{"booking_enabled":false}`); w.Code != http.StatusOK {
		t.Fatalf("booking off: %d", w.Code)
	}
	if w := call("GET", "/api/v1/events/config", "", "", "If-None-Match", etag); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"booking_enabled":false`) {
		t.Fatalf("config after toggle: %d %s", w.Code, w.Body.String())
	}

	// Deactivated ghats disappear from app config but stay for admins.
	if w := call("PUT", "/api/v1/admin/events/"+ev.ID+"/ghats/"+ghat.ID, adminToken, `{"name":"Pushkar Ghat","latitude":17.0005,"longitude":81.8040,"is_active":false}`); w.Code != http.StatusOK {
		t.Fatalf("ghat deactivate: %d %s", w.Code, w.Body.String())
	}
	w = call("GET", "/api/v1/events/config", "", "")
	_ = json.Unmarshal(w.Body.Bytes(), &cfg)
	if len(cfg.Event.Ghats) != 0 {
		t.Fatalf("inactive ghat leaked to apps: %s", w.Body.String())
	}

	// Ending the event turns switches off and removes it from app config.
	w = call("PATCH", "/api/v1/admin/events/"+ev.ID, adminToken, `{"status":"ENDED"}`)
	_ = json.Unmarshal(w.Body.Bytes(), &ev)
	if ev.BookingEnabled || ev.CardIssuanceEnabled {
		t.Fatalf("ended event must have switches off: %+v", ev)
	}
	if w := call("GET", "/api/v1/events/config", "", ""); !strings.Contains(w.Body.String(), `"event":null`) {
		t.Fatalf("ended event still live: %s", w.Body.String())
	}

	// Every change is audited and the audit log cannot be edited.
	w = call("GET", "/api/v1/admin/events/"+ev.ID+"/audit", adminToken, "")
	var audit []models.AuditEntry
	_ = json.Unmarshal(w.Body.Bytes(), &audit)
	if len(audit) < 6 {
		t.Fatalf("expected audit rows for create, ghat create, schedule, activate, toggle, ghat update, end; got %d", len(audit))
	}
	if _, err := storage.DB.Exec(`DELETE FROM audit_log`); err == nil {
		t.Fatal("audit_log must be append-only")
	}

	if w := call("GET", "/api/v1/admin/events/not-a-uuid", adminToken, ""); w.Code != http.StatusNotFound {
		t.Fatalf("malformed id: %d", w.Code)
	}
	if push.count() == 0 {
		t.Fatal("expected event-config push after admin changes")
	}
}
