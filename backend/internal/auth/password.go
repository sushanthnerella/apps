package auth

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters long")
	ErrPasswordTooLong    = errors.New("password exceeds maximum allowed length of 72 bytes")
	ErrPasswordBlank      = errors.New("password cannot be blank")
	ErrRecoveryRequired   = errors.New("account requires password setup/recovery before login")
	ErrInvalidCredentials = errors.New("invalid phone number or password")

	// dummyBcryptHash is a valid bcrypt hash of a random string used to equalize timing when an account is not found
	dummyBcryptHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")
)

const (
	BcryptCost   = bcrypt.DefaultCost // Cost factor 10
	MinPasswordLen = 8
	MaxPasswordLen = 72
)

// ValidatePassword checks password requirements (minimum 8 characters, maximum 72 bytes)
func ValidatePassword(password string) error {
	trimmed := strings.TrimSpace(password)
	if trimmed == "" {
		return ErrPasswordBlank
	}
	if len(password) < MinPasswordLen {
		return ErrPasswordTooShort
	}
	if len([]byte(password)) > MaxPasswordLen {
		return ErrPasswordTooLong
	}
	return nil
}

// HashPassword generates a bcrypt hash with cost 10 and a unique cryptographically secure salt
func HashPassword(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// IsBcryptHash validates whether a string matches a standard modular crypt format bcrypt hash ($2a$, $2b$, or $2y$)
func IsBcryptHash(hash string) bool {
	if len(hash) != 60 {
		return false
	}
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") && !strings.HasPrefix(hash, "$2y$") {
		return false
	}
	return true
}

// VerifyPassword compares a password against a stored hash.
// If the stored hash is blank or legacy/malformed plaintext, it equalizes timing and returns needsRecovery=true.
func VerifyPassword(storedHash, password string) (match bool, needsRecovery bool) {
	// If stored value is missing or not a valid bcrypt hash, this is a legacy account requiring recovery
	if !IsBcryptHash(storedHash) {
		// Run dummy compare to neutralize timing differences
		_ = bcrypt.CompareHashAndPassword(dummyBcryptHash, []byte(password))
		return false, true
	}

	err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		return false, false
	}
	return true, false
}

// EqualizeTiming executes a dummy bcrypt comparison when an account lookup fails
func EqualizeTiming(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyBcryptHash, []byte(password))
}
