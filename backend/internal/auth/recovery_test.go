package auth

import (
	"context"
	"strconv"
	"testing"
	"time"
)

func TestGenerateRecoveryCode(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		code, err := GenerateRecoveryCode()
		if err != nil {
			t.Fatalf("unexpected error generating code: %v", err)
		}
		if len(code) != 6 {
			t.Errorf("expected 6-digit code, got %q", code)
		}
		num, err := strconv.Atoi(code)
		if err != nil || num < 0 || num > 999999 {
			t.Errorf("expected numeric value 000000-999999, got %q", code)
		}
		seen[code] = true
	}
	// Verify randomness: 50 codes should generate more than 40 distinct codes
	if len(seen) < 40 {
		t.Errorf("expected high randomness across 50 iterations, got only %d distinct codes", len(seen))
	}
}

func TestRecoveryCodeHashingAndVerification(t *testing.T) {
	secret := "test-secret-key-for-pilot-signing-12345"
	code := "492817"

	hash := HashRecoveryCode(code, secret)
	if hash == "" || hash == code {
		t.Fatalf("hash should not be empty or equal to plaintext code")
	}

	// Correct code matches
	if !VerifyRecoveryCode(code, hash, secret) {
		t.Errorf("expected correct code to verify")
	}

	// Code with leading/trailing spaces matches due to TrimSpace
	if !VerifyRecoveryCode(" 492817 ", hash, secret) {
		t.Errorf("expected trimmed code to verify")
	}

	// Wrong code fails
	if VerifyRecoveryCode("492818", hash, secret) {
		t.Errorf("wrong code should not verify")
	}

	// Wrong secret fails
	if VerifyRecoveryCode(code, hash, "different-secret-key-1234567890123") {
		t.Errorf("verification with wrong secret should fail")
	}
}

func TestUnconfiguredSMSProvider(t *testing.T) {
	provider := &UnconfiguredSMSProvider{}
	if provider.IsConfigured() {
		t.Errorf("UnconfiguredSMSProvider should report IsConfigured = false")
	}
	err := provider.SendVerificationCode(context.Background(), "+919876543210", "123456")
	if err != ErrSMSUnavailable {
		t.Errorf("expected ErrSMSUnavailable, got %v", err)
	}
}

func TestRecoveryThrottlerCooldownAndLimits(t *testing.T) {
	throttler := NewRecoveryThrottler()
	accountKey := "role:devotee:phone:+919876543210"
	clientIP := "192.168.1.100"

	// 1. Initial attempt allowed
	allowed, _, err := throttler.CheckStartAllowed(accountKey, clientIP)
	if !allowed || err != nil {
		t.Fatalf("first attempt should be allowed: %v", err)
	}
	throttler.RecordStartAttempt(accountKey, clientIP)

	// 2. Second immediate attempt blocked by cooldown (60 seconds)
	allowed, retryAfter, err := throttler.CheckStartAllowed(accountKey, clientIP)
	if allowed || err != ErrRecoveryCooldown {
		t.Errorf("second immediate attempt should be blocked by cooldown, got allowed=%v err=%v", allowed, err)
	}
	if retryAfter <= 0 || retryAfter > 60*time.Second {
		t.Errorf("unexpected retryAfter duration: %v", retryAfter)
	}

	// 3. Different account from different IP allowed
	allowed2, _, err2 := throttler.CheckStartAllowed("role:devotee:phone:+919999999999", "192.168.1.101")
	if !allowed2 || err2 != nil {
		t.Errorf("different account should be allowed: %v", err2)
	}
}
