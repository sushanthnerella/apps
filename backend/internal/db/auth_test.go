package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/db/migrate"
	"github.com/purohit-platform/backend/internal/models"
)

func getAuthTestRootConnStr() string {
	connStr := os.Getenv("PG_TEST_CONN_STR")
	if connStr != "" {
		return connStr
	}
	connStr = os.Getenv("PG_CONN_STR")
	if connStr != "" {
		return connStr
	}
	return ""
}

func getAuthMigrationsDir(t *testing.T) string {
	candidates := []string{
		"../../migrations",
		"../migrations",
		"migrations",
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err == nil {
			if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
				return abs
			}
		}
	}
	t.Fatalf("could not locate migrations directory from candidates: %v", candidates)
	return ""
}

// createIsolatedAuthTestDB provisions an isolated database with all migrations applied
func createIsolatedAuthTestDB(t *testing.T) (*Storage, *sql.DB, string) {
	auth.SetJWTSecretForTesting("purohit_platform_test_jwt_secret_key_32bytes_long!")

	rootConnStr := getAuthTestRootConnStr()
	if rootConnStr == "" {
		t.Skip("Explicit PostgreSQL test connection is required")
	}
	rootDB, err := sql.Open("postgres", rootConnStr)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to root postgres (%v)", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rootDB.PingContext(ctx); err != nil {
		rootDB.Close()
		t.Skipf("Skipping test: postgres server unreachable (%v)", err)
	}

	randBytes := make([]byte, 4)
	_, _ = rand.Read(randBytes)
	testDBName := fmt.Sprintf("pv_auth_test_%d_%s", time.Now().Unix(), hex.EncodeToString(randBytes))

	_, err = rootDB.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, testDBName))
	if err != nil {
		rootDB.Close()
		t.Fatalf("failed to create isolated test database %s: %v", testDBName, err)
	}

	testURL, err := url.Parse(rootConnStr)
	if err != nil || testURL.Host == "" {
		t.Fatal("test connection must be a PostgreSQL URL")
	}
	testURL.Path = "/" + testDBName
	testConnStr := testURL.String()
	testDB, err := sql.Open("postgres", testConnStr)
	if err != nil {
		rootDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, testDBName))
		rootDB.Close()
		t.Fatalf("failed to connect to test db: %v", err)
	}

	// Apply migrations up to 000005
	migDir := getAuthMigrationsDir(t)
	migrator := migrate.New(testDB, migDir, "")
	if _, err := migrator.Up(context.Background(), false); err != nil {
		testDB.Close()
		rootDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, testDBName))
		rootDB.Close()
		t.Fatalf("failed to migrate test db: %v", err)
	}

	storage := &Storage{DB: testDB, Redis: nil}

	t.Cleanup(func() {
		testDB.Close()
		_, _ = rootDB.Exec(fmt.Sprintf(`
			SELECT pg_terminate_backend(pid) 
			FROM pg_stat_activity 
			WHERE datname = '%s' AND pid <> pg_backend_pid()
		`, testDBName))
		_, _ = rootDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, testDBName))
		rootDB.Close()
	})

	return storage, testDB, testDBName
}

// 1. Acceptance Check: Correct password authenticates the correct role/account
func TestCorrectPasswordAuthenticates(t *testing.T) {
	storage, _, _ := createIsolatedAuthTestDB(t)

	// Register Devotee
	devoteeReq := models.DevoteeRegistrationRequest{
		FullName:      "Ananya Devi",
		Phone:         "+91 98490 55555",
		Password:      "VedicDevotee@2026",
		Gotram:        "Kashyapa",
		Nakshatram:    "Rohini",
		Rashi:         "Vrishabha",
		City:          "Hyderabad",
		ConsentAgreed: true,
	}
	devoteeID, err := storage.RegisterDevotee(devoteeReq)
	if err != nil {
		t.Fatalf("RegisterDevotee failed: %v", err)
	}
	if devoteeID == "" {
		t.Fatalf("expected non-empty devotee ID")
	}

	// Login with correct password
	loginRes, err := storage.Login(models.LoginRequest{
		Phone:    "+91 98490 55555",
		Password: "VedicDevotee@2026",
		Role:     "devotee",
	})
	if err != nil {
		t.Fatalf("Login with correct password failed: %v", err)
	}
	if loginRes.AccessToken == "" {
		t.Errorf("expected access token to be generated")
	}
	if loginRes.User.Role != "devotee" {
		t.Errorf("expected role devotee, got %s", loginRes.User.Role)
	}

	// Verify token claims
	claims, err := auth.ValidateToken(loginRes.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken failed on issued token: %v", err)
	}
	if claims.UserID != devoteeID {
		t.Errorf("token UserID mismatch: got %s, expected %s", claims.UserID, devoteeID)
	}

	// Register Priest
	priestReq := models.PriestRegistrationRequest{
		FullName:        "Brahmasri Somayaji",
		Phone:           "+91 98490 66666",
		Password:        "VedicPurohit@2026",
		Sampradaya:      "Smartha",
		VedaShakha:      "Rigveda",
		ExperienceYears: 12,
		City:            "Hyderabad",
		ConsentAgreed:   true,
	}
	priestID, err := storage.RegisterPriest(priestReq)
	if err != nil {
		t.Fatalf("RegisterPriest failed: %v", err)
	}

	// Login as priest with correct password
	priestLogin, err := storage.Login(models.LoginRequest{
		Phone:    "9849066666", // Normalized resolution test
		Password: "VedicPurohit@2026",
		Role:     "priest",
	})
	if err != nil {
		t.Fatalf("Priest login failed: %v", err)
	}
	if priestLogin.User.Role != "priest" || priestLogin.User.ID != priestID {
		t.Errorf("priest login role/ID mismatch: %+v", priestLogin.User)
	}
}

