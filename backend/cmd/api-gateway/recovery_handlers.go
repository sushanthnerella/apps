package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/models"
)

// registerRecoveryRoutes wires up all password recovery and email verification endpoints.
func registerRecoveryRoutes(router *mux.Router, storage *db.Storage, secret string) {
	// 1. Password Recovery: Start (Sends single-use cryptographic email reset link)
	router.HandleFunc("/api/v1/auth/recovery/start", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req models.StartRecoveryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid_payload","message":"invalid start recovery payload format"}`, http.StatusBadRequest)
			return
		}

		clientIP := auth.ExtractClientIP(r)
		res, err := storage.StartPasswordRecovery(r.Context(), req, clientIP, secret)
		if err != nil {
			if errors.Is(err, auth.ErrEmailUnavailable) {
				http.Error(w, `{"error":"email_service_unavailable","message":"Email delivery service is currently unconfigured. Password recovery via email is unavailable."}`, http.StatusServiceUnavailable)
				return
			}
			if errors.Is(err, auth.ErrRecoveryCooldown) || errors.Is(err, auth.ErrRecoveryAttemptsExceeded) || strings.Contains(err.Error(), "retry in") {
				http.Error(w, fmt.Sprintf(`{"error":"too_many_requests","message":"%s"}`, err.Error()), http.StatusTooManyRequests)
				return
			}
			if strings.Contains(err.Error(), "role is required") || strings.Contains(err.Error(), "required") {
				http.Error(w, fmt.Sprintf(`{"error":"bad_request","message":"%s"}`, err.Error()), http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf(`{"error":"internal_error","message":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(res)
	}).Methods("POST", "OPTIONS")

	// 2. HTTPS Password Reset Form (GET: Validates token without consuming it)
	handleResetFormGET := func(w http.ResponseWriter, r *http.Request) {
		// Strict security headers: prevent token leakage via logs and referrers
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline';")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			renderResetPage(w, "", nil, "A valid reset token is required to access this page. Please use the complete link sent to your email.")
			return
		}

		tokenInfo, err := storage.ValidateResetToken(r.Context(), token, secret)
		if err != nil {
			renderResetPage(w, "", nil, err.Error())
			return
		}

		// Valid token: render form without consuming token
		renderResetPage(w, token, tokenInfo, "")
	}

	router.HandleFunc("/api/v1/auth/recovery/reset", handleResetFormGET).Methods("GET")
	router.HandleFunc("/reset-password", handleResetFormGET).Methods("GET")

	// 3. Password Recovery: Confirm (Atomically consumes token and resets password)
	handleResetConfirmPOST := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")

		isForm := strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")

		var req models.ConfirmRecoveryRequest
		if isForm {
			if err := r.ParseForm(); err != nil {
				renderResetPage(w, "", nil, "Invalid form submission.")
				return
			}
			req.Token = strings.TrimSpace(r.FormValue("token"))
			req.NewPassword = strings.TrimSpace(r.FormValue("new_password"))
			req.ConfirmPassword = strings.TrimSpace(r.FormValue("confirm_password"))
		} else {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid_payload","message":"invalid confirm recovery payload format"}`, http.StatusBadRequest)
				return
			}
		}

		clientIP := auth.ExtractClientIP(r)
		err := storage.ConfirmPasswordRecovery(r.Context(), req, clientIP, secret)
		if err != nil {
			if isForm {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				renderResetPage(w, req.Token, nil, err.Error())
				return
			}

			if strings.Contains(err.Error(), "too many failed") {
				http.Error(w, fmt.Sprintf(`{"error":"too_many_requests","message":"%s"}`, err.Error()), http.StatusTooManyRequests)
				return
			}
			if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "expired") || strings.Contains(err.Error(), "not match") || strings.Contains(err.Error(), "must be") || strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "already been used") {
				http.Error(w, fmt.Sprintf(`{"error":"bad_request","message":"%s"}`, err.Error()), http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf(`{"error":"internal_error","message":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		if isForm {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			renderSuccessPage(w)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "SUCCESS",
			"message": "Password updated successfully. Please log in with your new password.",
		})
	}

	router.HandleFunc("/api/v1/auth/recovery/confirm", handleResetConfirmPOST).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/auth/recovery/reset", handleResetConfirmPOST).Methods("POST")
	router.HandleFunc("/reset-password", handleResetConfirmPOST).Methods("POST")

	// 4. Authenticated Email Management: Request Verification & Check Status
	router.HandleFunc("/api/v1/user/email", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := validateCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%s"}`, err.Error()), http.StatusUnauthorized)
			return
		}

		if r.Method == "GET" {
			email, isVerified, err := storage.GetAccountEmail(r.Context(), claims.Role, claims.UserID)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"not_found","message":"%s"}`, err.Error()), http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"email":          email,
				"email_verified": isVerified,
				"role":           claims.Role,
			})
			return
		}

		var payload struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, `{"error":"invalid_payload","message":"invalid payload format"}`, http.StatusBadRequest)
			return
		}

		_, err = storage.RequestEmailVerification(r.Context(), claims.Role, claims.UserID, payload.Email, secret)
		if err != nil {
			if errors.Is(err, auth.ErrEmailUnavailable) {
				http.Error(w, `{"error":"email_service_unavailable","message":"Email service is currently unconfigured. Password recovery via email is unavailable."}`, http.StatusServiceUnavailable)
				return
			}
			if errors.Is(err, auth.ErrDeliveryFailed) || strings.Contains(err.Error(), "email_delivery_failed") {
				http.Error(w, `{"error":"email_delivery_failed","message":"Failed to deliver verification email. Please check your email address or try again later."}`, http.StatusBadGateway)
				return
			}
			http.Error(w, fmt.Sprintf(`{"error":"bad_request","message":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "SUCCESS",
			"message": "Verification email sent. Please check your inbox to confirm your email address.",
		})
	}).Methods("GET", "POST", "OPTIONS")

	// 5. Confirm Email Verification (Supports both browser GET and API GET with anti-leakage security headers)
	handleEmailVerificationGET := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline';")

		isHTML := strings.Contains(r.Header.Get("Accept"), "text/html")
		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			if isHTML {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				renderVerificationResultPage(w, false, "", "", "A valid verification token is required to verify your email.")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"bad_request","message":"verification token is required"}`, http.StatusBadRequest)
			return
		}

		verifiedEmail, role, err := storage.ConfirmEmailVerification(r.Context(), token, secret)
		if err != nil {
			if isHTML {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				renderVerificationResultPage(w, false, "", "", err.Error())
				return
			}
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, fmt.Sprintf(`{"error":"bad_request","message":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		if isHTML {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			renderVerificationResultPage(w, true, verifiedEmail, role, "")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "SUCCESS",
			"message": "Email verified successfully. You can now use this email for account recovery.",
			"email":   verifiedEmail,
			"role":    role,
		})
	}

	router.HandleFunc("/api/v1/auth/verify-email", handleEmailVerificationGET).Methods("GET")
	router.HandleFunc("/verify-email", handleEmailVerificationGET).Methods("GET")
}

