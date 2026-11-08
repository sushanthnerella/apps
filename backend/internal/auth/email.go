package auth

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrEmailUnavailable    = errors.New("email_service_unavailable: email delivery service is currently unconfigured")
	ErrInvalidEmail         = errors.New("invalid_email: invalid email address format")
	ErrInvalidResetBaseURL  = errors.New("invalid_reset_base_url: PASSWORD_RESET_BASE_URL must be a valid absolute HTTPS URL")
	ErrInvalidVerifyBaseURL = errors.New("invalid_verify_base_url: EMAIL_VERIFICATION_BASE_URL must be a valid absolute HTTPS URL")
	ErrTLSRequired          = errors.New("tls_required: SMTP delivery requires TLS encryption")
	ErrDeliveryFailed       = errors.New("email_delivery_failed: failed to send email via SMTP provider")
)

// SMTPConfig holds validated email delivery configuration.
type SMTPConfig struct {
	Host                string
	Port                int
	User                string
	Password            string
	From                string
	ResetBaseURL        string
	VerificationBaseURL string
	RequireTLS          bool
}

// LoadLocalEnv loads variables from .env files into the environment without
// overriding any variables already set in os.Environ().
func LoadLocalEnv(paths ...string) error {
	candidatePaths := paths
	if len(candidatePaths) == 0 {
		candidatePaths = []string{".env", "backend/.env", "../.env"}
	}

	for _, p := range candidatePaths {
		f, err := os.Open(p)
		if err != nil {
			continue // try next candidate if not found
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			// Strip optional quotes
			if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
				val = val[1 : len(val)-1]
			}

			// Do not override deployment environment values
			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, val)
			}
		}
		break // loaded first found
	}
	return nil
}

// LoadSMTPConfigFromEnv reads and parses SMTP configuration from environment variables.
func LoadSMTPConfigFromEnv() (*SMTPConfig, error) {
	_ = LoadLocalEnv()

	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	portStr := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	user := strings.TrimSpace(os.Getenv("SMTP_USER"))
	password := strings.TrimSpace(os.Getenv("SMTP_PASSWORD"))
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	resetBaseURL := strings.TrimSpace(os.Getenv("PASSWORD_RESET_BASE_URL"))
	verifyBaseURL := strings.TrimSpace(os.Getenv("EMAIL_VERIFICATION_BASE_URL"))

	if host == "" || from == "" || resetBaseURL == "" {
		// Considered unconfigured
		return nil, nil
	}

	if verifyBaseURL == "" {
		// Derive from resetBaseURL by routing to email verification endpoint
		u, err := url.Parse(resetBaseURL)
		if err == nil && u.IsAbs() {
			if strings.HasSuffix(u.Path, "/recovery/reset") {
				u.Path = strings.TrimSuffix(u.Path, "/recovery/reset") + "/verify-email"
			} else if strings.HasSuffix(u.Path, "/reset-password") {
				u.Path = strings.TrimSuffix(u.Path, "/reset-password") + "/verify-email"
			} else if strings.HasSuffix(u.Path, "/reset") {
				u.Path = strings.TrimSuffix(u.Path, "/reset") + "/verify-email"
			} else {
				u.Path = "/api/v1/auth/verify-email"
			}
			u.RawQuery = ""
			verifyBaseURL = u.String()
		}
	}

	port := 587
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil || p <= 0 || p > 65535 {
			return nil, fmt.Errorf("invalid SMTP_PORT '%s': must be a valid port number between 1 and 65535", portStr)
		}
		port = p
	}

	cfg := &SMTPConfig{
		Host:                host,
		Port:                port,
		User:                user,
		Password:            password,
		From:                from,
		ResetBaseURL:        resetBaseURL,
		VerificationBaseURL: verifyBaseURL,
		RequireTLS:          true,
	}

	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// ValidateConfig verifies that SMTP configuration meets strict security invariants.
