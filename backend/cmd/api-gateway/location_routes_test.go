package main

import (
	"bytes"
	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPriestLocationOwnershipAndDiscovery(t *testing.T) {
	_, storage, _ := setupBookingTestGateway(t)
	priest, err := storage.RegisterPriest(models.PriestRegistrationRequest{Phone: "+919876501765", FullName: "Location Test", Sampradaya: "Smartha", Password: "TestPass123!", ConsentAgreed: true})
	if err != nil {
		t.Fatal(err)
	}
	devotee, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{Phone: "+919876501766", FullName: "Location Devotee", Password: "TestPass123!", ConsentAgreed: true})
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	registerPriestLocationRoutes(router, storage)
	send := func(token, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("PUT", "/api/v1/priests/me/location", bytes.NewBufferString(body))
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	if w := send("", `{"latitude":12.9716,"longitude":77.5946}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous %d", w.Code)
	}
	if w := send(issueTestToken(t, devotee, "devotee", 1), `{"latitude":12.9716,"longitude":77.5946}`); w.Code != http.StatusForbidden {
		t.Fatalf("role %d", w.Code)
	}
	if _, err := storage.DB.Exec(`UPDATE priests SET is_online=true WHERE id=$1`, priest); err != nil {
		t.Fatal(err)
	}
	before, err := storage.GetNearbyPriests(12.9716, 77.5946, 50000, "")
	if err != nil || len(before) != 0 {
		t.Fatalf("unconfirmed priest appeared: %v %v", before, err)
	}
	token := issueTestToken(t, priest, "priest", 1)
	if w := send(token, `{"latitude":91,"longitude":77}`); w.Code != http.StatusBadRequest {
		t.Fatalf("invalid %d", w.Code)
	}
	if w := send(token, `{"latitude":12.9716,"longitude":77.5946}`); w.Code != http.StatusOK {
		t.Fatalf("update %d %s", w.Code, w.Body.String())
	}
	near, err := storage.GetNearbyPriests(12.9716, 77.5946, 50000, "")
	if err != nil || len(near) != 1 || near[0].ID != priest {
		t.Fatalf("nearby search: %v %v", near, err)
	}
	far, err := storage.GetNearbyPriests(17.385, 78.4867, 50000, "")
	if err != nil || len(far) != 0 {
		t.Fatalf("distant priest appeared: %v %v", far, err)
	}
	var lat, lng float64
	var verified bool
	if err := storage.DB.QueryRow(`SELECT latitude,longitude,location_confirmed FROM priests WHERE id=$1`, priest).Scan(&lat, &lng, &verified); err != nil || !verified || lat != 12.9716 || lng != 77.5946 {
		t.Fatalf("location not persisted correctly: %v", err)
	}
}
