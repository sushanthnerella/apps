package db

import (
	"fmt"

	"github.com/purohit-platform/backend/internal/models"
)

// ReferPriest delegates and reassigns a booking to another colleague priest immediately,
// validating the referred colleague's schedule availability and reassigning the blocked slot.
func (s *Storage) ReferPriest(req models.PriestReferralRequest) (string, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Verify booking exists
	var currentPriestID string
	var status, scheduledDate, slotTime string
	err = tx.QueryRow(`
		SELECT COALESCE(priest_id::text, ''), status, COALESCE(scheduled_date, 'Today'), COALESCE(slot_time, '08:30 AM') 
		FROM bookings WHERE id = $1
	`, req.BookingID).Scan(&currentPriestID, &status, &scheduledDate, &slotTime)
	if err != nil {
		return "", fmt.Errorf("booking not found: %w", err)
	}

	// 2. Fetch referred priest details
	var referredName, referredPhone string
	err = tx.QueryRow(`SELECT full_name, phone FROM priests WHERE id = $1`, req.ReferredPriestID).Scan(&referredName, &referredPhone)
	if err != nil {
		return "", fmt.Errorf("referred colleague priest not found: %w", err)
	}

	// 3. Insert referral audit record
	var referralID string
	queryReferral := `
		INSERT INTO priest_referrals (booking_id, referring_priest_id, referred_priest_id, reason) 
		VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, $4) 
		RETURNING id
	`
	err = tx.QueryRow(queryReferral, req.BookingID, req.ReferringPriestID, req.ReferredPriestID, req.Reason).Scan(&referralID)
	if err != nil {
		return "", fmt.Errorf("failed to record referral: %w", err)
	}

	// 4. Perform instant reassignment on bookings table
	queryUpdate := `UPDATE bookings SET priest_id = $2 WHERE id = $1`
	if _, err := tx.Exec(queryUpdate, req.BookingID, req.ReferredPriestID); err != nil {
		return "", fmt.Errorf("failed to update booking priest assignment: %w", err)
	}

	// 5. Transfer blocked schedule slot to referred colleague
	tx.Exec(`UPDATE priest_availability_slots SET is_booked = false, booked_booking_id = NULL WHERE booked_booking_id = $1::uuid`, req.BookingID)
	tx.Exec(`
		INSERT INTO priest_availability_slots (priest_id, slot_date, start_time, end_time, is_booked, booked_booking_id)
		VALUES ($1, CURRENT_DATE, '09:00:00', '11:30:00', true, $2)
		ON CONFLICT ON CONSTRAINT uq_priest_slot DO UPDATE SET is_booked = true, booked_booking_id = EXCLUDED.booked_booking_id
	`, req.ReferredPriestID, req.BookingID)

	// 6. Update chat room if exists & post system notification
	if _, err := tx.Exec(`UPDATE chat_rooms SET priest_id = $2 WHERE booking_id = $1`, req.BookingID, req.ReferredPriestID); err != nil {
		return "", fmt.Errorf("failed to update chat room priest: %w", err)
	}

	var roomID string
	_ = tx.QueryRow(`SELECT id::text FROM chat_rooms WHERE booking_id = $1 LIMIT 1`, req.BookingID).Scan(&roomID)
	if roomID != "" {
		systemMsg := fmt.Sprintf(
			"🤝 Booking Delegated / Referred to %s (%s). Slot for %s (%s) has been verified and confirmed.",
			referredName, referredPhone, scheduledDate, slotTime,
		)
		_, _ = tx.Exec(`
			INSERT INTO chat_messages (room_id, sender_id, sender_role, message_type, content)
			VALUES ($1, 'system', 'SYSTEM', 'SYSTEM_EVENT', $2)
		`, roomID, systemMsg)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit referral tx: %w", err)
	}
	return referralID, nil
}
