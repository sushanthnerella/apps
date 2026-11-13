package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/db"
)

type contextKey string

const callerClaimsKey contextKey = "caller_claims"

func getCallerClaims(r *http.Request) *auth.CustomClaims {
	if c, ok := r.Context().Value(callerClaimsKey).(*auth.CustomClaims); ok {
		return c
	}
	return nil
}

// requireRole creates a middleware enforcing caller session validity and authorized roles.
// Rejects unauthenticated callers with 401 Unauthorized and unauthorized roles with 403 Forbidden.
func requireRole(storage *db.Storage, allowedRoles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			claims, err := requireCallerSession(r, storage)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "unauthorized",
					"message": err.Error(),
				})
				return
			}

			roleAllowed := false
			for _, rRole := range allowedRoles {
				if strings.EqualFold(claims.Role, rRole) {
					roleAllowed = true
					break
				}
			}

			if !roleAllowed {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "forbidden",
					"message": "insufficient permissions for role: " + claims.Role,
				})
				return
			}

			ctx := context.WithValue(r.Context(), callerClaimsKey, claims)
			next(w, r.WithContext(ctx))
		}
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// requireIdempotency enforces mandatory Idempotency-Key on critical POST mutations using the reserve-first pattern.
func requireIdempotency(storage *db.Storage) func(http.HandlerFunc) http.HandlerFunc {
	return idempotencyMiddleware(storage, true)
}

// optionalIdempotency enables idempotency when Idempotency-Key is present, but passes through when absent.
func optionalIdempotency(storage *db.Storage) func(http.HandlerFunc) http.HandlerFunc {
	return idempotencyMiddleware(storage, false)
}

func idempotencyMiddleware(storage *db.Storage, required bool) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
			if key == "" {
				if required {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]string{
						"error":   "bad_request",
						"message": "Idempotency-Key header is required",
					})
					return
				}
				next(w, r)
				return
			}

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "bad_request",
					"message": "failed to read request body",
				})
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			callerID := "anonymous"
			if claims := getCallerClaims(r); claims != nil && claims.UserID != "" {
				callerID = claims.UserID
			} else {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
					if cl, err := auth.ValidateToken(tokenStr); err == nil && cl.UserID != "" {
						callerID = cl.UserID
					}
				}
			}

			routeTemplate := r.URL.Path
			if route := mux.CurrentRoute(r); route != nil {
				if t, err := route.GetPathTemplate(); err == nil && t != "" {
					routeTemplate = t
				}
			}

			hasher := sha256.New()
			hasher.Write([]byte(r.Method + ":" + r.URL.RequestURI() + ":"))
			hasher.Write(bodyBytes)
			reqHash := hex.EncodeToString(hasher.Sum(nil))

			if storage != nil && storage.DB != nil {
				// Retain keys indefinitely: a crash or response failure can leave a committed
				// mutation with an unknown outcome. Age alone never proves it is safe to rerun.
				reserved, err := storage.ReserveIdempotencyKey(r.Context(), key, callerID, routeTemplate, reqHash)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{
						"error":   "internal_error",
						"message": "failed to reserve idempotency lease",
					})
					return
				}

				if !reserved {
					// Key exists: inspect state
					stored, err := storage.GetIdempotencyKey(r.Context(), callerID, key)
					if err != nil || stored == nil {
						// Record could have been deleted concurrently, retry reservation
						reserved, _ = storage.ReserveIdempotencyKey(r.Context(), key, callerID, routeTemplate, reqHash)
					} else {
						if stored.RequestHash != reqHash {
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(422) // Unprocessable Entity
							json.NewEncoder(w).Encode(map[string]string{
								"error":   "unprocessable_entity",
								"message": "Idempotency key reused with different request payload",
							})
							return
						} else if stored.Status == "COMPLETED" {
							// Replay stored response
							w.Header().Set("Content-Type", "application/json")
							w.Header().Set("Idempotent-Replayed", "true")
							w.Header().Set("X-Idempotent-Replayed", "true")
							w.WriteHeader(stored.ResponseStatus)
							_, _ = w.Write([]byte(stored.ResponseBody))
							return
						} else {
							// Status is IN_PROGRESS
							if stored.LockedUntil != nil && stored.LockedUntil.After(time.Now()) {
								w.Header().Set("Content-Type", "application/json")
								w.Header().Set("Retry-After", "1")
								w.WriteHeader(http.StatusConflict) // 409 Conflict
								json.NewEncoder(w).Encode(map[string]string{
									"error":   "conflict",
									"message": "request with this idempotency key is currently in progress",
								})
								return
							}

							// Lease expired without a stored result: the first attempt may or may not have
							// committed (crash, or result save failed). Never re-run it; a re-run could duplicate
							// a committed order. The client must re-read state (e.g. GET /bookings) and use a new key.
							// A future transactional result record can make this outcome unambiguous.
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusConflict)
							json.NewEncoder(w).Encode(map[string]string{
								"error":   "outcome_unknown",
								"message": "a previous attempt with this idempotency key did not report its result; verify whether the operation completed before submitting a new request",
							})
							return
						}
					}
				}

				if reserved {
					rec := &responseRecorder{
						ResponseWriter: w,
						statusCode:     http.StatusOK,
					}
					next(rec, r)

					if rec.statusCode < 500 {
						// 2xx and 4xx: persist result
						// If saving still fails the row stays IN_PROGRESS; retries get 409 outcome_unknown, never a re-run.
						var saveErr error
						for i := 0; i < 3; i++ {
							if saveErr = storage.UpdateIdempotencySuccess(context.Background(), key, callerID, rec.statusCode, rec.body.String()); saveErr == nil {
								break
							}
							time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
						}
						if saveErr != nil {
							log.Printf("CRITICAL idempotency result not persisted caller=%s route=%s: %v", callerID, routeTemplate, saveErr)
						}
					}
					return
				}

				// Fail closed: never run the mutation without owning the key.
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "conflict",
					"message": "request with this idempotency key is currently in progress",
				})
				return
			}

			// Storage nil fallback (e.g. tests without DB)
			rec := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}
			next(rec, r)
		}
	}
}
