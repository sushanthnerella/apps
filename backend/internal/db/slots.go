package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/purohit-platform/backend/internal/lifecycle"
	"github.com/purohit-platform/backend/internal/models"
)

var (
	inMemorySlotLocks sync.Map // map[string]int64 (lockKey -> expiryUnix)
	inMemoryLockMutex sync.Mutex
)

// GetPriestSlots retrieves availability slots for a priest on a given date
func (s *Storage) GetPriestSlots(priestID, date string) ([]models.PriestSlot, error) {
	query := `
		SELECT id, priest_id, slot_date, start_time, end_time, is_booked, COALESCE(booked_booking_id::text, ''), created_at 
		FROM priest_availability_slots 
		WHERE priest_id = $1 AND slot_date = $2 
		ORDER BY start_time ASC
	`
	rows, err := s.DB.Query(query, priestID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to query priest slots: %w", err)
	}
	defer rows.Close()

	var slots []models.PriestSlot
	ctx := context.Background()

	for rows.Next() {
		var slot models.PriestSlot
		var st, et time.Time
		var sDate time.Time
		var bookedID string
		if err := rows.Scan(&slot.ID, &slot.PriestID, &sDate, &st, &et, &slot.IsBooked, &bookedID, &slot.CreatedAt); err != nil {
			continue
		}
		slot.SlotDate = sDate.Format("2006-01-02")
		slot.StartTime = st.Format("15:04")
		slot.EndTime = et.Format("15:04")
		slot.BookedBookingID = bookedID

		// Check if held in Redis
		if s.Redis != nil {
			lockKey := fmt.Sprintf("lock:slot:%s", slot.ID)
			val, _ := s.Redis.Get(ctx, lockKey).Result()
			if val != "" {
				slot.IsHeld = true
			}
		} else {
			lockKey := fmt.Sprintf("lock:slot:%s", slot.ID)
			now := time.Now().Unix()
			if val, ok := inMemorySlotLocks.Load(lockKey); ok {
				if holdInfo, ok := val.(slotHoldInfo); ok && holdInfo.expiresAt > now {
					slot.IsHeld = true
				}
			}
		}

		slots = append(slots, slot)
	}

	return slots, nil
}

// CreatePriestSlot inserts a new availability slot, preserving existing booked state on conflict
func (s *Storage) CreatePriestSlot(req models.CreateSlotRequest) (string, error) {
	query := `
		INSERT INTO priest_availability_slots (priest_id, slot_date, start_time, end_time) 
		VALUES ($1, $2, $3, $4) 
		ON CONFLICT ON CONSTRAINT uq_priest_slot DO UPDATE SET is_booked = priest_availability_slots.is_booked
		RETURNING id
	`
	var id string
	err := s.DB.QueryRow(query, req.PriestID, req.SlotDate, req.StartTime, req.EndTime).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to create priest slot: %w", err)
	}
	return id, nil
}

// HoldPriestSlot attempts a 10-minute hold lock in Redis for devotee checkout (with in-memory mutex fallback)
func (s *Storage) HoldPriestSlot(slotID, devoteeID string, durationSec int) (bool, int64, error) {
	if durationSec <= 0 {
		durationSec = 600 // 10 minutes default
	}

	ctx := context.Background()
	lockKey := fmt.Sprintf("lock:slot:%s", slotID)

	// First verify the slot exists and is not already booked in the database
	var isBooked bool
	err := s.DB.QueryRow(`SELECT is_booked FROM priest_availability_slots WHERE id = $1::uuid`, slotID).Scan(&isBooked)
	if err != nil {
		return false, 0, fmt.Errorf("slot not found: %w", err)
	}
	if isBooked {
		return false, 0, fmt.Errorf("slot is already booked")
	}

	if s.Redis != nil {
		success, err := s.Redis.SetNX(ctx, lockKey, devoteeID, time.Duration(durationSec)*time.Second).Result()
		if err == nil {
			if !success {
				// Check if the current holder is the same devotee (idempotent re-hold)
				val, _ := s.Redis.Get(ctx, lockKey).Result()
				if val != devoteeID {
					return false, 0, fmt.Errorf("slot is currently held by another devotee")
				}
			}
			expiryTimestamp := time.Now().Add(time.Duration(durationSec) * time.Second).Unix()
			return true, expiryTimestamp, nil
		}
		// Log transient Redis lock error and proceed with in-memory lock fallback + DB pessimistic lock safety
		// (Postgres row lock FOR UPDATE in VerifyAndConsumeSlotHold guarantees no double-booking regardless)
	}

	// Fallback to thread-safe in-memory hold mutex if Redis is unavailable or errored
	inMemoryLockMutex.Lock()
	defer inMemoryLockMutex.Unlock()
	now := time.Now().Unix()
	if val, ok := inMemorySlotLocks.Load(lockKey); ok {
		if holdInfo, ok := val.(slotHoldInfo); ok && holdInfo.expiresAt > now {
			if holdInfo.devoteeID != devoteeID {
				return false, 0, fmt.Errorf("slot is currently held by another devotee")
			}
		}
	}
	expiryTimestamp := now + int64(durationSec)
	inMemorySlotLocks.Store(lockKey, slotHoldInfo{devoteeID: devoteeID, expiresAt: expiryTimestamp})
	return true, expiryTimestamp, nil
}