// renderResetPage generates the HTML password reset form with zero external assets to eliminate referrer leakage.
func renderResetPage(w http.ResponseWriter, token string, info *models.ResetTokenInfo, errNotice string) {
	title := "Reset Your Password"
	accountDetails := ""
	if info != nil {
		accountDetails = fmt.Sprintf("For verified %s account (%s)", html.EscapeString(info.Role), html.EscapeString(info.Email))
	}

	errorBlock := ""
	if errNotice != "" {
		cleanErr := html.EscapeString(errNotice)
		errorBlock = fmt.Sprintf(`
			<div style="background-color: #fef2f2; border: 1px solid #fecaca; color: #b91c1c; padding: 12px 16px; border-radius: 8px; margin-bottom: 20px; font-size: 14px;">
				⚠️ %s
			</div>
		`, cleanErr)
	}

	formBlock := ""
	if token != "" && (errNotice == "" || strings.Contains(errNotice, "not match") || strings.Contains(errNotice, "characters")) {
		formBlock = fmt.Sprintf(`
			<form method="POST" action="/api/v1/auth/recovery/confirm">
				<input type="hidden" name="token" value="%s">
				
				<div style="margin-bottom: 16px; text-align: left;">
					<label style="display: block; font-size: 13px; font-weight: 600; color: #334155; margin-bottom: 6px;">New Password</label>
					<input type="password" name="new_password" required minlength="8" maxlength="72" placeholder="At least 8 characters"
						style="width: 100%%; box-sizing: border-box; padding: 10px 14px; border: 1px solid #cbd5e1; border-radius: 8px; font-size: 15px;">
				</div>

				<div style="margin-bottom: 24px; text-align: left;">
					<label style="display: block; font-size: 13px; font-weight: 600; color: #334155; margin-bottom: 6px;">Confirm New Password</label>
					<input type="password" name="confirm_password" required minlength="8" maxlength="72" placeholder="Repeat your new password"
						style="width: 100%%; box-sizing: border-box; padding: 10px 14px; border: 1px solid #cbd5e1; border-radius: 8px; font-size: 15px;">
				</div>

				<button type="submit"
					style="width: 100%%; background-color: #ea580c; color: #ffffff; border: none; padding: 13px; border-radius: 8px; font-size: 16px; font-weight: 600; cursor: pointer; transition: background 0.2s;">
					Update Password
				</button>
			</form>
		`, html.EscapeString(token))
	} else if errNotice != "" {
		formBlock = `
			<p style="color: #64748b; font-size: 14px; line-height: 1.6;">
				Please open the Puja Vaahini mobile app and request a new password reset link.
			</p>
		`
	}

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<meta name="referrer" content="no-referrer">
	<title>Puja Vaahini — %s</title>
	<style>
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
			background-color: #f8fafc;
			color: #0f172a;
			margin: 0;
			padding: 24px;
			display: flex;
			align-items: center;
			justify-content: center;
			min-height: 100vh;
			box-sizing: border-box;
		}
		.card {
			background: #ffffff;
			max-width: 440px;
			width: 100%%;
			border-radius: 16px;
			border: 1px solid #e2e8f0;
			padding: 32px;
			box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.05), 0 4px 6px -2px rgba(0, 0, 0, 0.025);
			text-align: center;
		}
	</style>
