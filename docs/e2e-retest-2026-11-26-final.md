# Final retest after staging redeploy — 26 November 2026 (HEAD 48f8e3a)

## Environment

- Emulator `emulator-5554`. Priest APK from 48f8e3a, devotee APK from 35835d1 (not rebuilt). GPS fix 81.78 16.99.
- Backend `https://pooja-vahini-staging.onrender.com`, `/health` UP, `2.5.0-railway-staging` (17:54 UTC).
- **Deploy confirmed** on the first check: offering ada07524 created through the API with `dakshina_fee_minor:110000` came back from `GET /api/v1/priests/custom-services?priest_id=…` with `"dakshina_fee_minor":110000`, `"samagri_kit_price_minor":25000` (previously 0).
- Starting state: priest Offline, event duty AVAILABLE at suskfnk, no offerings, no open bookings. font_scale 1.0, location_mode 3, Wi-Fi on, priest app English.
- Bookings were created through the devotee API (as in the previous run); all priest actions were done in the priest app.
- Screenshots: `output/e2e-2026-11-26-final-retest/`.

## Results

| # | Check | Result | Notes | Screenshots |
|---|---|---|---|---|
| 1a | Scheduled ghat request reaches the open app and can be accepted | **PASS** | a1defbf3, pinda pradanam, 27 Nov 06:00 IST. Alert shown in the running app ("Incoming Puja Dispatch Request!", 27 Nov, 06:00 AM, ₹501). Accept Duty → ACCEPTED. Duty Radar then went back to "Ready for booking requests" | V1-scheduled-accepted.png |
| 1b | "Now" request to the same priest shows the alert without restart | **PASS** | eb7e3b34 → `201 REQUESTED`. With the app left open on Duty Radar (no restart), the alert appeared **8 s** after the request (4:43 left on the timer) | V1-now-alert.png |
| 1c | Accept, start with code, complete | **PASS** | Accept → ACCEPTED card; Start Puja → code 2127 → "Puja is in progress" (POOJA_STARTED); Complete → "Booking status updated.", server COMPLETED | V1-now-accepted.png, V1-now-started.png, V1-now-completed.png |
| 1d | A Now ritual under way still blocks a second Now | **PASS** | With eb7e3b34 ACCEPTED, a second "Now" → `409 {"error":"priest_busy"}`, no booking | — |
| 2 | Fee fields in the offerings list | **PASS** | API minor fields (ada07524): 110000 / 25000. API legacy fields only (e6a3a5ac, `dakshina_fee:501`): `dakshina_fee_minor:50100`, `samagri_kit_price_minor:0`. App-created "E2E_App_Fee" (₹2100, ₹150, 40 min): `dakshina_fee_minor:210000`, `samagri_kit_price_minor:15000`. App lists ₹1100/+₹250, ₹501/+₹0, ₹2100/+₹150 | F2-my-offerings.png, F2-app-form.png, F2-my-offerings-3.png |

### Quick regression

| Check | Result | Notes |
|---|---|---|
| Duty Radar online toggle | PASS | Offline → Online ("Ready for booking requests", server `is_online:true`, priest listed at the ghat) and back to Offline (`is_online:false`). One toggle tap during cleanup did not register; a second tap worked (likely tapped while the screen was still loading) |
| My Offerings loads | PASS | Lists offerings, "No services yet" when empty; Remove confirmation ("Remove this offering?") works |
| Crashes | PASS | `adb logcat -b crash` empty for the whole run |

## New problems

None found. Minor test-harness note: the New Offering dialog shifts up when the keyboard opens, so fixed-coordinate taps land in the wrong field; not a user-facing defect.

## Test data and cleanup

- Bookings: a1defbf3 (ghat scheduled, accepted, cancelled by devotee via API → CANCELLED, full refund), eb7e3b34 (ghat Now, COMPLETED). One "Now" refused as `priest_busy` (no booking). Priest has no open bookings.
- Offerings: ada07524 and e6a3a5ac (API, deleted via API), 73fe258e "E2E_App_Fee" (app, removed in app). API returns `[]`.
- Priest set back to **Offline**; event duty untouched (AVAILABLE at suskfnk). font_scale 1.0, location_mode 3, Wi-Fi on, app language English. No passwords typed in the UI; no screenshot shows a secret. Nothing committed, no code changed.
