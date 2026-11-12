package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/db"
)

func validCoordinates(lat, lon float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lon) && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func validLocation(lat, lon float64) bool {
	return validCoordinates(lat, lon)
}

func registerPriestLocationRoutes(router *mux.Router, storage *db.Storage) {
	router.HandleFunc("/api/v1/priests/me/location", func(w http.ResponseWriter, r *http.Request) {
		claims, err := validateCallerSession(r, storage)
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if claims.Role != "priest" {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}

		var body struct {
			Latitude   *float64   `json:"latitude"`
			Longitude  *float64   `json:"longitude"`
			AccuracyM  *float64   `json:"accuracy_m,omitempty"`
			CapturedAt *time.Time `json:"captured_at"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body) != nil {
			http.Error(w, `{"error":"invalid_payload"}`, http.StatusBadRequest)
			return
		}

		if body.Latitude == nil || body.Longitude == nil || !validCoordinates(*body.Latitude, *body.Longitude) {
			http.Error(w, `{"error":"invalid_coordinates","message":"latitude must be between -90 and 90, longitude between -180 and 180"}`, http.StatusBadRequest)
			return
		}

		if body.AccuracyM != nil && (*body.AccuracyM < 0 || *body.AccuracyM > 10000 || math.IsNaN(*body.AccuracyM)) {
			http.Error(w, `{"error":"invalid_accuracy","message":"accuracy_m must be between 0 and 10000"}`, http.StatusBadRequest)
			return
		}
		accuracy := 0.0
		if body.AccuracyM != nil {
			accuracy = *body.AccuracyM
		}

		if body.CapturedAt == nil || body.CapturedAt.IsZero() {
			http.Error(w, `{"error":"missing_captured_at","message":"captured_at is required in RFC 3339 format"}`, http.StatusBadRequest)
			return
		}

		capturedTime := body.CapturedAt.UTC()
		now := time.Now().UTC()
		if capturedTime.After(now.Add(2 * time.Minute)) {
			http.Error(w, `{"error":"future_timestamp","message":"captured_at cannot be more than 2 minutes in the future"}`, http.StatusBadRequest)
			return
		}
		if capturedTime.Before(now.Add(-30 * time.Minute)) {
			http.Error(w, `{"error":"expired_timestamp","message":"captured_at cannot be older than 30 minutes"}`, http.StatusBadRequest)
			return
		}

		if storage != nil && storage.DB != nil {
			// Reject update if priest is currently off duty
			var isOnline bool
			err := storage.DB.QueryRowContext(r.Context(), `SELECT is_online FROM priests WHERE id=$1::uuid`, claims.UserID).Scan(&isOnline)
			if err != nil {
				http.Error(w, `{"error":"location_unavailable"}`, http.StatusInternalServerError)
				return
			}
			if !isOnline {
				http.Error(w, `{"error":"off_duty","message":"location updates are only accepted while on duty"}`, http.StatusConflict)
				return
			}

			// Atomic update: only updates if location_captured_at is null or older than this update
			query := `
				UPDATE priests 
				SET latitude=$2, longitude=$3, location_accuracy_m=$4, location_captured_at=$5, location_confirmed=true 
				WHERE id=$1::uuid AND is_online=true AND (location_captured_at IS NULL OR location_captured_at < $5)
				RETURNING id
			`
			var updatedID string
			err = storage.DB.QueryRowContext(r.Context(), query, claims.UserID, *body.Latitude, *body.Longitude, accuracy, capturedTime).Scan(&updatedID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"applied": false,
						"reason":  "stale",
					})
					return
				}
				http.Error(w, `{"error":"location_unavailable"}`, http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"applied": true,
			"status":  "updated",
		})
	}).Methods(http.MethodPut, http.MethodPost, "OPTIONS")
}