type slotHoldInfo struct {
	devoteeID string
	expiresAt int64
}

// ReleaseSlotHold releases the checkout hold on a slot
func (s *Storage) ReleaseSlotHold(slotID string) {
	ctx := context.Background()
	lockKey := fmt.Sprintf("lock:slot:%s", slotID)
	if s.Redis != nil {
		s.Redis.Del(ctx, lockKey)
	}
	inMemorySlotLocks.Delete(lockKey)
}

// VerifyAndConsumeSlotHold atomically checks and binds the availability slot to the booking inside tx.
// Returns the slot's date and formatted slot time string.
func (s *Storage) VerifyAndConsumeSlotHold(ctx context.Context, tx *sql.Tx, slotID, devoteeID, bookingID, expectedPriestID string) (string, string, error) {
	lockKey := fmt.Sprintf("lock:slot:%s", slotID)

	// Check hold lock ownership if active
	if s.Redis != nil {
		heldBy, err := s.Redis.Get(ctx, lockKey).Result()
		if err == nil && heldBy != "" && heldBy != devoteeID {
			return "", "", fmt.Errorf("%w: slot is held by another user", lifecycle.ErrSlotUnavailable)
		}
	}

	// Also check in-memory hold lock if present
	now := time.Now().Unix()
	if val, ok := inMemorySlotLocks.Load(lockKey); ok {
		if holdInfo, ok := val.(slotHoldInfo); ok && holdInfo.expiresAt > now {
			if holdInfo.devoteeID != devoteeID {
				return "", "", fmt.Errorf("%w: slot is held by another user", lifecycle.ErrSlotUnavailable)
			}
		}
	}

	// Pessimistic row-level lock on the availability slot
	var pID string
	var slotDate time.Time
	var startTime, endTime time.Time
	var isBooked bool
	var bookedBookingID sql.NullString

	query := `
		SELECT priest_id::text, slot_date, start_time, end_time, is_booked, booked_booking_id::text 
		FROM priest_availability_slots 
		WHERE id = $1::uuid 
		FOR UPDATE
	`
	err := tx.QueryRowContext(ctx, query, slotID).Scan(&pID, &slotDate, &startTime, &endTime, &isBooked, &bookedBookingID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", fmt.Errorf("slot not found: %s", slotID)
		}
		return "", "", fmt.Errorf("failed to lock slot for booking: %w", err)
	}

	if expectedPriestID != "" && pID != expectedPriestID {
		return "", "", fmt.Errorf("slot does not belong to the selected priest (%s != %s)", pID, expectedPriestID)
	}

	if isBooked {
		if bookingID == "" || (bookedBookingID.Valid && bookedBookingID.String != bookingID) {
			return "", "", fmt.Errorf("%w: priest availability slot is already booked", lifecycle.ErrSlotUnavailable)
		}
	}

	// Update slot to booked (booking ID will be linked after booking row insertion if empty)
	if bookingID != "" {
		updateQuery := `
			UPDATE priest_availability_slots 
			SET is_booked = true, booked_booking_id = $1::uuid 
			WHERE id = $2::uuid
		`
		if _, err := tx.ExecContext(ctx, updateQuery, bookingID, slotID); err != nil {
			return "", "", fmt.Errorf("failed to bind slot to booking: %w", err)
		}
	} else {
		updateQuery := `
			UPDATE priest_availability_slots 
			SET is_booked = true 
			WHERE id = $1::uuid
		`
		if _, err := tx.ExecContext(ctx, updateQuery, slotID); err != nil {
			return "", "", fmt.Errorf("failed to bind slot to booking: %w", err)
		}
	}

	// Clear the hold lock
	s.ReleaseSlotHold(slotID)

	slotDateStr := slotDate.Format("2006-01-02")
	slotTimeStr := fmt.Sprintf("%s - %s", startTime.Format("15:04"), endTime.Format("15:04"))
	return slotDateStr, slotTimeStr, nil
}
