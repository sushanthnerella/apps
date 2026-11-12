package auth

import (
	"context"
	"errors"
	"sync"
)

var (
	// ErrSMSUnavailable indicates that no real SMS provider credentials or gateway are configured
	ErrSMSUnavailable = errors.New("sms_service_unavailable: SMS delivery service is currently unconfigured. Password recovery via SMS is unavailable")
)

// SMSProvider defines the boundary interface for sending transactional SMS verification codes
type SMSProvider interface {
	SendVerificationCode(ctx context.Context, phone string, code string) error
	IsConfigured() bool
}

// UnconfiguredSMSProvider is the safe default provider for normal/release builds.
// It fails safely, never simulates fake deliveries or emits fixed codes.
type UnconfiguredSMSProvider struct{}

func (u *UnconfiguredSMSProvider) SendVerificationCode(ctx context.Context, phone string, code string) error {
	return ErrSMSUnavailable
}

func (u *UnconfiguredSMSProvider) IsConfigured() bool {
	return false
}

var (
	currentSMSProvider SMSProvider = &UnconfiguredSMSProvider{}
	smsMu              sync.RWMutex
)

// SetSMSProvider allows injecting an SMSProvider (used in automated tests)
func SetSMSProvider(p SMSProvider) {
	smsMu.Lock()
	defer smsMu.Unlock()
	if p == nil {
		currentSMSProvider = &UnconfiguredSMSProvider{}
		return
	}
	currentSMSProvider = p
}

// GetSMSProvider returns the active SMSProvider
func GetSMSProvider() SMSProvider {
	smsMu.RLock()
	defer smsMu.RUnlock()
	return currentSMSProvider
}
