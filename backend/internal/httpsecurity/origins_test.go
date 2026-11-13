package httpsecurity

import (
	"net/http/httptest"
	"testing"
)

func TestAllowedOrigins(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://cms.example.com")
	for _, tc := range []struct {
		origin string
		want   bool
	}{{"", true}, {"https://cms.example.com", true}, {"https://evil.example", false}, {"null", false}, {"https://cms.example.com.evil.test", false}} {
		r := httptest.NewRequest("GET", "http://gateway/", nil)
		r.Header.Set("Origin", tc.origin)
		if got := AllowedOrigin(r); got != tc.want {
			t.Errorf("origin %q: got %v", tc.origin, got)
		}
	}
}
