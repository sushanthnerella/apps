package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/db"
)

// HealthResponse represents the truthful health response returned to orchestrators like Railway
type HealthResponse struct {
	Status    string            `json:"status"` // "UP" if core DB is functional, "DEGRADED" if Redis is down, "DOWN" if DB is unreachable
	Timestamp string            `json:"timestamp"`
	Version   string            `json:"version"`
	Postgres  string            `json:"postgres"` // "CONNECTED" or "DISCONNECTED"
	Redis     string            `json:"redis"`    // "CONNECTED" or "UNAVAILABLE"
	Services  map[string]string `json:"services,omitempty"`
}

// registerHealthRoutes mounts truthful health and readiness check routes for Railway and k8s
func registerHealthRoutes(router *mux.Router, storage *db.Storage) {
	// 1. /health and /health/ready: Comprehensive dependency check
	// Returns 200 OK when Postgres is healthy (ready to serve requests)
	// Returns 503 Service Unavailable when Postgres is down
	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		pgStatus := "CONNECTED"
		redisStatus := "CONNECTED"
		overallStatus := "UP"
		httpCode := http.StatusOK

		if err := storage.PingDB(ctx); err != nil {
			pgStatus = "DISCONNECTED"
			overallStatus = "DOWN"
			httpCode = http.StatusServiceUnavailable
		}

		if err := storage.PingRedis(ctx); err != nil {
			redisStatus = "UNAVAILABLE"
			if overallStatus == "UP" {
				overallStatus = "DEGRADED"
			}
		}

		w.WriteHeader(httpCode)
		json.NewEncoder(w).Encode(HealthResponse{
			Status:    overallStatus,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "2.5.0-railway-staging",
			Postgres:  pgStatus,
			Redis:     redisStatus,
			Services: map[string]string{
				"postgres": pgStatus,
				"redis":    redisStatus,
			},
		})
	}

	router.HandleFunc("/health", healthHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/health/ready", healthHandler).Methods("GET", "OPTIONS")

	// 2. /health/live: Pure process liveness check (process running and accepting connections)
	router.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ALIVE",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}).Methods("GET", "OPTIONS")
}
