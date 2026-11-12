package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/models"
	"github.com/purohit-platform/backend/internal/notify"
)

func initTestJWT() {
	_ = auth.InitJWT("purohit_platform_test_jwt_secret_key_32bytes_long!")
}

// issueRoleToken creates an authenticated token for testing without touching the DB
func issueRoleToken(userID, role string) string {
	token, _ := auth.GenerateToken(userID, "+919999900000", role, "Test User", 1)
	return token
}

func TestUnauthenticatedWriteAndAdminRoutes(t *testing.T) {
	initTestJWT()
	storage := &db.Storage{} // DB is nil; cryptographic JWT validation and role checks execute in middleware
	router := buildGatewayRouter(storage, &notify.LoggingNoOpNotifier{})

	adminToken := issueRoleToken("admin-id-1", "admin")
	devoteeToken := issueRoleToken("devotee-id-1", "devotee")
	priestToken := issueRoleToken("priest-id-1", "priest")

	type routeTestCase struct {
		name         string
		method       string
		path         string
		body         string
		requiredRole string
		validToken   string
		wrongToken   string
	}

	testCases := []routeTestCase{
		{
			name:         "PUT /api/v1/services/translations",
			method:       "PUT",
			path:         "/api/v1/services/translations",
			body:         `{"service_code":"P01","language":"te","display_name":"Satyanarayana Swamy Pooja"}`,
			requiredRole: "admin",
			validToken:   adminToken,
			wrongToken:   devoteeToken,
		},
		{
			name:         "POST /api/v1/services/auto-translate",
			method:       "POST",
			path:         "/api/v1/services/auto-translate",
			body:         `{"target_language":"te"}`,
			requiredRole: "admin",
			validToken:   adminToken,
			wrongToken:   priestToken,
		},
		{
			name:         "POST /api/v1/poojas/{id}/samagri",
			method:       "POST",
			path:         "/api/v1/poojas/p1/samagri",
			body:         `{"item_name":"Turmeric","quantity":"100g","is_mandatory":true}`,
			requiredRole: "admin",
			validToken:   adminToken,
			wrongToken:   devoteeToken,
		},
		{
			name:         "DELETE /api/v1/poojas/samagri/{item_id}",
			method:       "DELETE",
			path:         "/api/v1/poojas/samagri/item-123",
			body:         ``,
			requiredRole: "admin",
			validToken:   adminToken,
			wrongToken:   priestToken,
		},
		{
			name:         "POST /api/v1/priests/custom-services",
			method:       "POST",
			path:         "/api/v1/priests/custom-services",
			body:         `{"custom_name":"Special Archana","price_minor":10000}`,
			requiredRole: "priest",
			validToken:   priestToken,
			wrongToken:   devoteeToken,
		},
		{
			name:         "POST /api/v1/store/orders",
			method:       "POST",
			path:         "/api/v1/store/orders",
			body:         `{"items":[{"item_id":"1","quantity":1}]}`,
			requiredRole: "devotee",
			validToken:   devoteeToken,
			wrongToken:   priestToken,
		},
		{
			name:         "PUT /api/v1/store/orders/{id}/status",
			method:       "PUT",
			path:         "/api/v1/store/orders/ord-123/status",
			body:         `{"status":"SHIPPED"}`,
			requiredRole: "admin",
			validToken:   adminToken,
			wrongToken:   devoteeToken,
		},
		{
			name:         "POST /api/v1/temples/tickets/book",
			method:       "POST",
			path:         "/api/v1/temples/tickets/book",
			body:         `{"temple_id":"t1","vip_puja_id":"p1","devotee_name":"Rama"}`,
			requiredRole: "devotee",
			validToken:   devoteeToken,
			wrongToken:   priestToken,
		},
		{
			name:         "POST /api/v1/temples/tickets/verify",
			method:       "POST",
			path:         "/api/v1/temples/tickets/verify",
			body:         `{"qr_token":"TICKET_123"}`,
			requiredRole: "admin",
			validToken:   adminToken,
			wrongToken:   devoteeToken,
		},
		{
			name:         "GET /api/v1/dispatch/radar",
			method:       "GET",
			path:         "/api/v1/dispatch/radar",
			body:         ``,
			requiredRole: "admin",
			validToken:   adminToken,
			wrongToken:   priestToken,
		},
		{
			name:         "POST /api/v1/dispatch/match",
			method:       "POST",
			path:         "/api/v1/dispatch/match",
			body:         `{"booking_id":"b1","devotee_latitude":17.385,"devotee_longitude":78.4867}`,
			requiredRole: "devotee",
			validToken:   devoteeToken,
			wrongToken:   priestToken,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+" - 401 on missing token", func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.method == http.MethodPost {
				req.Header.Set("Idempotency-Key", "test-key-401")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("%s without token expected 401, got %d (%s)", tc.name, w.Code, w.Body.String())
			}
		})

		t.Run(tc.name+" - 403 on wrong role", func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tc.wrongToken)
			if tc.method == http.MethodPost {
				req.Header.Set("Idempotency-Key", "test-key-403")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Fatalf("%s with wrong role expected 403, got %d (%s)", tc.name, w.Code, w.Body.String())
			}
		})
	}
}

