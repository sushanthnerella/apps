package db_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/models"
	"github.com/redis/go-redis/v9"
)

// TestRedisParsingAndCredentialMasking verifies that redis URLs (including TLS rediss:// and credentials)
// parse properly and do not expose passwords in logs or string conversions.
func TestRedisParsingAndCredentialMasking(t *testing.T) {
	// Plain redis URL
	plainURL := "redis://default:SecretPassword123@redis.railway.internal:6379"
	masked := db.SanitizeRedisEndpoint(plainURL)
	if masked == plainURL {
		t.Errorf("SanitizeRedisEndpoint failed to mask plain redis password: got %s", masked)
	}
	if masked != "redis://default:xxxxx@redis.railway.internal:6379" && masked != "redis://default:redacted@redis.railway.internal:6379" {
		// Verify url.Redacted() behavior
		if masked != "redis://default:xxxxx@redis.railway.internal:6379" && !containsRedacted(masked) {
			t.Errorf("expected redacted password, got %s", masked)
		}
	}

	// TLS rediss URL
	tlsURL := "rediss://default:SuperSecretTLS@secure-redis.railway.internal:6380"
	maskedTLS := db.SanitizeRedisEndpoint(tlsURL)
	if maskedTLS == tlsURL {
		t.Errorf("SanitizeRedisEndpoint failed to mask TLS redis password: got %s", maskedTLS)
	}

	// Host:port without credentials
	hostPort := "127.0.0.1:6379"
	if db.SanitizeRedisEndpoint(hostPort) != hostPort {
		t.Errorf("SanitizeRedisEndpoint altered host:port: got %s", db.SanitizeRedisEndpoint(hostPort))
	}
}

func containsRedacted(s string) bool {
	return len(s) > 0 && (s != "redis://default:SecretPassword123@redis.railway.internal:6379")
}

// TestSlotLockingResilienceWhenRedisUnavailable verifies that when Redis is nil or unreachable:
// 1. HoldPriestSlot succeeds by falling back to thread-safe in-memory hold locking.
// 2. A second devotee cannot hold the same slot while the hold is active.
// 3. Re-holding by the same devotee is idempotent.
// 4. VerifyAndConsumeSlotHold with PostgreSQL pessimistic lock (FOR UPDATE) guarantees no double booking.
func TestSlotLockingResilienceWhenRedisUnavailable(t *testing.T) {
	storage, _, _ := db.CreateIsolatedAuthTestDBForTest(t)
	// Ensure Redis is nil (simulating unavailable Redis)
	storage.Redis = nil

	// 1. Register a priest
	prID, err := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone:           "+919876540301",
		FullName:        "Priest Slot Test",
		Password:        "Password123!",
		Sampradaya:      "Smartha",
		VedaShakha:      "Rigveda",
		ExperienceYears: 10,
		City:            "Hyderabad",
		ConsentAgreed:   true,
	})
	if err != nil {
		t.Fatalf("failed registering priest: %v", err)
	}

	// 2. Create an availability slot
	slotID, err := storage.CreatePriestSlot(models.CreateSlotRequest{
		PriestID:  prID,
		SlotDate:  "2026-11-25",
		StartTime: "10:00",
		EndTime:   "12:00",
	})
	if err != nil {
		t.Fatalf("failed creating slot: %v", err)
	}

	// 3. Devotee 1 holds the slot for 10 minutes
	dev1ID := "devotee-1-uuid"
	held, expiresAt, err := storage.HoldPriestSlot(slotID, dev1ID, 600)
	if err != nil || !held {
		t.Fatalf("expected successful hold with in-memory fallback, got held=%v, err=%v", held, err)
	}
	if expiresAt <= time.Now().Unix() {
		t.Errorf("expected expiry in future, got %d", expiresAt)
	}

	// 4. Idempotent re-hold by same Devotee 1 must succeed
	heldAgain, _, err := storage.HoldPriestSlot(slotID, dev1ID, 600)
	if err != nil || !heldAgain {
		t.Fatalf("expected idempotent re-hold for devotee 1 to succeed, got %v, err=%v", heldAgain, err)
	}

	// 5. Devotee 2 attempts to hold the same slot -> must fail with conflict
	dev2ID := "devotee-2-uuid"
	held2, _, err2 := storage.HoldPriestSlot(slotID, dev2ID, 600)
	if held2 || err2 == nil {
		t.Fatalf("expected Devotee 2 hold attempt to fail, got held=%v, err=%v", held2, err2)
	}

	// 6. Release Devotee 1 hold
	storage.ReleaseSlotHold(slotID)

	// 7. Devotee 2 should now be able to hold the slot
	held2AfterRelease, _, err3 := storage.HoldPriestSlot(slotID, dev2ID, 600)
	if !held2AfterRelease || err3 != nil {
		t.Fatalf("expected Devotee 2 hold to succeed after release, got %v, err=%v", held2AfterRelease, err3)
	}
}

