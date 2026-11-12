package db

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/models"
)

// extractTokenFromURL parses the token query parameter from a reset URL
func extractTokenFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Query().Get("token")
}

// 1. Acceptance Test: Email verification, uniqueness per role, and profile binding
func TestEmailVerificationAndUniqueness(t *testing.T) {
	storage, _, _ := createIsolatedAuthTestDB(t)
	ctx := context.Background()
	testSecret := "purohit_platform_test_jwt_secret_key_32bytes_long!"

	emailMock := &auth.MockEmailProvider{Configured: true}
	auth.SetEmailProvider(emailMock)
	defer auth.SetEmailProvider(&auth.UnconfiguredEmailProvider{})

	// 1. Register two devotees
	user1ID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876543001",
		FullName:      "Devotee One",
		Password:      "SecurePassword123!",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("failed registering user 1: %v", err)
	}

	user2ID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876543002",
		FullName:      "Devotee Two",
		Password:      "SecurePassword123!",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("failed registering user 2: %v", err)
	}

	// 2. Request verification for user 1
	token, err := storage.RequestEmailVerification(ctx, "devotee", user1ID, "user1@example.com", testSecret)
	if err != nil {
		t.Fatalf("RequestEmailVerification failed: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty verification token")
	}

	// Invariant: Existing account email is preserved until new email is confirmed
	email, verified, err := storage.GetAccountEmail(ctx, "devotee", user1ID)
	if err != nil || verified || email != "" {
		t.Fatalf("expected unverified empty email for user 1 prior to confirmation, got email=%s, verified=%v, err=%v", email, verified, err)
	}

	// 3. Confirm email verification
	verifiedEmail, verifiedRole, err := storage.ConfirmEmailVerification(ctx, token, testSecret)
	if err != nil {
		t.Fatalf("ConfirmEmailVerification failed: %v", err)
	}
	if verifiedEmail != "user1@example.com" || verifiedRole != "devotee" {
		t.Fatalf("unexpected verified return values: email=%s, role=%s", verifiedEmail, verifiedRole)
	}

	// Now email is updated to user1@example.com and verified
	email, verified, err = storage.GetAccountEmail(ctx, "devotee", user1ID)
	if err != nil || !verified || email != "user1@example.com" {
		t.Fatalf("expected email to be user1@example.com and verified after confirmation, got: %s, verified=%v", email, verified)
	}

	// Reusing verification token fails
	if _, _, err := storage.ConfirmEmailVerification(ctx, token, testSecret); err == nil {
		t.Fatalf("expected reusing verification token to fail")
	}

	// 4. Unique email enforcement: user 2 cannot claim user 1's verified email
	err = storage.SetAccountVerifiedEmail(ctx, "devotee", user2ID, "user1@example.com", true)
	if err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("expected duplicate email error, got: %v", err)
	}
}

