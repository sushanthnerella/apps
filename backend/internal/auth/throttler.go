package auth

import (
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Operational Notice: Single-Process In-Memory Throttler Limitations
// 1. Single Node Scope: This throttler maintains state in process memory. In a distributed multi-node
//    deployment, rate limits are tracked per-node unless fronted by sticky sessions or backed by
//    shared distributed storage (e.g. Redis).
// 2. Process Restart: Restarting the API gateway resets in-memory failure counters and lockouts.
// 3. Bounded Memory: Memory is strictly bounded to maxEntries (default 5,000 entries ~ 500KB RAM),
//    preventing memory exhaustion under attack while protecting active lockouts.

type throttleRecord struct {
	failCount int
	firstFail time.Time
	lastFail  time.Time
	blockedTo time.Time
}

// LoginThrottler provides bounded, in-memory rate limiting for login attempts
// tracked by normalized account (phone + role) and client IP.
// Memory is strictly bounded to maxEntries to eliminate memory exhaustion vectors.
type LoginThrottler struct {
	mu           sync.Mutex
	records      map[string]*throttleRecord
	maxEntries   int
	accountLimit int
	ipLimit      int
	window       time.Duration
	blockTime    time.Duration
}

// NewLoginThrottler creates a throttler with bounded memory and sensible defaults
func NewLoginThrottler(maxEntries int) *LoginThrottler {
	if maxEntries <= 0 {
		maxEntries = 5000
	}
	return &LoginThrottler{
		records:      make(map[string]*throttleRecord),
		maxEntries:   maxEntries,
		accountLimit: 5,               // 5 failed attempts per account
		ipLimit:      10,              // 10 failed attempts per IP
		window:       15 * time.Minute, // Rolling 15-minute failure window
		blockTime:    15 * time.Minute, // 15-minute temporary lockout
	}
}

// DefaultThrottler is the singleton instance used by API Gateway
var DefaultThrottler = NewLoginThrottler(5000)

// BuildAccountThrottleKey generates a canonical account throttle key combining normalized phone and explicit role.
// Phone numbers in different valid formats (e.g. +91 98765 43210, 9876543210, 09876543210) normalize to the
// same canonical E.164 identity (+919876543210), sharing the rate limit.
func BuildAccountThrottleKey(role, phone string) string {
	cleanRole := strings.ToLower(strings.TrimSpace(role))
	if cleanRole == "" {
		cleanRole = "unknown"
	}
	normalized, err := NormalizePhone(phone)
	if err != nil {
		normalized = strings.TrimSpace(phone)
	}
	return fmt.Sprintf("role:%s:phone:%s", cleanRole, normalized)
}

var (
	trustedProxiesMu sync.RWMutex
	trustedProxies   []*net.IPNet
)

// ConfigureTrustedProxies sets the trusted proxy networks/IPs (CIDRs or single IPs).
// If empty, loopback interfaces (127.0.0.1, ::1) are trusted by default.
func ConfigureTrustedProxies(proxies []string) error {
	trustedProxiesMu.Lock()
	defer trustedProxiesMu.Unlock()

	var parsed []*net.IPNet
	for _, p := range proxies {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.Contains(p, "/") {
			_, ipNet, err := net.ParseCIDR(p)
			if err != nil {
				return fmt.Errorf("invalid trusted proxy CIDR %q: %w", p, err)
			}
			parsed = append(parsed, ipNet)
		} else {
			ip := net.ParseIP(p)
			if ip == nil {
				return fmt.Errorf("invalid trusted proxy IP %q", p)
			}
			var mask net.IPMask
			if ip.To4() != nil {
				mask = net.CIDRMask(32, 32)
			} else {
				mask = net.CIDRMask(128, 128)
			}
			parsed = append(parsed, &net.IPNet{IP: ip, Mask: mask})
		}
	}
	trustedProxies = parsed
	return nil
}

// ResetTrustedProxies clears configured trusted proxies, returning to default loopback trust.
func ResetTrustedProxies() {
	trustedProxiesMu.Lock()
	defer trustedProxiesMu.Unlock()
	trustedProxies = nil
}

func isTrustedProxy(ip net.IP) bool {
	trustedProxiesMu.RLock()
	defer trustedProxiesMu.RUnlock()

	if len(trustedProxies) == 0 {
		return ip.IsLoopback()
	}

	for _, ipNet := range trustedProxies {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// IsThrottled checks whether an account or IP is currently locked out.
// Returns throttled boolean and the remaining lockout duration.
func (t *LoginThrottler) IsThrottled(accountKey, ipKey string) (bool, time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()

	// Check account block
	if accountKey != "" {
		if rec, exists := t.records["acc:"+accountKey]; exists {
			if now.Before(rec.blockedTo) {
				return true, rec.blockedTo.Sub(now)
			}
		}
	}

	// Check IP block
	if ipKey != "" {
		if rec, exists := t.records["ip:"+ipKey]; exists {
			if now.Before(rec.blockedTo) {
				return true, rec.blockedTo.Sub(now)
			}
		}
	}

	return false, 0
}

// RecordFailure increments failed attempt counts for account and IP, applying lockout if threshold is exceeded
func (t *LoginThrottler) RecordFailure(accountKey, ipKey string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.evictIfNeeded(now)

	if accountKey != "" {
		k := "acc:" + accountKey
		rec := t.getOrCreate(k, now)
		rec.failCount++
		rec.lastFail = now
		if rec.failCount >= t.accountLimit {
			rec.blockedTo = now.Add(t.blockTime)
		}
	}

	if ipKey != "" {
		k := "ip:" + ipKey
		rec := t.getOrCreate(k, now)
		rec.failCount++
		rec.lastFail = now
		if rec.failCount >= t.ipLimit {
			rec.blockedTo = now.Add(t.blockTime)
		}
	}
}

// RecordSuccess clears failure counters for an account upon successful login
func (t *LoginThrottler) RecordSuccess(accountKey string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if accountKey != "" {
		delete(t.records, "acc:"+accountKey)
	}
}

// Reset clears all throttle records (used for test teardown)
func (t *LoginThrottler) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.records = make(map[string]*throttleRecord)
}

// EntryCount returns the current number of tracked keys (for tests)
func (t *LoginThrottler) EntryCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.records)
}

