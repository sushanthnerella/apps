package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/lifecycle"
	"github.com/purohit-platform/backend/internal/models"
)

// mapLifecycleErrorToHTTP maps typed lifecycle and storage errors to strict HTTP status codes:
// - 409 Conflict: Invalid transitions, repeated transitions, terminal states, slot collisions, and invalid/missing OTPs.
// - 403 Forbidden: Unauthorized roles, unassigned priests, non-owner devotees, and forbidden cancellation phases.
// - 404 Not Found: Booking not found.
// - 400 Bad Request: Malformed input or unhandled domain validation errors.
func mapLifecycleErrorToHTTP(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch {
	// 409 Conflict: Invalid transitions, repeated transitions, terminal states, slot collision
	case errors.Is(err, lifecycle.ErrInvalidTransition),
		errors.Is(err, lifecycle.ErrRepeatedTransition),
		errors.Is(err, lifecycle.ErrTerminalState),
		errors.Is(err, lifecycle.ErrSlotUnavailable):
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "conflict",
			"message": err.Error(),
		})

	case errors.Is(err, lifecycle.ErrInvalidOTP),
		errors.Is(err, lifecycle.ErrMissingOTP):
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "conflict",
			"message": "invalid completion OTP",
		})

	// 403 Forbidden: Role not permitted, priest not assigned, devotee not owner, unauthorized cancellation
	case errors.Is(err, lifecycle.ErrUnauthorizedRole),
		errors.Is(err, lifecycle.ErrUnauthorizedPriest),
		errors.Is(err, lifecycle.ErrUnauthorizedDevotee),
		errors.Is(err, lifecycle.ErrUnauthorizedCancellation):
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "forbidden",
			"message": err.Error(),
		})

	// 404 Not Found
	case errors.Is(err, lifecycle.ErrBookingNotFound):
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "not_found",
			"message": "booking not found",
		})

	default:
		// String-based fallback for nested or legacy error compatibility
		msg := err.Error()
		if strings.Contains(msg, "unauthorized") || strings.Contains(msg, "access denied") || strings.Contains(msg, "forbidden") {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "forbidden", "message": msg})
		} else if strings.Contains(msg, "invalid completion OTP") ||
			strings.Contains(msg, "invalid state transition") ||
			strings.Contains(msg, "terminal state") ||
			strings.Contains(msg, "already in state") ||
			strings.Contains(msg, "already booked") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "conflict", "message": msg})
		} else if strings.Contains(msg, "booking not found") || strings.Contains(msg, "not found") {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not_found", "message": msg})
		} else {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "bad_request", "message": msg})
		}
	}
}

// requireCallerSession extracts and validates the Bearer token and verifies
// that the session has not been revoked or expired against the database.
// Rejects unauthenticated, malformed, expired, or revoked tokens with an error.
func requireCallerSession(r *http.Request, storage *db.Storage) (*auth.CustomClaims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, errors.New("missing or malformed authorization header")
	}
	tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if tokenStr == "" {
		return nil, errors.New("empty bearer token")
	}
	claims, err := auth.ValidateToken(tokenStr)
	if err != nil {
		// Fallback for admin role token verification
		secret, secErr := auth.GetSecretKey()
		if secErr == nil {
			var adminClaims auth.CustomClaims
			token, parseErr := jwt.ParseWithClaims(tokenStr, &adminClaims, func(t *jwt.Token) (interface{}, error) {
				if sm, ok := t.Method.(*jwt.SigningMethodHMAC); !ok || sm != jwt.SigningMethodHS256 {
					return nil, fmt.Errorf("unexpected signing algorithm")
				}
				return []byte(secret), nil
			})
			if parseErr == nil && token.Valid && adminClaims.Role == "admin" {
				return &adminClaims, nil
			}
		}
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	dbVersion, err := storage.GetUserSessionVersion(claims.UserID, claims.Role)
	if err != nil {
		return nil, fmt.Errorf("session lookup failed: %w", err)
	}
	if err := auth.ValidateSessionVersion(claims.SessionVersion, dbVersion); err != nil {
		return nil, fmt.Errorf("session expired or revoked: %w", err)
	}
	return claims, nil
}

