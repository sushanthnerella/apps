package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/db"
)

func TestTruthfulHealthEndpointReportsConnectedWhenHealthy(t *testing.T) {
	// 1. Setup isolated test DB
	storage, _, _ := db.CreateIsolatedAuthTestDBForTest(t)

	router := mux.NewRouter()
	registerHealthRoutes(router, storage)

	// Since isolated test DB does not have Redis, storage.Redis is nil.
	// PingRedis will return "redis not initialized", so status is DEGRADED, postgres is CONNECTED, redis is UNAVAILABLE.
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 for healthy PostgreSQL, got %d", rr.Code)
	}

	var resp HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed decoding health response: %v", err)
	}

	if resp.Postgres != "CONNECTED" {
		t.Errorf("expected Postgres CONNECTED, got %s", resp.Postgres)
	}
	if resp.Redis != "UNAVAILABLE" {
		t.Errorf("expected Redis UNAVAILABLE without redis, got %s", resp.Redis)
	}
	if resp.Status != "DEGRADED" {
		t.Errorf("expected overall status DEGRADED without Redis, got %s", resp.Status)
	}
}

func TestTruthfulHealthEndpointReportsDownWhenDatabaseUnavailable(t *testing.T) {
	// Storage pointing to closed / invalid DB
	closedDB, _ := sql.Open("postgres", "postgres://invalid:invalid@127.0.0.1:54329/nonexistent?sslmode=disable&connect_timeout=1")
	closedStorage := &db.Storage{
		DB:    closedDB,
		Redis: nil,
	}
	defer closedDB.Close()

	router := mux.NewRouter()
	registerHealthRoutes(router, closedStorage)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected HTTP 503 when PostgreSQL is down, got %d", rr.Code)
	}

	var resp HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed decoding health response: %v", err)
	}

	if resp.Postgres != "DISCONNECTED" {
		t.Errorf("expected Postgres DISCONNECTED, got %s", resp.Postgres)
	}
	if resp.Status != "DOWN" {
		t.Errorf("expected overall status DOWN, got %s", resp.Status)
	}
}

func TestLivenessEndpointAlwaysReturnsOK(t *testing.T) {
	router := mux.NewRouter()
	registerHealthRoutes(router, &db.Storage{})

	req := httptest.NewRequest("GET", "/health/live", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 for liveness check, got %d", rr.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed decoding liveness response: %v", err)
	}
	if body["status"] != "ALIVE" {
		t.Errorf("expected status ALIVE, got %v", body["status"])
	}
}