// 2. Acceptance Test: Devotee email recovery flow, link validation (without consuming), reset, and session revocation
func TestEmailRecoveryDevoteeSuccessfulFlow(t *testing.T) {
	storage, db, _ := createIsolatedAuthTestDB(t)
	ctx := context.Background()
	testSecret := "purohit_platform_test_jwt_secret_key_32bytes_long!"

	emailMock := &auth.MockEmailProvider{Configured: true}
	auth.SetEmailProvider(emailMock)
	defer auth.SetEmailProvider(&auth.UnconfiguredEmailProvider{})
	auth.DefaultRecoveryThrottler.ResetForTesting()

	// Register devotee
	userID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876543210",
		FullName:      "Ananya Sharma",
		Password:      "OldPassword123!",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("failed registering devotee: %v", err)
	}

	// Bind and verify email
	verifiedEmail := "ananya.sharma@example.com"
	if err := storage.SetAccountVerifiedEmail(ctx, "devotee", userID, verifiedEmail, true); err != nil {
		t.Fatalf("SetAccountVerifiedEmail failed: %v", err)
	}

	// Log in to obtain active JWT session
	loginRes, err := storage.Login(models.LoginRequest{
		Phone:    "+919876543210",
		Password: "OldPassword123!",
		Role:     "devotee",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	oldToken := loginRes.AccessToken
	claims, err := auth.ValidateToken(oldToken)
	if err != nil {
		t.Fatalf("token validation failed: %v", err)
	}
	dbVer, _ := storage.GetUserSessionVersion(userID, "devotee")
	if err := auth.ValidateSessionVersion(claims.SessionVersion, dbVer); err != nil {
		t.Fatalf("session version should match before reset: %v", err)
	}

	// 1. Start email password recovery
	startRes, err := storage.StartPasswordRecovery(ctx, models.StartRecoveryRequest{
		Email: verifiedEmail,
		Role:  "devotee",
	}, "127.0.0.1", testSecret)
	if err != nil {
		t.Fatalf("StartPasswordRecovery failed: %v", err)
	}
	if startRes.Status != "ACCEPTED" {
		t.Errorf("expected status ACCEPTED, got %s", startRes.Status)
	}

	// 2. Verify email dispatch
	resetURL := emailMock.LastSentResetURL(verifiedEmail)
	if resetURL == "" {
		t.Fatalf("expected reset email sent to %s", verifiedEmail)
	}
	rawToken := extractTokenFromURL(resetURL)
	if len(rawToken) != 64 {
		t.Fatalf("expected 64-char hex token in reset URL, got: %s", rawToken)
	}

	// 3. Verify challenge in DB stores HMAC hash, NEVER plaintext token
	var codeHash, status string
	err = db.QueryRowContext(ctx, "SELECT code_hash, status FROM password_recovery_challenges WHERE account_id = $1", userID).Scan(&codeHash, &status)
	if err != nil {
		t.Fatalf("failed querying challenge: %v", err)
	}
	if status != "ACTIVE" {
		t.Errorf("expected challenge status ACTIVE, got %s", status)
	}
	if codeHash == rawToken {
		t.Fatalf("CRITICAL SECURITY ERROR: Plaintext token stored in database!")
	}
	if !auth.VerifyResetToken(rawToken, codeHash, testSecret) {
		t.Fatalf("stored code_hash does not match candidate token HMAC")
	}

	// 4. Invariant: Visiting the reset link (GET) DOES NOT consume the token
	tokenInfo, err := storage.ValidateResetToken(ctx, rawToken, testSecret)
	if err != nil {
		t.Fatalf("ValidateResetToken failed: %v", err)
	}
	if tokenInfo.AccountID != userID || tokenInfo.Role != "devotee" {
		t.Errorf("unexpected token info: %+v", tokenInfo)
	}
	// Verify still ACTIVE in DB
	_ = db.QueryRowContext(ctx, "SELECT status FROM password_recovery_challenges WHERE account_id = $1", userID).Scan(&status)
	if status != "ACTIVE" {
		t.Fatalf("visiting reset link must not consume token; status is %s", status)
	}

	// 5. Confirm password reset with new password
	newPassword := "NewDevoteePassword2026!"
	err = storage.ConfirmPasswordRecovery(ctx, models.ConfirmRecoveryRequest{
		Token:           rawToken,
		NewPassword:     newPassword,
		ConfirmPassword: newPassword,
	}, "127.0.0.1", testSecret)
	if err != nil {
		t.Fatalf("ConfirmPasswordRecovery failed: %v", err)
	}

	// Verify challenge marked CONSUMED
	_ = db.QueryRowContext(ctx, "SELECT status FROM password_recovery_challenges WHERE account_id = $1", userID).Scan(&status)
	if status != "CONSUMED" {
		t.Errorf("expected challenge to be CONSUMED, got %s", status)
	}

	// 6. Old password no longer authenticates
	_, err = storage.Login(models.LoginRequest{
		Phone:    "+919876543210",
		Password: "OldPassword123!",
		Role:     "devotee",
	})
	if err == nil {
		t.Fatalf("expected old password to fail login, but succeeded")
	}

	// 7. New password authenticates
	newLogin, err := storage.Login(models.LoginRequest{
		Phone:    "+919876543210",
		Password: newPassword,
		Role:     "devotee",
	})
	if err != nil {
		t.Fatalf("new password failed to authenticate: %v", err)
	}
	if newLogin.AccessToken == "" {
		t.Errorf("expected access token")
	}

	// 8. Session revocation: old token's session_version is revoked
	newDBVer, _ := storage.GetUserSessionVersion(userID, "devotee")
	if newDBVer <= dbVer {
		t.Errorf("expected session_version to increment from %d, got %d", dbVer, newDBVer)
	}
	if err := auth.ValidateSessionVersion(claims.SessionVersion, newDBVer); err == nil {
		t.Errorf("expected old token session version to fail validation after password reset")
	}
}

// 3. Acceptance Test: Priest recovery & legacy account manual recovery
func TestEmailRecoveryPriestAndLegacyCredentials(t *testing.T) {
	storage, db, _ := createIsolatedAuthTestDB(t)
	ctx := context.Background()
	testSecret := "purohit_platform_test_jwt_secret_key_32bytes_long!"

	emailMock := &auth.MockEmailProvider{Configured: true}
	auth.SetEmailProvider(emailMock)
	defer auth.SetEmailProvider(&auth.UnconfiguredEmailProvider{})
	auth.DefaultRecoveryThrottler.ResetForTesting()

	// Insert legacy priest with plaintext password and NO verified email
	priestID := "00000000-0000-0000-0000-000000000777"
	_, err := db.ExecContext(ctx, `
		INSERT INTO priests (id, phone, full_name, password_hash, sampradaya, veda_shakha, experience_years, consent_agreed, is_online, session_version)
		VALUES ($1, '+919876500001', 'Sri Venkateswara Ghanapati', 'legacy_unhashed_pass', 'Smartha', 'Rigveda', 15, true, false, 1)
	`, priestID)
	if err != nil {
		t.Fatalf("failed inserting legacy priest: %v", err)
	}

	// Legacy priest without verified email cannot recover via self-service phone claiming
	res, err := storage.StartPasswordRecovery(ctx, models.StartRecoveryRequest{
		Phone: "+919876500001",
		Role:  "priest",
	}, "127.0.0.1", testSecret)
	if err != nil {
		t.Fatalf("unexpected error on unverified account recovery request: %v", err)
	}
	if res.Status != "ACCEPTED" {
		t.Errorf("expected generic ACCEPTED status")
	}
	if emailMock.LastSentResetURL("any") != "" {
		t.Errorf("no email should be sent when account has no verified email")
	}

	// Manual verified recovery: operator/support verifies email for legacy priest
	priestEmail := "ghanapati@temple.org"
	if err := storage.SetAccountVerifiedEmail(ctx, "priest", priestID, priestEmail, true); err != nil {
		t.Fatalf("failed setting verified email: %v", err)
	}

	// Now priest can recover via verified email
	auth.DefaultRecoveryThrottler.ResetForTesting()
	_, err = storage.StartPasswordRecovery(ctx, models.StartRecoveryRequest{
		Email: priestEmail,
		Role:  "priest",
	}, "127.0.0.1", testSecret)
	if err != nil {
		t.Fatalf("StartPasswordRecovery failed: %v", err)
	}

	resetURL := emailMock.LastSentResetURL(priestEmail)
	if resetURL == "" {
		t.Fatalf("expected reset email sent to %s", priestEmail)
	}
	rawToken := extractTokenFromURL(resetURL)

	// Confirm password reset
	newPriestPassword := "VedicGhanapati#2026"
	err = storage.ConfirmPasswordRecovery(ctx, models.ConfirmRecoveryRequest{
		Token:           rawToken,
		NewPassword:     newPriestPassword,
		ConfirmPassword: newPriestPassword,
	}, "127.0.0.1", testSecret)
	if err != nil {
		t.Fatalf("ConfirmPasswordRecovery failed: %v", err)
	}

	// Priest logs in with new password
	priestLogin, err := storage.Login(models.LoginRequest{
		Phone:    "+919876500001",
		Password: newPriestPassword,
		Role:     "priest",
	})
	if err != nil {
		t.Fatalf("priest failed to log in after recovery: %v", err)
	}
	if priestLogin.User.FullName != "Sri Venkateswara Ghanapati" {
		t.Errorf("unexpected priest profile: %v", priestLogin.User)
	}
}

// 4. Regression & Security Test: Cross-role, unverified email, expired token, reused token, and superseded token
func TestEmailRecoveryFailureModes(t *testing.T) {
	storage, db, _ := createIsolatedAuthTestDB(t)
	ctx := context.Background()
	testSecret := "purohit_platform_test_jwt_secret_key_32bytes_long!"

	emailMock := &auth.MockEmailProvider{Configured: true}
	auth.SetEmailProvider(emailMock)
	defer auth.SetEmailProvider(&auth.UnconfiguredEmailProvider{})
	auth.DefaultRecoveryThrottler.ResetForTesting()

	devoteeID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876543211",
		FullName:      "Suresh Babu",
		Password:      "OriginalPass123!",
		ConsentAgreed: true,
	})

	// Unverified email: recovery must not send email
	_ = storage.SetAccountVerifiedEmail(ctx, "devotee", devoteeID, "unverified@example.com", false)
	res, _ := storage.StartPasswordRecovery(ctx, models.StartRecoveryRequest{
		Email: "unverified@example.com",
		Role:  "devotee",
	}, "127.0.0.1", testSecret)
	if res.Status != "ACCEPTED" {
		t.Errorf("expected generic response for unverified email")
	}
	if emailMock.LastSentResetURL("unverified@example.com") != "" {
		t.Errorf("unverified email must NOT receive password reset link")
	}

	// Verify email
	_ = storage.SetAccountVerifiedEmail(ctx, "devotee", devoteeID, "suresh@example.com", true)

	auth.DefaultRecoveryThrottler.ResetForTesting()
	_, _ = storage.StartPasswordRecovery(ctx, models.StartRecoveryRequest{
		Email: "suresh@example.com",
		Role:  "devotee",
	}, "127.0.0.1", testSecret)
	validURL := emailMock.LastSentResetURL("suresh@example.com")
	validToken := extractTokenFromURL(validURL)

	// 1. Wrong token fails
	err := storage.ConfirmPasswordRecovery(ctx, models.ConfirmRecoveryRequest{
		Token:           "0000000000000000000000000000000000000000000000000000000000000000",
		NewPassword:     "NewPassword123!",
		ConfirmPassword: "NewPassword123!",
	}, "127.0.0.1", testSecret)
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Errorf("expected invalid token error, got: %v", err)
	}

	// 2. Expired token fails
	_, _ = db.ExecContext(ctx, "UPDATE password_recovery_challenges SET expires_at = NOW() - INTERVAL '1 hour' WHERE account_id = $1", devoteeID)
	err = storage.ConfirmPasswordRecovery(ctx, models.ConfirmRecoveryRequest{
		Token:           validToken,
		NewPassword:     "NewPassword123!",
		ConfirmPassword: "NewPassword123!",
	}, "127.0.0.1", testSecret)
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected expired token error, got: %v", err)
	}

	// 3. Superseded token fails
	auth.DefaultRecoveryThrottler.ResetForTesting()
	_, _ = storage.StartPasswordRecovery(ctx, models.StartRecoveryRequest{
		Email: "suresh@example.com",
		Role:  "devotee",
	}, "127.0.0.1", testSecret)
	token2 := extractTokenFromURL(emailMock.LastSentResetURL("suresh@example.com"))

	auth.DefaultRecoveryThrottler.ResetForTesting()
	_, _ = storage.StartPasswordRecovery(ctx, models.StartRecoveryRequest{
		Email: "suresh@example.com",
		Role:  "devotee",
	}, "127.0.0.1", testSecret)
	token3 := extractTokenFromURL(emailMock.LastSentResetURL("suresh@example.com"))

	// token2 is now superseded by token3
	err = storage.ConfirmPasswordRecovery(ctx, models.ConfirmRecoveryRequest{
		Token:           token2,
		NewPassword:     "NewPassword123!",
		ConfirmPassword: "NewPassword123!",
	}, "127.0.0.1", testSecret)
	if err == nil {
		t.Errorf("superseded token2 should be rejected")
	}

	// token3 succeeds
	err = storage.ConfirmPasswordRecovery(ctx, models.ConfirmRecoveryRequest{
		Token:           token3,
		NewPassword:     "NewPassword123!",
		ConfirmPassword: "NewPassword123!",
	}, "127.0.0.1", testSecret)
	if err != nil {
		t.Errorf("active token3 should succeed, got: %v", err)
	}

	// Reused token3 fails
	err = storage.ConfirmPasswordRecovery(ctx, models.ConfirmRecoveryRequest{
		Token:           token3,
		NewPassword:     "AnotherPassword123!",
		ConfirmPassword: "AnotherPassword123!",
	}, "127.0.0.1", testSecret)
	if err == nil || !strings.Contains(err.Error(), "already been used") {
		t.Errorf("reusing consumed token3 should be rejected with already used error, got: %v", err)
	}
}

