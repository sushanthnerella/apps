package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/purohit-platform/backend/internal/models"
)

// GetOrCreateChatRoomTx ensures a chat room exists for a booking within a transaction
func (s *Storage) GetOrCreateChatRoomTx(ctx context.Context, tx *sql.Tx, bookingID, userID, priestID string) (string, error) {
	var roomID string
	queryFind := `SELECT id FROM chat_rooms WHERE booking_id = $1`
	err := tx.QueryRowContext(ctx, queryFind, bookingID).Scan(&roomID)
	if err == nil && roomID != "" {
		return roomID, nil
	}

	queryInsert := `
		INSERT INTO chat_rooms (booking_id, user_id, priest_id) 
		VALUES ($1, $2, NULLIF($3, '')::uuid) 
		ON CONFLICT (booking_id) DO UPDATE SET is_active = TRUE 
		RETURNING id
	`
	err = tx.QueryRowContext(ctx, queryInsert, bookingID, userID, priestID).Scan(&roomID)
	if err != nil {
		return "", fmt.Errorf("failed to create chat room: %w", err)
	}
	return roomID, nil
}

// GetOrCreateChatRoom ensures a chat room exists for a booking
func (s *Storage) GetOrCreateChatRoom(bookingID, userID, priestID string) (string, error) {
	var roomID string
	queryFind := `SELECT id FROM chat_rooms WHERE booking_id = $1`
	err := s.DB.QueryRow(queryFind, bookingID).Scan(&roomID)
	if err == nil && roomID != "" {
		return roomID, nil
	}

	queryInsert := `
		INSERT INTO chat_rooms (booking_id, user_id, priest_id) 
		VALUES ($1, $2, NULLIF($3, '')::uuid) 
		ON CONFLICT (booking_id) DO UPDATE SET is_active = TRUE 
		RETURNING id
	`
	err = s.DB.QueryRow(queryInsert, bookingID, userID, priestID).Scan(&roomID)
	if err != nil {
		return "", fmt.Errorf("failed to create chat room: %w", err)
	}
	return roomID, nil
}

// GetChatRoomParticipants returns the booking ID, devotee owner user_id, and assigned priest_id for a chat room.
// It joins with bookings to reflect any active reassignment/referral.
func (s *Storage) GetChatRoomParticipants(roomID string) (bookingID, userID, priestID string, err error) {
	query := `
		SELECT cr.booking_id::text, cr.user_id::text, COALESCE(cr.priest_id::text, ''),
		       COALESCE(b.user_id::text, ''), COALESCE(b.priest_id::text, '')
		FROM chat_rooms cr
		LEFT JOIN bookings b ON b.id = cr.booking_id
		WHERE cr.id = $1
	`
	var bID, rUID, crPID, bkUID, bkPID string
	err = s.DB.QueryRow(query, roomID).Scan(&bID, &rUID, &crPID, &bkUID, &bkPID)
	if err != nil {
		return "", "", "", fmt.Errorf("chat room not found: %w", err)
	}
	effectiveUID := rUID
	if bkUID != "" {
		effectiveUID = bkUID
	}
	effectivePID := crPID
	if bkPID != "" {
		effectivePID = bkPID
	}
	return bID, effectiveUID, effectivePID, nil
}

// VerifyChatRoomParticipant checks whether the authenticated caller is an authorized participant
// in the booking corresponding to the chat room.
func (s *Storage) VerifyChatRoomParticipant(roomID, callerID, callerRole string) (string, error) {
	bookingID, ownerUID, assignedPID, err := s.GetChatRoomParticipants(roomID)
	if err != nil {
		return "", err
	}
	if callerRole == "devotee" || callerRole == "user" {
		if ownerUID != callerID {
			return "", fmt.Errorf("access denied: you are not the devotee for this chat room")
		}
		return bookingID, nil
	}
	if callerRole == "priest" {
		if assignedPID == "" || assignedPID != callerID {
			return "", fmt.Errorf("access denied: you are not the assigned priest for this chat room")
		}
		return bookingID, nil
	}
	return "", fmt.Errorf("access denied: unknown role %s", callerRole)
}

// SaveChatMessageTx persists an incoming message within a transaction
func (s *Storage) SaveChatMessageTx(ctx context.Context, tx *sql.Tx, msg models.ChatMessage) (string, error) {
	senderID := msg.SenderID
	if senderID == "" || senderID == "system" {
		senderID = "00000000-0000-0000-0000-000000000000"
	}
	query := `
		INSERT INTO chat_messages (room_id, sender_id, sender_role, message_type, content, attachment_url) 
		VALUES ($1, $2::uuid, $3, $4, $5, $6) 
		RETURNING id
	`
	var id string
	err := tx.QueryRowContext(
		ctx, query, msg.RoomID, senderID, msg.SenderRole,
		msg.MessageType, msg.Content, msg.AttachmentURL,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("failed to save chat message: %w", err)
	}
	return id, nil
}

// SaveChatMessage persists an incoming message from devotee, priest, or system
func (s *Storage) SaveChatMessage(msg models.ChatMessage) (string, error) {
	senderID := msg.SenderID
	if senderID == "" || senderID == "system" {
		senderID = "00000000-0000-0000-0000-000000000000"
	}
	query := `
		INSERT INTO chat_messages (room_id, sender_id, sender_role, message_type, content, attachment_url) 
		VALUES ($1, $2::uuid, $3, $4, $5, $6) 
		RETURNING id
	`
	var id string
	err := s.DB.QueryRow(
		query, msg.RoomID, senderID, msg.SenderRole,
		msg.MessageType, msg.Content, msg.AttachmentURL,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("failed to save chat message: %w", err)
	}
	return id, nil
}

// GetChatMessages retrieves chronological chat history for a room
func (s *Storage) GetChatMessages(roomID string) ([]models.ChatMessage, error) {
	query := `
		SELECT id, room_id, sender_id, sender_role, message_type, content, COALESCE(attachment_url, ''), is_read, created_at 
		FROM chat_messages 
		WHERE room_id = $1 
		ORDER BY created_at ASC
	`
	rows, err := s.DB.Query(query, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chat messages: %w", err)
	}
	defer rows.Close()

	var messages []models.ChatMessage
	for rows.Next() {
		var m models.ChatMessage
		if err := rows.Scan(
			&m.ID, &m.RoomID, &m.SenderID, &m.SenderRole,
			&m.MessageType, &m.Content, &m.AttachmentURL, &m.IsRead, &m.CreatedAt,
		); err != nil {
			continue
		}
		messages = append(messages, m)
	}
	return messages, nil
}