</head>
<body>
	<div class="card">
		<h1 style="color: #ea580c; margin: 0 0 6px 0; font-size: 26px;">🕉️ Puja Vaahini</h1>
		<h2 style="font-size: 19px; color: #0f172a; margin: 0 0 4px 0;">%s</h2>
		<p style="color: #64748b; font-size: 13px; margin: 0 0 20px 0;">%s</p>

		%s
		%s

		<div style="margin-top: 28px; padding-top: 16px; border-top: 1px solid #f1f5f9; font-size: 12px; color: #94a3b8;">
			Puja Vaahini Platform &bull; Hyderabad Pilot Service &bull; All Sessions Terminated upon Reset
		</div>
	</div>
</body>
</html>`, title, title, accountDetails, errorBlock, formBlock)

	_, _ = w.Write([]byte(htmlContent))
}

// renderSuccessPage generates a confirmation message upon successful password reset.
func renderSuccessPage(w http.ResponseWriter) {
	htmlContent := `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<meta name="referrer" content="no-referrer">
	<title>Puja Vaahini — Password Reset Successful</title>
	<style>
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
			background-color: #f8fafc;
			color: #0f172a;
			margin: 0;
			padding: 24px;
			display: flex;
			align-items: center;
			justify-content: center;
			min-height: 100vh;
			box-sizing: border-box;
		}
		.card {
			background: #ffffff;
			max-width: 440px;
			width: 100%;
			border-radius: 16px;
			border: 1px solid #e2e8f0;
			padding: 32px;
			box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.05);
			text-align: center;
		}
	</style>
</head>
<body>
	<div class="card">
		<div style="font-size: 48px; margin-bottom: 12px;">✅</div>
		<h1 style="color: #ea580c; margin: 0 0 6px 0; font-size: 24px;">🕉️ Puja Vaahini</h1>
		<h2 style="font-size: 18px; color: #15803d; margin: 0 0 12px 0;">Password Reset Successful!</h2>
		<p style="color: #334155; font-size: 14px; line-height: 1.6; margin-bottom: 24px;">
			Your account password has been updated. All previous active logins have been safely signed out.
		</p>
		<p style="color: #64748b; font-size: 13px; line-height: 1.5;">
			You can now open the <strong>Puja Vaahini</strong> mobile app and log in with your new password.
		</p>
		<div style="margin-top: 28px; padding-top: 16px; border-top: 1px solid #f1f5f9; font-size: 12px; color: #94a3b8;">
			Puja Vaahini Platform &bull; Hyderabad Pilot Service
		</div>
	</div>