// 5. Security Test: Unconfigured email provider fails safely
func TestEmailRecoveryUnconfiguredProviderFailsSafely(t *testing.T) {
	storage, db, _ := createIsolatedAuthTestDB(t)
	ctx := context.Background()
	testSecret := "purohit_platform_test_jwt_secret_key_32bytes_long!"

	auth.SetEmailProvider(&auth.UnconfiguredEmailProvider{})
	auth.DefaultRecoveryThrottler.ResetForTesting()

	userID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876543212",
		FullName:      "Kavita Rao",
		Password:      "KavitaSecurePass1!",
		ConsentAgreed: true,
	})
	_ = storage.SetAccountVerifiedEmail(ctx, "devotee", userID, "kavita@example.com", true)

	_, err := storage.StartPasswordRecovery(ctx, models.StartRecoveryRequest{
		Email: "kavita@example.com",
		Role:  "devotee",
	}, "127.0.0.1", testSecret)
	if !errors.Is(err, auth.ErrEmailUnavailable) {
		t.Fatalf("expected ErrEmailUnavailable, got: %v", err)
	}

	var count int
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM password_recovery_challenges WHERE account_id = $1 AND status = 'ACTIVE'", userID).Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 active challenges when email unconfigured, got %d", count)
	}
}

// 6. Concurrency Test: Concurrent confirmations on same token produce exactly one success
func TestConcurrentConfirmationsProduceOneReset(t *testing.T) {
	storage, _, _ := createIsolatedAuthTestDB(t)
	ctx := context.Background()
	testSecret := "purohit_platform_test_jwt_secret_key_32bytes_long!"

	emailMock := &auth.MockEmailProvider{Configured: true}
	auth.SetEmailProvider(emailMock)
	defer auth.SetEmailProvider(&auth.UnconfiguredEmailProvider{})
	auth.DefaultRecoveryThrottler.ResetForTesting()

	userID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876543213",
		FullName:      "Vijay Kumar",
		Password:      "PasswordInit123!",
		ConsentAgreed: true,
	})
	_ = storage.SetAccountVerifiedEmail(ctx, "devotee", userID, "vijay@example.com", true)

	_, _ = storage.StartPasswordRecovery(ctx, models.StartRecoveryRequest{
		Email: "vijay@example.com",
		Role:  "devotee",
	}, "127.0.0.1", testSecret)
	token := extractTokenFromURL(emailMock.LastSentResetURL("vijay@example.com"))

	concurrency := 6
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			newPwd := "ConcurrentResetPassword123!"
			err := storage.ConfirmPasswordRecovery(ctx, models.ConfirmRecoveryRequest{
				Token:           token,
				NewPassword:     newPwd,
				ConfirmPassword: newPwd,
			}, "127.0.0.1", testSecret)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if successCount != 1 {
		t.Errorf("expected exactly 1 concurrent confirmation to succeed, got %d", successCount)
	}
}

