package auth

import (
	"strings"
	"testing"
)

func TestPasswordValidation(t *testing.T) {
	if err := ValidatePassword(""); err != ErrPasswordBlank {
		t.Errorf("expected ErrPasswordBlank, got %v", err)
	}
	if err := ValidatePassword("short"); err != ErrPasswordTooShort {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
	if err := ValidatePassword("1234567"); err != ErrPasswordTooShort {
		t.Errorf("expected ErrPasswordTooShort for 7 chars, got %v", err)
	}
	if err := ValidatePassword("valid_password_123"); err != nil {
		t.Errorf("expected valid password to pass, got %v", err)
	}

	tooLong := strings.Repeat("a", 73)
	if err := ValidatePassword(tooLong); err != ErrPasswordTooLong {
		t.Errorf("expected ErrPasswordTooLong for 73 bytes, got %v", err)
	}
}

func TestPasswordHashing(t *testing.T) {
	plain := "Purohit@Secure2026"

	hash1, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Password hashes differ from plaintext
	if hash1 == plain {
		t.Fatalf("hash matches plaintext!")
	}
	if !IsBcryptHash(hash1) {
		t.Fatalf("expected bcrypt modular crypt format ($2a$...), got %s", hash1)
	}

	// Repeated hashing uses fresh salts
	hash2, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("second HashPassword failed: %v", err)
	}
	if hash1 == hash2 {
		t.Errorf("repeated hashing produced identical hash; salts must be unique: %s == %s", hash1, hash2)
	}

	// Verification
	match, needsRec := VerifyPassword(hash1, plain)
	if !match || needsRec {
		t.Errorf("expected match=true, needsRec=false; got match=%v, needsRec=%v", match, needsRec)
	}

	// Wrong password cannot authenticate
	match, needsRec = VerifyPassword(hash1, "WrongPassword123")
	if match || needsRec {
		t.Errorf("wrong password authenticated: match=%v, needsRec=%v", match, needsRec)
	}
}

func TestLegacyCredentialsRequireRecovery(t *testing.T) {
	plain := "TestPassword123"

	// 1. Stored value is blank
	match, needsRec := VerifyPassword("", plain)
	if match || !needsRec {
		t.Errorf("blank stored password must trigger recovery: match=%v, needsRec=%v", match, needsRec)
	}

	// 2. Stored value is legacy plaintext
	match, needsRec = VerifyPassword("legacy_plaintext_secret", plain)
	if match || !needsRec {
		t.Errorf("legacy plaintext password must trigger recovery: match=%v, needsRec=%v", match, needsRec)
	}

	// 3. Stored value is malformed hash
	match, needsRec = VerifyPassword("$2a$invalid_short_hash", plain)
	if match || !needsRec {
		t.Errorf("malformed hash must trigger recovery: match=%v, needsRec=%v", match, needsRec)
	}
}

func TestEqualizeTiming(t *testing.T) {
	// Must execute without panic
	EqualizeTiming("any_attempted_password")
}
