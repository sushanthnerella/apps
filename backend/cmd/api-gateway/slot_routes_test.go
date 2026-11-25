package main

import (
	"errors"
	"testing"
	"time"

	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/models"
)

func TestValidateNewSlotRejectsPast(t *testing.T) {
	now := time.Date(2026, 11, 23, 10, 0, 0, 0, istZone)
	req := func(d, s string) models.CreateSlotRequest {
		return models.CreateSlotRequest{SlotDate: d, StartTime: s, EndTime: "23:00"}
	}
	if err := validateNewSlot(req("2026-11-22", "11:00"), now); !errors.Is(err, errSlotInPast) {
		t.Fatalf("yesterday: %v", err)
	}
	if err := validateNewSlot(req("2026-11-23", "09:59"), now); !errors.Is(err, errSlotInPast) {
		t.Fatalf("earlier today: %v", err)
	}
	if err := validateNewSlot(req("2026-11-23", "10:30"), now); err != nil {
		t.Fatalf("later today: %v", err)
	}
	if err := validateNewSlot(req("bad", "10:30"), now); !errors.Is(err, db.ErrSlotInvalid) {
		t.Fatalf("bad date: %v", err)
	}
}
