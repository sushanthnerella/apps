package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/models"
	"github.com/purohit-platform/backend/internal/notify"
)

// ----------------------------------------------------------------------------
// DEFECT 4 TESTS: Priest Location Stale-Write Race & Capture Time Validation
// ----------------------------------------------------------------------------

func TestLocationRejectsMissingCapturedAt(t *testing.T) {
	initTestJWT()
	storage := &db.Storage{}
	router := mux.NewRouter()
	registerPriestLocationRoutes(router, storage)
	priestToken := issueRoleToken("priest-test-loc-1", "priest")

	req := httptest.NewRequest("PUT", "/api/v1/priests/me/location", bytes.NewBufferString(`{"latitude":17.385,"longitude":78.4867}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+priestToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when captured_at is missing, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLocationRejectsFutureAndTooOld(t *testing.T) {
	initTestJWT()
	storage := &db.Storage{}
	router := mux.NewRouter()
	registerPriestLocationRoutes(router, storage)
	priestToken := issueRoleToken("priest-test-loc-2", "priest")

	// 1. Future timestamp (> now + 2m)
	futureTime := time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339)
	futureBody := fmt.Sprintf(`{"latitude":17.385,"longitude":78.4867,"captured_at":"%s"}`, futureTime)
	req1 := httptest.NewRequest("PUT", "/api/v1/priests/me/location", bytes.NewBufferString(futureBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+priestToken)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for future captured_at, got %d: %s", w1.Code, w1.Body.String())
	}

	// 2. Too old timestamp (> 30m ago)
	oldTime := time.Now().UTC().Add(-45 * time.Minute).Format(time.RFC3339)
	oldBody := fmt.Sprintf(`{"latitude":17.385,"longitude":78.4867,"captured_at":"%s"}`, oldTime)
	req2 := httptest.NewRequest("PUT", "/api/v1/priests/me/location", bytes.NewBufferString(oldBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+priestToken)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for expired captured_at, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestLocationRejectsInvalidCoordinatesAndAccuracy(t *testing.T) {
	initTestJWT()
	storage := &db.Storage{}
	router := mux.NewRouter()
	registerPriestLocationRoutes(router, storage)
	priestToken := issueRoleToken("priest-test-loc-3", "priest")
	validTime := time.Now().UTC().Format(time.RFC3339)

	testCases := []struct {
		name string
		body string
	}{
		{"latitude > 90", fmt.Sprintf(`{"latitude":95.0,"longitude":78.4867,"captured_at":"%s"}`, validTime)},
		{"latitude < -90", fmt.Sprintf(`{"latitude":-95.0,"longitude":78.4867,"captured_at":"%s"}`, validTime)},
		{"longitude > 180", fmt.Sprintf(`{"latitude":17.385,"longitude":185.0,"captured_at":"%s"}`, validTime)},
		{"longitude < -180", fmt.Sprintf(`{"latitude":17.385,"longitude":-185.0,"captured_at":"%s"}`, validTime)},
		{"accuracy negative", fmt.Sprintf(`{"latitude":17.385,"longitude":78.4867,"accuracy_m":-5.0,"captured_at":"%s"}`, validTime)},
		{"accuracy > 10000", fmt.Sprintf(`{"latitude":17.385,"longitude":78.4867,"accuracy_m":12000.0,"captured_at":"%s"}`, validTime)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", "/api/v1/priests/me/location", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+priestToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for %s, got %d: %s", tc.name, w.Code, w.Body.String())
			}
		})
	}
}

func TestLocationOffDutyRejected(t *testing.T) {
	_, storage, _ := setupBookingTestGateway(t)
	priestID, err := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone:         "+919876543201",
		FullName:      "Off Duty Priest",
		Sampradaya:    "Smartha",
		Password:      "SecurePass123!",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Explicitly set priest offline
	if _, err := storage.DB.Exec(`UPDATE priests SET is_online=false WHERE id=$1`, priestID); err != nil {
		t.Fatal(err)
	}

	router := mux.NewRouter()
	registerPriestLocationRoutes(router, storage)
	token := issueRoleToken(priestID, "priest")

	now := time.Now().UTC().Format(time.RFC3339)
	req := httptest.NewRequest("PUT", "/api/v1/priests/me/location", bytes.NewBufferString(fmt.Sprintf(`{"latitude":17.385,"longitude":78.4867,"captured_at":"%s"}`, now)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict when priest is off duty, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLocationConcurrentStaleWrite(t *testing.T) {
	_, storage, _ := setupBookingTestGateway(t)
	priestID, err := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone:         "+919876543202",
		FullName:      "Stale Test Priest",
		Sampradaya:    "Smartha",
		Password:      "SecurePass123!",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Set priest online
	if _, err := storage.DB.Exec(`UPDATE priests SET is_online=true WHERE id=$1`, priestID); err != nil {
		t.Fatal(err)
	}

	router := mux.NewRouter()
	registerPriestLocationRoutes(router, storage)
	token := issueRoleToken(priestID, "priest")

	// 1. Send modern point T2 (now)
	t2 := time.Now().UTC()
	reqT2 := httptest.NewRequest("PUT", "/api/v1/priests/me/location", bytes.NewBufferString(
		fmt.Sprintf(`{"latitude":17.3850,"longitude":78.4867,"accuracy_m":5.0,"captured_at":"%s"}`, t2.Format(time.RFC3339)),
	))
	reqT2.Header.Set("Content-Type", "application/json")
	reqT2.Header.Set("Authorization", "Bearer "+token)
	wT2 := httptest.NewRecorder()
	router.ServeHTTP(wT2, reqT2)

	if wT2.Code != http.StatusOK {
		t.Fatalf("expected 200 for T2, got %d: %s", wT2.Code, wT2.Body.String())
	}

	// 2. Concurrently send 10 older points T1 (now - 15s)
	t1 := t2.Add(-15 * time.Second)
	t1Body := fmt.Sprintf(`{"latitude":12.9716,"longitude":77.5946,"accuracy_m":10.0,"captured_at":"%s"}`, t1.Format(time.RFC3339))

	var wg sync.WaitGroup
	var staleCount int64

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("PUT", "/api/v1/priests/me/location", bytes.NewBufferString(t1Body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				var resp map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&resp); err == nil {
					if applied, ok := resp["applied"].(bool); ok && !applied {
						if reason, ok := resp["reason"].(string); ok && reason == "stale" {
							atomic.AddInt64(&staleCount, 1)
						}
					}
				}
			}
		}()
	}
	wg.Wait()

	if staleCount != 10 {
		t.Fatalf("expected all 10 concurrent older writes to be marked stale, got %d", staleCount)
	}

	// Verify database still holds T2 coordinates
	var lat, lon float64
	var dbCapturedAt time.Time
	err = storage.DB.QueryRow(`SELECT latitude, longitude, location_captured_at FROM priests WHERE id=$1`, priestID).Scan(&lat, &lon, &dbCapturedAt)
	if err != nil {
		t.Fatalf("failed querying priest location: %v", err)
	}

	if lat != 17.3850 || lon != 78.4867 {
		t.Fatalf("stored coordinates overwritten by stale write: got lat=%v lon=%v", lat, lon)
	}
}

// ----------------------------------------------------------------------------
// DEFECT 3 TESTS: Store Orders Minor Units & Server-Authoritative Pricing
// ----------------------------------------------------------------------------

func TestStoreOrderInvalidPayloadValidation(t *testing.T) {
	initTestJWT()
	storage := &db.Storage{}
	router := buildGatewayRouter(storage, &notify.LoggingNoOpNotifier{})
	devoteeToken := issueRoleToken("devotee-store-user-1", "devotee")

	testCases := []struct {
		name string
		body string
	}{
		{
			name: "item with quantity 0 returns 422",
			body: `{"vendor_id":"v1","delivery_address":"Hyderabad","items":[{"inventory_item_id":"p1","quantity":0}]}`,
		},
		{
			name: "item with quantity 101 returns 422",
			body: `{"vendor_id":"v1","delivery_address":"Hyderabad","items":[{"inventory_item_id":"p1","quantity":101}]}`,
		},
		{
			name: "item with empty id returns 422",
			body: `{"vendor_id":"v1","delivery_address":"Hyderabad","items":[{"inventory_item_id":"","quantity":2}]}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/store/orders", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+devoteeToken)
			req.Header.Set("Idempotency-Key", "idemp-val-"+tc.name)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != 422 {
				t.Fatalf("expected 422 for %s, got %d: %s", tc.name, w.Code, w.Body.String())
			}
		})
	}
}