</body>
</html>`
	_, _ = w.Write([]byte(htmlContent))
}

// renderVerificationResultPage generates an accessible, anti-leakage HTML page showing email verification outcome.
func renderVerificationResultPage(w http.ResponseWriter, success bool, email, role, errMessage string) {
	title := "Email Verified Successfully"
	icon := "✅"
	statusColor := "#15803d"
	headline := "Email Verification Confirmed!"
	contentBlock := fmt.Sprintf(`
		<p style="color: #334155; font-size: 15px; line-height: 1.6; margin-bottom: 20px;">
			Your email <strong>%s</strong> has been verified for your <strong>%s</strong> account.
		</p>
		<div style="background-color: #f0fdf4; border: 1px solid #bbf7d0; border-radius: 8px; padding: 14px; margin-bottom: 24px; text-align: left; font-size: 13px; color: #166534; line-height: 1.5;">
			🔒 <strong>Password Recovery Active:</strong> You can now use this verified email address to recover your account if you ever forget your password.
		</div>
		<p style="color: #64748b; font-size: 13px; line-height: 1.5;">
			You can now safely return to the <strong>Puja Vaahini</strong> app.
		</p>
	`, html.EscapeString(email), html.EscapeString(strings.ToUpper(role)))

	if !success {
		title = "Email Verification Failed"
		icon = "⚠️"
		statusColor = "#b91c1c"
		headline = "Verification Link Expired or Invalid"

		scannerNotice := ""
		if strings.Contains(errMessage, "already been used") {
			scannerNotice = `
			<div style="background-color: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 12px 14px; margin-top: 14px; font-size: 13px; color: #475569; text-align: left; line-height: 1.5;">
				🛡️ <strong>Email Security Notice:</strong> If your corporate email gateway or link protection software (e.g. Defender, Proofpoint) automatically scanned this link, your email address may have already been verified. Please check your account profile in the Puja Vaahini mobile app to confirm.
			</div>`
		}

		contentBlock = fmt.Sprintf(`
			<div style="background-color: #fef2f2; border: 1px solid #fecaca; color: #b91c1c; padding: 14px 16px; border-radius: 8px; margin-bottom: 20px; font-size: 14px; line-height: 1.5;">
				%s
			</div>
			%s
			<p style="color: #64748b; font-size: 13px; line-height: 1.6; margin-top: 16px;">
				Please open the <strong>Puja Vaahini</strong> mobile app, go to your profile, and request a fresh verification link.
			</p>
		`, html.EscapeString(errMessage), scannerNotice)
	}

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<meta name="referrer" content="no-referrer">
	<title>Puja Vaahini — %s</title>
	<style>
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
			background-color: #f8fafc;
			color: #0f172a;
			margin: 0;
			padding: 24px;
			display: flex;
			align-items: center;
			justify-content: center;
			min-height: 100vh;
			box-sizing: border-box;
		}
		.card {
			background: #ffffff;
			max-width: 440px;
			width: 100%%;
			border-radius: 16px;
			border: 1px solid #e2e8f0;
			padding: 32px;
			box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.05);
			text-align: center;
		}
	</style>
</head>
<body>
	<div class="card">
		<div style="font-size: 48px; margin-bottom: 12px;">%s</div>
		<h1 style="color: #ea580c; margin: 0 0 6px 0; font-size: 24px;">🕉️ Puja Vaahini</h1>
		<h2 style="font-size: 18px; color: %s; margin: 0 0 16px 0;">%s</h2>
		%s
		<div style="margin-top: 28px; padding-top: 16px; border-top: 1px solid #f1f5f9; font-size: 12px; color: #94a3b8;">
			Puja Vaahini Platform &bull; Hyderabad Pilot Service
		</div>
	</div>
</body>
</html>`, title, icon, statusColor, headline, contentBlock)

	_, _ = w.Write([]byte(htmlContent))
}
