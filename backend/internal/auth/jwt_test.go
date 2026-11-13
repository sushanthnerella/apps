package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "purohit_platform_test_jwt_secret_key_32bytes_long!"

func TestJWTConfiguration(t *testing.T) {
	// Missing / empty secret
	ResetJWTSecretForTesting()
	err := InitJWT("")
	if err == nil {
		t.Fatalf("expected InitJWT(\"\") to fail, but succeeded")
	}

	// Secret too short (< 32 chars)
	err = InitJWT("short_secret_under_32_bytes")
	if err == nil {
		t.Fatalf("expected short secret to fail, but succeeded")
	}

	// Valid 32+ bytes secret
	err = InitJWT(testSecret)
	if err != nil {
		t.Fatalf("expected valid secret to succeed, got %v", err)
	}
}

func TestTokenLifecycleAndClaims(t *testing.T) {
	SetJWTSecretForTesting(testSecret)

	userID := "8e54366e-2f3b-485a-8b1a-b6058e469599"
	phone := "+919849012345"
	role := "devotee"
	fullName := "Sushanth Sastry"
	sessionVersion := 2

	tokenStr, err := GenerateToken(userID, phone, role, fullName, sessionVersion)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != userID || claims.Role != role || claims.SessionVersion != sessionVersion {
		t.Errorf("claims mismatch: got %+v", claims)
	}
	if claims.Issuer != ExpectedJWTIssuer {
		t.Errorf("issuer mismatch: got %s, expected %s", claims.Issuer, ExpectedJWTIssuer)
	}
}

func TestTokenValidationFailures(t *testing.T) {
	SetJWTSecretForTesting(testSecret)

	userID := "8e54366e-2f3b-485a-8b1a-b6058e469599"

	// 1. Expired token
	expiredClaims := CustomClaims{
		UserID:         userID,
		Phone:          "+919876543210",
		Role:           "devotee",
		SessionVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    ExpectedJWTIssuer,
			Subject:   userID,
		},
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredStr, _ := expiredToken.SignedString([]byte(testSecret))
	if _, err := ValidateToken(expiredStr); err == nil {
		t.Errorf("expired token was accepted")
	}

	// 2. Missing expiry claim (exp = nil)
	noExpClaims := CustomClaims{
		UserID:         userID,
		Phone:          "+919876543210",
		Role:           "devotee",
		SessionVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:  ExpectedJWTIssuer,
			Subject: userID,
		},
	}
	noExpToken := jwt.NewWithClaims(jwt.SigningMethodHS256, noExpClaims)
	noExpStr, _ := noExpToken.SignedString([]byte(testSecret))
	if _, err := ValidateToken(noExpStr); err == nil {
		t.Errorf("token without exp claim was accepted")
	}

	// 3. Mismatched subject and user_id
	mismatchedSubClaims := CustomClaims{
		UserID:         userID,
		Phone:          "+919876543210",
		Role:           "devotee",
		SessionVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			Issuer:    ExpectedJWTIssuer,
			Subject:   "different-subject-user-id",
		},
	}
	mismatchedSubToken := jwt.NewWithClaims(jwt.SigningMethodHS256, mismatchedSubClaims)
	mismatchedSubStr, _ := mismatchedSubToken.SignedString([]byte(testSecret))
	if _, err := ValidateToken(mismatchedSubStr); err == nil {
		t.Errorf("token with mismatched subject was accepted")
	}

	// 4. Missing subject claim
	noSubClaims := CustomClaims{
		UserID:         userID,
		Phone:          "+919876543210",
		Role:           "devotee",
		SessionVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			Issuer:    ExpectedJWTIssuer,
			Subject:   "",
		},
	}
	noSubToken := jwt.NewWithClaims(jwt.SigningMethodHS256, noSubClaims)
	noSubStr, _ := noSubToken.SignedString([]byte(testSecret))
	if _, err := ValidateToken(noSubStr); err == nil {
		t.Errorf("token with missing subject was accepted")
	}

	// 5. Zero / missing session version
	zeroSessionClaims := CustomClaims{
		UserID:         userID,
		Phone:          "+919876543210",
		Role:           "devotee",
		SessionVersion: 0,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			Issuer:    ExpectedJWTIssuer,
			Subject:   userID,
		},
	}
	zeroSessionToken := jwt.NewWithClaims(jwt.SigningMethodHS256, zeroSessionClaims)
	zeroSessionStr, _ := zeroSessionToken.SignedString([]byte(testSecret))
	if _, err := ValidateToken(zeroSessionStr); err == nil {
		t.Errorf("token with zero session_version was accepted")
	}

	// 6. Wrong signing algorithm (none algorithm exploit prevention)
	noneClaims := CustomClaims{
		UserID:         userID,
		Phone:          "+919876543210",
		Role:           "devotee",
		SessionVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			Issuer:    ExpectedJWTIssuer,
			Subject:   userID,
		},
	}
	noneToken := jwt.NewWithClaims(jwt.SigningMethodNone, noneClaims)
	noneStr, _ := noneToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := ValidateToken(noneStr); err == nil {
		t.Errorf("none algorithm token was accepted")
	}

	// 7. Wrong issuer
	wrongIssuerClaims := CustomClaims{
		UserID:         userID,
		Phone:          "+919876543210",
		Role:           "devotee",
		SessionVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			Issuer:    "evil-issuer-unauthorized",
			Subject:   userID,
		},
	}
	wrongIssuerToken := jwt.NewWithClaims(jwt.SigningMethodHS256, wrongIssuerClaims)
	wrongIssuerStr, _ := wrongIssuerToken.SignedString([]byte(testSecret))
	if _, err := ValidateToken(wrongIssuerStr); err == nil {
		t.Errorf("wrong issuer token was accepted")
	}

	// 8. Forged token / wrong secret
	differentSecret := strings.Repeat("x", 40)
	forgedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, CustomClaims{
		UserID:         userID,
		Phone:          "+919876543210",
		Role:           "devotee",
		SessionVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			Issuer:    ExpectedJWTIssuer,
			Subject:   userID,
		},
	})
	forgedStr, _ := forgedToken.SignedString([]byte(differentSecret))
	if _, err := ValidateToken(forgedStr); err == nil {
		t.Errorf("forged token with different secret was accepted")
	}
}

