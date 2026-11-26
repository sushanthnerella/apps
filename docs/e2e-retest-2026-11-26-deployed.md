# Retest after staging redeploy — 26 November 2026 (HEAD 35835d1)

## Environment

- Emulator `emulator-5554`, both apps from 35835d1 (already installed, not rebuilt). GPS fix 81.78 16.99.
- Backend `https://pooja-vahini-staging.onrender.com`, `/health` UP, `2.5.0-railway-staging`.
- **Deploy confirmed** at 17:29 UTC: `GET /api/v1/bookings?priest_id=…` now carries `scheduled_at` on EVENT_* bookings (e.g. 26babb7a `2026-11-27T00:30:00Z`). INSTANT_DISPATCH rows omit it.
- Starting state: priest Offline, on duty at suskfnk (AVAILABLE), no offerings, no open bookings. font_scale 1.0, location_mode 3, Wi-Fi on, both apps in English.
- Screenshots: `output/e2e-2026-11-26-deployed-retest/`.

## Results

| # | Check | Result | Notes | Screenshots |
|---|---|---|---|---|
| 1a | Server: "Now" request while an accepted scheduled ritual is >2 h ahead | **PASS** | Scheduled ghat booking b0721e92 for 27 Nov 06:00 IST, accepted in the priest app. "Now" request 487f0a07 to the same priest → `201 REQUESTED` (no `priest_busy`) | V1-scheduled-accepted.png |
| 1b | Priest app shows the incoming alert | **FAIL** | With the priest app open on Duty Radar (showing the accepted 27 Nov card), no alert appeared within ~40 s. After force-stop and relaunch the alert appeared within ~8 s ("Incoming Puja Dispatch Request!", 3:24 left). Cause (`PriestApp.kt` ~line 168): accepting a booking sets `hasActiveBooking = true`, and while it is true the poll loop only refreshes that one booking and never fetches REQUESTED bookings. The new `scheduled_at` check only runs in the `!hasActiveBooking` branch, so it helps after a cold start but not after accepting in-session | V1-now-alert.png (no alert), V1-now-alert-after-restart.png |
| 1c | Ritual under way still blocks a second "Now" | **PASS** | With 487f0a07 ACCEPTED, a second "Now" → `409 {"error":"priest_busy"}` | V1-now-accepted.png |
| 2 | My Offerings with a `samagri_checklist`-less row | **PASS** | Offering 51ab8f3f created through the API with no checklist. API returns `"samagri_checklist":[]`. App lists "E2E NullChecklist", ₹1100, +₹0, 45 mins (no "Could not load services"). Deleted through the API → `[]` | V2-my-offerings.png |
| 3 | `POST /api/v1/bookings` with empty `pooja_id` | **PASS** | `""` and `"  "` both → `400 {"error":"bad_request","message":"pooja_id is required"}`; no booking created. Devotee app home visit (Namakaranam, 27 Nov 2–6 PM) → "Request sent to priest" (19e6a3b5), then cancelled in the app → CANCELLED | V3-homevisit-sent.png |
| 4a | Blank samagri cost accepted as 0 | **PASS** | "E2E Deployed 1126", ₹1100, 45 min, samagri empty → published, "Samagri Kit: +₹0" | V4-blank-samagri-published.png |
| 4b | Back with text typed keeps the form open | **PASS** | First Back closed the keyboard, second Back left the dialog open with the title intact | V4-back-with-text.png |
| 4c | Back on an empty form closes it | **PASS** | | V4-back-empty-closed.png |

### Quick regression

| Check | Result | Notes |
|---|---|---|
| Devotee login | PASS | Cold start restores the session and opens Services; API login 200 for both accounts. In-app password login was not repeated |
| Services load | PASS | Pooja list and the Pushkaralu banner ("LIVE · Day 9 of 24") | 
| Priest Duty Radar | PASS | Online/Offline toggle works, "Ready for booking requests", server `is_online` follows |
| Ghat Complete | PASS | 487f0a07: Start Puja → code → "Puja is in progress" (POOJA_STARTED) → Complete → COMPLETED, "Booking status updated." |

Remove confirmation ("Remove this offering?") still works. No lines in `adb logcat -b crash`.

## New problems

1. **Priest misses new requests after accepting a scheduled booking in the same session (medium).** See 1b. Fix direction: in the poll loop, keep polling REQUESTED bookings while the active booking is an ACCEPTED one scheduled more than an hour out, instead of switching entirely to the single-booking refresh.
2. **`dakshina_fee_minor` is 0 in the custom-services list (low, observation).** Both the API-created (`dakshina_fee_minor:110000` sent) and the app-created offering come back with `"dakshina_fee_minor":0` and `"dakshina_fee":1100`. The app shows ₹1100 correctly, so it reads the legacy field. Anything reading the minor field would see 0.

## Test data and cleanup

- Bookings: b0721e92 (ghat scheduled, accepted, cancelled by devotee via API → CANCELLED), 487f0a07 (ghat Now, COMPLETED), 19e6a3b5 (home visit, cancelled in app → CANCELLED). One "Now" refused as `priest_busy` (no booking); two empty-`pooja_id` posts refused (no booking). No open bookings.
- Offerings: 51ab8f3f (API, deleted via API), ce39b79f "E2E Deployed 1126" (app, removed in app). Priest has none.
- Slots: none created.
- Priest set back to **Offline**; event duty AVAILABLE at suskfnk. font_scale 1.0, location_mode 3, Wi-Fi on, both apps English. Nothing committed, no code changed.
