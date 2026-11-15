package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/models"
	"github.com/purohit-platform/backend/internal/notify"
)

// registerEventOpsRoutes mounts participation, priest prices, ghat duty, discovery, event bookings,
// the public card status page and the CMS review endpoints (PRD v2 PRI, SRV, DUT, DEV, BKG, ID, CMS).
func registerEventOpsRoutes(router *mux.Router, storage *db.Storage, push notify.PushNotifier) {
	if push == nil {
		push = &notify.LoggingNoOpNotifier{}
	}
	priest := func(h http.HandlerFunc) http.HandlerFunc { return requireRole(storage, "priest")(h) }
	devotee := func(h http.HandlerFunc) http.HandlerFunc { return requireRole(storage, "devotee", "user")(h) }
	admin := func(h http.HandlerFunc) http.HandlerFunc { return requireRole(storage, "admin")(h) }
	decode := func(w http.ResponseWriter, r *http.Request, v any) bool {
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(v) != nil {
			writeEventError(w, db.ErrInvalidEventInput)
			return false
		}
		return true
	}
	ok := func(w http.ResponseWriter, v any) { _ = json.NewEncoder(w).Encode(v) }

	// ----- Priest -----
	router.HandleFunc("/api/v1/events/{id}/participation/me", priest(func(w http.ResponseWriter, r *http.Request) {
		p, err := storage.GetParticipation(r.Context(), mux.Vars(r)["id"], getCallerClaims(r).UserID)
		if err != nil {
			writeEventError(w, err)
			return
		}
		ok(w, p)
	})).Methods(http.MethodGet)

	router.HandleFunc("/api/v1/events/{id}/participation/me", priest(func(w http.ResponseWriter, r *http.Request) {
		id, pid := mux.Vars(r)["id"], getCallerClaims(r).UserID
		if err := storage.SubmitParticipation(r.Context(), id, pid); err != nil {
			writeEventError(w, err)
			return
		}
		p, err := storage.GetParticipation(r.Context(), id, pid)
		if err != nil {
			writeEventError(w, err)
			return
		}
		ok(w, p)
	})).Methods(http.MethodPost)

	router.HandleFunc("/api/v1/events/{id}/services/me", priest(func(w http.ResponseWriter, r *http.Request) {
		var req models.SetServicesRequest
		if !decode(w, r, &req) {
			return
		}
		id, pid := mux.Vars(r)["id"], getCallerClaims(r).UserID
		if err := storage.SetPriestServices(r.Context(), id, pid, req); err != nil {
			writeEventError(w, err)
			return
		}
		p, err := storage.GetParticipation(r.Context(), id, pid)
		if err != nil {
			writeEventError(w, err)
			return
		}
		ok(w, p)
	})).Methods(http.MethodPut)

	router.HandleFunc("/api/v1/events/{id}/duty/me", priest(func(w http.ResponseWriter, r *http.Request) {
		var req models.DutyRequest
		if !decode(w, r, &req) {
			return
		}
		d, err := storage.SetDuty(r.Context(), mux.Vars(r)["id"], getCallerClaims(r).UserID, req)
		if err != nil {
			writeEventError(w, err)
			return
		}
		ok(w, d)
	})).Methods(http.MethodPut)

	router.HandleFunc("/api/v1/events/{id}/duty/me", priest(func(w http.ResponseWriter, r *http.Request) {
		if err := storage.EndDuty(r.Context(), mux.Vars(r)["id"], getCallerClaims(r).UserID); err != nil {
			writeEventError(w, err)
			return
		}
		ok(w, map[string]bool{"on_duty": false})
	})).Methods(http.MethodDelete)

	// ----- Devotee -----
	router.HandleFunc("/api/v1/events/{id}/ghats/availability", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		a, err := storage.GhatAvailability(r.Context(), mux.Vars(r)["id"])
		if err != nil {
			writeEventError(w, err)
			return
		}
		ok(w, a)
	}).Methods(http.MethodGet)

	router.HandleFunc("/api/v1/events/{id}/ghats/{ghatId}/priests", devotee(func(w http.ResponseWriter, r *http.Request) {
		v := mux.Vars(r)
		list, err := storage.ListGhatPriests(r.Context(), v["id"], v["ghatId"])
		if err != nil {
			writeEventError(w, err)
			return
		}
		ok(w, list)
	})).Methods(http.MethodGet)

	router.HandleFunc("/api/v1/events/{id}/bookings", devotee(requireIdempotency(storage)(func(w http.ResponseWriter, r *http.Request) {
		var req models.CreateEventBookingRequest
		if !decode(w, r, &req) {
			return
		}
		res, err := storage.CreateEventBooking(r.Context(), getCallerClaims(r).UserID, mux.Vars(r)["id"], req, time.Now())
		if err != nil {
			writeEventError(w, err)
			return
		}
		go func(priestID, bookingID string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			tokens, _ := storage.GetDeviceTokensForUser(ctx, priestID)
			_ = push.SendToTokens(ctx, tokens, notify.NotificationMessage{
				Title: "New ghat request",
				Body:  "A devotee is requesting a ritual at your ghat.",
				Data:  map[string]string{"booking_id": bookingID, "type": "NEW_BOOKING"},
			})
		}(req.PriestID, res.BookingID)
		w.WriteHeader(http.StatusCreated)
		ok(w, res)
	}))).Methods(http.MethodPost)

	// ----- Public QR card page (status only) -----
	router.HandleFunc("/api/v1/cards/{code}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=60")
		c, err := storage.GetPublicCardStatus(r.Context(), mux.Vars(r)["code"])
		if err != nil {
			writeEventError(w, err)
			return
		}
		ok(w, c)
	}).Methods(http.MethodGet)

	// ----- CMS -----
	router.HandleFunc("/api/v1/admin/events/{id}/rituals", admin(func(w http.ResponseWriter, r *http.Request) {
		list, err := storage.ListRituals(r.Context(), mux.Vars(r)["id"], false)
		if err != nil {
			writeEventError(w, err)
			return
		}
		ok(w, list)
	})).Methods(http.MethodGet)

	upsertRitual := func(w http.ResponseWriter, r *http.Request) {
		var req models.UpsertRitualRequest
		if !decode(w, r, &req) {
			return
		}
		v := mux.Vars(r)
		rit, err := storage.UpsertRitual(r.Context(), v["id"], v["ritualId"], req, getCallerClaims(r).UserID)
		if err != nil {
			writeEventError(w, err)
			return
		}
		ok(w, rit)
	}
	router.HandleFunc("/api/v1/admin/events/{id}/rituals", admin(upsertRitual)).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/admin/events/{id}/rituals/{ritualId}", admin(upsertRitual)).Methods(http.MethodPut)

	router.HandleFunc("/api/v1/admin/events/{id}/participations", admin(func(w http.ResponseWriter, r *http.Request) {
		list, err := storage.ListParticipations(r.Context(), mux.Vars(r)["id"])
		if err != nil {
			writeEventError(w, err)
			return
		}
		ok(w, list)
	})).Methods(http.MethodGet)

	router.HandleFunc("/api/v1/admin/events/{id}/participations/{priestId}", admin(func(w http.ResponseWriter, r *http.Request) {
		var req models.ReviewParticipationRequest
		if !decode(w, r, &req) {
			return
		}
		v := mux.Vars(r)
		if err := storage.ReviewParticipation(r.Context(), v["id"], v["priestId"], req, getCallerClaims(r).UserID); err != nil {
			writeEventError(w, err)
			return
		}
		ok(w, map[string]string{"status": req.Status})
	})).Methods(http.MethodPut)
}

// startEventRequestExpiry cancels unanswered event requests every 30 s (PRD v2 BKG-003).
func startEventRequestExpiry(ctx context.Context, storage *db.Storage) {
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if n, err := storage.ExpireEventRequests(ctx); err != nil {
					log.Printf("event request expiry failed: %v", err)
				} else if n > 0 {
					log.Printf("expired %d unanswered event requests", n)
				}
			}
		}
	}()
}
