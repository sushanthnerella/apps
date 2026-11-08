package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/db"
	"net/http"
)

func registerDutyRoutes(router *mux.Router, storage *db.Storage) {
	router.HandleFunc("/api/v1/priests/me/duty", func(w http.ResponseWriter, r *http.Request) {
		claims, err := validateCallerSession(r, storage)
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if claims.Role != "priest" {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		var online bool
		if r.Method == http.MethodPut {
			var body struct {
				IsOnline *bool `json:"is_online"`
			}
			if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body) != nil || body.IsOnline == nil {
				http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
				return
			}
			err = storage.DB.QueryRowContext(r.Context(), `UPDATE priests SET is_online=$1 WHERE id=$2::uuid RETURNING is_online`, *body.IsOnline, claims.UserID).Scan(&online)
		} else {
			err = storage.DB.QueryRowContext(r.Context(), `SELECT is_online FROM priests WHERE id=$1::uuid`, claims.UserID).Scan(&online)
		}
		if err != nil {
			http.Error(w, `{"error":"duty_unavailable"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"is_online": online})
	}).Methods(http.MethodGet, http.MethodPut)
}
