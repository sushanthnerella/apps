package db

import (
	"context"
	"fmt"
)

// RegisterDeviceToken stores or refreshes an FCM device token for an authenticated user.
func (s *Storage) RegisterDeviceToken(ctx context.Context, userID, deviceToken, platform string) error {
	if platform == "" {
		platform = "android"
	}
	query := `
		INSERT INTO device_tokens (user_id, device_token, platform, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id, device_token) DO UPDATE SET
			platform = EXCLUDED.platform,
			updated_at = NOW()
	`
	_, err := s.DB.ExecContext(ctx, query, userID, deviceToken, platform)
	if err != nil {
		return fmt.Errorf("failed to register device token: %w", err)
	}
	return nil
}

// GetDeviceTokensForUser retrieves all active device tokens registered to a user.
func (s *Storage) GetDeviceTokensForUser(ctx context.Context, userID string) ([]string, error) {
	query := `SELECT device_token FROM device_tokens WHERE user_id = $1::uuid ORDER BY updated_at DESC`
	rows, err := s.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query device tokens: %w", err)
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var tok string
		if err := rows.Scan(&tok); err == nil && tok != "" {
			tokens = append(tokens, tok)
		}
	}
	return tokens, nil
}
