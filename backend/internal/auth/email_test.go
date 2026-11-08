package auth

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateConfigAndHTTPSRequirement(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *SMTPConfig
		wantError bool
		errMsg    string
	}{
		{
			name: "Valid HTTPS configuration",
			cfg: &SMTPConfig{
				Host:         "smtp.example.com",
				Port:         587,
				From:         "support@pujavaahini.com",
				ResetBaseURL: "https://pujavaahini.com/reset-password",
				RequireTLS:   true,
			},
			wantError: false,
		},
		{
			name: "Valid localhost HTTP configuration for local development",
			cfg: &SMTPConfig{
				Host:         "localhost",
				Port:         1025,
				From:         "dev@localhost",
				ResetBaseURL: "http://localhost:8085/api/v1/auth/recovery/reset",
				RequireTLS:   true,
			},
			wantError: false,
		},
		{
			name: "Reject non-HTTPS remote URL",
			cfg: &SMTPConfig{
				Host:         "smtp.example.com",
				Port:         587,
				From:         "support@pujavaahini.com",
				ResetBaseURL: "http://pujavaahini.com/reset-password",
				RequireTLS:   true,
			},
			wantError: true,
			errMsg:    "public URL scheme must be HTTPS",
		},
		{
			name: "Reject invalid email in From",
			cfg: &SMTPConfig{
				Host:         "smtp.example.com",
				Port:         587,
				From:         "not-an-email",
				ResetBaseURL: "https://pujavaahini.com/reset-password",
				RequireTLS:   true,
			},
			wantError: true,
			errMsg:    "invalid SMTP_FROM",
		},
		{
			name: "Reject empty host",
			cfg: &SMTPConfig{
				Host:         "",
				Port:         587,
				From:         "support@pujavaahini.com",
				ResetBaseURL: "https://pujavaahini.com/reset-password",
				RequireTLS:   true,
			},
			wantError: true,
			errMsg:    "SMTP_HOST cannot be empty",
		},
		{
			name: "Reject invalid port",
			cfg: &SMTPConfig{
				Host:         "smtp.example.com",
				Port:         70000,
				From:         "support@pujavaahini.com",
				ResetBaseURL: "https://pujavaahini.com/reset-password",
				RequireTLS:   true,
			},
			wantError: true,
			errMsg:    "invalid SMTP_PORT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.cfg)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error containing '%s', got nil", tt.errMsg)
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Fatalf("expected error containing '%s', got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBuildPasswordResetURLNeverUsesHostHeader(t *testing.T) {
	baseURL := "https://pujavaahini.com/auth/reset?existing=1"
	token := "4a5b6c7d8e9f00112233445566778899aabbccddeeff"

	resetURL, err := BuildPasswordResetURL(baseURL, token)
	if err != nil {
		t.Fatalf("failed building reset URL: %v", err)
	}

	if !strings.HasPrefix(resetURL, "https://pujavaahini.com/auth/reset") {
		t.Errorf("expected reset URL to start with base URL, got: %s", resetURL)
	}
	if !strings.Contains(resetURL, "token="+token) {
		t.Errorf("expected token parameter in URL, got: %s", resetURL)
	}
	if !strings.Contains(resetURL, "existing=1") {
		t.Errorf("expected existing query parameter preserved, got: %s", resetURL)
	}
}

func TestResetTokenGenerationAndVerification(t *testing.T) {
	secret := "super_secret_test_signing_key_32bytes_long!"

	token1, err := GenerateResetToken()
	if err != nil {
		t.Fatalf("GenerateResetToken failed: %v", err)
	}
	if len(token1) != 64 {
		t.Fatalf("expected 64-char hex token (32 bytes), got %d chars: %s", len(token1), token1)
	}

	token2, err := GenerateResetToken()
	if err != nil {
		t.Fatalf("GenerateResetToken failed: %v", err)
	}
	if token1 == token2 {
		t.Fatalf("expected distinct random tokens, got identical tokens")
	}

	hash1 := HashResetToken(token1, secret)
	if len(hash1) != 64 {
		t.Fatalf("expected 64-char HMAC hash, got: %s", hash1)
	}

	// Correct token verifies
	if !VerifyResetToken(token1, hash1, secret) {
		t.Fatalf("expected VerifyResetToken to pass for correct token")
	}

	// Wrong token fails
	if VerifyResetToken(token2, hash1, secret) {
		t.Fatalf("expected VerifyResetToken to fail for wrong token")
	}

	// Wrong secret fails
	if VerifyResetToken(token1, hash1, "wrong_secret_key_32bytes_long_1234") {
		t.Fatalf("expected VerifyResetToken to fail for wrong secret")
	}
}

func TestLoadLocalEnvDoesNotOverrideDeploymentEnvironment(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	// Pre-set existing deployment env var
	_ = os.Setenv("DEPLOYMENT_ENV_VAR", "original_deployment_value")
	defer os.Unsetenv("DEPLOYMENT_ENV_VAR")
	defer os.Unsetenv("LOCAL_ONLY_VAR")

	envContent := `
DEPLOYMENT_ENV_VAR=overridden_value
LOCAL_ONLY_VAR=loaded_from_file
`
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("failed writing test env: %v", err)
	}

	if err := LoadLocalEnv(envPath); err != nil {
		t.Fatalf("LoadLocalEnv failed: %v", err)
	}

	// Verify original deployment value preserved
	if val := os.Getenv("DEPLOYMENT_ENV_VAR"); val != "original_deployment_value" {
		t.Fatalf("expected deployment value 'original_deployment_value' preserved, got '%s'", val)
	}

	// Verify un-set variable was loaded
	if val := os.Getenv("LOCAL_ONLY_VAR"); val != "loaded_from_file" {
		t.Fatalf("expected 'loaded_from_file', got '%s'", val)
	}
}

