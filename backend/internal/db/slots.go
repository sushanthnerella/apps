package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/purohit-platform/backend/internal/models"
)

var (
	inMemorySlotLocks sync.Map // map[string]int64 (lockKey -> expiryUnix)
	inMemoryLockMutex sync.Mutex
)

// GetPriestSlots retrieves availability slots for a priest on a given date
func (s *Storage) GetPriestSlots(priestID, date string) ([]models.PriestSlot, error) {
	query := `
		SELECT id, priest_id, slot_date, start_time, end_time, is_booked, created_at 
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
		if err := rows.Scan(&slot.ID, &slot.PriestID, &sDate, &st, &et, &slot.IsBooked, &slot.CreatedAt); err != nil {
			continue
		}
		slot.SlotDate = sDate.Format("2006-01-02")
		slot.StartTime = st.Format("15:04")
		slot.EndTime = et.Format("15:04")

		// Check if held in Redis
		if s.Redis != nil {
			lockKey := fmt.Sprintf("lock:slot:%s", slot.ID)
			val, _ := s.Redis.Get(ctx, lockKey).Result()
			if val != "" {
				slot.IsHeld = true
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

	if s.Redis != nil {
		success, err := s.Redis.SetNX(ctx, lockKey, devoteeID, time.Duration(durationSec)*time.Second).Result()
		if err != nil {
			return false, 0, fmt.Errorf("redis lock error: %w", err)
		}
		if !success {
			return false, 0, fmt.Errorf("slot is currently held by another devotee or already booked")
		}
		expiryTimestamp := time.Now().Add(time.Duration(durationSec) * time.Second).Unix()
		return true, expiryTimestamp, nil
	}

	// Fallback to thread-safe in-memory hold mutex if Redis is unavailable
	inMemoryLockMutex.Lock()
	defer inMemoryLockMutex.Unlock()
	now := time.Now().Unix()
	if val, ok := inMemorySlotLocks.Load(lockKey); ok {
		if exp, isInt := val.(int64); isInt && exp > now {
			return false, 0, fmt.Errorf("slot is currently held by another devotee or already booked")
		}
	}
	expiryTimestamp := now + int64(durationSec)
	inMemorySlotLocks.Store(lockKey, expiryTimestamp)
	return true, expiryTimestamp, nil
}