// 2. Acceptance Check: Wrong/blank password and arbitrary OTP cannot authenticate
func TestWrongOrBlankPasswordCannotAuthenticate(t *testing.T) {
	storage, _, _ := createIsolatedAuthTestDB(t)

	_, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		FullName:      "Kavitha Sharma",
		Phone:         "+919849077777",
		Password:      "CorrectPassword@123",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Wrong password
	_, err = storage.Login(models.LoginRequest{
		Phone:    "+919849077777",
		Password: "WrongPassword@999",
		Role:     "devotee",
	})
	if err == nil {
		t.Errorf("login with wrong password should have failed, but succeeded")
	}

	// Blank password
	_, err = storage.Login(models.LoginRequest{
		Phone:    "+919849077777",
		Password: "",
		Role:     "devotee",
	})
	if err == nil {
		t.Errorf("login with blank password should have failed, but succeeded")
	}

	// Arbitrary OTP attempt via LoginWithOTP
	_, err = storage.LoginWithOTP("+919849077777", "1234", "devotee")
	if err == nil {
		t.Errorf("arbitrary OTP login should have failed, but succeeded")
	}
}

// 3. Acceptance Check: Duplicate registration cannot overwrite a profile or password
func TestDuplicateRegistrationCannotOverwrite(t *testing.T) {
	storage, db, _ := createIsolatedAuthTestDB(t)

	originalReq := models.DevoteeRegistrationRequest{
		FullName:      "Original User",
		Phone:         "+919849011111",
		Password:      "OriginalSecret123",
		Gotram:        "Bharadwaja",
		ConsentAgreed: true,
	}
	origID, err := storage.RegisterDevotee(originalReq)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Attempt duplicate registration with modified name and new password
	duplicateReq := models.DevoteeRegistrationRequest{
		FullName:      "Attacker Impersonator",
		Phone:         "+91 98490 11111", // Space variant
		Password:      "HackedPassword999",
		Gotram:        "Vashishta",
		ConsentAgreed: true,
	}
	_, err = storage.RegisterDevotee(duplicateReq)
	if err == nil {
		t.Fatalf("expected duplicate registration to be rejected with conflict, but succeeded")
	}

	// Verify original profile and password remain uncorrupted
	var currentName, currentHash string
	err = db.QueryRow("SELECT full_name, password_hash FROM users WHERE id = $1", origID).Scan(&currentName, &currentHash)
	if err != nil {
		t.Fatalf("failed to query user: %v", err)
	}
	if currentName != "Original User" {
		t.Errorf("profile full_name was overwritten: got %s", currentName)
	}

	// Original password must still authenticate
	_, err = storage.Login(models.LoginRequest{
		Phone:    "+919849011111",
		Password: "OriginalSecret123",
		Role:     "devotee",
	})
	if err != nil {
		t.Errorf("original password failed to authenticate after duplicate attempt: %v", err)
	}

	// Attacker password must fail
	_, err = storage.Login(models.LoginRequest{
		Phone:    "+919849011111",
		Password: "HackedPassword999",
		Role:     "devotee",
	})
	if err == nil {
		t.Errorf("attacker password authenticated!")
	}
}

