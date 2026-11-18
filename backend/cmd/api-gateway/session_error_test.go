package main

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A database outage during the session check must not look like a revoked session,
// or clients wipe a valid login.
func TestSessionLookupErrorStatus(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{sessionLookupError(errors.New("driver: bad connection")), http.StatusServiceUnavailable},
		{sessionLookupError(sql.ErrNoRows), http.StatusUnauthorized},
		{errors.New("invalid token"), http.StatusUnauthorized},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		writeSessionError(rec, c.err, `{"error":"unauthorized"}`)
		if rec.Code != c.want {
			t.Errorf("%v: got %d, want %d", c.err, rec.Code, c.want)
		}
	}
}