// 7. Security Test: Unknown email does not reveal account absence
func TestRecoveryUnknownAccountDoesNotLeak(t *testing.T) {
	storage, _, _ := createIsolatedAuthTestDB(t)
	ctx := context.Background()
	testSecret := "purohit_platform_test_jwt_secret_key_32bytes_long!"

	emailMock := &auth.MockEmailProvider{Configured: true}
	auth.SetEmailProvider(emailMock)
	defer auth.SetEmailProvider(&auth.UnconfiguredEmailProvider{})
	auth.DefaultRecoveryThrottler.ResetForTesting()

	res, err := storage.StartPasswordRecovery(ctx, models.StartRecoveryRequest{
		Email: "nonexistent.user@example.com",
		Role:  "devotee",
	}, "127.0.0.1", testSecret)
	if err != nil {
		t.Fatalf("unexpected error on unknown account: %v", err)
	}
	if res.Status != "ACCEPTED" {
		t.Errorf("expected generic status ACCEPTED, got %s", res.Status)
	}
	if emailMock.LastSentResetURL("nonexistent.user@example.com") != "" {
		t.Errorf("no email should be dispatched for unknown account")
	}
}

// 8. Acceptance Test: Changing email from A to B preserves A until B confirmed,
// supersedes earlier requests, and ensures older link can never verify newer email.
func TestEmailChangePreservesOldUntilNewConfirmed(t *testing.T) {
	storage, db, _ := createIsolatedAuthTestDB(t)
	ctx := context.Background()
	testSecret := "purohit_platform_test_jwt_secret_key_32bytes_long!"

	emailMock := &auth.MockEmailProvider{Configured: true}
	auth.SetEmailProvider(emailMock)
	defer auth.SetEmailProvider(&auth.UnconfiguredEmailProvider{})

	// 1. Register devotee and verify initial email A
	userID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876543999",
		FullName:      "Multi Email User",
		Password:      "InitialSecret123!",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("failed registering user: %v", err)
	}

	emailA := "email.a@example.com"
	if err := storage.SetAccountVerifiedEmail(ctx, "devotee", userID, emailA, true); err != nil {
		t.Fatalf("failed setting verified email A: %v", err)
	}

	// 2. Request change to email B
	emailB := "email.b@example.com"
	tokenB, err := storage.RequestEmailVerification(ctx, "devotee", userID, emailB, testSecret)
	if err != nil {
		t.Fatalf("failed requesting verification for email B: %v", err)
	}

	// CRITICAL INVARIANT: Existing verified email A must NOT be overwritten!
	currentEmail, isVerified, err := storage.GetAccountEmail(ctx, "devotee", userID)
	if err != nil || currentEmail != emailA || !isVerified {
		t.Fatalf("CRITICAL BUG: Email A was modified before B was verified! currentEmail=%s, isVerified=%v", currentEmail, isVerified)
	}

	// Verify email B was dispatched
	verifyURLB := emailMock.LastSentVerificationURL(emailB)
	if verifyURLB == "" {
		t.Fatalf("expected verification email dispatched to email B")
	}
	extractedTokenB := extractTokenFromURL(verifyURLB)
	if extractedTokenB != tokenB {
		t.Fatalf("token mismatch between returned and emailed: got %s vs %s", extractedTokenB, tokenB)
	}

	// 3. Before confirming B, user requests email C (superseding B)
	emailC := "email.c@example.com"
	tokenC, err := storage.RequestEmailVerification(ctx, "devotee", userID, emailC, testSecret)
	if err != nil {
		t.Fatalf("failed requesting verification for email C: %v", err)
	}

	// Invariant: Email is still A!
	currentEmail, isVerified, _ = storage.GetAccountEmail(ctx, "devotee", userID)
	if currentEmail != emailA || !isVerified {
		t.Fatalf("email should still be A, got %s", currentEmail)
	}

	// 4. Invariant: Older token B is now SUPERSEDED and must be rejected!
	_, _, err = storage.ConfirmEmailVerification(ctx, tokenB, testSecret)
	if err == nil || !strings.Contains(err.Error(), "superseded") {
		t.Fatalf("expected older token B to be rejected as superseded, got: %v", err)
	}

	// Token B must NEVER verify email C or any email!
	currentEmail, _, _ = storage.GetAccountEmail(ctx, "devotee", userID)
	if currentEmail != emailA {
		t.Fatalf("email should remain A after rejected token B, got %s", currentEmail)
	}

	// 5. Confirm token C
	verifiedEmail, role, err := storage.ConfirmEmailVerification(ctx, tokenC, testSecret)
	if err != nil {
		t.Fatalf("ConfirmEmailVerification for token C failed: %v", err)
	}
	if verifiedEmail != emailC || role != "devotee" {
		t.Fatalf("expected verified email C (%s, devotee), got (%s, %s)", emailC, verifiedEmail, role)
	}

	// Account now has email C and is verified
	currentEmail, isVerified, _ = storage.GetAccountEmail(ctx, "devotee", userID)
	if currentEmail != emailC || !isVerified {
		t.Fatalf("expected account email to be updated to C, got %s, verified=%v", currentEmail, isVerified)
	}

	// 6. Token C cannot be reused
	_, _, err = storage.ConfirmEmailVerification(ctx, tokenC, testSecret)
	if err == nil || !strings.Contains(err.Error(), "already been used") {
		t.Fatalf("expected already used error on reuse of token C, got: %v", err)
	}

	// Verify challenges state in database
	var countActive int
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM password_recovery_challenges WHERE account_id = $1 AND status = 'ACTIVE'", userID).Scan(&countActive)
	if countActive != 0 {
		t.Fatalf("expected 0 active challenges remaining, got %d", countActive)
	}
}

