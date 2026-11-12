package auth

import (
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestThrottlerAccountLockout(t *testing.T) {
	th := NewLoginThrottler(100)
	accountKey := BuildAccountThrottleKey("devotee", "+919849012345")
	ipKey := "192.168.1.100"

	// 4 failures should not throttle
	for i := 0; i < 4; i++ {
		th.RecordFailure(accountKey, ipKey)
		throttled, _ := th.IsThrottled(accountKey, ipKey)
		if throttled {
			t.Fatalf("throttled prematurely at attempt %d", i+1)
		}
	}

	// 5th failure should trigger account lockout
	th.RecordFailure(accountKey, ipKey)
	throttled, rem := th.IsThrottled(accountKey, ipKey)
	if !throttled {
		t.Fatalf("expected account to be throttled after 5 failures")
	}
	if rem <= 0 {
		t.Fatalf("expected positive remaining lockout duration")
	}

	// Success clears the account lockout
	th.RecordSuccess(accountKey)
	throttled, _ = th.IsThrottled(accountKey, "")
	if throttled {
		t.Fatalf("expected account to be unlocked after RecordSuccess")
	}
}

func TestPhoneFormatVariantsShareThrottleLimit(t *testing.T) {
	th := NewLoginThrottler(100)

	variants := []string{
		"+91 98765 43210",
		"9876543210",
		"+91-98765-43210",
		"09876543210",
	}

	// Fail with first 4 variants (1 each + 1 repeat)
	th.RecordFailure(BuildAccountThrottleKey("devotee", variants[0]), "192.168.1.1")
	th.RecordFailure(BuildAccountThrottleKey("devotee", variants[1]), "192.168.1.2")
	th.RecordFailure(BuildAccountThrottleKey("devotee", variants[2]), "192.168.1.3")
	th.RecordFailure(BuildAccountThrottleKey("devotee", variants[0]), "192.168.1.4")

	// 4 failures total: should not yet be throttled
	throttled, _ := th.IsThrottled(BuildAccountThrottleKey("devotee", variants[3]), "")
	if throttled {
		t.Fatalf("should not be throttled at 4 attempts")
	}

	// 5th failure using variant 2
	th.RecordFailure(BuildAccountThrottleKey("devotee", variants[1]), "192.168.1.5")

	// 5th failure reached: variant 3 must now be locked out because they share normalized identity
	throttled, rem := th.IsThrottled(BuildAccountThrottleKey("devotee", variants[3]), "")
	if !throttled || rem <= 0 {
		t.Fatalf("expected variant 3 to share lockout after 5 total failures across format variants")
	}

	// Role scoping: priest account with same phone number should NOT be locked out
	priestKey := BuildAccountThrottleKey("priest", variants[0])
	priestThrottled, _ := th.IsThrottled(priestKey, "")
	if priestThrottled {
		t.Fatalf("priest role account was locked out by devotee failures; role scoping is broken")
	}
}

func TestExtractClientIPSecurityAndProxies(t *testing.T) {
	ResetTrustedProxies()
	defer ResetTrustedProxies()

	// 1. Untrusted caller cannot spoof IP via X-Forwarded-For
	untrustedReq := httptest.NewRequest("POST", "/login", nil)
	untrustedReq.RemoteAddr = "203.0.113.195:45892"
	untrustedReq.Header.Set("X-Forwarded-For", "1.2.3.4")
	untrustedReq.Header.Set("X-Real-IP", "5.6.7.8")

	ip := ExtractClientIP(untrustedReq)
	if ip != "203.0.113.195" {
		t.Errorf("security violation: untrusted caller spoofed IP as %s instead of remote %s", ip, "203.0.113.195")
	}

	// 2. Loopback is trusted by default: X-Forwarded-For is parsed
	loopbackReq := httptest.NewRequest("POST", "/login", nil)
	loopbackReq.RemoteAddr = "127.0.0.1:8085"
	loopbackReq.Header.Set("X-Forwarded-For", "198.51.100.42")

	ip2 := ExtractClientIP(loopbackReq)
	if ip2 != "198.51.100.42" {
		t.Errorf("expected 198.51.100.42 from loopback proxy, got %s", ip2)
	}

	// 3. Configured trusted proxy network safely parses proxy chain right-to-left
	err := ConfigureTrustedProxies([]string{"10.0.0.0/8", "127.0.0.1/32"})
	if err != nil {
		t.Fatalf("ConfigureTrustedProxies failed: %v", err)
	}

	chainReq := httptest.NewRequest("POST", "/login", nil)
	chainReq.RemoteAddr = "10.0.0.1:45123"
	// Client: 203.0.113.50, Intermediate trusted proxy: 10.0.0.2
	chainReq.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.2")

	ip3 := ExtractClientIP(chainReq)
	if ip3 != "203.0.113.50" {
		t.Errorf("expected 203.0.113.50 from proxy chain, got %s", ip3)
	}

	// 4. Fallback to X-Real-IP from trusted proxy
	realIPReq := httptest.NewRequest("POST", "/login", nil)
	realIPReq.RemoteAddr = "10.0.0.1:45123"
	realIPReq.Header.Set("X-Real-IP", "198.51.100.88")

	ip4 := ExtractClientIP(realIPReq)
	if ip4 != "198.51.100.88" {
		t.Errorf("expected 198.51.100.88 from trusted proxy X-Real-IP, got %s", ip4)
	}
}

func TestCapacityPressurePreservesActiveLockouts(t *testing.T) {
	// Create small throttler with limit of 30 entries
	maxLimit := 30
	th := NewLoginThrottler(maxLimit)

	// Step 1: Lock out target account (5 failures)
	targetKey := BuildAccountThrottleKey("devotee", "+919999900000")
	for i := 0; i < 5; i++ {
		th.RecordFailure(targetKey, "")
	}
	throttled, _ := th.IsThrottled(targetKey, "")
	if !throttled {
		t.Fatalf("target account was not locked out")
	}

	// Step 2: Inundate throttler with 200 dummy accounts with 1 failure each (unblocked)
	for i := 1; i <= 200; i++ {
		dummyKey := fmt.Sprintf("role:devotee:phone:+9198000%05d", i)
		th.RecordFailure(dummyKey, "")
	}

	// Step 3: Verify target account's active lockout was NOT trivially evicted by capacity pressure
	throttled, rem := th.IsThrottled(targetKey, "")
	if !throttled || rem <= 0 {
		t.Fatalf("active lockout was evicted by capacity pressure! Attackers can bypass lockouts by flooding keys.")
	}

	// Verify memory remains strictly bounded
	count := th.EntryCount()
	if count > maxLimit*2 {
		t.Errorf("entry count %d exceeds memory bound %d", count, maxLimit*2)
	}
}

func TestConcurrentThrottleAttempts(t *testing.T) {
	th := NewLoginThrottler(500)
	accountKey := BuildAccountThrottleKey("devotee", "+919876500000")

	var wg sync.WaitGroup
	concurrency := 20

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ip := fmt.Sprintf("192.168.1.%d", workerID)
			th.RecordFailure(accountKey, ip)
			_, _ = th.IsThrottled(accountKey, ip)
		}(i)
	}

	wg.Wait()

	// 20 concurrent failures should guarantee account is locked out
	throttled, _ := th.IsThrottled(accountKey, "")
	if !throttled {
		t.Fatalf("expected account to be locked out after 20 concurrent failures")
	}
}