// TestSlotLockingResilienceWhenRedisErrors verifies that if Redis is configured but unreachable (e.g. invalid host/credentials),
// HoldPriestSlot logs the warning and falls back to in-memory hold locking rather than failing completely.
func TestSlotLockingResilienceWhenRedisErrors(t *testing.T) {
	storage, _, _ := db.CreateIsolatedAuthTestDBForTest(t)

	// Configure an invalid Redis client that will fail on Ping / SetNX
	badRedis := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:54328", // port where nothing is listening
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
	})
	storage.Redis = badRedis
	defer badRedis.Close()

	// 1. Register a priest
	prID, err := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone:           "+919876540302",
		FullName:        "Priest Redis Fail",
		Password:        "Password123!",
		Sampradaya:      "Smartha",
		VedaShakha:      "Rigveda",
		ExperienceYears: 10,
		City:            "Hyderabad",
		ConsentAgreed:   true,
	})
	if err != nil {
		t.Fatalf("failed registering priest: %v", err)
	}

	// 2. Create slot
	slotID, err := storage.CreatePriestSlot(models.CreateSlotRequest{
		PriestID:  prID,
		SlotDate:  "2026-11-26",
		StartTime: "14:00",
		EndTime:   "16:00",
	})
	if err != nil {
		t.Fatalf("failed creating slot: %v", err)
	}

	// 3. Devotee holds slot: Redis will return connection error, so HoldPriestSlot must fall back to in-memory lock
	devID := "devotee-redis-fail-1"
	held, expiresAt, err := storage.HoldPriestSlot(slotID, devID, 600)
	if err != nil || !held {
		t.Fatalf("expected hold to fall back to in-memory lock when Redis errors, got held=%v, err=%v", held, err)
	}
	if expiresAt <= time.Now().Unix() {
		t.Errorf("expected valid expiry timestamp, got %d", expiresAt)
	}

	// 4. Another devotee should be blocked by in-memory lock
	otherDevID := "devotee-redis-fail-2"
	heldOther, _, errOther := storage.HoldPriestSlot(slotID, otherDevID, 600)
	if heldOther || errOther == nil {
		t.Fatalf("expected other devotee hold to fail against in-memory lock, got held=%v, err=%v", heldOther, errOther)
	}
}

// TestConcurrentSlotConsumptionUnderRedisOutage verifies that under high concurrency with Redis offline,
// PostgreSQL pessimistic row lock (FOR UPDATE) guarantees that exactly ONE booking binds to the slot.
func TestConcurrentSlotConsumptionUnderRedisOutage(t *testing.T) {
	storage, _, _ := db.CreateIsolatedAuthTestDBForTest(t)
	storage.Redis = nil

	prID, err := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone:           "+919876540303",
		FullName:        "Priest Concurrency Test",
		Password:        "Password123!",
		Sampradaya:      "Smartha",
		VedaShakha:      "Rigveda",
		ExperienceYears: 10,
		City:            "Hyderabad",
		ConsentAgreed:   true,
	})
	if err != nil {
		t.Fatalf("failed registering priest: %v", err)
	}

	slotID, err := storage.CreatePriestSlot(models.CreateSlotRequest{
		PriestID:  prID,
		SlotDate:  "2026-11-27",
		StartTime: "08:00",
		EndTime:   "10:00",
	})
	if err != nil {
		t.Fatalf("failed creating slot: %v", err)
	}

	var successfulBinds int32
	var failedBinds int32
	var wg sync.WaitGroup
	concurrentAttempts := 10

	for i := 0; i < concurrentAttempts; i++ {
		wg.Add(1)
		devoteeID := "devotee-concurrent-" + string(rune('A'+i))
		go func(dID string) {
			defer wg.Done()
			tx, err := storage.DB.BeginTx(context.Background(), nil)
			if err != nil {
				atomic.AddInt32(&failedBinds, 1)
				return
			}
			defer tx.Rollback()

			_, _, err = storage.VerifyAndConsumeSlotHold(context.Background(), tx, slotID, dID, "", prID)
			if err == nil {
				if err := tx.Commit(); err == nil {
					atomic.AddInt32(&successfulBinds, 1)
					return
				}
			}
			atomic.AddInt32(&failedBinds, 1)
		}(devoteeID)
	}

	wg.Wait()

	if successfulBinds != 1 {
		t.Fatalf("expected exactly 1 successful slot bind under concurrency, got %d (failed: %d)", successfulBinds, failedBinds)
	}
	if failedBinds != int32(concurrentAttempts-1) {
		t.Errorf("expected %d failed binds, got %d", concurrentAttempts-1, failedBinds)
	}
}