// 9. Failure Mode Test: Delivery failure or unconfigured email provider does not report success
// and preserves account state intact.
func TestEmailVerificationDeliveryFailureHandling(t *testing.T) {
	storage, _, _ := createIsolatedAuthTestDB(t)
	ctx := context.Background()
	testSecret := "purohit_platform_test_jwt_secret_key_32bytes_long!"

	userID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876543888",
		FullName:      "Delivery Failure User",
		Password:      "SecurePassword123!",
		ConsentAgreed: true,
	})
	_ = storage.SetAccountVerifiedEmail(ctx, "devotee", userID, "safe.original@example.com", true)

	// 1. Unconfigured provider returns ErrEmailUnavailable immediately
	auth.SetEmailProvider(&auth.UnconfiguredEmailProvider{})
	_, err := storage.RequestEmailVerification(ctx, "devotee", userID, "new.attempt@example.com", testSecret)
	if !errors.Is(err, auth.ErrEmailUnavailable) {
		t.Fatalf("expected ErrEmailUnavailable when unconfigured, got: %v", err)
	}

	// Account email remains untouched
	email, verified, _ := storage.GetAccountEmail(ctx, "devotee", userID)
	if email != "safe.original@example.com" || !verified {
		t.Fatalf("account email was corrupted on unconfigured delivery!")
	}

	// 2. Simulated SMTP delivery error returns ErrDeliveryFailed
	failingMock := &auth.MockEmailProvider{
		Configured:  true,
		SimulateErr: errors.New("connection reset by peer on port 587"),
	}
	auth.SetEmailProvider(failingMock)

	_, err = storage.RequestEmailVerification(ctx, "devotee", userID, "new.attempt@example.com", testSecret)
	if err == nil || !errors.Is(err, auth.ErrDeliveryFailed) {
		t.Fatalf("expected ErrDeliveryFailed when sending fails, got: %v", err)
	}

	// Account email remains untouched
	email, verified, _ = storage.GetAccountEmail(ctx, "devotee", userID)
	if email != "safe.original@example.com" || !verified {
		t.Fatalf("account email was corrupted on delivery failure!")
	}
}

