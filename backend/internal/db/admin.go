package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/models"
)

// CreateAdminUser inserts a new administrator account with a bcrypt-hashed password.
func (s *Storage) CreateAdminUser(email, password string) (*models.AdminUser, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, errors.New("valid email is required")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	var admin models.AdminUser
	query := `
		INSERT INTO admin_users (email, password_hash, role, session_version, created_at)
		VALUES ($1, $2, 'admin', 1, NOW())
		RETURNING id, email, role, session_version, created_at
	`
	err = s.DB.QueryRow(query, email, hash).Scan(
		&admin.ID, &admin.Email, &admin.Role, &admin.SessionVersion, &admin.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin: %w", err)
	}
	return &admin, nil
}

// LoginAdmin verifies email, password, and session status for administrator accounts.
func (s *Storage) LoginAdmin(email, password string) (*models.AuthResponse, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, errors.New("email is required")
	}
	if password == "" {
		return nil, errors.New("password is required")
	}

	var id, storedHash, role string
	var sessionVersion int
	var disabledAt *time.Time

	query := `SELECT id, password_hash, role, session_version, disabled_at FROM admin_users WHERE email = $1`
	err := s.DB.QueryRow(query, email).Scan(&id, &storedHash, &role, &sessionVersion, &disabledAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			auth.EqualizeTiming(password)
			return nil, auth.ErrInvalidCredentials
		}
		return nil, err
	}

	if disabledAt != nil {
		auth.EqualizeTiming(password)
		return nil, errors.New("account disabled")
	}

	matched, _ := auth.VerifyPassword(storedHash, password)
	if !matched {
		return nil, auth.ErrInvalidCredentials
	}

	token, err := auth.GenerateToken(id, email, "admin", "Admin", sessionVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.AuthResponse{
		Status:      "SUCCESS",
		AccessToken: token,
		TokenType:   "Bearer",
		User: models.UserProfile{
			ID:       id,
			Phone:    email,
			FullName: "Admin",
			Role:     "admin",
		},
		Message: "Admin authenticated successfully",
	}, nil
}