func TestIdentityConflictReturns403(t *testing.T) {
	initTestJWT()
	storage := &db.Storage{}
	router := buildGatewayRouter(storage, &notify.LoggingNoOpNotifier{})

	devoteeToken := issueRoleToken("devotee-111", "devotee")
	priestToken := issueRoleToken("priest-222", "priest")

	// 1. POST /api/v1/priests/custom-services with conflicting priest_id in body
	t.Run("priest custom-services identity mismatch gives 403", func(t *testing.T) {
		body := `{"priest_id":"conflicting-priest-999","custom_name":"Abhishekam","price_minor":15000}`
		req := httptest.NewRequest("POST", "/api/v1/priests/custom-services", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+priestToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 on priest identity conflict, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 2. POST /api/v1/store/orders with conflicting user_id in body
	t.Run("store orders user_id conflict gives 403", func(t *testing.T) {
		body := `{"user_id":"conflicting-devotee-999","items":[{"item_id":"1","quantity":1}]}`
		req := httptest.NewRequest("POST", "/api/v1/store/orders", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+devoteeToken)
		req.Header.Set("Idempotency-Key", "key-conflict-1")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 on store order identity conflict, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 3. POST /api/v1/temples/tickets/book with conflicting user_id in body
	t.Run("temple ticket book user_id conflict gives 403", func(t *testing.T) {
		body := `{"user_id":"conflicting-devotee-999","temple_id":"t1","vip_puja_id":"p1"}`
		req := httptest.NewRequest("POST", "/api/v1/temples/tickets/book", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+devoteeToken)
		req.Header.Set("Idempotency-Key", "key-conflict-2")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 on temple ticket identity conflict, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 4. POST /api/v1/dispatch/match with conflicting devotee_id in body
	t.Run("dispatch match devotee_id conflict gives 403", func(t *testing.T) {
		body := `{"devotee_id":"conflicting-devotee-999","booking_id":"b1"}`
		req := httptest.NewRequest("POST", "/api/v1/dispatch/match", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+devoteeToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 on dispatch match identity conflict, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIdempotencyMiddlewareEnforcement(t *testing.T) {
	initTestJWT()
	storage := &db.Storage{}
	router := buildGatewayRouter(storage, &notify.LoggingNoOpNotifier{})

	devoteeToken := issueRoleToken("devotee-111", "devotee")

	idempotentPostRoutes := []struct {
		name string
		path string
		body string
	}{
		{"POST /api/v1/bookings", "/api/v1/bookings", `{"pooja_id":"p1","booking_type":"DIRECT"}`},
		{"POST /api/v1/store/orders", "/api/v1/store/orders", `{"items":[]}`},
		{"POST /api/v1/temples/tickets/book", "/api/v1/temples/tickets/book", `{"temple_id":"t1"}`},
		{"POST /api/v1/priests/slots/hold", "/api/v1/priests/slots/hold", `{"priest_id":"p1","slot_id":"s1"}`},
		{"POST /api/v1/bookings/cancel", "/api/v1/bookings/cancel", `{"booking_id":"b1"}`},
	}

	for _, ep := range idempotentPostRoutes {
		t.Run(ep.name+" missing Idempotency-Key gives 400", func(t *testing.T) {
			req := httptest.NewRequest("POST", ep.path, bytes.NewBufferString(ep.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+devoteeToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 when missing Idempotency-Key on %s, got %d (%s)", ep.name, w.Code, w.Body.String())
			}
		})
	}
}

func TestDispatchMatchBehavior(t *testing.T) {
	initTestJWT()
	storage := &db.Storage{}
	router := buildGatewayRouter(storage, &notify.LoggingNoOpNotifier{})

	devoteeToken := issueRoleToken("devotee-111", "devotee")

	// 1. Invalid JSON body returns 400
	t.Run("invalid json returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/dispatch/match", bytes.NewBufferString("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+devoteeToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 on malformed json, got %d (%s)", w.Code, w.Body.String())
		}
	})

	// 2. Valid request when DB has no priests returns 200 with empty candidate list (no fake priests)
	t.Run("empty candidates returns 200 with no fake priest", func(t *testing.T) {
		body := `{"booking_id":"b1","devotee_latitude":17.385,"devotee_longitude":78.4867}`
		req := httptest.NewRequest("POST", "/api/v1/dispatch/match", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+devoteeToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 with empty list, got %d (%s)", w.Code, w.Body.String())
		}

		var res struct {
			Status          string                   `json:"status"`
			Reason          string                   `json:"reason"`
			TotalCandidates int                      `json:"total_candidates"`
			Candidates      []models.PriestCandidate `json:"candidates"`
		}
		if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
			t.Fatalf("failed decoding dispatch response: %v", err)
		}

		if len(res.Candidates) != 0 {
			t.Fatalf("expected 0 candidates, got %d (fake priest returned!)", len(res.Candidates))
		}
		if res.Reason == "" {
			t.Errorf("expected explanatory reason when no candidates found")
		}
	})
}