// 10. Security Test: Legacy SMS code path cannot bypass verified email requirements.
func TestLegacySMSCodePathCannotBypassVerifiedEmail(t *testing.T) {
	storage, db, _ := createIsolatedAuthTestDB(t)
	ctx := context.Background()
	testSecret := "purohit_platform_test_jwt_secret_key_32bytes_long!"

	auth.SetEmailProvider(&auth.MockEmailProvider{Configured: true})
	defer auth.SetEmailProvider(&auth.UnconfiguredEmailProvider{})
	auth.DefaultThrottler.Reset()
	defer auth.DefaultThrottler.Reset()

	// 1. Account with NO verified email cannot reset password via code
	userID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876543777",
		FullName:      "Unverified User",
		Password:      "OldPass123!",
		ConsentAgreed: true,
	})
	// Insert an active challenge in DB manually as if legacy system created it
	codeHash := auth.HashRecoveryCode("123456", testSecret)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO password_recovery_challenges
			(account_id, account_role, phone, email, purpose, code_hash, expires_at, status)
		VALUES
			($1, 'devotee', '+919876543777', '', 'password_reset', $2, CURRENT_TIMESTAMP + INTERVAL '10 minutes', 'ACTIVE')
	`, userID, codeHash)

	// Attempting to confirm via legacy code MUST be rejected because account has no verified email!
	err := storage.ConfirmPasswordRecovery(ctx, models.ConfirmRecoveryRequest{
		Phone:           "+919876543777",
		Role:            "devotee",
		Code:            "123456",
		NewPassword:     "NewBypassPass123!",
		ConfirmPassword: "NewBypassPass123!",
	}, "127.0.0.1", testSecret)

	if err == nil || !strings.Contains(err.Error(), "verified email address") {
		t.Fatalf("CRITICAL SECURITY ERROR: Legacy code bypassed verified email rule! err: %v", err)
	}

	// 2. An email_verification challenge code cannot be used to reset password
	auth.DefaultThrottler.Reset()
	_ = storage.SetAccountVerifiedEmail(ctx, "devotee", userID, "user@example.com", true)
	_, _ = db.ExecContext(ctx, "DELETE FROM password_recovery_challenges WHERE account_id = $1", userID)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO password_recovery_challenges
			(account_id, account_role, phone, email, purpose, code_hash, expires_at, status)
		VALUES
			($1, 'devotee', '+919876543777', 'user@example.com', 'email_verification', $2, CURRENT_TIMESTAMP + INTERVAL '10 minutes', 'ACTIVE')
	`, userID, codeHash)

	err = storage.ConfirmPasswordRecovery(ctx, models.ConfirmRecoveryRequest{
		Phone:           "+919876543777",
		Role:            "devotee",
		Code:            "123456",
		NewPassword:     "NewBypassPass123!",
		ConfirmPassword: "NewBypassPass123!",
	}, "127.0.0.1", testSecret)

	if err == nil || !strings.Contains(err.Error(), "invalid or expired") {
		t.Fatalf("email_verification challenge should not be accepted for password reset, got: %v", err)
	}
}

