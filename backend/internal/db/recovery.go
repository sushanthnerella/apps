package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/models"
)

// StartPasswordRecovery initiates verified password recovery by validating the account,
// checking rate limits/cooldown, generating an opaque single-use cryptographic token,
// and dispatching a secure reset link to the account's already-verified email address.
func (s *Storage) StartPasswordRecovery(ctx context.Context, req models.StartRecoveryRequest, clientIP, secret string) (*models.StartRecoveryResponse, error) {
	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role != "devotee" && role != "priest" {
		return nil, fmt.Errorf("role is required and must be either 'devotee' or 'priest'")
	}

	emailInput := strings.TrimSpace(req.Email)
	phoneInput := strings.TrimSpace(req.Phone)

	if emailInput == "" && phoneInput == "" {
		return nil, fmt.Errorf("either email or phone number is required for password recovery")
	}

	var (
		accountID     string
		accountPhone  string
		verifiedEmail string
		isVerified    bool
		throttleKey   string
	)

	// Mode A: Recovery requested with Email address (Primary flow)
	if emailInput != "" {
		cleanEmail, err := auth.ValidateEmailFormat(emailInput)
		if err != nil {
			auth.EqualizeTiming(emailInput)
			// Return generic response without revealing email formatting/absence
			return &models.StartRecoveryResponse{
				Status:  "ACCEPTED",
				Message: "If a verified account exists with this email address, a password reset link will be sent.",
			}, nil
		}

		throttleKey = auth.BuildEmailThrottleKey(role, cleanEmail)

		// Check rate limits & cooldown
		allowed, retryAfter, err := auth.DefaultRecoveryThrottler.CheckStartAllowed(throttleKey, clientIP)
		if !allowed {
			return nil, fmt.Errorf("%w: please retry in %d seconds", err, int(retryAfter.Seconds()))
		}

		// Fail safely if email provider is unconfigured
		if !auth.GetEmailProvider().IsConfigured() {
			return nil, auth.ErrEmailUnavailable
		}

		// Query account by verified email
		var query string
		if role == "devotee" {
			query = `SELECT id, phone, email, email_verified FROM users WHERE LOWER(email) = LOWER($1)`
		} else {
			query = `SELECT id, phone, email, email_verified FROM priests WHERE LOWER(email) = LOWER($1)`
		}

		err = s.DB.QueryRowContext(ctx, query, cleanEmail).Scan(&accountID, &accountPhone, &verifiedEmail, &isVerified)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				auth.EqualizeTiming(cleanEmail)
				auth.DefaultRecoveryThrottler.RecordStartAttempt(throttleKey, clientIP)
				return &models.StartRecoveryResponse{
					Status:  "ACCEPTED",
					Message: "If a verified account exists with this email address, a password reset link will be sent.",
				}, nil
			}
			return nil, fmt.Errorf("failed resolving account for password recovery: %w", err)
		}

		// Recovery must use an email already verified for that account/role
		if !isVerified {
			auth.EqualizeTiming(cleanEmail)
			auth.DefaultRecoveryThrottler.RecordStartAttempt(throttleKey, clientIP)
			return &models.StartRecoveryResponse{
				Status:  "ACCEPTED",
				Message: "If a verified account exists with this email address, a password reset link will be sent.",
			}, nil
		}
	} else {
		// Mode B: Recovery requested with Phone number
		// Invariant: Never claim an account using only its phone number.
		// If the account has a verified email, send the reset link to THAT verified email.
		// If the account does not have a verified email, return generic response (legacy accounts require manual recovery).
		canonicalPhone, err := auth.NormalizePhone(phoneInput)
		if err != nil {
			auth.EqualizeTiming(phoneInput)
			return &models.StartRecoveryResponse{
				Status:  "ACCEPTED",
				Message: "If a verified account exists with this phone number, a password reset link will be sent.",
			}, nil
		}

		throttleKey = auth.BuildAccountThrottleKey(role, canonicalPhone)
		allowed, retryAfter, err := auth.DefaultRecoveryThrottler.CheckStartAllowed(throttleKey, clientIP)
		if !allowed {
			return nil, fmt.Errorf("%w: please retry in %d seconds", err, int(retryAfter.Seconds()))
		}

		if !auth.GetEmailProvider().IsConfigured() {
			return nil, auth.ErrEmailUnavailable
		}

		variants, _ := auth.PhoneLookupVariants(phoneInput)
		var query string
		if role == "devotee" {
			query = `SELECT id, phone, COALESCE(email, ''), email_verified FROM users WHERE phone = ANY($1)`
		} else {
			query = `SELECT id, phone, COALESCE(email, ''), email_verified FROM priests WHERE phone = ANY($1)`
		}

		rows, err := s.DB.QueryContext(ctx, query, pq.Array(variants))
		if err != nil {
			return nil, fmt.Errorf("failed resolving account: %w", err)
		}
		defer rows.Close()

		var matchedAccounts []struct {
			id         string
			phone      string
			email      string
			isVerified bool
		}
		for rows.Next() {
			var acc struct {
				id         string
				phone      string
				email      string
				isVerified bool
			}
			if err := rows.Scan(&acc.id, &acc.phone, &acc.email, &acc.isVerified); err == nil {
				matchedAccounts = append(matchedAccounts, acc)
			}
		}

		if len(matchedAccounts) != 1 || !matchedAccounts[0].isVerified || matchedAccounts[0].email == "" {
			auth.EqualizeTiming(phoneInput)
			auth.DefaultRecoveryThrottler.RecordStartAttempt(throttleKey, clientIP)
			return &models.StartRecoveryResponse{
				Status:  "ACCEPTED",
				Message: "If a verified account exists with this phone number, a password reset link will be sent.",
			}, nil
		}

		accountID = matchedAccounts[0].id
		accountPhone = matchedAccounts[0].phone
		verifiedEmail = matchedAccounts[0].email
		isVerified = true
	}

	// Generate cryptographically secure single-use reset token
	rawToken, err := auth.GenerateResetToken()
	if err != nil {
		return nil, fmt.Errorf("failed generating reset token: %w", err)
	}

	tokenHash := auth.HashResetToken(rawToken, secret)

	// Build reset link using configured HTTPS endpoint (never Host header)
	baseURL := strings.TrimSpace(os.Getenv("PASSWORD_RESET_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:8085/api/v1/auth/recovery/reset"
	}
	resetURL, err := auth.BuildPasswordResetURL(baseURL, rawToken)
	if err != nil {
		return nil, fmt.Errorf("failed constructing password reset URL: %w", err)
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed starting recovery transaction: %w", err)
	}
	defer tx.Rollback()

	// Invalidate earlier active challenges for this account (supersede)
	_, err = tx.ExecContext(ctx, `
		UPDATE password_recovery_challenges
		SET status = 'SUPERSEDED'
		WHERE account_role = $1 AND account_id = $2 AND status = 'ACTIVE'
	`, role, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed superseding active recovery challenges: %w", err)
	}

	// Insert new challenge record
	var challengeID string
	insertQuery := `
		INSERT INTO password_recovery_challenges
			(account_id, account_role, phone, email, purpose, code_hash, expires_at, status, delivery_status)
		VALUES
			($1, $2, $3, $4, 'password_reset', $5, CURRENT_TIMESTAMP + INTERVAL '15 minutes', 'ACTIVE', 'PENDING')
		RETURNING id
	`
	err = tx.QueryRowContext(ctx, insertQuery, accountID, role, accountPhone, verifiedEmail, tokenHash).Scan(&challengeID)
	if err != nil {
		return nil, fmt.Errorf("failed persisting recovery challenge: %w", err)
	}

	// Dispatch email via provider
	if err := auth.GetEmailProvider().SendPasswordResetEmail(ctx, verifiedEmail, resetURL); err != nil {
		_, _ = tx.ExecContext(ctx, "UPDATE password_recovery_challenges SET status = 'FAILED', delivery_status = 'FAILED' WHERE id = $1", challengeID)
		return nil, fmt.Errorf("failed dispatching password reset email: %w", err)
	}

	// Update delivery status to SENT
	_, err = tx.ExecContext(ctx, "UPDATE password_recovery_challenges SET delivery_status = 'SENT' WHERE id = $1", challengeID)
	if err != nil {
		return nil, fmt.Errorf("failed updating challenge delivery status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed committing recovery challenge: %w", err)
	}

	auth.DefaultRecoveryThrottler.RecordStartAttempt(throttleKey, clientIP)

	return &models.StartRecoveryResponse{
		Status:  "ACCEPTED",
		Message: "If a verified account exists with this email address, a password reset link will be sent.",
	}, nil
}

