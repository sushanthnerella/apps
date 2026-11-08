package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/models"
)

func setupTestGateway(t *testing.T) (*mux.Router, *db.Storage, string, *auth.MockEmailProvider) {
	testSecret := "purohit_platform_test_jwt_secret_key_32bytes_long!"
	_ = auth.InitJWT(testSecret)

	storage, _, _ := db.CreateIsolatedAuthTestDBForTest(t)

	emailMock := &auth.MockEmailProvider{Configured: true}
	auth.SetEmailProvider(emailMock)

	auth.DefaultRecoveryThrottler.ResetForTesting()

	router := mux.NewRouter()
	registerRecoveryRoutes(router, storage, testSecret)

	return router, storage, testSecret, emailMock
}

func TestHTTPRecoveryStartAndSecurity(t *testing.T) {
	router, storage, _, emailMock := setupTestGateway(t)
	ctx := context.Background()

	// Register devotee with verified email
	devoteeID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876540001",
		FullName:      "HTTP Test Devotee",
		Password:      "InitialPass123!",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("failed registering devotee: %v", err)
	}
	verifiedEmail := "http.devotee@example.com"
	_ = storage.SetAccountVerifiedEmail(ctx, "devotee", devoteeID, verifiedEmail, true)

	// 1. Valid recovery request returns generic 200
	payload := `{"email":"http.devotee@example.com","role":"devotee"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/recovery/start", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d: %s", w.Code, w.Body.String())
	}
	var res models.StartRecoveryResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Status != "ACCEPTED" {
		t.Errorf("expected status ACCEPTED, got %s", res.Status)
	}

	// Email was dispatched
	resetURL := emailMock.LastSentResetURL(verifiedEmail)
	if resetURL == "" {
		t.Fatalf("expected reset email sent to %s", verifiedEmail)
	}
	u, _ := url.Parse(resetURL)
	rawToken := u.Query().Get("token")
	if len(rawToken) != 64 {
		t.Fatalf("expected 64-character token in reset URL")
	}

	// 2. Cooldown rejection: immediate resend triggers HTTP 429
	req2 := httptest.NewRequest("POST", "/api/v1/auth/recovery/start", bytes.NewBufferString(payload))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("expected HTTP 429 on immediate retry, got %d", w2.Code)
	}

	// 3. Unconfigured email service returns HTTP 503
	auth.SetEmailProvider(&auth.UnconfiguredEmailProvider{})
	auth.DefaultRecoveryThrottler.ResetForTesting()
	req3 := httptest.NewRequest("POST", "/api/v1/auth/recovery/start", bytes.NewBufferString(payload))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	if w3.Code != http.StatusServiceUnavailable {
		t.Errorf("expected HTTP 503 when email unconfigured, got %d", w3.Code)
	}
	if !strings.Contains(w3.Body.String(), "email_service_unavailable") {
		t.Errorf("expected email_service_unavailable error body")
	}
}

func TestHTTPResetFormGETAndSecurityHeaders(t *testing.T) {
	router, storage, _, emailMock := setupTestGateway(t)
	ctx := context.Background()

	devoteeID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876540002",
		FullName:      "Form GET Devotee",
		Password:      "InitialPass123!",
		ConsentAgreed: true,
	})
	_ = storage.SetAccountVerifiedEmail(ctx, "devotee", devoteeID, "form.get@example.com", true)

	_, _ = storage.StartPasswordRecovery(ctx, models.StartRecoveryRequest{
		Email: "form.get@example.com",
		Role:  "devotee",
	}, "127.0.0.1", "purohit_platform_test_jwt_secret_key_32bytes_long!")

	u, _ := url.Parse(emailMock.LastSentResetURL("form.get@example.com"))
	rawToken := u.Query().Get("token")

	// 1. GET valid reset form
	req := httptest.NewRequest("GET", "/api/v1/auth/recovery/reset?token="+rawToken, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", w.Code)
	}

	// Verify strict security headers protecting token and preventing leakage
	if w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Errorf("expected Referrer-Policy: no-referrer")
	}
	if !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
		t.Errorf("expected Cache-Control: no-store")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("expected X-Frame-Options: DENY")
	}

	body := w.Body.String()
	if !strings.Contains(body, "Reset Your Password") || !strings.Contains(body, "New Password") {
		t.Errorf("expected password reset form in response body")
	}

	// 2. Invariant: Visiting the page DID NOT consume the token
	reqVisitAgain := httptest.NewRequest("GET", "/api/v1/auth/recovery/reset?token="+rawToken, nil)
	wVisitAgain := httptest.NewRecorder()
	router.ServeHTTP(wVisitAgain, reqVisitAgain)
	if wVisitAgain.Code != http.StatusOK {
		t.Fatalf("visiting page must not consume token; got status %d", wVisitAgain.Code)
	}

	// 3. GET with invalid token renders friendly error
	reqInvalid := httptest.NewRequest("GET", "/api/v1/auth/recovery/reset?token=invalid_token_123", nil)
	wInvalid := httptest.NewRecorder()
	router.ServeHTTP(wInvalid, reqInvalid)
	if !strings.Contains(wInvalid.Body.String(), "⚠️") {
		t.Errorf("expected error banner for invalid token")
	}
}

func TestHTTPResetFormPOSTConfirmationAndReuseRejection(t *testing.T) {
	router, storage, _, emailMock := setupTestGateway(t)
	ctx := context.Background()

	devoteeID, _ := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876540003",
		FullName:      "Form POST Devotee",
		Password:      "InitialPass123!",
		ConsentAgreed: true,
	})
	_ = storage.SetAccountVerifiedEmail(ctx, "devotee", devoteeID, "form.post@example.com", true)

	_, _ = storage.StartPasswordRecovery(ctx, models.StartRecoveryRequest{
		Email: "form.post@example.com",
		Role:  "devotee",
	}, "127.0.0.1", "purohit_platform_test_jwt_secret_key_32bytes_long!")

	u, _ := url.Parse(emailMock.LastSentResetURL("form.post@example.com"))
	rawToken := u.Query().Get("token")

	// 1. Submit HTML form
	form := url.Values{}
	form.Set("token", rawToken)
	form.Set("new_password", "BrandNewSecurePass#2026")
	form.Set("confirm_password", "BrandNewSecurePass#2026")

	req := httptest.NewRequest("POST", "/api/v1/auth/recovery/confirm", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 on form submit, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Password Reset Successful") {
		t.Errorf("expected success page rendered")
	}

	// 2. Reusing token fails (single-use enforced)
	reqReuse := httptest.NewRequest("POST", "/api/v1/auth/recovery/confirm", strings.NewReader(form.Encode()))
	reqReuse.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wReuse := httptest.NewRecorder()
	router.ServeHTTP(wReuse, reqReuse)

	if !strings.Contains(wReuse.Body.String(), "already been used") && !strings.Contains(wReuse.Body.String(), "invalid") {
		t.Errorf("expected already been used error on reuse")
	}

	// 3. Visiting the GET link after consumption displays already used
	reqGETConsumed := httptest.NewRequest("GET", "/api/v1/auth/recovery/reset?token="+rawToken, nil)
	wGETConsumed := httptest.NewRecorder()
	router.ServeHTTP(wGETConsumed, reqGETConsumed)
	if !strings.Contains(wGETConsumed.Body.String(), "already been used") {
		t.Errorf("expected already been used notice on consumed GET link")
	}
}

func TestHTTPEmailVerificationLinkAndSecurity(t *testing.T) {
	router, storage, _, emailMock := setupTestGateway(t)
	ctx := context.Background()

	devoteeID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876540004",
		FullName:      "Verify Link Devotee",
		Password:      "SecurePassword123!",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("failed registering devotee: %v", err)
	}

	// 1. Generate JWT for authenticated devotee
	tokenStr, err := auth.GenerateToken(devoteeID, "+919876540004", "devotee", "Verify Link Devotee", 1)
	if err != nil {
		t.Fatalf("failed generating token: %v", err)
	}

	// 2. Request email verification via POST /api/v1/user/email
	payload := `{"email":"verify.link@example.com"}`
	reqPost := httptest.NewRequest("POST", "/api/v1/user/email", bytes.NewBufferString(payload))
	reqPost.Header.Set("Authorization", "Bearer "+tokenStr)
	reqPost.Header.Set("Content-Type", "application/json")
	wPost := httptest.NewRecorder()
	router.ServeHTTP(wPost, reqPost)

	if wPost.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d: %s", wPost.Code, wPost.Body.String())
	}

	// Verify email was dispatched
	verifyURL := emailMock.LastSentVerificationURL("verify.link@example.com")
	if verifyURL == "" {
		t.Fatalf("expected verification email dispatched to verify.link@example.com")
	}

	// Invariant: Verification URL must point to verify-email endpoint, NOT password-reset form!
	if strings.Contains(verifyURL, "recovery/reset") || strings.Contains(verifyURL, "reset-password") {
		t.Fatalf("CRITICAL BUG: Verification URL points to password reset form! URL: %s", verifyURL)
	}
	if !strings.Contains(verifyURL, "verify-email") {
		t.Fatalf("expected verify-email endpoint in URL: %s", verifyURL)
	}

	u, _ := url.Parse(verifyURL)
	rawToken := u.Query().Get("token")
	if len(rawToken) != 64 {
		t.Fatalf("expected 64-char hex token in verification link, got: %s", rawToken)
	}

	// 3. Browser GET verification link with text/html Accept header
	reqBrowser := httptest.NewRequest("GET", "/api/v1/auth/verify-email?token="+rawToken, nil)
	reqBrowser.Header.Set("Accept", "text/html,application/xhtml+xml")
	wBrowser := httptest.NewRecorder()
	router.ServeHTTP(wBrowser, reqBrowser)

	if wBrowser.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 for browser GET, got %d", wBrowser.Code)
	}

	// Invariant: Anti-leakage security headers must be present
	if wBrowser.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Errorf("expected Referrer-Policy: no-referrer")
	}
	if !strings.Contains(wBrowser.Header().Get("Cache-Control"), "no-store") {
		t.Errorf("expected Cache-Control: no-store")
	}
	if wBrowser.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("expected X-Frame-Options: DENY")
	}

	htmlBody := wBrowser.Body.String()
	if !strings.Contains(htmlBody, "Email Verification Confirmed") || !strings.Contains(htmlBody, "verify.link@example.com") {
		t.Errorf("expected confirmation card with email, got: %s", htmlBody)
	}

	// 4. Verify account in database now has email_verified = true
	email, isVerified, err := storage.GetAccountEmail(ctx, "devotee", devoteeID)
	if err != nil || email != "verify.link@example.com" || !isVerified {
		t.Fatalf("expected verified email in DB, got email=%s, isVerified=%v, err=%v", email, isVerified, err)
	}

	// 5. Visiting the link again renders friendly already used error
	reqBrowserAgain := httptest.NewRequest("GET", "/api/v1/auth/verify-email?token="+rawToken, nil)
	reqBrowserAgain.Header.Set("Accept", "text/html")
	wBrowserAgain := httptest.NewRecorder()
	router.ServeHTTP(wBrowserAgain, reqBrowserAgain)

	if !strings.Contains(wBrowserAgain.Body.String(), "already been used") {
		t.Errorf("expected already been used notice on reused verification link, got: %s", wBrowserAgain.Body.String())
	}
}

func TestHTTPUserEmailEndpoints(t *testing.T) {
	router, storage, _, emailMock := setupTestGateway(t)
	ctx := context.Background()

	priestID, err := storage.RegisterPriest(models.PriestRegistrationRequest{
		Phone:           "+919876540005",
		FullName:        "API Priest",
		Password:        "VedicPassword123!",
		Sampradaya:      "Smartha",
		VedaShakha:      "Rigveda",
		ExperienceYears: 10,
		ConsentAgreed:   true,
	})
	if err != nil {
		t.Fatalf("failed registering priest: %v", err)
	}

	tokenStr, _ := auth.GenerateToken(priestID, "+919876540005", "priest", "API Priest", 1)

	// 1. Initial GET /api/v1/user/email returns empty unverified
	reqGet := httptest.NewRequest("GET", "/api/v1/user/email", nil)
	reqGet.Header.Set("Authorization", "Bearer "+tokenStr)
	wGet := httptest.NewRecorder()
	router.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", wGet.Code)
	}
	var getRes map[string]interface{}
	_ = json.Unmarshal(wGet.Body.Bytes(), &getRes)
	if getRes["email_verified"] != false {
		t.Errorf("expected email_verified false, got: %v", getRes["email_verified"])
	}

	// 2. POST unconfigured email provider returns 503
	auth.SetEmailProvider(&auth.UnconfiguredEmailProvider{})
	reqUnconf := httptest.NewRequest("POST", "/api/v1/user/email", bytes.NewBufferString(`{"email":"priest@temple.org"}`))
	reqUnconf.Header.Set("Authorization", "Bearer "+tokenStr)
	reqUnconf.Header.Set("Content-Type", "application/json")
	wUnconf := httptest.NewRecorder()
	router.ServeHTTP(wUnconf, reqUnconf)

	if wUnconf.Code != http.StatusServiceUnavailable {
		t.Errorf("expected HTTP 503 when email unconfigured, got %d: %s", wUnconf.Code, wUnconf.Body.String())
	}

	// 3. POST with configured provider succeeds
	auth.SetEmailProvider(emailMock)
	reqPost := httptest.NewRequest("POST", "/api/v1/user/email", bytes.NewBufferString(`{"email":"priest@temple.org"}`))
	reqPost.Header.Set("Authorization", "Bearer "+tokenStr)
	reqPost.Header.Set("Content-Type", "application/json")
	wPost := httptest.NewRecorder()
	router.ServeHTTP(wPost, reqPost)

	if wPost.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d: %s", wPost.Code, wPost.Body.String())
	}

	// 4. API GET verification with JSON Accept header
	rawURL := emailMock.LastSentVerificationURL("priest@temple.org")
	u, _ := url.Parse(rawURL)
	token := u.Query().Get("token")

	reqApiVerify := httptest.NewRequest("GET", "/api/v1/auth/verify-email?token="+token, nil)
	reqApiVerify.Header.Set("Accept", "application/json")
	wApiVerify := httptest.NewRecorder()
	router.ServeHTTP(wApiVerify, reqApiVerify)

	if wApiVerify.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d: %s", wApiVerify.Code, wApiVerify.Body.String())
	}
	var verifyRes map[string]interface{}
	_ = json.Unmarshal(wApiVerify.Body.Bytes(), &verifyRes)
	if verifyRes["status"] != "SUCCESS" || verifyRes["email"] != "priest@temple.org" || verifyRes["role"] != "priest" {
		t.Errorf("unexpected API verification response: %v", verifyRes)
	}

	// 5. GET /api/v1/user/profile now reflects email and email_verified=true
	profile, err := storage.GetUserProfile(priestID, "priest")
	if err != nil || profile.Email != "priest@temple.org" || !profile.EmailVerified {
		t.Fatalf("expected priest profile to have verified email, got email=%s, verified=%v", profile.Email, profile.EmailVerified)
	}
	_ = ctx
}

func TestEmailSecurityScannerVisitAndUserClick(t *testing.T) {
	router, storage, _, emailMock := setupTestGateway(t)
	ctx := context.Background()

	devoteeID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876540009",
		FullName:      "Scanner User",
		Password:      "SecurePassword123!",
		ConsentAgreed: true,
	})
	if err != nil {
		t.Fatalf("failed registering devotee: %v", err)
	}

	tokenStr, _ := auth.GenerateToken(devoteeID, "+919876540009", "devotee", "Scanner User", 1)

	// 1. User requests email verification
	payload := `{"email":"scanner.user@example.com"}`
	reqPost := httptest.NewRequest("POST", "/api/v1/user/email", bytes.NewBufferString(payload))
	reqPost.Header.Set("Authorization", "Bearer "+tokenStr)
	reqPost.Header.Set("Content-Type", "application/json")
	wPost := httptest.NewRecorder()
	router.ServeHTTP(wPost, reqPost)

	if wPost.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", wPost.Code)
	}

	verifyURL := emailMock.LastSentVerificationURL("scanner.user@example.com")
	u, _ := url.Parse(verifyURL)
	rawToken := u.Query().Get("token")

	// 2. Email security scanner automatically prefetches GET link with corporate scanner User-Agent
	reqScanner := httptest.NewRequest("GET", "/api/v1/auth/verify-email?token="+rawToken, nil)
	reqScanner.Header.Set("Accept", "text/html,application/xhtml+xml")
	reqScanner.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36 (Proofpoint-Link-Scanner)")
	wScanner := httptest.NewRecorder()
	router.ServeHTTP(wScanner, reqScanner)

	if wScanner.Code != http.StatusOK {
		t.Fatalf("scanner request should succeed with HTTP 200, got %d", wScanner.Code)
	}
	if !strings.Contains(wScanner.Body.String(), "Email Verification Confirmed") {
		t.Fatalf("scanner should confirm verification, got: %s", wScanner.Body.String())
	}

	// 3. Database state: Account email is verified in DB
	email, isVerified, err := storage.GetAccountEmail(ctx, "devotee", devoteeID)
	if err != nil || email != "scanner.user@example.com" || !isVerified {
		t.Fatalf("expected email to be verified in DB, got email=%s, isVerified=%v", email, isVerified)
	}

	// 4. Human user subsequently clicks the link in their email client
	reqUser := httptest.NewRequest("GET", "/api/v1/auth/verify-email?token="+rawToken, nil)
	reqUser.Header.Set("Accept", "text/html,application/xhtml+xml")
	reqUser.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148")
	wUser := httptest.NewRecorder()
	router.ServeHTTP(wUser, reqUser)

	userHtml := wUser.Body.String()
	// Invariant: Card explains that the link was already used and gives reassuring Email Security Notice
	if !strings.Contains(userHtml, "already been used") {
		t.Errorf("expected already been used notice, got: %s", userHtml)
	}
	if !strings.Contains(userHtml, "Email Security Notice") {
		t.Errorf("expected Email Security Notice explaining scanner prefetch, got: %s", userHtml)
	}
}