// 11. Concurrency Test (Devotee): Serializes verification requests and confirmations per account.
// Proves concurrent requests cannot leave multiple active links, an older link cannot overwrite a newer request,
// and existing verified email is preserved until replacement confirmation.
func TestConcurrentEmailVerificationSerializationDevotee(t *testing.T) {
	storage, db, _ := createIsolatedAuthTestDB(t)
	ctx := context.Background()
	testSecret := "purohit_platform_test_jwt_secret_key_32bytes_long!"

	auth.SetEmailProvider(&auth.MockEmailProvider{Configured: true})
	defer auth.SetEmailProvider(&auth.UnconfiguredEmailProvider{})

	// 1. Setup devotee with initial verified email A
	devoteeID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876540111",
		FullName:      "Concurrent Devotee",
		Password:      "SecurePassword123!",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("failed registering devotee: %v", err)
	}
	emailA := "initial.verified.devotee@example.com"
	if err := storage.SetAccountVerifiedEmail(ctx, "devotee", devoteeID, emailA, true); err != nil {
		t.Fatalf("failed setting initial verified email: %v", err)
	}

	// 2. Launch 8 concurrent verification requests for different emails
	const numRequests = 8
	tokens := make([]string, numRequests)
	emails := make([]string, numRequests)
	for i := 0; i < numRequests; i++ {
		emails[i] = fmt.Sprintf("devotee.replacement.%d@example.com", i)
	}

	var wg sync.WaitGroup
	errs := make([]error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tok, err := storage.RequestEmailVerification(ctx, "devotee", devoteeID, emails[idx], testSecret)
			tokens[idx] = tok
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("RequestEmailVerification %d failed: %v", i, err)
		}
		if tokens[i] == "" {
			t.Fatalf("expected non-empty token for request %d", i)
		}
	}

	// INVARIANT 1: There must be EXACTLY ONE ACTIVE challenge in DB for this account!
	var activeCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM password_recovery_challenges
		WHERE account_role = 'devotee' AND account_id = $1 AND purpose = 'email_verification' AND status = 'ACTIVE'
	`, devoteeID).Scan(&activeCount)
	if err != nil {
		t.Fatalf("failed querying active challenge count: %v", err)
	}
	if activeCount != 1 {
		t.Fatalf("CRITICAL CONCURRENCY VIOLATION: Expected exactly 1 ACTIVE challenge, got %d", activeCount)
	}

	// INVARIANT 2: Exactly (numRequests - 1) challenges must be SUPERSEDED!
	var supersededCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM password_recovery_challenges
		WHERE account_role = 'devotee' AND account_id = $1 AND purpose = 'email_verification' AND status = 'SUPERSEDED'
	`, devoteeID).Scan(&supersededCount)
	if err != nil {
		t.Fatalf("failed querying superseded challenge count: %v", err)
	}
	if supersededCount != numRequests-1 {
		t.Fatalf("Expected %d SUPERSEDED challenges, got %d", numRequests-1, supersededCount)
	}

	// INVARIANT 3: Currently verified email A is strictly PRESERVED on the account!
	currentEmail, isVerified, err := storage.GetAccountEmail(ctx, "devotee", devoteeID)
	if err != nil || currentEmail != emailA || !isVerified {
		t.Fatalf("CRITICAL: Initial email A was overwritten prematurely! current=%s, verified=%v", currentEmail, isVerified)
	}

	// Find active token and superseded tokens
	var activeToken string
	var supersededTokens []string
	for _, tok := range tokens {
		h := auth.HashResetToken(tok, testSecret)
		var st string
		_ = db.QueryRowContext(ctx, "SELECT status FROM password_recovery_challenges WHERE code_hash = $1", h).Scan(&st)
		if st == "ACTIVE" {
			activeToken = tok
		} else if st == "SUPERSEDED" {
			supersededTokens = append(supersededTokens, tok)
		}
	}
	if activeToken == "" {
		t.Fatalf("failed to find active token among generated tokens")
	}

	// INVARIANT 4: Concurrently attempting to confirm all superseded tokens MUST be rejected!
	var confirmWg sync.WaitGroup
	supersededConfirmErrors := make([]error, len(supersededTokens))
	for i, supTok := range supersededTokens {
		confirmWg.Add(1)
		go func(idx int, tStr string) {
			defer confirmWg.Done()
			_, _, err := storage.ConfirmEmailVerification(ctx, tStr, testSecret)
			supersededConfirmErrors[idx] = err
		}(i, supTok)
	}
	confirmWg.Wait()

	for i, cErr := range supersededConfirmErrors {
		if cErr == nil || (!strings.Contains(cErr.Error(), "superseded") && !strings.Contains(cErr.Error(), "no longer valid")) {
			t.Fatalf("CRITICAL SECURITY ERROR: Superseded token %d was accepted or not marked superseded! err: %v", i, cErr)
		}
	}

	// Account email MUST STILL BE email A!
	currentEmail, isVerified, _ = storage.GetAccountEmail(ctx, "devotee", devoteeID)
	if currentEmail != emailA || !isVerified {
		t.Fatalf("Superseded token confirmation modified account email! current=%s", currentEmail)
	}

	// INVARIANT 5: Confirming the single ACTIVE token succeeds and updates the email atomically
	verifiedEmail, role, err := storage.ConfirmEmailVerification(ctx, activeToken, testSecret)
	if err != nil {
		t.Fatalf("Failed confirming active token: %v", err)
	}
	if role != "devotee" {
		t.Fatalf("expected role devotee, got %s", role)
	}

	// Account email is now verifiedEmail
	currentEmail, isVerified, _ = storage.GetAccountEmail(ctx, "devotee", devoteeID)
	if currentEmail != verifiedEmail || !isVerified {
		t.Fatalf("expected account email to be updated to %s, got %s", verifiedEmail, currentEmail)
	}

	// Confirming active token AGAIN fails as already used
	_, _, err = storage.ConfirmEmailVerification(ctx, activeToken, testSecret)
	if err == nil || !strings.Contains(err.Error(), "already been used") {
		t.Fatalf("expected already been used on repeat confirmation, got: %v", err)
	}
}