func TestValidateEmailFormat(t *testing.T) {
	valid := []string{
		"user@example.com",
		"devotee.sharma+pooja@pujavaahini.org",
		"priest@temple.co.in",
	}
	for _, email := range valid {
		normalized, err := ValidateEmailFormat(email)
		if err != nil {
			t.Errorf("expected valid email for '%s', got error: %v", email, err)
		}
		if normalized != strings.ToLower(strings.TrimSpace(email)) {
			t.Errorf("expected normalized lowercase, got %s", normalized)
		}
	}

	invalid := []string{
		"",
		"   ",
		"notanemail",
		"missingat.com",
		"@missinguser.com",
		"user@domain",
	}
	for _, email := range invalid {
		_, err := ValidateEmailFormat(email)
		if err == nil {
			t.Errorf("expected error for invalid email '%s', got nil", email)
		}
	}
}

func TestUnconfiguredEmailProviderRejects(t *testing.T) {
	p := &UnconfiguredEmailProvider{}
	if p.IsConfigured() {
		t.Fatalf("expected UnconfiguredEmailProvider to return false for IsConfigured")
	}

	err := p.SendPasswordResetEmail(context.Background(), "user@example.com", "https://pujavaahini.com/reset")
	if err == nil || !strings.Contains(err.Error(), "email_service_unavailable") {
		t.Fatalf("expected email_service_unavailable error, got: %v", err)
	}
}

func TestMockEmailProviderRecordsMessages(t *testing.T) {
	mock := &MockEmailProvider{Configured: true}
	ctx := context.Background()

	err := mock.SendPasswordResetEmail(ctx, "devotee@example.com", "https://pujavaahini.com/reset?token=abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.SentMessages) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(mock.SentMessages))
	}
	if mock.LastSentResetURL("devotee@example.com") != "https://pujavaahini.com/reset?token=abc" {
		t.Fatalf("LastSentResetURL mismatch")
	}
}

func TestSMTPAddressFormatting(t *testing.T) {
	tests := []struct {
		host     string
		port     int
		expected string
	}{
		{"smtp.example.com", 587, "smtp.example.com:587"},
		{"127.0.0.1", 25, "127.0.0.1:25"},
		{"::1", 587, "[::1]:587"},
		{"2001:db8::1", 465, "[2001:db8::1]:465"},
	}

	for _, tt := range tests {
		addr := FormatSMTPAddress(tt.host, tt.port)
		if addr != tt.expected {
			t.Errorf("FormatSMTPAddress(%q, %d) = %q; want %q", tt.host, tt.port, addr, tt.expected)
		}
	}
}

