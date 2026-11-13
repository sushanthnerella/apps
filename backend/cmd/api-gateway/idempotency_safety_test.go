package main

import (
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIdempotencyUncertainOutcomeNeverReexecutes(t *testing.T) {
	for _, scenario := range []string{"server_error", "panic", "expired"} {
		t.Run(scenario, func(t *testing.T) {
			_, storage, _ := setupBookingTestGateway(t)
			calls := 0
			handler := requireIdempotency(storage)(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if scenario == "panic" {
					panic("after commit")
				}
				if scenario == "server_error" {
					http.Error(w, "after commit", 500)
					return
				}
				w.WriteHeader(http.StatusCreated)
			})
			call := func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				r := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
				r.Header.Set("Idempotency-Key", "uncertain-key")
				func() { defer func() { _ = recover() }(); handler(w, r) }()
				return w
			}
			call()
			if _, err := storage.DB.Exec("UPDATE idempotency_keys SET created_at=NOW()-INTERVAL '25 hours', locked_until=NOW()-INTERVAL '1 hour' WHERE key=$1", "uncertain-key"); err != nil {
				t.Fatal(err)
			}
			result := call()
			if calls != 1 {
				t.Fatalf("uncertain operation ran %d times; want 1", calls)
			}
			if scenario != "expired" && result.Code != http.StatusConflict {
				t.Fatalf("want conflict, got %d", result.Code)
			}
			if scenario == "expired" && result.Code != http.StatusCreated {
				t.Fatalf("want replay, got %d", result.Code)
			}
		})
	}
}

func TestIdempotencyKeyCannotReplayAnotherResource(t *testing.T) {
	_, storage, _ := setupBookingTestGateway(t)
	calls := 0
	router := mux.NewRouter()
	router.HandleFunc("/items/{id}", requireIdempotency(storage)(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(http.StatusCreated) }))
	call := func(path string) int {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", path, strings.NewReader("{}"))
		r.Header.Set("Idempotency-Key", "same-key")
		router.ServeHTTP(w, r)
		return w.Code
	}
	if got := call("/items/one"); got != http.StatusCreated {
		t.Fatalf("first status %d", got)
	}
	if got := call("/items/two"); got != http.StatusUnprocessableEntity {
		t.Fatalf("different resource status %d; calls %d", got, calls)
	}
}