// ValidateResetToken checks if a reset token is valid, active, and unexpired.
// Crucially: visiting a reset link DOES NOT consume the token.
func (s *Storage) ValidateResetToken(ctx context.Context, token, secret string) (*models.ResetTokenInfo, error) {
	rawToken := strings.TrimSpace(token)
	if rawToken == "" {
		return nil, errors.New("reset token is required")
	}

	tokenHash := auth.HashResetToken(rawToken, secret)

	var info models.ResetTokenInfo
	var status string
	var expiresAt time.Time
	var attemptCount, maxAttempts int

	query := `
		SELECT account_id, account_role, COALESCE(email, ''), expires_at, status, attempt_count, max_attempts
		FROM password_recovery_challenges
		WHERE code_hash = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := s.DB.QueryRowContext(ctx, query, tokenHash).Scan(
		&info.AccountID, &info.Role, &info.Email, &expiresAt, &status, &attemptCount, &maxAttempts,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid or non-existent reset token")
		}
		return nil, fmt.Errorf("failed querying reset token: %w", err)
	}

	info.ExpiresAt = expiresAt

	if status == "CONSUMED" {
		return nil, errors.New("this password reset link has already been used")
	}
	if status == "SUPERSEDED" {
		return nil, errors.New("this password reset link has been superseded by a newer request")
	}
	if status != "ACTIVE" {
		return nil, errors.New("this password reset link is no longer valid")
	}
	if time.Now().After(expiresAt) {
		_, _ = s.DB.ExecContext(ctx, "UPDATE password_recovery_challenges SET status = 'EXPIRED' WHERE code_hash = $1", tokenHash)
		return nil, errors.New("this password reset link has expired (15-minute validity window)")
	}
	if attemptCount >= maxAttempts {
		return nil, errors.New("maximum reset attempts exceeded for this link")
	}

	return &info, nil
}

// ConfirmPasswordRecovery atomically validates the challenge token, updates the bcrypt password hash,
// increments session_version (immediately revoking existing JWT sessions), and marks the challenge CONSUMED.
func (s *Storage) ConfirmPasswordRecovery(ctx context.Context, req models.ConfirmRecoveryRequest, clientIP, secret string) error {
	if req.NewPassword == "" || req.ConfirmPassword == "" {
		return fmt.Errorf("new password and confirmation password are required")
	}

	if req.NewPassword != req.ConfirmPassword {
		return fmt.Errorf("new password and confirmation password do not match")
	}

	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		return err
	}

	rawToken := strings.TrimSpace(req.Token)

	// If token is provided, execute primary email token confirmation
	if rawToken != "" {
		tokenHash := auth.HashResetToken(rawToken, secret)

		throttleKey := "token:" + tokenHash[:16]
		if throttled, rem := auth.DefaultThrottler.IsThrottled(throttleKey, clientIP); throttled {
			return fmt.Errorf("too many failed verification attempts. Please retry in %d seconds", int(rem.Seconds()))
		}

		newHash, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			return fmt.Errorf("failed hashing password: %w", err)
		}

		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed beginning confirmation transaction: %w", err)
		}
		defer tx.Rollback()

		var (
			challengeID  string
			accountID    string
			role         string
			expiresAt    time.Time
			status       string
			maxAttempts  int
			attemptCount int
		)

		queryChallenge := `
			SELECT id, account_id, account_role, expires_at, status, max_attempts, attempt_count
			FROM password_recovery_challenges
			WHERE code_hash = $1
			FOR UPDATE
		`
		err = tx.QueryRowContext(ctx, queryChallenge, tokenHash).Scan(
			&challengeID, &accountID, &role, &expiresAt, &status, &maxAttempts, &attemptCount,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				auth.DefaultThrottler.RecordFailure(throttleKey, clientIP)
				return fmt.Errorf("invalid or expired reset token")
			}
			return fmt.Errorf("failed querying challenge: %w", err)
		}

		if status == "CONSUMED" {
			auth.DefaultThrottler.RecordFailure(throttleKey, clientIP)
			return fmt.Errorf("this reset token has already been used")
		}
		if status != "ACTIVE" {
			auth.DefaultThrottler.RecordFailure(throttleKey, clientIP)
			return fmt.Errorf("invalid reset token")
		}
		if time.Now().After(expiresAt) {
			_, _ = tx.ExecContext(ctx, "UPDATE password_recovery_challenges SET status = 'EXPIRED' WHERE id = $1", challengeID)
			_ = tx.Commit()
			auth.DefaultThrottler.RecordFailure(throttleKey, clientIP)
			return fmt.Errorf("reset token has expired. Please request a new password reset")
		}
		if attemptCount >= maxAttempts {
			_, _ = tx.ExecContext(ctx, "UPDATE password_recovery_challenges SET status = 'FAILED' WHERE id = $1", challengeID)
			_ = tx.Commit()
			auth.DefaultThrottler.RecordFailure(throttleKey, clientIP)
			return fmt.Errorf("maximum reset attempts exceeded. Please request a new reset link")
		}

		// Atomically consume challenge
		attemptCount++
		_, err = tx.ExecContext(ctx, `
			UPDATE password_recovery_challenges
			SET status = 'CONSUMED', consumed_at = CURRENT_TIMESTAMP, attempt_count = $1
			WHERE id = $2
		`, attemptCount, challengeID)
		if err != nil {
			return fmt.Errorf("failed consuming recovery challenge: %w", err)
		}

		// Invalidate any remaining recovery challenges for this account
		_, err = tx.ExecContext(ctx, `
			UPDATE password_recovery_challenges
			SET status = 'SUPERSEDED'
			WHERE account_role = $1 AND account_id = $2 AND id != $3 AND status = 'ACTIVE'
		`, role, accountID, challengeID)
		if err != nil {
			return fmt.Errorf("failed superseding remaining challenges: %w", err)
		}

		// Update password hash and atomically bump session_version to revoke existing sessions
		if role == "devotee" {
			res, err := tx.ExecContext(ctx, "UPDATE users SET password_hash = $1, session_version = session_version + 1 WHERE id = $2", newHash, accountID)
			if err != nil {
				return fmt.Errorf("failed updating devotee password: %w", err)
			}
			if rows, _ := res.RowsAffected(); rows == 0 {
				return fmt.Errorf("devotee account not found for update")
			}
		} else {
			res, err := tx.ExecContext(ctx, "UPDATE priests SET password_hash = $1, session_version = session_version + 1 WHERE id = $2", newHash, accountID)
			if err != nil {
				return fmt.Errorf("failed updating priest password: %w", err)
			}
			if rows, _ := res.RowsAffected(); rows == 0 {
				return fmt.Errorf("priest account not found for update")
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed committing password reset: %w", err)
		}

		auth.DefaultThrottler.RecordSuccess(throttleKey)
		log.Printf("🔑 Password reset successful via email link for %s (%s). Session version incremented.", role, accountID)
		return nil
	}

	// Fallback for code-based submission if code is supplied
	code := strings.TrimSpace(req.Code)
	if len(code) != 6 {
		return fmt.Errorf("reset token or 6-digit verification code is required")
	}

	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role != "devotee" && role != "priest" {
		return fmt.Errorf("role is required and must be either 'devotee' or 'priest'")
	}

	canonicalPhone, err := auth.NormalizePhone(req.Phone)
	if err != nil {
		return fmt.Errorf("invalid phone number format")
	}

	accountKey := auth.BuildAccountThrottleKey(role, canonicalPhone)
	if throttled, rem := auth.DefaultThrottler.IsThrottled(accountKey, clientIP); throttled {
		return fmt.Errorf("too many failed verification attempts. Please retry in %d seconds", int(rem.Seconds()))
	}

	variants, _ := auth.PhoneLookupVariants(req.Phone)
	var accountIDs []string
	var accountEmail string
	var emailVerified bool

	if role == "devotee" {
		rows, err := s.DB.QueryContext(ctx, "SELECT id, COALESCE(email, ''), email_verified FROM users WHERE phone = ANY($1)", pq.Array(variants))
		if err != nil {
			return fmt.Errorf("failed querying account: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id, em string
			var ver bool
			if err := rows.Scan(&id, &em, &ver); err == nil {
				accountIDs = append(accountIDs, id)
				accountEmail = em
				emailVerified = ver
			}
		}
	} else {
		rows, err := s.DB.QueryContext(ctx, "SELECT id, COALESCE(email, ''), email_verified FROM priests WHERE phone = ANY($1)", pq.Array(variants))
		if err != nil {
			return fmt.Errorf("failed querying account: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id, em string
			var ver bool
			if err := rows.Scan(&id, &em, &ver); err == nil {
				accountIDs = append(accountIDs, id)
				accountEmail = em
				emailVerified = ver
			}
		}
	}

	if len(accountIDs) != 1 {
		auth.DefaultThrottler.RecordFailure(accountKey, clientIP)
		return fmt.Errorf("invalid or expired verification code")
	}

	// Invariant: Self-service recovery requires a verified email address. Phone-only resets are disabled.
	if !emailVerified || strings.TrimSpace(accountEmail) == "" {
		auth.DefaultThrottler.RecordFailure(accountKey, clientIP)
		return fmt.Errorf("account does not have a verified email address. Self-service password recovery is disabled; manual verified recovery is required")
	}

	accountID := accountIDs[0]
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed hashing password: %w", err)
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed beginning confirmation transaction: %w", err)
	}
	defer tx.Rollback()

	var (
		challengeID    string
		codeHash       string
		expiresAt      time.Time
		maxAttempts    int
		attemptCount   int
		challengeEmail string
	)

	queryChallenge := `
		SELECT id, code_hash, expires_at, max_attempts, attempt_count, COALESCE(email, '')
		FROM password_recovery_challenges
		WHERE account_role = $1 AND account_id = $2 AND purpose = 'password_reset' AND status = 'ACTIVE'
		ORDER BY created_at DESC
		LIMIT 1
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, queryChallenge, role, accountID).Scan(
		&challengeID, &codeHash, &expiresAt, &maxAttempts, &attemptCount, &challengeEmail,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			auth.DefaultThrottler.RecordFailure(accountKey, clientIP)
			return fmt.Errorf("invalid or expired verification code")
		}
		return fmt.Errorf("failed querying challenge: %w", err)
	}

	// Ensure challenge matches the account's verified email
	if !strings.EqualFold(challengeEmail, accountEmail) {
		auth.DefaultThrottler.RecordFailure(accountKey, clientIP)
		return fmt.Errorf("recovery challenge email mismatch")
	}

	if time.Now().After(expiresAt) {
		_, _ = tx.ExecContext(ctx, "UPDATE password_recovery_challenges SET status = 'EXPIRED' WHERE id = $1", challengeID)
		_ = tx.Commit()
		auth.DefaultThrottler.RecordFailure(accountKey, clientIP)
		return fmt.Errorf("verification code has expired. Please request a new code")
	}

	if attemptCount >= maxAttempts {
		_, _ = tx.ExecContext(ctx, "UPDATE password_recovery_challenges SET status = 'FAILED' WHERE id = $1", challengeID)
		_ = tx.Commit()
		auth.DefaultThrottler.RecordFailure(accountKey, clientIP)
		return fmt.Errorf("maximum verification attempts exceeded. Please request a new code")
	}

	attemptCount++
	if !auth.VerifyRecoveryCode(code, codeHash, secret) {
		_, _ = tx.ExecContext(ctx, "UPDATE password_recovery_challenges SET attempt_count = $1 WHERE id = $2", attemptCount, challengeID)
		_ = tx.Commit()
		auth.DefaultThrottler.RecordFailure(accountKey, clientIP)
		return fmt.Errorf("invalid verification code")
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE password_recovery_challenges
		SET status = 'CONSUMED', consumed_at = CURRENT_TIMESTAMP, attempt_count = $1
		WHERE id = $2
	`, attemptCount, challengeID)
	if err != nil {
		return fmt.Errorf("failed consuming recovery challenge: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE password_recovery_challenges
		SET status = 'SUPERSEDED'
		WHERE account_role = $1 AND account_id = $2 AND id != $3 AND status = 'ACTIVE'
	`, role, accountID, challengeID)
	if err != nil {
		return fmt.Errorf("failed superseding remaining challenges: %w", err)
	}

	if role == "devotee" {
		res, err := tx.ExecContext(ctx, "UPDATE users SET password_hash = $1, session_version = session_version + 1 WHERE id = $2", newHash, accountID)
		if err != nil {
			return fmt.Errorf("failed updating devotee password: %w", err)
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			return fmt.Errorf("devotee account not found for update")
		}
	} else {
		res, err := tx.ExecContext(ctx, "UPDATE priests SET password_hash = $1, session_version = session_version + 1 WHERE id = $2", newHash, accountID)
		if err != nil {
			return fmt.Errorf("failed updating priest password: %w", err)
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			return fmt.Errorf("priest account not found for update")
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed committing password reset: %w", err)
	}

	auth.DefaultThrottler.RecordSuccess(accountKey)
	return nil
}