func TestSessionRevocation(t *testing.T) {
	// 1. Equal session version -> valid
	if err := ValidateSessionVersion(1, 1); err != nil {
		t.Errorf("expected session version 1 to match DB version 1, got: %v", err)
	}
	if err := ValidateSessionVersion(5, 5); err != nil {
		t.Errorf("expected session version 5 to match DB version 5, got: %v", err)
	}

	// 2. Token version older than DB version -> revoked
	if err := ValidateSessionVersion(1, 2); err != ErrSessionRevoked {
		t.Errorf("expected ErrSessionRevoked when token session is 1 and DB is 2, got: %v", err)
	}

	// 3. Future session version -> rejected
	if err := ValidateSessionVersion(3, 2); err == nil {
		t.Errorf("expected future token session version 3 with DB version 2 to be rejected")
	}

	// 4. Zero or negative session version -> rejected
	if err := ValidateSessionVersion(0, 1); err == nil {
		t.Errorf("expected zero token session version to be rejected")
	}
	if err := ValidateSessionVersion(1, 0); err == nil {
		t.Errorf("expected zero DB session version to be rejected")
	}
}

func TestAdminTokenValidationSecurity(t *testing.T) {
	SetJWTSecretForTesting(testSecret)
	adminID := "c1111111-2f3b-485a-8b1a-b6058e469599"

	// 1. Valid admin token
	validToken, err := GenerateToken(adminID, "+919999900000", "admin", "Admin User", 1)
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}
	claims, err := ValidateToken(validToken)
	if err != nil {
		t.Fatalf("expected valid admin token, got: %v", err)
	}
	if claims.Role != "admin" || claims.UserID != adminID || claims.SessionVersion != 1 {
		t.Errorf("unexpected claims in admin token: %+v", claims)
	}

	// 2. Expired admin token
	expiredClaims := CustomClaims{
		UserID:         adminID,
		Phone:          "+919999900000",
		Role:           "admin",
		SessionVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			Issuer:    ExpectedJWTIssuer,
			Subject:   adminID,
		},
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredStr, _ := expiredToken.SignedString([]byte(testSecret))
	if _, err := ValidateToken(expiredStr); err == nil {
		t.Errorf("expired admin token was unexpectedly accepted")
	}

	// 3. Admin token with no exp
	noExpClaims := CustomClaims{
		UserID:         adminID,
		Phone:          "+919999900000",
		Role:           "admin",
		SessionVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:  ExpectedJWTIssuer,
			Subject: adminID,
		},
	}
	noExpToken := jwt.NewWithClaims(jwt.SigningMethodHS256, noExpClaims)
	noExpStr, _ := noExpToken.SignedString([]byte(testSecret))
	if _, err := ValidateToken(noExpStr); err == nil {
		t.Errorf("admin token without exp was unexpectedly accepted")
	}

	// 4. Admin token after session_version bump (revocation check)
	if err := ValidateSessionVersion(claims.SessionVersion, 2); err != ErrSessionRevoked {
		t.Errorf("expected ErrSessionRevoked after session_version bump to 2, got: %v", err)
	}

	// 5. Forged admin token signed with random secret
	forgedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, CustomClaims{
		UserID:         adminID,
		Role:           "admin",
		SessionVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			Issuer:    ExpectedJWTIssuer,
			Subject:   adminID,
		},
	})
	forgedStr, _ := forgedToken.SignedString([]byte("some_random_unauthorized_key_12345678"))
	if _, err := ValidateToken(forgedStr); err == nil {
		t.Errorf("forged admin token was unexpectedly accepted")
	}
}
