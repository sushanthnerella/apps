package db_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/models"
)

func TestPriestSlotValidationOverlapAndRemoval(t *testing.T) {
	storage, _, _ := db.CreateIsolatedAuthTestDBForTest(t)
	storage.Redis = nil
	prID, err := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone: "+919876540401", FullName: "Slot Rules", Password: "Password123!",
		Sampradaya: "Smartha", VedaShakha: "Rigveda", ExperienceYears: 5, City: "Hyderabad", ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("register priest: %v", err)
	}
	slot := func(start, end string) (string, error) {
		return storage.CreatePriestSlot(models.CreateSlotRequest{PriestID: prID, SlotDate: "2030-01-10", StartTime: start, EndTime: end})
	}

	if _, err := slot("10:00", "09:00"); !errors.Is(err, db.ErrSlotInvalid) {
		t.Fatalf("end before start: want ErrSlotInvalid, got %v", err)
	}
	if _, err := slot("10:00", "10:00"); !errors.Is(err, db.ErrSlotInvalid) {
		t.Fatalf("zero length: want ErrSlotInvalid, got %v", err)
	}
	first, err := slot("09:00", "10:00")
	if err != nil {
		t.Fatalf("first slot: %v", err)
	}
	for _, c := range [][2]string{{"09:00", "10:00"}, {"09:30", "10:30"}, {"08:30", "09:01"}, {"08:00", "11:00"}} {
		if _, err := slot(c[0], c[1]); !errors.Is(err, db.ErrSlotOverlap) {
			t.Fatalf("%v overlaps 09:00-10:00: want ErrSlotOverlap, got %v", c, err)
		}
	}
	if _, err := slot("10:00", "11:00"); err != nil {
		t.Fatalf("adjacent slot must be allowed: %v", err)
	}

	// Concurrent overlapping publishes: exactly one wins.
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := slot("14:00", "15:00"); err == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("concurrent overlapping publishes: want 1 winner, got %d", wins)
	}

	// Held slot cannot be removed; once released it can.
	if ok, _, err := storage.HoldPriestSlot(first, "devotee-x", 600); err != nil || !ok {
		t.Fatalf("hold: %v", err)
	}
	if err := storage.DeletePriestSlot(prID, first); !errors.Is(err, db.ErrSlotNotRemovable) {
		t.Fatalf("held slot: want ErrSlotNotRemovable, got %v", err)
	}
	storage.ReleaseSlotHold(first)

	// Booked slot cannot be removed.
	if _, err := storage.DB.Exec(`UPDATE priest_availability_slots SET is_booked = true WHERE id = $1::uuid`, first); err != nil {
		t.Fatal(err)
	}
	if err := storage.DeletePriestSlot(prID, first); !errors.Is(err, db.ErrSlotNotRemovable) {
		t.Fatalf("booked slot: want ErrSlotNotRemovable, got %v", err)
	}

	// Another priest cannot remove it; owner removes an available slot.
	second, _ := slot("16:00", "17:00")
	if err := storage.DeletePriestSlot("00000000-0000-0000-0000-000000000000", second); !errors.Is(err, db.ErrSlotNotFound) {
		t.Fatalf("foreign priest: want ErrSlotNotFound, got %v", err)
	}
	if err := storage.DeletePriestSlot(prID, second); err != nil {
		t.Fatalf("delete available: %v", err)
	}
	if _, err := slot("16:00", "17:00"); err != nil {
		t.Fatalf("time freed after delete should be publishable: %v", err)
	}
}
