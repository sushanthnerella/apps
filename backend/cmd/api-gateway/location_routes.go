package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/db"
	"math"
	"net/http"
)

func validLocation(lat, lon float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lon) && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180 && (lat != 0 || lon != 0)
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
			Latitude  *float64 `json:"latitude"`
			Longitude *float64 `json:"longitude"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body) != nil || body.Latitude == nil || body.Longitude == nil || !validLocation(*body.Latitude, *body.Longitude) {
			http.Error(w, `{"error":"invalid_location"}`, http.StatusBadRequest)
			return
		}
		result, err := storage.DB.ExecContext(r.Context(), `UPDATE priests SET latitude=$1, longitude=$2, location_confirmed=true WHERE id=$3::uuid`, *body.Latitude, *body.Longitude, claims.UserID)
		if err != nil {
			http.Error(w, `{"error":"location_unavailable"}`, http.StatusInternalServerError)
			return
		}
		n, err := result.RowsAffected()
		if err != nil || n != 1 {
			http.Error(w, `{"error":"location_unavailable"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}).Methods(http.MethodPut)
}