// 12. Concurrency Test (Priest): Serializes verification requests and confirmations per priest account.
func TestConcurrentEmailVerificationSerializationPriest(t *testing.T) {
	storage, db, _ := createIsolatedAuthTestDB(t)
	ctx := context.Background()
	testSecret := "purohit_platform_test_jwt_secret_key_32bytes_long!"

	auth.SetEmailProvider(&auth.MockEmailProvider{Configured: true})
	defer auth.SetEmailProvider(&auth.UnconfiguredEmailProvider{})

	// 1. Setup priest with initial verified email A
	priestID, err := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone:           "+919876540222",
		FullName:        "Concurrent Priest",
		Password:        "VedicPassword123!",
		Sampradaya:      "Smartha",
		VedaShakha:      "Rigveda",
		ExperienceYears: 12,
		ConsentAgreed:   true,
	})
	if err != nil {
		t.Fatalf("failed registering priest: %v", err)
	}
	emailA := "initial.verified.priest@example.com"
	if err := storage.SetAccountVerifiedEmail(ctx, "priest", priestID, emailA, true); err != nil {
		t.Fatalf("failed setting initial verified email for priest: %v", err)
	}

	// 2. Launch 8 concurrent verification requests for different emails
	const numRequests = 8
	tokens := make([]string, numRequests)
	emails := make([]string, numRequests)
	for i := 0; i < numRequests; i++ {
		emails[i] = fmt.Sprintf("priest.replacement.%d@example.com", i)
	}

	var wg sync.WaitGroup
	errs := make([]error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tok, err := storage.RequestEmailVerification(ctx, "priest", priestID, emails[idx], testSecret)
			tokens[idx] = tok
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("RequestEmailVerification for priest %d failed: %v", i, err)
		}
		if tokens[i] == "" {
			t.Fatalf("expected non-empty token for priest request %d", i)
		}
	}

	// INVARIANT 1: There must be EXACTLY ONE ACTIVE challenge in DB for this priest account!
	var activeCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM password_recovery_challenges
		WHERE account_role = 'priest' AND account_id = $1 AND purpose = 'email_verification' AND status = 'ACTIVE'
	`, priestID).Scan(&activeCount)
	if err != nil {
		t.Fatalf("failed querying active challenge count: %v", err)
	}
	if activeCount != 1 {
		t.Fatalf("CRITICAL CONCURRENCY VIOLATION: Expected exactly 1 ACTIVE challenge for priest, got %d", activeCount)
	}

	// INVARIANT 2: Exactly (numRequests - 1) challenges must be SUPERSEDED!
	var supersededCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM password_recovery_challenges
		WHERE account_role = 'priest' AND account_id = $1 AND purpose = 'email_verification' AND status = 'SUPERSEDED'
	`, priestID).Scan(&supersededCount)
	if err != nil {
		t.Fatalf("failed querying superseded challenge count: %v", err)
	}
	if supersededCount != numRequests-1 {
		t.Fatalf("Expected %d SUPERSEDED challenges for priest, got %d", numRequests-1, supersededCount)
	}

	// INVARIANT 3: Currently verified email A is strictly PRESERVED on the priest account!
	currentEmail, isVerified, err := storage.GetAccountEmail(ctx, "priest", priestID)
	if err != nil || currentEmail != emailA || !isVerified {
		t.Fatalf("CRITICAL: Initial priest email A was overwritten prematurely! current=%s, verified=%v", currentEmail, isVerified)
	}

	// Find active token and superseded tokens
	var activeToken string
	var supersededTokens []string
	for _, tok := range tokens {
		h := auth.HashResetToken(tok, testSecret)
		var st string
		_ = db.QueryRowContext(ctx, "SELECT status FROM password_recovery_challenges WHERE code_hash = $1", h).Scan(&st)
		if st == "ACTIVE" {
			activeToken = tok
		} else if st == "SUPERSEDED" {
			supersededTokens = append(supersededTokens, tok)
		}
	}
	if activeToken == "" {
		t.Fatalf("failed to find active token among generated priest tokens")
	}

	// INVARIANT 4: Concurrently attempting to confirm all superseded tokens MUST be rejected!
	var confirmWg sync.WaitGroup
	supersededConfirmErrors := make([]error, len(supersededTokens))
	for i, supTok := range supersededTokens {
		confirmWg.Add(1)
		go func(idx int, tStr string) {
			defer confirmWg.Done()
			_, _, err := storage.ConfirmEmailVerification(ctx, tStr, testSecret)
			supersededConfirmErrors[idx] = err
		}(i, supTok)
	}
	confirmWg.Wait()

	for i, cErr := range supersededConfirmErrors {
		if cErr == nil || (!strings.Contains(cErr.Error(), "superseded") && !strings.Contains(cErr.Error(), "no longer valid")) {
			t.Fatalf("CRITICAL SECURITY ERROR: Superseded priest token %d was accepted! err: %v", i, cErr)
		}
	}

	// Priest email MUST STILL BE email A!
	currentEmail, isVerified, _ = storage.GetAccountEmail(ctx, "priest", priestID)
	if currentEmail != emailA || !isVerified {
		t.Fatalf("Superseded token confirmation modified priest email! current=%s", currentEmail)
	}

	// INVARIANT 5: Confirming the single ACTIVE token succeeds and updates the email atomically
	verifiedEmail, role, err := storage.ConfirmEmailVerification(ctx, activeToken, testSecret)
	if err != nil {
		t.Fatalf("Failed confirming active priest token: %v", err)
	}
	if role != "priest" {
		t.Fatalf("expected role priest, got %s", role)
	}

	// Priest email is now verifiedEmail
	currentEmail, isVerified, _ = storage.GetAccountEmail(ctx, "priest", priestID)
	if currentEmail != verifiedEmail || !isVerified {
		t.Fatalf("expected priest email to be updated to %s, got %s", verifiedEmail, currentEmail)
	}
}