func TestStoreOrderServerCalculatedPrice(t *testing.T) {
	_, storage, _ := setupBookingTestGateway(t)
	devoteeID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone: "+919988776657", FullName: "Store Devotee", Password: "SecureDev123!", ConsentAgreed: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create test vendor and inventory items with exact minor prices
	var vendorID string
	err = storage.DB.QueryRow(`
		INSERT INTO vendors (shop_name, owner_name, phone, address, city, latitude, longitude)
		VALUES ('Vedic Store', 'Owner One', '+919900011223', 'Temple Rd', 'Hyderabad', 17.385, 78.4867)
		RETURNING id
	`).Scan(&vendorID)
	if err != nil {
		t.Fatal(err)
	}

	var itemID string
	// price_minor = 333 (₹3.33)
	err = storage.DB.QueryRow(`
		INSERT INTO vendor_inventory (vendor_id, item_name, category, price, price_minor, stock_quantity, unit, is_available)
		VALUES ($1, 'Pure Ghee 100g', 'Samagri', 3.33, 333, 50, 'pack', true)
		RETURNING id
	`, vendorID).Scan(&itemID)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Devotee buys 3 items. Client attempts to pass fake price 1.00 (₹1)
	req := models.CreateStoreOrderRequest{
		UserID:          devoteeID,
		VendorID:        vendorID,
		DeliveryAddress: "Hyderabad",
		Items: []models.StoreOrderItem{
			{
				InventoryItemID: itemID,
				Quantity:        3,
				UnitPrice:       1.00, // fake price from client
				Subtotal:        3.00,
			},
		},
	}

	orderID, totalMinor, err := storage.CreateStoreOrder(req)
	if err != nil {
		t.Fatalf("CreateStoreOrder failed: %v", err)
	}

	// 3 items * 333 paise = exactly 999 paise (₹9.99), ignoring client price 1.00!
	if totalMinor != 999 {
		t.Fatalf("expected total_minor to be exactly 999, got %d", totalMinor)
	}

	// Check stock decremented by 3 (50 -> 47)
	var remainingStock int
	if err := storage.DB.QueryRow(`SELECT stock_quantity FROM vendor_inventory WHERE id=$1`, itemID).Scan(&remainingStock); err != nil {
		t.Fatal(err)
	}
	if remainingStock != 47 {
		t.Fatalf("expected remaining stock 47, got %d", remainingStock)
	}

	// 2. Out of stock check: attempting to order 48 items when 47 remain
	reqOutOfStock := models.CreateStoreOrderRequest{
		UserID:          "devotee-db-1",
		VendorID:        vendorID,
		DeliveryAddress: "Hyderabad",
		Items: []models.StoreOrderItem{
			{
				InventoryItemID: itemID,
				Quantity:        48,
			},
		},
	}
	_, _, err = storage.CreateStoreOrder(reqOutOfStock)
	if err != db.ErrStoreItemOutOfStock {
		t.Fatalf("expected ErrStoreItemOutOfStock, got %v", err)
	}

	// 3. Vendor mismatch check: item belonging to another vendor
	var otherVendorID string
	err = storage.DB.QueryRow(`
		INSERT INTO vendors (shop_name, owner_name, phone, address, city, latitude, longitude)
		VALUES ('Other Store', 'Owner Two', '+919900011224', 'Market St', 'Hyderabad', 17.385, 78.4867)
		RETURNING id
	`).Scan(&otherVendorID)
	if err != nil {
		t.Fatal(err)
	}

	reqMismatch := models.CreateStoreOrderRequest{
		UserID:          "devotee-db-1",
		VendorID:        otherVendorID, // mismatched vendor
		DeliveryAddress: "Hyderabad",
		Items: []models.StoreOrderItem{
			{
				InventoryItemID: itemID,
				Quantity:        1,
			},
		},
	}
	_, _, err = storage.CreateStoreOrder(reqMismatch)
	if err != db.ErrStoreItemVendorMismatch {
		t.Fatalf("expected ErrStoreItemVendorMismatch, got %v", err)
	}

	_ = orderID
}

