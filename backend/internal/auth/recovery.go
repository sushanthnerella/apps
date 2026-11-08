package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

var (
	ErrRecoveryCooldown       = errors.New("recovery_cooldown: please wait before requesting another recovery code")
	ErrRecoveryAttemptsExceeded = errors.New("recovery_rate_limit: too many recovery attempts. Please try again later")
)

// GenerateRecoveryCode generates a cryptographically secure 6-digit numeric string
func GenerateRecoveryCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("failed to generate secure recovery code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// HashRecoveryCode computes an HMAC-SHA256 keyed digest of the verification code
// using the application signing secret as the cryptographic key.
func HashRecoveryCode(code, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strings.TrimSpace(code)))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyRecoveryCode verifies a candidate code against the stored keyed digest using constant-time comparison
func VerifyRecoveryCode(code, storedHash, secret string) bool {
	candidateHash := HashRecoveryCode(code, secret)
	return subtle.ConstantTimeCompare([]byte(candidateHash), []byte(storedHash)) == 1
}

// AccountLockID computes a deterministic 64-bit integer lock key for PostgreSQL advisory locking
// scoped to the account role and ID to strictly serialize concurrent recovery and verification operations.
func AccountLockID(role, accountID string) int64 {
	h := sha256.Sum256([]byte("account_verify:" + strings.ToLower(strings.TrimSpace(role)) + ":" + strings.TrimSpace(accountID)))
	var val int64
	for i := 0; i < 8; i++ {
		val = (val << 8) | int64(h[i])
	}
	return val
}

// recoveryCooldownEntry tracks timestamps of recovery start attempts
type recoveryCooldownEntry struct {
	lastStart    time.Time
	attemptCount int
	firstAttempt time.Time
}

// RecoveryThrottler manages rate limits and resend cooldowns for password recovery
type RecoveryThrottler struct {
	mu         sync.Mutex
	entries    map[string]*recoveryCooldownEntry
	ipEntries  map[string]*recoveryCooldownEntry
	cooldown   time.Duration
	maxAccount int
	maxIP      int
	window     time.Duration
}

// NewRecoveryThrottler creates a recovery throttler with standard pilot parameters
func NewRecoveryThrottler() *RecoveryThrottler {
	return &RecoveryThrottler{
		entries:    make(map[string]*recoveryCooldownEntry),
		ipEntries:  make(map[string]*recoveryCooldownEntry),
		cooldown:   60 * time.Second, // 60s cooldown between code sends
		maxAccount: 3,                // Max 3 recovery starts per account per 15m
		maxIP:      10,               // Max 10 recovery starts per IP per 15m
		window:     15 * time.Minute,
	}
}

// DefaultRecoveryThrottler is the singleton recovery throttler instance
var DefaultRecoveryThrottler = NewRecoveryThrottler()

// CheckStartAllowed checks whether an account or IP is in cooldown or has exceeded rate limits
func (rt *RecoveryThrottler) CheckStartAllowed(accountKey, clientIP string) (allowed bool, retryAfter time.Duration, err error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	now := time.Now()

	// 1. Check account cooldown & limit
	if entry, ok := rt.entries[accountKey]; ok {
		// Clean up expired window
		if now.Sub(entry.firstAttempt) > rt.window {
			entry.attemptCount = 0
			entry.firstAttempt = now
		}

		// Cooldown check (60 seconds)
		if elapsed := now.Sub(entry.lastStart); elapsed < rt.cooldown {
			rem := rt.cooldown - elapsed
			return false, rem, ErrRecoveryCooldown
		}

		// Account rate limit check
		if entry.attemptCount >= rt.maxAccount {
			rem := rt.window - now.Sub(entry.firstAttempt)
			if rem < 0 {
				rem = rt.window
			}
			return false, rem, ErrRecoveryAttemptsExceeded
		}
	}

	// 2. Check IP rate limit
	if ipEntry, ok := rt.ipEntries[clientIP]; ok {
		if now.Sub(ipEntry.firstAttempt) > rt.window {
			ipEntry.attemptCount = 0
			ipEntry.firstAttempt = now
		}
		if ipEntry.attemptCount >= rt.maxIP {
			rem := rt.window - now.Sub(ipEntry.firstAttempt)
			if rem < 0 {
				rem = rt.window
			}
			return false, rem, ErrRecoveryAttemptsExceeded
		}
	}

	return true, 0, nil
}

// RecordStartAttempt records a recovery start attempt, updating cooldown and attempt counters
func (rt *RecoveryThrottler) RecordStartAttempt(accountKey, clientIP string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	now := time.Now()

	// Update account entry
	entry, ok := rt.entries[accountKey]
	if !ok || now.Sub(entry.firstAttempt) > rt.window {
		entry = &recoveryCooldownEntry{
			firstAttempt: now,
		}
		rt.entries[accountKey] = entry
	}
	entry.lastStart = now
	entry.attemptCount++

	// Update IP entry
	ipEntry, ok := rt.ipEntries[clientIP]
	if !ok || now.Sub(ipEntry.firstAttempt) > rt.window {
		ipEntry = &recoveryCooldownEntry{
			firstAttempt: now,
		}
		rt.ipEntries[clientIP] = ipEntry
	}
	ipEntry.lastStart = now
	ipEntry.attemptCount++

	// Prune old entries if map grows large
	if len(rt.entries) > 5000 {
		for k, v := range rt.entries {
			if now.Sub(v.firstAttempt) > rt.window && now.Sub(v.lastStart) > rt.cooldown {
				delete(rt.entries, k)
			}
		}
	}
	if len(rt.ipEntries) > 5000 {
		for k, v := range rt.ipEntries {
			if now.Sub(v.firstAttempt) > rt.window {
				delete(rt.ipEntries, k)
			}
		}
	}
}

// ResetForTesting resets the recovery throttler state for automated test isolation
func (rt *RecoveryThrottler) ResetForTesting() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.entries = make(map[string]*recoveryCooldownEntry)
	rt.ipEntries = make(map[string]*recoveryCooldownEntry)
}
