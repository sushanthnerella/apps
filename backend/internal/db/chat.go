package db

import (
	"fmt"

	"github.com/purohit-platform/backend/internal/models"
)

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
		VALUES ($1, $2, $3) 
		ON CONFLICT (booking_id) DO UPDATE SET is_active = TRUE 
		RETURNING id
	`
	err = s.DB.QueryRow(queryInsert, bookingID, userID, priestID).Scan(&roomID)
	if err != nil {
		return "", fmt.Errorf("failed to create chat room: %w", err)
	}
	return roomID, nil
}

// SaveChatMessage persists an incoming message from devotee, priest, or system
func (s *Storage) SaveChatMessage(msg models.ChatMessage) (string, error) {
	query := `
		INSERT INTO chat_messages (room_id, sender_id, sender_role, message_type, content, attachment_url) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id
	`
	var id string
	err := s.DB.QueryRow(
		query, msg.RoomID, msg.SenderID, msg.SenderRole,
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
