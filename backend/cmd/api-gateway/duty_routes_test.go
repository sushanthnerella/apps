package main

import (
	"bytes"
	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDutyRequiresPriestAndPersists(t *testing.T) {
	_, storage, _ := setupBookingTestGateway(t)
	priest, err := storage.RegisterPriest(models.PriestRegistrationRequest{Phone: "+919876501234", FullName: "Duty Test", Sampradaya: "Smartha", Password: "TestPass123!", ConsentAgreed: true})
	if err != nil {
		t.Fatal(err)
	}
	devotee, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{Phone: "+919876501235", FullName: "Devotee Test", Password: "TestPass123!", ConsentAgreed: true})
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	registerDutyRoutes(router, storage)
	call := func(method, token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/api/v1/priests/me/duty", bytes.NewBufferString(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	if w := call("GET", "", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", w.Code)
	}
	if w := call("PUT", issueTestToken(t, devotee, "devotee", 1), `{"is_online":true}`); w.Code != http.StatusForbidden {
		t.Fatalf("devotee: %d", w.Code)
	}
	token := issueTestToken(t, priest, "priest", 1)
	if w := call("PUT", token, `{"is_online":true}`); w.Code != http.StatusOK {
		t.Fatalf("set: %d %s", w.Code, w.Body.String())
	}
	if w := call("GET", token, ""); w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"is_online":true`)) {
		t.Fatalf("get: %d %s", w.Code, w.Body.String())
	}
	if w := call("PUT", token, `{}`); w.Code != http.StatusBadRequest {
		t.Fatalf("missing: %d", w.Code)
	}
	if w := call("PUT", token, `{"is_online":false}`); w.Code != http.StatusOK {
		t.Fatalf("off: %d", w.Code)
	}
}
