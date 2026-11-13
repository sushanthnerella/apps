package notify

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// NotificationMessage encapsulates user-facing push payload and data payload.
type NotificationMessage struct {
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Data  map[string]string `json:"data,omitempty"`
}

// PushNotifier defines the interface for delivering push notifications to client devices.
type PushNotifier interface {
	SendToToken(ctx context.Context, token string, msg NotificationMessage) error
	SendToTokens(ctx context.Context, tokens []string, msg NotificationMessage) error
}

// LoggingNoOpNotifier is used when FCM is not configured (e.g. development/testing).
type LoggingNoOpNotifier struct{}

func (n *LoggingNoOpNotifier) SendToToken(ctx context.Context, token string, msg NotificationMessage) error {
	log.Printf("[Push][NoOp] To token %s: %s - %s", token, msg.Title, msg.Body)
	return nil
}

func (n *LoggingNoOpNotifier) SendToTokens(ctx context.Context, tokens []string, msg NotificationMessage) error {
	for _, t := range tokens {
		_ = n.SendToToken(ctx, t, msg)
	}
	return nil
}

type serviceAccountCredentials struct {
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	TokenURI    string `json:"token_uri"`
}

// FCMPushNotifier delivers real push notifications via FCM HTTP v1 using standard library OAuth2 exchange.
type FCMPushNotifier struct {
	projectID   string
	clientEmail string
	tokenURI    string
	privateKey  *rsa.PrivateKey
	httpClient  *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// NewFCMPushNotifier parses a service account JSON string and builds an FCM HTTP v1 client.
func NewFCMPushNotifier(saJSON string) (*FCMPushNotifier, error) {
	var sa serviceAccountCredentials
	if err := json.Unmarshal([]byte(saJSON), &sa); err != nil {
		return nil, fmt.Errorf("invalid service account JSON: %w", err)
	}
	if sa.ProjectID == "" || sa.ClientEmail == "" || sa.PrivateKey == "" {
		return nil, errors.New("incomplete service account credentials")
	}
	if sa.TokenURI == "" {
		sa.TokenURI = "https://oauth2.googleapis.com/token"
	}

	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing private key")
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Fallback to PKCS1
		var pkcs1Err error
		parsedKey, pkcs1Err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if pkcs1Err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w (pkcs8 err: %v)", pkcs1Err, err)
		}
	}

	rsaKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not an RSA private key")
	}

	return &FCMPushNotifier{
		projectID:   sa.ProjectID,
		clientEmail: sa.ClientEmail,
		tokenURI:    sa.TokenURI,
		privateKey:  rsaKey,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (f *FCMPushNotifier) getAccessToken(ctx context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.accessToken != "" && time.Now().Before(f.tokenExpiry.Add(-2*time.Minute)) {
		return f.accessToken, nil
	}

	now := time.Now()
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	claims := map[string]interface{}{
		"iss":   f.clientEmail,
		"scope": "https://www.googleapis.com/auth/firebase.messaging",
		"aud":   f.tokenURI,
		"exp":   now.Add(time.Hour).Unix(),
		"iat":   now.Unix(),
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
	payload := headerB64 + "." + claimsB64

	hashed := sha256.Sum256([]byte(payload))
	sig, err := rsa.SignPKCS1v15(rand.Reader, f.privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign assertion: %w", err)
	}

	assertion := payload + "." + base64.RawURLEncoding.EncodeToString(sig)

	formData := url.Values{}
	formData.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	formData.Set("assertion", assertion)

	req, err := http.NewRequestWithContext(ctx, "POST", f.tokenURI, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("oauth token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenRes struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
		return "", fmt.Errorf("decode token response failed: %w", err)
	}

	f.accessToken = tokenRes.AccessToken
	f.tokenExpiry = time.Now().Add(time.Duration(tokenRes.ExpiresIn) * time.Second)
	return f.accessToken, nil
}

func (f *FCMPushNotifier) SendToToken(ctx context.Context, token string, msg NotificationMessage) error {
	accessToken, err := f.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to obtain FCM access token: %w", err)
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", f.projectID)

	payload := map[string]interface{}{
		"message": map[string]interface{}{
			"token": token,
			"notification": map[string]string{
				"title": msg.Title,
				"body":  msg.Body,
			},
			"data": msg.Data,
		},
	}

	reqBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json; UTF-8")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fcm send error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("fcm send failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (f *FCMPushNotifier) SendToTokens(ctx context.Context, tokens []string, msg NotificationMessage) error {
	var firstErr error
	for _, t := range tokens {
		if err := f.SendToToken(ctx, t, msg); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// NewPushNotifierFromEnv initializes an FCM HTTP v1 notifier if FCM_SERVICE_ACCOUNT_JSON is set,
// or defaults to a LoggingNoOpNotifier.
func NewPushNotifierFromEnv() PushNotifier {
	saJSON := os.Getenv("FCM_SERVICE_ACCOUNT_JSON")
	if strings.TrimSpace(saJSON) == "" {
		log.Println("ℹ️  FCM_SERVICE_ACCOUNT_JSON not set; using LoggingNoOpNotifier")
		return &LoggingNoOpNotifier{}
	}

	notifier, err := NewFCMPushNotifier(saJSON)
	if err != nil {
		log.Printf("⚠️  Failed to initialize FCM notifier from environment: %v; falling back to LoggingNoOpNotifier", err)
		return &LoggingNoOpNotifier{}
	}

	log.Println("✅ FCM HTTP v1 push notifications initialized successfully")
	return notifier
}