// ----------------------------------------------------------------------------
// DEFECT 2 TESTS: Reserve-First Idempotency Middleware & Replays
// ----------------------------------------------------------------------------

func TestIdempotencyDifferentBodyReusedKey(t *testing.T) {
	_, storage, _ := setupBookingTestGateway(t)
	devoteeID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919988776655",
		FullName:      "Idemp Devotee",
		Password:      "SecureDev123!",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	router := buildGatewayRouter(storage, &notify.LoggingNoOpNotifier{})
	token := issueRoleToken(devoteeID, "devotee")
	idempKey := "key-diff-body-test-123"
	poojaID := firstPoojaID(t, storage)

	// 1. Initial request with Body A
	bodyA := `{"pooja_id":"`+poojaID+`","scheduled_date":"2026-12-01","total_amount":1500.0,"event_address":"Address A"}`
	reqA := httptest.NewRequest("POST", "/api/v1/bookings", bytes.NewBufferString(bodyA))
	reqA.Header.Set("Content-Type", "application/json")
	reqA.Header.Set("Authorization", "Bearer "+token)
	reqA.Header.Set("Idempotency-Key", idempKey)
	wA := httptest.NewRecorder()
	router.ServeHTTP(wA, reqA)

	if wA.Code != http.StatusOK {
		t.Fatalf("initial request failed: %d %s", wA.Code, wA.Body.String())
	}

	// 2. Second request reusing key with Body B (different payload) -> must return 422 Unprocessable Entity
	bodyB := `{"pooja_id":"`+poojaID+`","scheduled_date":"2026-12-01","total_amount":1500.0,"event_address":"DIFFERENT Address B"}`
	reqB := httptest.NewRequest("POST", "/api/v1/bookings", bytes.NewBufferString(bodyB))
	reqB.Header.Set("Content-Type", "application/json")
	reqB.Header.Set("Authorization", "Bearer "+token)
	reqB.Header.Set("Idempotency-Key", idempKey)
	wB := httptest.NewRecorder()
	router.ServeHTTP(wB, reqB)

	if wB.Code != 422 {
		t.Fatalf("expected 422 when reusing key with different payload, got %d: %s", wB.Code, wB.Body.String())
	}
}