// ResetPassword validates through the recovery verification path.
func (s *Storage) ResetPassword(req models.ResetPasswordRequest) error {
	secret, err := auth.GetSecretKey()
	if err != nil {
		return err
	}
	confirmReq := models.ConfirmRecoveryRequest{
		Phone:           req.Phone,
		Role:            req.Role,
		Code:            req.OTP,
		NewPassword:     req.NewPassword,
		ConfirmPassword: req.NewPassword,
	}
	return s.ConfirmPasswordRecovery(context.Background(), confirmReq, "127.0.0.1", secret)
}

// SetAccountVerifiedEmail assigns and optionally verifies an email address for an account.
// Used for verified authenticated profile email additions and manual recovery for legacy accounts.
func (s *Storage) SetAccountVerifiedEmail(ctx context.Context, role, accountID, email string, verified bool) error {
	cleanRole := strings.ToLower(strings.TrimSpace(role))
	if cleanRole != "devotee" && cleanRole != "priest" {
		return fmt.Errorf("invalid role: must be 'devotee' or 'priest'")
	}

	cleanEmail, err := auth.ValidateEmailFormat(email)
	if err != nil {
		return err
	}

	// Verify no other account in the same role has claimed this email
	var collisionQuery string
	if cleanRole == "devotee" {
		collisionQuery = `SELECT COUNT(*) FROM users WHERE LOWER(email) = LOWER($1) AND id != $2`
	} else {
		collisionQuery = `SELECT COUNT(*) FROM priests WHERE LOWER(email) = LOWER($1) AND id != $2`
	}

	var existingCount int
	err = s.DB.QueryRowContext(ctx, collisionQuery, cleanEmail, accountID).Scan(&existingCount)
	if err != nil {
		return fmt.Errorf("failed checking email uniqueness: %w", err)
	}
	if existingCount > 0 {
		return fmt.Errorf("email '%s' is already registered to another %s account", cleanEmail, cleanRole)
	}

	// Update account
	var updateQuery string
	if cleanRole == "devotee" {
		updateQuery = `UPDATE users SET email = $1, email_verified = $2 WHERE id = $3`
	} else {
		updateQuery = `UPDATE priests SET email = $1, email_verified = $2 WHERE id = $3`
	}

	res, err := s.DB.ExecContext(ctx, updateQuery, cleanEmail, verified, accountID)
	if err != nil {
		return fmt.Errorf("failed updating account email: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("%s account not found", cleanRole)
	}

	return nil
}

// RequestEmailVerification generates a single-use token and sends a verification email for authenticated users.
// Invariant: The existing verified email is PRESERVED on the account until the new email is confirmed.
// Concurrency: Serializes per-account via PostgreSQL transaction advisory locks to guarantee that concurrent
// requests cannot leave multiple active links or allow an older request to replace a newer one.
func (s *Storage) RequestEmailVerification(ctx context.Context, role, accountID, email, secret string) (string, error) {
	cleanRole := strings.ToLower(strings.TrimSpace(role))
	if cleanRole != "devotee" && cleanRole != "priest" {
		return "", fmt.Errorf("invalid role: must be 'devotee' or 'priest'")
	}

	cleanEmail, err := auth.ValidateEmailFormat(email)
	if err != nil {
		return "", err
	}

	// Strictly require email provider to be configured
	if !auth.GetEmailProvider().IsConfigured() {
		return "", auth.ErrEmailUnavailable
	}

	rawToken, err := auth.GenerateResetToken()
	if err != nil {
		return "", fmt.Errorf("failed generating verification token: %w", err)
	}

	tokenHash := auth.HashResetToken(rawToken, secret)

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed starting verification transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Serialize per-account: Acquire transaction advisory lock scoped to this account
	lockID := auth.AccountLockID(cleanRole, accountID)
	_, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", lockID)
	if err != nil {
		return "", fmt.Errorf("failed acquiring account verification lock: %w", err)
	}

	// 2. Lock and retrieve account phone
	var phone string
	if cleanRole == "devotee" {
		err = tx.QueryRowContext(ctx, "SELECT phone FROM users WHERE id = $1 FOR UPDATE", accountID).Scan(&phone)
	} else {
		err = tx.QueryRowContext(ctx, "SELECT phone FROM priests WHERE id = $1 FOR UPDATE", accountID).Scan(&phone)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("account not found")
		}
		return "", fmt.Errorf("failed querying account: %w", err)
	}

	// 3. Check if this email is already registered to another account of the same role
	var collisionQuery string
	if cleanRole == "devotee" {
		collisionQuery = `SELECT COUNT(*) FROM users WHERE LOWER(email) = LOWER($1) AND id != $2`
	} else {
		collisionQuery = `SELECT COUNT(*) FROM priests WHERE LOWER(email) = LOWER($1) AND id != $2`
	}

	var existingCount int
	err = tx.QueryRowContext(ctx, collisionQuery, cleanEmail, accountID).Scan(&existingCount)
	if err != nil {
		return "", fmt.Errorf("failed checking email uniqueness: %w", err)
	}
	if existingCount > 0 {
		return "", fmt.Errorf("email '%s' is already registered to another %s account", cleanEmail, cleanRole)
	}

	// 4. Invalidate and supersede any existing ACTIVE email verification challenges for this account
	_, err = tx.ExecContext(ctx, `
		UPDATE password_recovery_challenges
		SET status = 'SUPERSEDED'
		WHERE account_role = $1 AND account_id = $2 AND purpose = 'email_verification' AND status = 'ACTIVE'
	`, cleanRole, accountID)
	if err != nil {
		return "", fmt.Errorf("failed superseding previous verification challenges: %w", err)
	}

	var challengeID string
	insertQuery := `
		INSERT INTO password_recovery_challenges
			(account_id, account_role, phone, email, purpose, code_hash, expires_at, status, delivery_status)
		VALUES
			($1, $2, $3, $4, 'email_verification', $5, CURRENT_TIMESTAMP + INTERVAL '24 hours', 'ACTIVE', 'PENDING')
		RETURNING id
	`
	err = tx.QueryRowContext(ctx, insertQuery, accountID, cleanRole, phone, cleanEmail, tokenHash).Scan(&challengeID)
	if err != nil {
		return "", fmt.Errorf("failed creating verification challenge: %w", err)
	}

	// Resolve trusted verification URL (never from Host header)
	cfg, _ := auth.LoadSMTPConfigFromEnv()
	var baseURL string
	if cfg != nil && cfg.VerificationBaseURL != "" {
		baseURL = cfg.VerificationBaseURL
	} else {
		baseURL = strings.TrimSpace(os.Getenv("EMAIL_VERIFICATION_BASE_URL"))
		if baseURL == "" {
			resetBase := strings.TrimSpace(os.Getenv("PASSWORD_RESET_BASE_URL"))
			if resetBase != "" {
				u, err := url.Parse(resetBase)
				if err == nil {
					u.Path = "/api/v1/auth/verify-email"
					u.RawQuery = ""
					baseURL = u.String()
				}
			}
		}
	}
	if baseURL == "" {
		baseURL = "http://localhost:8085/api/v1/auth/verify-email"
	}

	verifyURL, err := auth.BuildEmailVerificationURL(baseURL, rawToken)
	if err != nil {
		return "", fmt.Errorf("failed constructing verification URL: %w", err)
	}

	// Dispatch email via provider
	if err := auth.GetEmailProvider().SendVerificationEmail(ctx, cleanEmail, verifyURL); err != nil {
		_, _ = tx.ExecContext(ctx, "UPDATE password_recovery_challenges SET status = 'FAILED', delivery_status = 'FAILED' WHERE id = $1", challengeID)
		return "", fmt.Errorf("%w: %v", auth.ErrDeliveryFailed, err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE password_recovery_challenges SET delivery_status = 'SENT' WHERE id = $1", challengeID)
	if err != nil {
		return "", fmt.Errorf("failed updating challenge delivery status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed committing verification challenge: %w", err)
	}

	return rawToken, nil
}

// ConfirmEmailVerification validates an email verification token, marks the challenge CONSUMED,
// and updates the account's email and email_verified status atomically.
// Concurrency: Serializes per-account via PostgreSQL transaction advisory locks to guarantee that concurrent
// confirmations cannot allow an older request to replace a newer one or double-consume tokens.
func (s *Storage) ConfirmEmailVerification(ctx context.Context, token, secret string) (string, string, error) {
	rawToken := strings.TrimSpace(token)
	if rawToken == "" {
		return "", "", errors.New("verification token is required")
	}

	tokenHash := auth.HashResetToken(rawToken, secret)

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback()

	// 1. Plain read to resolve account identity without row locks (preventing lock-inversion deadlock)
	var (
		accountID string
		role      string
	)
	lookupQuery := `
		SELECT account_id, account_role
		FROM password_recovery_challenges
		WHERE code_hash = $1 AND purpose = 'email_verification'
	`
	err = tx.QueryRowContext(ctx, lookupQuery, tokenHash).Scan(&accountID, &role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", errors.New("invalid or expired email verification token")
		}
		return "", "", err
	}

	// 2. Serialize per-account: Acquire transaction advisory lock scoped to this account
	lockID := auth.AccountLockID(role, accountID)
	_, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", lockID)
	if err != nil {
		return "", "", fmt.Errorf("failed acquiring account verification lock: %w", err)
	}

	// 3. Lock challenge row FOR UPDATE and inspect latest committed status
	var (
		challengeID string
		email       string
		status      string
		expiresAt   time.Time
	)
	query := `
		SELECT id, COALESCE(email, ''), status, expires_at
		FROM password_recovery_challenges
		WHERE code_hash = $1 AND purpose = 'email_verification'
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, query, tokenHash).Scan(
		&challengeID, &email, &status, &expiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", errors.New("invalid or expired email verification token")
		}
		return "", "", err
	}

	if status == "CONSUMED" {
		return "", "", errors.New("this email verification link has already been used")
	}
	if status == "SUPERSEDED" {
		return "", "", errors.New("this verification link is superseded and no longer valid because a newer verification email was requested")
	}
	if status != "ACTIVE" {
		return "", "", errors.New("this email verification link is no longer valid")
	}
	if time.Now().After(expiresAt) {
		_, _ = tx.ExecContext(ctx, "UPDATE password_recovery_challenges SET status = 'EXPIRED' WHERE id = $1", challengeID)
		_ = tx.Commit()
		return "", "", errors.New("this email verification link has expired")
	}

	// 4. Re-verify no other account has claimed this email in the interim under account lock
	var collisionQuery string
	if role == "devotee" {
		collisionQuery = `SELECT COUNT(*) FROM users WHERE LOWER(email) = LOWER($1) AND id != $2`
	} else {
		collisionQuery = `SELECT COUNT(*) FROM priests WHERE LOWER(email) = LOWER($1) AND id != $2`
	}
	var collisionCount int
	err = tx.QueryRowContext(ctx, collisionQuery, email, accountID).Scan(&collisionCount)
	if err != nil {
		return "", "", fmt.Errorf("failed checking email conflict: %w", err)
	}
	if collisionCount > 0 {
		return "", "", fmt.Errorf("email '%s' has already been registered to another %s account", email, role)
	}

	// Mark challenge CONSUMED
	_, err = tx.ExecContext(ctx, "UPDATE password_recovery_challenges SET status = 'CONSUMED', consumed_at = CURRENT_TIMESTAMP WHERE id = $1", challengeID)
	if err != nil {
		return "", "", err
	}

	// Supersede any remaining email_verification challenges for this account
	_, err = tx.ExecContext(ctx, `
		UPDATE password_recovery_challenges
		SET status = 'SUPERSEDED'
		WHERE account_role = $1 AND account_id = $2 AND id != $3 AND purpose = 'email_verification' AND status = 'ACTIVE'
	`, role, accountID, challengeID)
	if err != nil {
		return "", "", fmt.Errorf("failed superseding remaining challenges: %w", err)
	}

	// Atomically set email and mark verified on account
	if role == "devotee" {
		_, err = tx.ExecContext(ctx, "UPDATE users SET email = $1, email_verified = TRUE WHERE id = $2", email, accountID)
	} else {
		_, err = tx.ExecContext(ctx, "UPDATE priests SET email = $1, email_verified = TRUE WHERE id = $2", email, accountID)
	}
	if err != nil {
		return "", "", fmt.Errorf("failed updating verified email on account: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", "", fmt.Errorf("failed committing email verification: %w", err)
	}

	return email, role, nil
}

// GetAccountEmail retrieves the registered email and verification status for an account.
func (s *Storage) GetAccountEmail(ctx context.Context, role, accountID string) (string, bool, error) {
	var email sql.NullString
	var isVerified bool

	var query string
	if role == "devotee" {
		query = `SELECT email, email_verified FROM users WHERE id = $1`
	} else {
		query = `SELECT email, email_verified FROM priests WHERE id = $1`
	}

	err := s.DB.QueryRowContext(ctx, query, accountID).Scan(&email, &isVerified)
	if err != nil {
		return "", false, err
	}

	return email.String, isVerified, nil
}