func ValidateConfig(cfg *SMTPConfig) error {
	if cfg == nil {
		return errors.New("nil SMTP configuration")
	}

	if strings.TrimSpace(cfg.Host) == "" {
		return errors.New("SMTP_HOST cannot be empty")
	}

	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid SMTP_PORT %d", cfg.Port)
	}

	if _, err := mail.ParseAddress(cfg.From); err != nil {
		return fmt.Errorf("invalid SMTP_FROM email address '%s': %w", cfg.From, err)
	}

	u, err := url.Parse(cfg.ResetBaseURL)
	if err != nil || !u.IsAbs() {
		return ErrInvalidResetBaseURL
	}

	isLocal := strings.EqualFold(u.Hostname(), "localhost") || u.Hostname() == "127.0.0.1" || strings.HasPrefix(u.Hostname(), "127.")
	if u.Scheme != "https" && !isLocal {
		return fmt.Errorf("%w: public URL scheme must be HTTPS (got %s)", ErrInvalidResetBaseURL, u.Scheme)
	}

	if cfg.VerificationBaseURL != "" {
		uv, err := url.Parse(cfg.VerificationBaseURL)
		if err != nil || !uv.IsAbs() {
			return ErrInvalidVerifyBaseURL
		}
		isLocalV := strings.EqualFold(uv.Hostname(), "localhost") || uv.Hostname() == "127.0.0.1" || strings.HasPrefix(uv.Hostname(), "127.")
		if uv.Scheme != "https" && !isLocalV {
			return fmt.Errorf("%w: public URL scheme must be HTTPS (got %s)", ErrInvalidVerifyBaseURL, uv.Scheme)
		}
	}

	return nil
}

