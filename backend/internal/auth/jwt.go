package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrJWTSecretUnconfigured = errors.New("JWT signing secret is missing or unsuitable (minimum 32 characters required)")
	ErrInvalidToken          = errors.New("invalid or expired JWT token")
	ErrInvalidAlgorithm      = errors.New("unexpected JWT signing algorithm: only HS256 is accepted")
	ErrInvalidIssuer         = errors.New("invalid JWT issuer")
	ErrInvalidClaims         = errors.New("missing or invalid required token claims")
	ErrSessionRevoked        = errors.New("session has been revoked; please log in again")

	jwtMu        sync.RWMutex
	jwtSecretKey []byte
)

const (
	ExpectedJWTIssuer = "purohit-platform-auth"
	TokenDuration     = 7 * 24 * time.Hour
	MinSecretLength   = 32
)

// InitJWT loads and validates the JWT signing key from configuration.
// It fails fast if the secret is missing, empty, or shorter than 32 characters (256 bits).
func InitJWT(secret string) error {
	jwtMu.Lock()
	defer jwtMu.Unlock()

	s := secret
	if s == "" {
		s = os.Getenv("JWT_SECRET")
		if s == "" {
			s = os.Getenv("PUROHIT_JWT_SECRET")
		}
	}

	if len(s) < MinSecretLength {
		jwtSecretKey = nil
		return fmt.Errorf("%w: provided secret has %d characters", ErrJWTSecretUnconfigured, len(s))
	}

	jwtSecretKey = []byte(s)
	return nil
}

// SetJWTSecretForTesting sets the secret key during test execution
func SetJWTSecretForTesting(secret string) {
	jwtMu.Lock()
	defer jwtMu.Unlock()
	jwtSecretKey = []byte(secret)
}

// ResetJWTSecretForTesting clears the secret for testing configuration errors
func ResetJWTSecretForTesting() {
	jwtMu.Lock()
	defer jwtMu.Unlock()
	jwtSecretKey = nil
}

func getSecret() ([]byte, error) {
	jwtMu.RLock()
	defer jwtMu.RUnlock()
	if len(jwtSecretKey) < MinSecretLength {
		return nil, ErrJWTSecretUnconfigured
	}
	return jwtSecretKey, nil
}

// GetSecretKey returns the validated application secret key
func GetSecretKey() (string, error) {
	s, err := getSecret()
	if err != nil {
		return "", err
	}
	return string(s), nil
}

type CustomClaims struct {
	UserID         string `json:"user_id"`
	Phone          string `json:"phone"`
	Role           string `json:"role"` // "devotee" or "priest"
	FullName       string `json:"full_name"`
	SessionVersion int    `json:"session_version"`
	jwt.RegisteredClaims
}

// GenerateToken issues a cryptographically signed JWT token valid for 7 days
func GenerateToken(userID, phone, role, fullName string, sessionVersion int) (string, error) {
	secret, err := getSecret()
	if err != nil {
		return "", err
	}

	if userID == "" {
		return "", fmt.Errorf("user_id claim is required")
	}
	if role != "devotee" && role != "priest" {
		return "", fmt.Errorf("invalid role claim %q: must be 'devotee' or 'priest'", role)
	}
	if sessionVersion <= 0 {
		sessionVersion = 1
	}

	now := time.Now()
	claims := CustomClaims{
		UserID:         userID,
		Phone:          phone,
		Role:           role,
		FullName:       fullName,
		SessionVersion: sessionVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-1 * time.Minute)), // Clock skew allowance
			Issuer:    ExpectedJWTIssuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return signedToken, nil
}

// ValidateToken parses, verifies signature, algorithm, issuer, expiration, and required identity claims of the JWT token
func ValidateToken(tokenString string) (*CustomClaims, error) {
	secret, err := getSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Strict algorithm enforcement: reject none, RS256, HS384, HS512, etc.
		if sm, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || sm != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("%w: %v", ErrInvalidAlgorithm, token.Header["alg"])
		}
		return secret, nil
	}, jwt.WithIssuer(ExpectedJWTIssuer), jwt.WithExpirationRequired(), jwt.WithLeeway(1*time.Minute))

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// 1. Expiry must be explicitly present and in the future
	if claims.ExpiresAt == nil {
		return nil, fmt.Errorf("%w: expiration claim (exp) is required", ErrInvalidClaims)
	}
	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, ErrInvalidToken
	}

	// 2. Validate required identity claims
	if strings.TrimSpace(claims.UserID) == "" {
		return nil, fmt.Errorf("%w: user_id claim is required", ErrInvalidClaims)
	}
	if strings.TrimSpace(claims.Phone) == "" {
		return nil, fmt.Errorf("%w: phone claim is required", ErrInvalidClaims)
	}
	if claims.Role != "devotee" && claims.Role != "priest" {
		return nil, fmt.Errorf("%w: invalid role claim %q", ErrInvalidClaims, claims.Role)
	}

	// 3. Subject and User-ID consistency
	if strings.TrimSpace(claims.Subject) == "" || claims.Subject != claims.UserID {
		return nil, fmt.Errorf("%w: subject claim %q does not match user_id %q", ErrInvalidClaims, claims.Subject, claims.UserID)
	}

	// 4. Positive session version in token
	if claims.SessionVersion <= 0 {
		return nil, fmt.Errorf("%w: session_version must be positive (got %d)", ErrInvalidClaims, claims.SessionVersion)
	}

	return claims, nil
}

// ValidateSessionVersion verifies that the token session version equals the current database version.
// Tokens with future, zero, or missing session versions are rejected.
func ValidateSessionVersion(tokenVersion, dbVersion int) error {
	if tokenVersion <= 0 || dbVersion <= 0 {
		return fmt.Errorf("%w: session version must be positive", ErrInvalidClaims)
	}
	if tokenVersion < dbVersion {
		return ErrSessionRevoked
	}
	if tokenVersion > dbVersion {
		return fmt.Errorf("%w: token session version %d cannot exceed current database version %d", ErrInvalidClaims, tokenVersion, dbVersion)
	}
	return nil
}