func (t *LoginThrottler) getOrCreate(key string, now time.Time) *throttleRecord {
	rec, exists := t.records[key]
	if !exists {
		rec = &throttleRecord{
			firstFail: now,
			lastFail:  now,
		}
		t.records[key] = rec
		return rec
	}

	// Reset window if past failure window and not currently blocked
	if now.Sub(rec.firstFail) > t.window && now.After(rec.blockedTo) {
		rec.failCount = 0
		rec.firstFail = now
		rec.blockedTo = time.Time{}
	}
	return rec
}

// evictIfNeeded purges expired entries when map approaches capacity.
// Under capacity pressure, it prioritizes evicting unblocked entries (oldest activity first)
// and protects active lockout records so an attacker cannot trivially flush out active protections.
func (t *LoginThrottler) evictIfNeeded(now time.Time) {
	if len(t.records) < t.maxEntries {
		return
	}

	// 1. Purge expired non-blocked entries
	for k, rec := range t.records {
		if now.After(rec.blockedTo) && now.Sub(rec.lastFail) > t.window {
			delete(t.records, k)
		}
	}

	if len(t.records) < t.maxEntries {
		return
	}

	// 2. Capacity pressure: Evict unblocked entries first, sorted by lastFail (oldest first)
	type candidate struct {
		key      string
		lastFail time.Time
	}
	var unblocked []candidate
	for k, rec := range t.records {
		if now.After(rec.blockedTo) {
			unblocked = append(unblocked, candidate{key: k, lastFail: rec.lastFail})
		}
	}

	sort.Slice(unblocked, func(i, j int) bool {
		return unblocked[i].lastFail.Before(unblocked[j].lastFail)
	})

	targetEvict := t.maxEntries / 5
	if targetEvict < 1 {
		targetEvict = 1
	}

	evicted := 0
	for _, cand := range unblocked {
		delete(t.records, cand.key)
		evicted++
		if evicted >= targetEvict || len(t.records) < t.maxEntries {
			break
		}
	}

	// 3. Fallback only if entire map is actively blocked: evict entries expiring soonest
	if len(t.records) >= t.maxEntries {
		type blockedCandidate struct {
			key       string
			blockedTo time.Time
		}
		var blocked []blockedCandidate
		for k, rec := range t.records {
			blocked = append(blocked, blockedCandidate{key: k, blockedTo: rec.blockedTo})
		}
		sort.Slice(blocked, func(i, j int) bool {
			return blocked[i].blockedTo.Before(blocked[j].blockedTo)
		})
		for _, cand := range blocked {
			delete(t.records, cand.key)
			if len(t.records) < t.maxEntries {
				break
			}
		}
	}
}

// ExtractClientIP extracts the trusted client IP from the request.
// Forwarded headers (X-Forwarded-For, X-Real-IP) are ONLY trusted if RemoteAddr is a configured trusted proxy
// (or loopback by default). Untrusted callers cannot spoof client IP via request headers.
// When trusted, the X-Forwarded-For chain is parsed right-to-left to find the first untrusted upstream IP.
func ExtractClientIP(r *http.Request) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = strings.TrimSpace(r.RemoteAddr)
	}

	remoteIP := net.ParseIP(remoteHost)
	if remoteIP == nil {
		return remoteHost
	}

	// If RemoteAddr is NOT a trusted proxy, do NOT trust forwarded headers
	if !isTrustedProxy(remoteIP) {
		return remoteIP.String()
	}

	// RemoteAddr is trusted. Parse X-Forwarded-For chain right-to-left
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			part := strings.TrimSpace(parts[i])
			ip := net.ParseIP(part)
			if ip == nil {
				continue
			}
			if !isTrustedProxy(ip) {
				return ip.String()
			}
		}
		// If all proxies in chain were trusted, return leftmost valid IP
		for _, part := range parts {
			ip := net.ParseIP(strings.TrimSpace(part))
			if ip != nil {
				return ip.String()
			}
		}
	}

	// Fallback to X-Real-IP header if present
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		if ip := net.ParseIP(strings.TrimSpace(xri)); ip != nil {
			return ip.String()
		}
	}

	return remoteIP.String()
}