func TestIdempotencyConcurrentSameKey(t *testing.T) {
	_, storage, _ := setupBookingTestGateway(t)
	devoteeID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919988776656",
		FullName:      "Concurrent Devotee",
		Password:      "SecureDev123!",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	router := buildGatewayRouter(storage, &notify.LoggingNoOpNotifier{})
	token := issueRoleToken(devoteeID, "devotee")
	sharedKey := "shared-concurrent-key-999"
	poojaID := firstPoojaID(t, storage)
	body := `{"pooja_id":"`+poojaID+`","scheduled_date":"2026-12-01","total_amount":2500.0,"event_address":"Concurrent Address"}`

	var wg sync.WaitGroup
	var successCount int64
	var conflictCount int64

	// Launch 20 concurrent requests with the identical idempotency key and body
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/api/v1/bookings", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Idempotency-Key", sharedKey)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
			} else if w.Code == http.StatusConflict {
				if retryAfter := w.Header().Get("Retry-After"); retryAfter != "" {
					atomic.AddInt64(&conflictCount, 1)
				}
			}
		}()
	}
	wg.Wait()

	if successCount < 1 {
		t.Fatalf("expected at least 1 success among concurrent requests, got %d", successCount)
	}

	// Verify only 1 booking was created in the database for this user
	var bookingCount int
	err = storage.DB.QueryRow(`SELECT count(*) FROM bookings WHERE user_id=$1`, devoteeID).Scan(&bookingCount)
	if err != nil {
		t.Fatal(err)
	}
	if bookingCount != 1 {
		t.Fatalf("expected exactly 1 booking in database, got %d (idempotency race failure!)", bookingCount)
	}

	// Replaying after completion must return 200 with Idempotent-Replayed: true
	reqReplay := httptest.NewRequest("POST", "/api/v1/bookings", bytes.NewBufferString(body))
	reqReplay.Header.Set("Content-Type", "application/json")
	reqReplay.Header.Set("Authorization", "Bearer "+token)
	reqReplay.Header.Set("Idempotency-Key", sharedKey)
	wReplay := httptest.NewRecorder()
	router.ServeHTTP(wReplay, reqReplay)

	if wReplay.Code != http.StatusOK {
		t.Fatalf("expected 200 on replay, got %d", wReplay.Code)
	}
	if wReplay.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatalf("expected Idempotent-Replayed: true header, got %q", wReplay.Header().Get("Idempotent-Replayed"))
	}
}

func firstPoojaID(t *testing.T, storage *db.Storage) string {
	var id string
	if err := storage.DB.QueryRow(`SELECT id FROM pooja_catalog LIMIT 1`).Scan(&id); err != nil {
		t.Fatalf("no pooja in catalog: %v", err)
	}
	return id
}
