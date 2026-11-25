package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/models"
	"github.com/purohit-platform/backend/internal/notify"
)

// registerEventRoutes mounts the public event config endpoint and the admin event controls (PRD v2 EVT, CMS).
func registerEventRoutes(router *mux.Router, storage *db.Storage, push notify.PushNotifier) {
	if push == nil {
		push = &notify.LoggingNoOpNotifier{}
	}

	// Public: apps fetch this on launch, on resume when older than 24 h, and on an event-config push.
	router.HandleFunc("/api/v1/events/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		event, err := storage.GetLiveEventConfig(r.Context())
		if err != nil {
			writeEventError(w, err)
			return
		}
		etag := `"none"`
		if event != nil {
			etag = fmt.Sprintf(`"%s-%d"`, event.ID, event.ConfigVersion)
		}
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "public, max-age=60")
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"event": event})
	}).Methods(http.MethodGet)

	router.HandleFunc("/api/v1/priests/events/config", requireRole(storage, "priest")(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		event, err := storage.GetPriestEventConfig(r.Context())
		if err != nil {
			writeEventError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"event": event})
	})).Methods(http.MethodGet)

	admin := func(h http.HandlerFunc) http.HandlerFunc { return requireRole(storage, "admin")(h) }

	router.HandleFunc("/api/v1/admin/events", admin(func(w http.ResponseWriter, r *http.Request) {
		events, err := storage.ListEvents(r.Context())
		if err != nil {
			writeEventError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(events)
	})).Methods(http.MethodGet)

	router.HandleFunc("/api/v1/admin/events", admin(func(w http.ResponseWriter, r *http.Request) {
		var req models.CreateEventRequest
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req) != nil {
			writeEventError(w, db.ErrInvalidEventInput)
			return
		}
		event, err := storage.CreateEvent(r.Context(), req, getCallerClaims(r).UserID)
		if err != nil {
			writeEventError(w, err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(event)
	})).Methods(http.MethodPost)

	router.HandleFunc("/api/v1/admin/events/{id}", admin(func(w http.ResponseWriter, r *http.Request) {
		event, err := storage.GetEvent(r.Context(), mux.Vars(r)["id"])
		if err != nil {
			writeEventError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(event)
	})).Methods(http.MethodGet)

	router.HandleFunc("/api/v1/admin/events/{id}", admin(func(w http.ResponseWriter, r *http.Request) {
		var req models.UpdateEventRequest
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req) != nil {
			writeEventError(w, db.ErrInvalidEventInput)
			return
		}
		event, err := storage.UpdateEvent(r.Context(), mux.Vars(r)["id"], req, getCallerClaims(r).UserID)
		if err != nil {
			writeEventError(w, err)
			return
		}
		notifyEventConfigChanged(push, event.ConfigVersion)
		_ = json.NewEncoder(w).Encode(event)
	})).Methods(http.MethodPatch)

	router.HandleFunc("/api/v1/admin/events/{id}/ghats", admin(func(w http.ResponseWriter, r *http.Request) {
		var req models.UpsertGhatRequest
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req) != nil {
			writeEventError(w, db.ErrInvalidEventInput)
			return
		}
		ghat, err := storage.CreateGhat(r.Context(), mux.Vars(r)["id"], req, getCallerClaims(r).UserID)
		if err != nil {
			writeEventError(w, err)
			return
		}
		notifyEventConfigChanged(push, 0)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(ghat)
	})).Methods(http.MethodPost)

	router.HandleFunc("/api/v1/admin/events/{id}/ghats/{ghatId}", admin(func(w http.ResponseWriter, r *http.Request) {
		var req models.UpsertGhatRequest
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req) != nil {
			writeEventError(w, db.ErrInvalidEventInput)
			return
		}
		vars := mux.Vars(r)
		ghat, err := storage.UpdateGhat(r.Context(), vars["id"], vars["ghatId"], req, getCallerClaims(r).UserID)
		if err != nil {
			writeEventError(w, err)
			return
		}
		notifyEventConfigChanged(push, 0)
		_ = json.NewEncoder(w).Encode(ghat)
	})).Methods(http.MethodPut)

	router.HandleFunc("/api/v1/admin/events/{id}/audit", admin(func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		entries, err := storage.ListEventAudit(r.Context(), mux.Vars(r)["id"], limit)
		if err != nil {
			writeEventError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(entries)
	})).Methods(http.MethodGet)
}

// notifyEventConfigChanged asks every app to refetch config. Delivery failure is logged, never fatal:
// apps still refresh on launch and on resume.
func notifyEventConfigChanged(push notify.PushNotifier, version int64) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		data := map[string]string{"type": "event_config", "version": strconv.FormatInt(version, 10)}
		if err := push.SendDataToTopic(ctx, notify.EventConfigTopic, data); err != nil {
			log.Printf("event config push failed: %v", err)
		}
	}()
}

func writeEventError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	status, code := http.StatusInternalServerError, "server_error"
	var pqErr *pq.Error
	switch {
	case errors.As(err, &pqErr) && pqErr.Code == "22P02": // malformed UUID in path
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, db.ErrEventNotFound), errors.Is(err, db.ErrGhatNotFound):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, db.ErrInvalidEventInput):
		status, code = http.StatusBadRequest, "invalid_request"
	case errors.Is(err, db.ErrInvalidEventTransition):
		status, code = http.StatusConflict, "invalid_transition"
	case errors.Is(err, db.ErrLiveEventConflict):
		status, code = http.StatusConflict, "another_event_live"
	case errors.Is(err, db.ErrDuplicateEventData):
		status, code = http.StatusConflict, "duplicate"
	case errors.Is(err, db.ErrParticipationMissing):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, db.ErrEventBookingClosed):
		status, code = http.StatusConflict, "event_booking_disabled"
	case errors.Is(err, db.ErrEventNotOpen):
		status, code = http.StatusConflict, "event_not_open"
	case errors.Is(err, db.ErrParticipationState):
		status, code = http.StatusConflict, "participation_state"
	case errors.Is(err, db.ErrNotApprovedForEvent):
		status, code = http.StatusForbidden, "not_approved"
	case errors.Is(err, db.ErrPriestUnavailable):
		status, code = http.StatusConflict, "priest_unavailable"
	case errors.Is(err, db.ErrPriestBusy):
		status, code = http.StatusConflict, "priest_busy"
	case errors.Is(err, db.ErrServiceUnavailable):
		status, code = http.StatusConflict, "service_unavailable"
	case errors.Is(err, db.ErrInvalidSchedule):
		status, code = http.StatusBadRequest, "invalid_schedule"
	default:
		log.Printf("event route error: %v", err)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}