// BuildPasswordResetURL constructs a reset URL from the configured base URL and raw token,
// never deriving it from any untrusted HTTP Host header.
func BuildPasswordResetURL(baseURL, rawToken string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil || !u.IsAbs() {
		return "", ErrInvalidResetBaseURL
	}

	isLocal := strings.EqualFold(u.Hostname(), "localhost") || u.Hostname() == "127.0.0.1" || strings.HasPrefix(u.Hostname(), "127.")
	if u.Scheme != "https" && !isLocal {
		return "", fmt.Errorf("%w: public URL scheme must be HTTPS (got %s)", ErrInvalidResetBaseURL, u.Scheme)
	}

	q := u.Query()
	q.Set("token", strings.TrimSpace(rawToken))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// BuildEmailVerificationURL constructs an email verification URL from the configured trusted base URL
// and raw cryptographic token, never deriving it from any untrusted HTTP Host header.
func BuildEmailVerificationURL(baseURL, rawToken string) (string, error) {
	cleanBase := strings.TrimSpace(baseURL)
	if cleanBase == "" {
		cleanBase = "http://localhost:8085/api/v1/auth/verify-email"
	}

	u, err := url.Parse(cleanBase)
	if err != nil || !u.IsAbs() {
		return "", ErrInvalidVerifyBaseURL
	}

	isLocal := strings.EqualFold(u.Hostname(), "localhost") || u.Hostname() == "127.0.0.1" || strings.HasPrefix(u.Hostname(), "127.")
	if u.Scheme != "https" && !isLocal {
		return "", fmt.Errorf("%w: public URL scheme must be HTTPS (got %s)", ErrInvalidVerifyBaseURL, u.Scheme)
	}

	q := u.Query()
	q.Set("token", strings.TrimSpace(rawToken))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// GenerateResetToken creates a cryptographically secure 32-byte URL-safe hex reset token.
func GenerateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed generating cryptographic reset token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// HashResetToken computes an HMAC-SHA256 keyed digest of the reset token using the server secret.
func HashResetToken(token, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyResetToken compares a candidate token against a stored digest in constant time.
func VerifyResetToken(token, storedHash, secret string) bool {
	candidateHash := HashResetToken(token, secret)
	return subtle.ConstantTimeCompare([]byte(candidateHash), []byte(storedHash)) == 1
}

// BuildEmailThrottleKey generates a canonical role-scoped email throttle key.
func BuildEmailThrottleKey(role, email string) string {
	cleanRole := strings.ToLower(strings.TrimSpace(role))
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	return fmt.Sprintf("role:%s:email:%s", cleanRole, cleanEmail)
}

// ValidateEmailFormat validates that the email string is a well-formed address.
func ValidateEmailFormat(email string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(email))
	if trimmed == "" {
		return "", fmt.Errorf("%w: email address cannot be empty", ErrInvalidEmail)
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil || addr.Address != trimmed || !strings.Contains(trimmed, "@") || !strings.Contains(trimmed, ".") {
		return "", fmt.Errorf("%w: '%s'", ErrInvalidEmail, email)
	}
	return trimmed, nil
}

// EmailProvider defines the interface for delivering transactional emails.
type EmailProvider interface {
	SendPasswordResetEmail(ctx context.Context, toEmail, resetURL string) error
	SendVerificationEmail(ctx context.Context, toEmail, verifyURL string) error
	IsConfigured() bool
}

// UnconfiguredEmailProvider safely rejects delivery requests when SMTP is unconfigured.
type UnconfiguredEmailProvider struct{}

func (u *UnconfiguredEmailProvider) SendPasswordResetEmail(ctx context.Context, toEmail, resetURL string) error {
	return ErrEmailUnavailable
}

func (u *UnconfiguredEmailProvider) SendVerificationEmail(ctx context.Context, toEmail, verifyURL string) error {
	return ErrEmailUnavailable
}

func (u *UnconfiguredEmailProvider) IsConfigured() bool {
	return false
}

// SMTPEmailProvider delivers emails via SMTP enforcing TLS encryption.
type SMTPEmailProvider struct {
	config *SMTPConfig
}

// NewSMTPEmailProvider creates an active SMTP email provider.
func NewSMTPEmailProvider(cfg *SMTPConfig) (*SMTPEmailProvider, error) {
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}
	return &SMTPEmailProvider{config: cfg}, nil
}

func (p *SMTPEmailProvider) IsConfigured() bool {
	return p.config != nil && p.config.Host != ""
}

func (p *SMTPEmailProvider) SendPasswordResetEmail(ctx context.Context, toEmail, resetURL string) error {
	cleanTo, err := ValidateEmailFormat(toEmail)
	if err != nil {
		return err
	}

	subject := "Puja Vaahini — Password Reset Request"
	plainBody := fmt.Sprintf(
		"Hello,\r\n\r\nWe received a request to reset your password for Puja Vaahini.\r\n\r\n"+
			"Please use the following single-use link to reset your password (valid for 15 minutes):\r\n%s\r\n\r\n"+
			"If you did not request this password reset, please ignore this email. Your account remains secure.\r\n\r\n"+
			"— The Puja Vaahini Team",
		resetURL,
	)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>Password Reset</title></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f8fafc; margin: 0; padding: 24px; color: #1e293b;">
  <div style="max-width: 540px; margin: 0 auto; background: #ffffff; border-radius: 12px; border: 1px solid #e2e8f0; padding: 32px; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.05);">
    <div style="text-align: center; margin-bottom: 24px;">
      <h2 style="color: #ea580c; margin: 0; font-size: 24px;">🕉️ Puja Vaahini</h2>
      <p style="color: #64748b; font-size: 14px; margin: 4px 0 0 0;">Vedic Pujas &amp; Purohit Services</p>
    </div>
    <h3 style="font-size: 18px; color: #0f172a; margin-top: 0;">Password Reset Request</h3>
    <p style="font-size: 14px; line-height: 1.6; color: #334155;">
      We received a request to reset the password for your verified account. Click the button below to set a new password:
    </p>
    <div style="text-align: center; margin: 28px 0;">
      <a href="%s" style="background-color: #ea580c; color: #ffffff; text-decoration: none; padding: 12px 24px; border-radius: 8px; font-weight: 600; font-size: 15px; display: inline-block;">Reset Password</a>
    </div>
    <p style="font-size: 13px; color: #64748b; line-height: 1.5;">
      ⚠️ <strong>Notice:</strong> This link is single-use and will expire in <strong>15 minutes</strong>. If you did not make this request, you can safely ignore this email; your existing password will remain active.
    </p>
    <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
    <p style="font-size: 11px; color: #94a3b8; text-align: center; margin: 0;">
      Puja Vaahini Platform &bull; Hyderabad Pilot Service &bull; Do not reply to this email
    </p>
  </div>
</body>
</html>`, resetURL)

	return p.sendMail(cleanTo, subject, plainBody, htmlBody)
}

func (p *SMTPEmailProvider) SendVerificationEmail(ctx context.Context, toEmail, verifyURL string) error {
	cleanTo, err := ValidateEmailFormat(toEmail)
	if err != nil {
		return err
	}

	subject := "Puja Vaahini — Verify Your Email Address"
	plainBody := fmt.Sprintf(
		"Hello,\r\n\r\nPlease verify your email address for your Puja Vaahini account by visiting this single-use link:\r\n%s\r\n\r\n"+
			"This link is valid for 24 hours. Once verified, this email will be linked to your account for official notifications and password recovery.\r\n\r\n"+
			"If you did not request this verification, you can safely ignore this email.\r\n\r\n"+
			"— The Puja Vaahini Team",
		verifyURL,
	)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>Verify Email Address</title></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f8fafc; margin: 0; padding: 24px; color: #1e293b;">
  <div style="max-width: 540px; margin: 0 auto; background: #ffffff; border-radius: 12px; border: 1px solid #e2e8f0; padding: 32px; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.05);">
    <div style="text-align: center; margin-bottom: 24px;">
      <h2 style="color: #ea580c; margin: 0; font-size: 24px;">🕉️ Puja Vaahini</h2>
      <p style="color: #64748b; font-size: 14px; margin: 4px 0 0 0;">Vedic Pujas &amp; Purohit Services</p>
    </div>
    <h3 style="font-size: 18px; color: #0f172a; margin-top: 0;">Verify Your Email Address</h3>
    <p style="font-size: 14px; line-height: 1.6; color: #334155;">
      Please confirm this email address for your Puja Vaahini account. Once verified, this email can be used for secure password recovery and booking confirmations.
    </p>
    <div style="text-align: center; margin: 28px 0;">
      <a href="%s" style="background-color: #ea580c; color: #ffffff; text-decoration: none; padding: 12px 24px; border-radius: 8px; font-weight: 600; font-size: 15px; display: inline-block;">Verify Email Address</a>
    </div>
    <p style="font-size: 13px; color: #64748b; line-height: 1.5;">
      ⚠️ <strong>Notice:</strong> This verification link is single-use and will expire in <strong>24 hours</strong>. If you did not make this request from your Puja Vaahini profile, you can safely ignore this message.
    </p>
    <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
    <p style="font-size: 11px; color: #94a3b8; text-align: center; margin: 0;">
      Puja Vaahini Platform &bull; Hyderabad Pilot Service &bull; Do not reply to this email
    </p>
  </div>
 </body>
</html>`, verifyURL)

	return p.sendMail(cleanTo, subject, plainBody, htmlBody)
}

func (p *SMTPEmailProvider) sendMail(to, subject, plainBody, htmlBody string) error {
	boundary := fmt.Sprintf("pv_boundary_%d", time.Now().UnixNano())

	msg := strings.Builder{}
	msg.WriteString(fmt.Sprintf("From: %s\r\n", p.config.From))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
	msg.WriteString("\r\n")

	// Plaintext section
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	msg.WriteString(plainBody)
	msg.WriteString("\r\n\r\n")

	// HTML section
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	msg.WriteString(htmlBody)
	msg.WriteString("\r\n\r\n")

	msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	addr := FormatSMTPAddress(p.config.Host, p.config.Port)

	// Port 465 uses direct TLS
	if p.config.Port == 465 {
		tlsConfig := &tls.Config{
			ServerName: p.config.Host,
		}
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("%w: failed to connect via SMTPS: %v", ErrDeliveryFailed, err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, p.config.Host)
		if err != nil {
			return fmt.Errorf("%w: failed to create SMTP client: %v", ErrDeliveryFailed, err)
		}
		defer client.Close()

		if p.config.User != "" && p.config.Password != "" {
			auth := smtp.PlainAuth("", p.config.User, p.config.Password, p.config.Host)
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("%w: SMTP authentication failed: %v", ErrDeliveryFailed, err)
			}
		}

		if err := client.Mail(p.config.From); err != nil {
			return fmt.Errorf("%w: MAIL FROM failed: %v", ErrDeliveryFailed, err)
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("%w: RCPT TO failed: %v", ErrDeliveryFailed, err)
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("%w: DATA failed: %v", ErrDeliveryFailed, err)
		}
		_, err = w.Write([]byte(msg.String()))
		if err != nil {
			_ = w.Close()
			return fmt.Errorf("%w: write message failed: %v", ErrDeliveryFailed, err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("%w: close message failed: %v", ErrDeliveryFailed, err)
		}
		return client.Quit()
	}

	// Port 587 (and others) connects plain TCP and requires STARTTLS
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("%w: failed to connect to SMTP server: %v", ErrDeliveryFailed, err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, p.config.Host)
	if err != nil {
		return fmt.Errorf("%w: failed to initialize SMTP client: %v", ErrDeliveryFailed, err)
	}
	defer client.Close()

	// Strict TLS requirement: STARTTLS must be offered
	hasStartTLS, _ := client.Extension("STARTTLS")
	if !hasStartTLS {
		return ErrTLSRequired
	}

	tlsConfig := &tls.Config{
		ServerName: p.config.Host,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("%w: STARTTLS failed: %v", ErrTLSRequired, err)
	}

	if p.config.User != "" && p.config.Password != "" {
		auth := smtp.PlainAuth("", p.config.User, p.config.Password, p.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("%w: SMTP authentication failed: %v", ErrDeliveryFailed, err)
		}
	}

	if err := client.Mail(p.config.From); err != nil {
		return fmt.Errorf("%w: MAIL FROM failed: %v", ErrDeliveryFailed, err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("%w: RCPT TO failed: %v", ErrDeliveryFailed, err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("%w: DATA failed: %v", ErrDeliveryFailed, err)
	}
	_, err = w.Write([]byte(msg.String()))
	if err != nil {
		_ = w.Close()
		return fmt.Errorf("%w: write message failed: %v", ErrDeliveryFailed, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("%w: close message failed: %v", ErrDeliveryFailed, err)
	}
	return client.Quit()
}

// MockEmailProvider is an in-memory fake email provider used exclusively for automated tests.
type MockEmailProvider struct {
	mu           sync.Mutex
	SentMessages []MockSentEmail
	SimulateErr  error
	Configured   bool
}

// MockSentEmail records delivery details for test assertions.
type MockSentEmail struct {
	ToEmail string
	URL     string
	Kind    string
}

func (m *MockEmailProvider) SendPasswordResetEmail(ctx context.Context, toEmail, resetURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SimulateErr != nil {
		return m.SimulateErr
	}
	m.SentMessages = append(m.SentMessages, MockSentEmail{ToEmail: toEmail, URL: resetURL, Kind: "PASSWORD_RESET"})
	return nil
}

func (m *MockEmailProvider) SendVerificationEmail(ctx context.Context, toEmail, verifyURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SimulateErr != nil {
		return m.SimulateErr
	}
	m.SentMessages = append(m.SentMessages, MockSentEmail{ToEmail: toEmail, URL: verifyURL, Kind: "VERIFY_EMAIL"})
	return nil
}

func (m *MockEmailProvider) IsConfigured() bool {
	return m.Configured
}

// LastSentResetURL returns the most recent reset URL sent to the given email in tests.
func (m *MockEmailProvider) LastSentResetURL(email string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.SentMessages) - 1; i >= 0; i-- {
		if strings.EqualFold(m.SentMessages[i].ToEmail, email) && m.SentMessages[i].Kind == "PASSWORD_RESET" {
			return m.SentMessages[i].URL
		}
	}
	return ""
}

// LastSentVerificationURL returns the most recent verification URL sent to the given email in tests.
func (m *MockEmailProvider) LastSentVerificationURL(email string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.SentMessages) - 1; i >= 0; i-- {
		if strings.EqualFold(m.SentMessages[i].ToEmail, email) && m.SentMessages[i].Kind == "VERIFY_EMAIL" {
			return m.SentMessages[i].URL
		}
	}
	return ""
}

// Reset clears the mock state.
func (m *MockEmailProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SentMessages = nil
	m.SimulateErr = nil
}

var (
	emailProviderMu sync.RWMutex
	globalEmailProv EmailProvider = &UnconfiguredEmailProvider{}
)

// GetEmailProvider returns the active EmailProvider.
func GetEmailProvider() EmailProvider {
	emailProviderMu.RLock()
	defer emailProviderMu.RUnlock()
	return globalEmailProv
}

// SetEmailProvider sets the active EmailProvider.
func SetEmailProvider(p EmailProvider) {
	emailProviderMu.Lock()
	defer emailProviderMu.Unlock()
	if p == nil {
		globalEmailProv = &UnconfiguredEmailProvider{}
	} else {
		globalEmailProv = p
	}
}

// InitEmailProviderFromEnv loads SMTP config and configures the email provider.
func InitEmailProviderFromEnv() {
	cfg, err := LoadSMTPConfigFromEnv()
	if err != nil {
		fmt.Printf("⚠️ SMTP configuration validation error: %v. Email delivery disabled.\n", err)
		SetEmailProvider(&UnconfiguredEmailProvider{})
		return
	}
	if cfg == nil {
		SetEmailProvider(&UnconfiguredEmailProvider{})
		return
	}

	provider, err := NewSMTPEmailProvider(cfg)
	if err != nil {
		fmt.Printf("⚠️ Failed to initialize SMTPEmailProvider: %v. Email delivery disabled.\n", err)
		SetEmailProvider(&UnconfiguredEmailProvider{})
		return
	}

	SetEmailProvider(provider)
	fmt.Printf("📧 SMTP Email Provider initialized (Host: %s, From: %s, TLS: required)\n", FormatSMTPAddress(cfg.Host, cfg.Port), cfg.From)
}

// FormatSMTPAddress safely constructs a host:port address compatible with IPv4, IPv6, and hostnames.
func FormatSMTPAddress(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