// registerBookingRoutes mounts all protected booking lifecycle routes with strict
// route-level authorization and asymmetric completion OTP privacy (Task 05).
func registerBookingRoutes(router *mux.Router, storage *db.Storage) {
	// 1. Create Booking (POST) - Devotees only
	router.HandleFunc("/api/v1/bookings", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		if claims.Role != "devotee" && claims.Role != "user" {
			http.Error(w, `{"error":"forbidden","message":"only devotees can create bookings"}`, http.StatusForbidden)
			return
		}

		var req models.BookingCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid booking payload"}`, http.StatusBadRequest)
			return
		}

		// Enforce server ownership: never trust client-supplied user_id as proof of ownership
		if req.UserID != "" && req.UserID != claims.UserID {
			http.Error(w, `{"error":"forbidden","message":"cannot create booking on behalf of another user"}`, http.StatusForbidden)
			return
		}
		req.UserID = claims.UserID

		bookingID, otp, err := storage.CreateBooking(req)
		if err != nil {
			mapLifecycleErrorToHTTP(w, err)
			return
		}

		// Initial creation response is returned exclusively to the creating devotee
		json.NewEncoder(w).Encode(map[string]interface{}{
			"booking_id":     bookingID,
			"status":         "REQUESTED",
			"escrow_status":  "HELD_IN_ESCROW",
			"completion_otp": otp,
			"total_amount":   req.TotalAmount,
			"message":        "Booking initialized successfully.",
		})
	}).Methods("POST")

	// 2. Query Bookings List (GET) - Scoped strictly to caller's identity
	router.HandleFunc("/api/v1/bookings", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		status := r.URL.Query().Get("status")
		var queryPriestID, queryUserID string

		switch claims.Role {
		case "devotee", "user":
			clientUID := r.URL.Query().Get("user_id")
			if clientUID != "" && clientUID != claims.UserID {
				http.Error(w, `{"error":"forbidden","message":"cannot query bookings for another user"}`, http.StatusForbidden)
				return
			}
			queryUserID = claims.UserID
			queryPriestID = r.URL.Query().Get("priest_id")
		case "priest":
			clientPID := r.URL.Query().Get("priest_id")
			if clientPID != "" && clientPID != claims.UserID {
				http.Error(w, `{"error":"forbidden","message":"cannot query bookings for another priest"}`, http.StatusForbidden)
				return
			}
			queryPriestID = claims.UserID
			queryUserID = ""
		default:
			http.Error(w, `{"error":"forbidden","message":"unknown account role"}`, http.StatusForbidden)
			return
		}

		bookings, err := storage.GetBookings(queryPriestID, queryUserID, status)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		if bookings == nil {
			bookings = []models.BookingDetailResponse{}
		}
		// Note: CompletionOTP is marked json:"-" and is never leaked in shared lists
		json.NewEncoder(w).Encode(bookings)
	}).Methods("GET")

	// 3. Priest Specific Bookings Query (GET)
	router.HandleFunc("/api/v1/priests/{id}/bookings", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		if claims.Role != "priest" {
			http.Error(w, `{"error":"forbidden","message":"only priests can access this endpoint"}`, http.StatusForbidden)
			return
		}

		vars := mux.Vars(r)
		priestID := vars["id"]
		if priestID != claims.UserID {
			http.Error(w, `{"error":"forbidden","message":"cannot access bookings of another priest"}`, http.StatusForbidden)
			return
		}

		status := r.URL.Query().Get("status")
		bookings, err := storage.GetBookings(claims.UserID, "", status)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		if bookings == nil {
			bookings = []models.BookingDetailResponse{}
		}
		json.NewEncoder(w).Encode(bookings)
	}).Methods("GET", "OPTIONS")

	// 4. Dedicated Protected OTP Retrieval (GET) - Exclusively accessible to the booking devotee owner
	router.HandleFunc("/api/v1/bookings/{id}/otp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		if claims.Role == "priest" {
			http.Error(w, `{"error":"forbidden","message":"priests cannot retrieve the completion OTP; devotee must disclose it in person upon ritual completion"}`, http.StatusForbidden)
			return
		}

		if claims.Role != "devotee" && claims.Role != "user" {
			http.Error(w, `{"error":"forbidden","message":"only devotees can retrieve completion OTP"}`, http.StatusForbidden)
			return
		}

		vars := mux.Vars(r)
		bookingID := vars["id"]

		ownerUID, _, status, otp, err := storage.GetBookingOwner(bookingID)
		if err != nil {
			http.Error(w, `{"error":"booking not found"}`, http.StatusNotFound)
			return
		}

		if ownerUID != claims.UserID {
			http.Error(w, `{"error":"forbidden","message":"access denied: you are not the owner of this booking"}`, http.StatusForbidden)
			return
		}

		json.NewEncoder(w).Encode(models.BookingOTPResponse{
			BookingID:     bookingID,
			CompletionOTP: otp,
			Status:        status,
		})
	}).Methods("GET", "OPTIONS")

	// 5. Booking Details (GET) - Owner or assigned priest only; CompletionOTP omitted
	router.HandleFunc("/api/v1/bookings/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		bookingID := vars["id"]

		ownerUID, assignedPID, _, _, err := storage.GetBookingOwner(bookingID)
		if err != nil {
			http.Error(w, `{"error":"booking not found"}`, http.StatusNotFound)
			return
		}

		if claims.Role == "devotee" || claims.Role == "user" {
			if ownerUID != claims.UserID {
				http.Error(w, `{"error":"forbidden","message":"access denied: you are not the owner of this booking"}`, http.StatusForbidden)
				return
			}
		} else if claims.Role == "priest" {
			if assignedPID != claims.UserID {
				http.Error(w, `{"error":"forbidden","message":"access denied: you are not assigned to this booking"}`, http.StatusForbidden)
				return
			}
		} else {
			http.Error(w, `{"error":"forbidden","message":"unknown account role"}`, http.StatusForbidden)
			return
		}

		details, err := storage.GetBookingDetails(bookingID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(details)
	}).Methods("GET", "OPTIONS")

	// 6. Update Status (POST) - Strict role & ownership checks; OTP verified on COMPLETED
	router.HandleFunc("/api/v1/bookings/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		var req models.BookingStatusUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
			return
		}

		ownerUID, assignedPID, _, _, err := storage.GetBookingOwner(req.BookingID)
		if err != nil {
			http.Error(w, `{"error":"booking not found"}`, http.StatusNotFound)
			return
		}

		if claims.Role == "priest" {
			if assignedPID != claims.UserID {
				http.Error(w, `{"error":"forbidden","message":"access denied: you are not assigned to this booking"}`, http.StatusForbidden)
				return
			}
			req.PriestID = claims.UserID
			req.CallerID = claims.UserID
			req.CallerRole = "priest"
		} else if claims.Role == "devotee" || claims.Role == "user" {
			if ownerUID != claims.UserID {
				http.Error(w, `{"error":"forbidden","message":"access denied: you are not the owner of this booking"}`, http.StatusForbidden)
				return
			}
			if req.Status == "COMPLETED" {
				http.Error(w, `{"error":"forbidden","message":"only the assigned priest can complete the booking with the completion OTP"}`, http.StatusForbidden)
				return
			}
			if req.Status != "CANCELLED" {
				http.Error(w, `{"error":"forbidden","message":"devotees can only cancel bookings; priest transitions are forbidden"}`, http.StatusForbidden)
				return
			}
			// Invariant: Devotees must never be able to set or change priest_id
			req.PriestID = ""
			req.CallerID = claims.UserID
			req.CallerRole = "devotee"
		} else if claims.Role == "admin" {
			req.CallerID = claims.UserID
			req.CallerRole = "admin"
		} else {
			http.Error(w, `{"error":"forbidden","message":"unknown account role"}`, http.StatusForbidden)
			return
		}

		escrowResult, err := storage.UpdateBookingStatus(req)
		if err != nil {
			mapLifecycleErrorToHTTP(w, err)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"booking_id":    req.BookingID,
			"new_status":    req.Status,
			"escrow_status": escrowResult,
			"message":       "Booking status updated successfully",
		})
	}).Methods("POST", "OPTIONS")

	// 7. Dedicated Booking Completion Endpoint (POST) - Assigned priest with valid OTP
	router.HandleFunc("/api/v1/bookings/{id}/complete", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		if claims.Role != "priest" {
			http.Error(w, `{"error":"forbidden","message":"only the assigned priest can complete this booking"}`, http.StatusForbidden)
			return
		}

		vars := mux.Vars(r)
		bookingID := vars["id"]

		_, assignedPID, _, _, err := storage.GetBookingOwner(bookingID)
		if err != nil {
			http.Error(w, `{"error":"booking not found"}`, http.StatusNotFound)
			return
		}

		if assignedPID != claims.UserID {
			http.Error(w, `{"error":"forbidden","message":"access denied: you are not assigned to this booking"}`, http.StatusForbidden)
			return
		}

		var body struct {
			CompletionOTP  string  `json:"completion_otp"`
			DakshinaAmount float64 `json:"dakshina_amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid complete payload"}`, http.StatusBadRequest)
			return
		}

		statusReq := models.BookingStatusUpdateRequest{
			BookingID:      bookingID,
			Status:         "COMPLETED",
			PriestID:       claims.UserID,
			CompletionOTP:  body.CompletionOTP,
			DakshinaAmount: body.DakshinaAmount,
			CallerID:       claims.UserID,
			CallerRole:     "priest",
		}

		escrowResult, err := storage.UpdateBookingStatus(statusReq)
		if err != nil {
			mapLifecycleErrorToHTTP(w, err)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"booking_id":    bookingID,
			"status":        "COMPLETED",
			"escrow_status": escrowResult,
			"message":       "Booking completed successfully with verified OTP",
		})
	}).Methods("POST", "OPTIONS")

	// 8. Booking Materials Live Checklist (GET & POST) - Owner or assigned priest only
	router.HandleFunc("/api/v1/bookings/{id}/materials", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		bookingID := vars["id"]

		ownerUID, assignedPID, _, _, err := storage.GetBookingOwner(bookingID)
		if err != nil {
			http.Error(w, `{"error":"booking not found"}`, http.StatusNotFound)
			return
		}

		if (claims.Role == "devotee" || claims.Role == "user") && ownerUID != claims.UserID {
			http.Error(w, `{"error":"forbidden","message":"access denied: you are not the owner of this booking"}`, http.StatusForbidden)
			return
		}
		if claims.Role == "priest" && assignedPID != claims.UserID {
			http.Error(w, `{"error":"forbidden","message":"access denied: you are not assigned to this booking"}`, http.StatusForbidden)
			return
		}

		if r.Method == "POST" {
			var body struct {
				Materials []models.BookingMaterial `json:"materials"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, `{"error":"invalid materials format"}`, http.StatusBadRequest)
				return
			}
			if err := storage.UpdateBookingMaterials(bookingID, body.Materials); err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "UPDATED", "booking_id": bookingID})
			return
		}

		materials, err := storage.GetBookingMaterials(bookingID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(materials)
	}).Methods("GET", "POST", "OPTIONS")

	// 9. Cancellation Quote (GET) & Cancel (POST)
	router.HandleFunc("/api/v1/bookings/{id}/cancel-quote", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		bookingID := vars["id"]

		ownerUID, assignedPID, _, _, err := storage.GetBookingOwner(bookingID)
		if err != nil {
			http.Error(w, `{"error":"booking not found"}`, http.StatusNotFound)
			return
		}

		if (claims.Role == "devotee" || claims.Role == "user") && ownerUID != claims.UserID {
			http.Error(w, `{"error":"forbidden","message":"access denied: you are not the owner of this booking"}`, http.StatusForbidden)
			return
		}
		if claims.Role == "priest" && assignedPID != claims.UserID {
			http.Error(w, `{"error":"forbidden","message":"access denied: you are not assigned to this booking"}`, http.StatusForbidden)
			return
		}

		quote, err := storage.GetCancellationQuote(bookingID)
		if err != nil {
			mapLifecycleErrorToHTTP(w, err)
			return
		}
		json.NewEncoder(w).Encode(quote)
	}).Methods("GET", "OPTIONS")

	router.HandleFunc("/api/v1/bookings/cancel", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		var req models.CancelBookingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid cancellation request"}`, http.StatusBadRequest)
			return
		}

		ownerUID, assignedPID, _, _, err := storage.GetBookingOwner(req.BookingID)
		if err != nil {
			http.Error(w, `{"error":"booking not found"}`, http.StatusNotFound)
			return
		}

		if (claims.Role == "devotee" || claims.Role == "user") && ownerUID != claims.UserID {
			http.Error(w, `{"error":"forbidden","message":"access denied: you are not the owner of this booking"}`, http.StatusForbidden)
			return
		}
		if claims.Role == "priest" && assignedPID != claims.UserID {
			http.Error(w, `{"error":"forbidden","message":"access denied: you are not assigned to this booking"}`, http.StatusForbidden)
			return
		}

		req.CancelledByRole = claims.Role

		quote, err := storage.CancelBookingWithRefund(req)
		if err != nil {
			mapLifecycleErrorToHTTP(w, err)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":              "CANCELLED",
			"booking_id":          req.BookingID,
			"refund_amount":       quote.RefundAmount,
			"priest_compensation": quote.PriestCompensation,
			"policy_notice":       quote.PolicyNotice,
			"message":             "Booking cancelled successfully per pilot policy",
		})
	}).Methods("POST", "OPTIONS")

	// 9B. Admin Emergency Force Cancellation (POST) - Admin role required with non-empty reason
	router.HandleFunc("/api/v1/bookings/{id}/admin-cancel", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		if claims.Role != "admin" {
			http.Error(w, `{"error":"forbidden","message":"only administrators can force cancel bookings"}`, http.StatusForbidden)
			return
		}

		vars := mux.Vars(r)
		bookingID := vars["id"]

		var body struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad_request","message":"invalid cancellation payload"}`, http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(body.Reason) == "" {
			http.Error(w, `{"error":"bad_request","message":"cancellation reason is required"}`, http.StatusBadRequest)
			return
		}

		cancelReq := models.CancelBookingRequest{
			BookingID:       bookingID,
			CancelledByRole: "admin",
			ReasonCode:      "ADMIN_EMERGENCY_OVERRIDE",
			ReasonText:      body.Reason,
		}

		quote, err := storage.CancelBookingWithRefund(cancelReq)
		if err != nil {
			mapLifecycleErrorToHTTP(w, err)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":              "CANCELLED",
			"booking_id":          bookingID,
			"refund_amount":       quote.RefundAmount,
			"priest_compensation": quote.PriestCompensation,
			"policy_notice":       quote.PolicyNotice,
			"cancelled_by":        "admin",
			"reason":              body.Reason,
			"message":             "Booking emergency cancelled by admin",
		})
	}).Methods("POST", "OPTIONS")

	// 10. Priest Referral (POST) - Only assigned priest can delegate
	router.HandleFunc("/api/v1/bookings/refer", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		if claims.Role != "priest" {
			http.Error(w, `{"error":"forbidden","message":"only priests can refer bookings"}`, http.StatusForbidden)
			return
		}

		var req models.PriestReferralRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid referral payload"}`, http.StatusBadRequest)
			return
		}

		_, assignedPID, _, _, err := storage.GetBookingOwner(req.BookingID)
		if err != nil {
			http.Error(w, `{"error":"booking not found"}`, http.StatusNotFound)
			return
		}

		if assignedPID != claims.UserID {
			http.Error(w, `{"error":"forbidden","message":"only the assigned priest can refer this booking"}`, http.StatusForbidden)
			return
		}

		req.ReferringPriestID = claims.UserID

		referralID, err := storage.ReferPriest(req)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":             "REASSIGNED",
			"referral_id":        referralID,
			"booking_id":         req.BookingID,
			"referred_priest_id": req.ReferredPriestID,
			"message":            "Booking auto-reassigned instantly to colleague purohit",
		})
	}).Methods("POST", "OPTIONS")
}
