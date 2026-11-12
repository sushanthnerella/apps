package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/models"
	"net/http"
	"strings"
)

func registerChatRoutes(router *mux.Router, storage *db.Storage) {
	router.HandleFunc("/api/v1/chat/rooms", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		var body struct {
			BookingID string `json:"booking_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.BookingID == "" {
			http.Error(w, `{"error":"invalid room payload: booking_id is required"}`, http.StatusBadRequest)
			return
		}

		ownerUID, assignedPID, err := storage.VerifyBookingParticipant(body.BookingID, claims.UserID, claims.Role)
		if err != nil {
			if strings.Contains(err.Error(), "booking not found") {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf(`{"error":"forbidden","message":"%v"}`, err), http.StatusForbidden)
			}
			return
		}

		roomID, err := storage.GetOrCreateChatRoom(body.BookingID, ownerUID, assignedPID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"room_id":    roomID,
			"booking_id": body.BookingID,
		})
	}).Methods("POST", "OPTIONS")

	router.HandleFunc("/api/v1/chat/messages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		if r.Method == "POST" {
			var msg models.ChatMessage
			if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
				http.Error(w, `{"error":"invalid message payload"}`, http.StatusBadRequest)
				return
			}
			if msg.RoomID == "" {
				http.Error(w, `{"error":"missing room_id"}`, http.StatusBadRequest)
				return
			}

			_, err := storage.VerifyChatRoomParticipant(msg.RoomID, claims.UserID, claims.Role)
			if err != nil {
				if strings.Contains(err.Error(), "chat room not found") {
					http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusNotFound)
				} else {
					http.Error(w, fmt.Sprintf(`{"error":"forbidden","message":"%v"}`, err), http.StatusForbidden)
				}
				return
			}

			// Invariant: Never trust client-supplied sender IDs or roles!
			msg.SenderID = claims.UserID
			if claims.Role == "priest" {
				msg.SenderRole = "PRIEST"
			} else {
				msg.SenderRole = "DEVOTEE"
			}
			if msg.MessageType == "" {
				msg.MessageType = "TEXT"
			}

			msgID, err := storage.SaveChatMessage(msg)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
				return
			}
			msg.ID = msgID
			msgBytes, _ := json.Marshal(msg)
			chatHub.BroadcastToRoom(msg.SenderID, msg.RoomID, msgBytes)

			json.NewEncoder(w).Encode(msg)
			return
		}

		roomID := r.URL.Query().Get("room_id")
		if roomID == "" {
			http.Error(w, `{"error":"missing room_id parameter"}`, http.StatusBadRequest)
			return
		}

		_, err = storage.VerifyChatRoomParticipant(roomID, claims.UserID, claims.Role)
		if err != nil {
			if strings.Contains(err.Error(), "chat room not found") {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf(`{"error":"forbidden","message":"%v"}`, err), http.StatusForbidden)
			}
			return
		}

		messages, err := storage.GetChatMessages(roomID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		if messages == nil {
			messages = []models.ChatMessage{}
		}
		json.NewEncoder(w).Encode(messages)
	}).Methods("GET", "POST", "OPTIONS")

	router.HandleFunc("/api/v1/chat/room-by-booking", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		bookingID := r.URL.Query().Get("booking_id")
		if bookingID == "" {
			http.Error(w, `{"error":"missing booking_id parameter"}`, http.StatusBadRequest)
			return
		}

		ownerUID, assignedPID, err := storage.VerifyBookingParticipant(bookingID, claims.UserID, claims.Role)
		if err != nil {
			if strings.Contains(err.Error(), "booking not found") {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf(`{"error":"forbidden","message":"%v"}`, err), http.StatusForbidden)
			}
			return
		}

		roomID, err := storage.GetOrCreateChatRoom(bookingID, ownerUID, assignedPID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"room_id":    roomID,
			"booking_id": bookingID,
		})
	}).Methods("GET", "OPTIONS")

	router.HandleFunc("/ws/chat", func(w http.ResponseWriter, r *http.Request) {
		// 1. Authenticate caller: check Authorization header, or token/authToken query parameter
		var tokenStr string
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		}
		if tokenStr == "" {
			tokenStr = r.URL.Query().Get("token")
		}
		if tokenStr == "" {
			tokenStr = r.URL.Query().Get("authToken")
		}
		if tokenStr == "" {
			http.Error(w, `{"error":"unauthorized","message":"missing bearer token or token query parameter"}`, http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}
		dbVersion, err := storage.GetUserSessionVersion(claims.UserID, claims.Role)
		if err != nil || auth.ValidateSessionVersion(claims.SessionVersion, dbVersion) != nil {
			http.Error(w, `{"error":"unauthorized","message":"session expired or revoked"}`, http.StatusUnauthorized)
			return
		}

		roomID := r.URL.Query().Get("roomId")
		if roomID == "" {
			http.Error(w, `{"error":"bad_request","message":"missing roomId query parameter"}`, http.StatusBadRequest)
			return
		}

		_, err = storage.VerifyChatRoomParticipant(roomID, claims.UserID, claims.Role)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"forbidden","message":"%v"}`, err), http.StatusForbidden)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		// Invariant: Never trust client-supplied sender ID or role!
		clientID := claims.UserID
		var role string
		if claims.Role == "priest" {
			role = "PRIEST"
		} else {
			role = "DEVOTEE"
		}

		client := &Client{
			ID:     clientID,
			RoomID: roomID,
			Conn:   conn,
			Send:   make(chan []byte, 256),
		}

		conn.SetReadLimit(65536)
		chatHub.Register(client)

		go func() {
			defer conn.Close()
			for msg := range client.Send {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					break
				}
			}
		}()

		defer func() {
			chatHub.Unregister(client)
			conn.Close()
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			var msg models.ChatMessage
			if err := json.Unmarshal(message, &msg); err == nil {
				msg.RoomID = roomID
				// Server-enforced identity
				msg.SenderID = clientID
				msg.SenderRole = role
				if msg.MessageType == "" {
					msg.MessageType = "TEXT"
				}
				msgID, err := storage.SaveChatMessage(msg)
				if err != nil {
					continue
				}
				msg.ID = msgID

				broadcastData, _ := json.Marshal(msg)
				chatHub.BroadcastToRoom(clientID, roomID, broadcastData)
			}
		}
	})
}