// 4. Acceptance Check: Missing/plaintext/malformed legacy credentials cannot claim an account
func TestLegacyCredentialsCannotClaimAccount(t *testing.T) {
	storage, db, _ := createIsolatedAuthTestDB(t)

	// Insert legacy devotee with plaintext password directly into DB
	legacyPlaintextPhone := "+919849022222"
	_, err := db.Exec(`
		INSERT INTO users (phone, full_name, password_hash, consent_agreed, session_version)
		VALUES ($1, 'Legacy Devotee', 'plain_unhashed_secret', true, 1)
	`, legacyPlaintextPhone)
	if err != nil {
		t.Fatalf("failed to insert legacy fixture: %v", err)
	}

	// Attempt login with the plaintext string
	_, err = storage.Login(models.LoginRequest{
		Phone:    legacyPlaintextPhone,
		Password: "plain_unhashed_secret",
		Role:     "devotee",
	})
	if !errors.Is(err, auth.ErrRecoveryRequired) {
		t.Errorf("expected ErrRecoveryRequired for legacy plaintext account, got: %v", err)
	}

	// Insert legacy devotee with empty password
	legacyEmptyPhone := "+919849033333"
	_, err = db.Exec(`
		INSERT INTO users (phone, full_name, password_hash, consent_agreed, session_version)
		VALUES ($1, 'Legacy Empty Devotee', '', true, 1)
	`, legacyEmptyPhone)
	if err != nil {
		t.Fatalf("failed to insert empty legacy fixture: %v", err)
	}

	_, err = storage.Login(models.LoginRequest{
		Phone:    legacyEmptyPhone,
		Password: "SomeAttemptedPassword123",
		Role:     "devotee",
	})
	if !errors.Is(err, auth.ErrRecoveryRequired) {
		t.Errorf("expected ErrRecoveryRequired for legacy blank account, got: %v", err)
	}
}

// 5. Acceptance Check: Phone-format variants resolve safely; collisions fail explicitly
func TestPhoneFormatVariantsAndCollisions(t *testing.T) {
	storage, _, _ := createIsolatedAuthTestDB(t)

	// Register with spaced format
	_, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		FullName:      "Sita Devi",
		Phone:         "+91 98490 44444",
		Password:      "SitaSecure@2026",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Login with 10 digits without prefix
	res1, err := storage.Login(models.LoginRequest{
		Phone:    "9849044444",
		Password: "SitaSecure@2026",
		Role:     "devotee",
	})
	if err != nil {
		t.Errorf("login with 10-digit variant failed: %v", err)
	}
	if res1.User.FullName != "Sita Devi" {
		t.Errorf("unexpected name from 10-digit variant: %s", res1.User.FullName)
	}

	// Login with 0-trunk prefix
	res2, err := storage.Login(models.LoginRequest{
		Phone:    "09849044444",
		Password: "SitaSecure@2026",
		Role:     "devotee",
	})
	if err != nil {
		t.Errorf("login with 0-trunk variant failed: %v", err)
	}
	if res2.User.FullName != "Sita Devi" {
		t.Errorf("unexpected name from 0-trunk variant: %s", res2.User.FullName)
	}

	// Role Collision: Cannot register same phone number as priest
	_, err = storage.RegisterPriest(models.PriestRegistrationRequest{
		FullName:      "Conflicting Priest",
		Phone:         "+919849044444",
		Password:      "PriestPassword@2026",
		Sampradaya:    "Smartha",
		ConsentAgreed: true,
	})
	if err == nil {
		t.Errorf("expected priest registration with existing devotee phone to be rejected, but succeeded")
	}
}

// 6. Acceptance Check: Session revocation invalidates tokens
func TestSessionRevocationInDB(t *testing.T) {
	storage, _, _ := createIsolatedAuthTestDB(t)

	userID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		FullName:      "Revocation Test User",
		Phone:         "+919849088888",
		Password:      "UserPass@2026",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Login to issue token with session_version 1
	loginRes, err := storage.Login(models.LoginRequest{
		Phone:    "+919849088888",
		Password: "UserPass@2026",
		Role:     "devotee",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	claims, err := auth.ValidateToken(loginRes.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	dbVersion, err := storage.GetUserSessionVersion(userID, "devotee")
	if err != nil {
		t.Fatalf("GetUserSessionVersion failed: %v", err)
	}
	if err := auth.ValidateSessionVersion(claims.SessionVersion, dbVersion); err != nil {
		t.Errorf("valid session was rejected: %v", err)
	}

	// Revoke sessions (e.g. on password reset)
	err = storage.RevokeUserSessions(userID, "devotee")
	if err != nil {
		t.Fatalf("RevokeUserSessions failed: %v", err)
	}

	// Verify DB version increased
	newDbVersion, _ := storage.GetUserSessionVersion(userID, "devotee")
	if newDbVersion != dbVersion+1 {
		t.Errorf("expected new DB version %d, got %d", dbVersion+1, newDbVersion)
	}

	// Previously issued token must now fail session validation
	if err := auth.ValidateSessionVersion(claims.SessionVersion, newDbVersion); err != auth.ErrSessionRevoked {
		t.Errorf("expected ErrSessionRevoked for old token, got: %v", err)
	}
}
