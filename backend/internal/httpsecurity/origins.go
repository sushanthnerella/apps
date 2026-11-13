package httpsecurity

import (
	"net/http"
	"os"
	"strings"
)

// AllowedOrigin permits native clients without Origin and explicitly configured web clients.
func AllowedOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	for _, allowed := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		if strings.TrimSpace(allowed) == origin && origin != "null" && origin != "*" {
			return true
		}
	}
	return false
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		if !AllowedOrigin(r) {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept-Language, Idempotency-Key")
			w.Header().Set("Access-Control-Expose-Headers", "Idempotent-Replayed, Retry-After")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
