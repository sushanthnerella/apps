package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/models"
)

var istZone = time.FixedZone("IST", 5*3600+1800)

// validateNewSlot rejects past slots. Slot date/time are interpreted in IST.
func validateNewSlot(req models.CreateSlotRequest, now time.Time) error {
	start, err := time.ParseInLocation("2006-01-02 15:04", req.SlotDate+" "+req.StartTime, istZone)
	if err != nil {
		return db.ErrSlotInvalid
	}
	if !start.After(now) {
		return errSlotInPast
	}
	return nil
}

var errSlotInPast = errors.New("slot start time is in the past")

func writeSlotError(w http.ResponseWriter, err error) {
	code, status := "SLOT_ERROR", http.StatusInternalServerError
	switch {
	case errors.Is(err, db.ErrSlotInvalid):
		code, status = "SLOT_INVALID", http.StatusBadRequest
	case errors.Is(err, errSlotInPast):
		code, status = "SLOT_IN_PAST", http.StatusBadRequest
	case errors.Is(err, db.ErrSlotOverlap):
		code, status = "SLOT_OVERLAP", http.StatusConflict
	case errors.Is(err, db.ErrSlotNotRemovable):
		code, status = "SLOT_NOT_REMOVABLE", http.StatusConflict
	case errors.Is(err, db.ErrSlotNotFound):
		code, status = "SLOT_NOT_FOUND", http.StatusNotFound
	}
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error(), "code": code})
}

func registerPriestSlotRoutes(router *mux.Router, storage *db.Storage) {
	router.HandleFunc("/api/v1/priests/slots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			priestID := r.URL.Query().Get("priest_id")
			date := r.URL.Query().Get("date")
			if date == "" {
				date = time.Now().In(istZone).Format("2006-01-02")
			}
			slots, err := storage.GetPriestSlots(priestID, date)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(slots)
			return
		}
		if r.Method == "POST" {
			claims, err := validateCallerSession(r, storage)
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			if claims.Role != "priest" {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			var req models.CreateSlotRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid slot payload"}`, http.StatusBadRequest)
				return
			}
			if req.PriestID != claims.UserID {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			if err := validateNewSlot(req, time.Now()); err != nil {
				writeSlotError(w, err)
				return
			}
			slotID, err := storage.CreatePriestSlot(req)
			if err != nil {
				writeSlotError(w, err)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "SUCCESS",
				"slot_id": slotID,
				"message": "Availability slot published successfully",
			})
		}
	}).Methods("GET", "POST", "OPTIONS")

	router.HandleFunc("/api/v1/priests/slots/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := validateCallerSession(r, storage)
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if claims.Role != "priest" {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		if err := storage.DeletePriestSlot(claims.UserID, mux.Vars(r)["id"]); err != nil {
			writeSlotError(w, err)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "SUCCESS"})
	}).Methods("DELETE", "OPTIONS")
}
