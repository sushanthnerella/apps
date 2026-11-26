# E2E run — 25 November 2026, deployed staging backend

## Environment

- Host: Windows, about 3.8 GB of RAM free at the start. One emulator only: `emulator-5554` (Android 35, 1080x2400), running both apps.
- APKs: devotee and priest debug builds from the current working tree (includes commit e924b9f). Built 25 Nov 21:20 and installed at the start of the run.
- Backend URL: `BackendConfig.kt` uses `STAGING` → `https://pooja-vahini-staging.onrender.com`. `/health` returned UP, `2.5.0-railway-staging`, with Postgres and Redis connected.
- GPS fix: Rajahmundry (16.99, 81.78).
- Screenshots: `output/e2e-2026-11-25-deployed/`.

## Is the e924b9f backend fix on staging?

| Check | Result |
|---|---|
| `GET /api/v1/priests/slots` for an empty date | Returns `[]` with HTTP 200. **Deployed.** The app shows "No slots published for this date yet." |
| Unanswered home-visit request expires | Bookings now carry `request_expires_at` (3 min). Unanswered requests moved to CANCELLED about 20 s after the deadline, and the slot became bookable again. **Deployed.** |
| Priest decline: REQUESTED → declined | The server accepts it. A priest `POST /bookings/status` with `CANCELLED` returns 200 `new_status: CANCELLED`. The **app** still doesn't get the decline through (see bug D1). |
| Service matching (bug 2) | `/priests/nearby?service=test` still returns 0 priests. **Not deployed**, or the migration has not run. |

## Results

| Step | Result | Notes | Screenshot |
|---|---|---|---|
| A1 | PASS | Devotee opens on Services | A1-devotee-launch.png |
| A2 | PASS | 3 force-stops, still logged in, no "revoked" dialog | A2-devotee-relaunch3.png |
| A3 | PASS | Priest opens on Duty Radar, in English | A3-priest-duty-radar.png |
| A4 | PASS | 3 force-stops, still logged in, no crash loop | A4-priest-relaunch3.png |
| B1 | PASS | Online. "Location updated. Devotees nearby can find you." shown. Priest appears in `/priests/nearby` at 0 km | B1-priest-online-location.png |
| B2 | PASS (adapted) | The Puja Store tab no longer exists in the nav, so I tested Pushkaralu "Sort by nearest" instead. No location error, 0.0 km | B2-D2-sort-nearest.png |
| B3 | PASS | With `location_mode 2`, location still found (0.0 km). Restored to 3 | B3-location-mode2.png |
| C1 | PASS | Published 29 Nov 09:00–18:00 from the UI; "Slot published" shown. 26 Nov already had the prior run's slot. Fixture slots for 30 Nov were added via API | C1-slot-published.png |
| C2 | PASS | Namakaranam booking: address, date, slot, priest, review, send | C2-request-sent.png |
| C3 | PASS | Alert shows devotee, gotram/nakshatram, address, ₹3216 and a 2:50 countdown (matches the new 3-min expiry). "Sankalpam:" line is empty | C3-priest-alert.png |
| C4 | PASS | Only the next step is enabled each time (1→2→3→4) | C4-* |
| C5 | PASS | On the way → Arrived → Puja in progress. Accepted seen in C8. Cancel shown while Accepted/On the way, hidden from Arrived | C4-C5-*, C5-C8-devotee-accepted.png |
| C6 | PASS | Wrong code: "Incorrect code…", booking stays in progress. Reopened dialog has an empty field (bug 3 fixed). Right code completes | C6-* |
| C7 | PASS | Devotee Bookings shows "Completed" | C7-devotee-completed.png |
| C8 | PASS | Devotee cancelled while Accepted (reason required). Cancelled on both sides | C8-* |
| C9 | **FAIL** | Real double tap ("Tap again to decline"). The server stays REQUESTED and the alert comes back on the next poll until the request expires. The same call made by hand works, and the devotee then sees "The priest declined…" after about 75 s | C9c-decline-armed.png, C9c-priest-after-decline.png |
| Unanswered normal request expiry | PASS (server) | CANCELLED about 20 s after `request_expires_at`. Slot freed. The devotee is told "The priest declined", which is wrong for an expiry | C9-expiry-devotee.png |
| D1 | NOT TESTED (needs admin) | Existing state: event "godavari pushkarlu" ACTIVE, `booking_enabled: true`, ghat suskfnk, ritual "pinda pradanam" | — |
| D2 | PASS | LIVE banner, ghat list, sort by nearest shows 0.0 km | B2-D2-sort-nearest.png |
| E1 | PASS (existing state) | Already approved. "Prices saved" for pinda pradanam ₹501 | E1-prices-saved.png |
| E2 | NOT TESTED (needs admin) | Already approved | — |
| E3 | PASS | Approved; Active card PV-7nfttixrznsk7hepdor5hpz7, 2026-11-18 – 2026-12-11 | E3-priest-pushkaralu.png |
| E4 | PASS (existing state) | Already on duty at suskfnk. Available / Performing / Break chips shown | E4-duty.png |
| F1 | PASS | Priest available at the ghat, "from ₹501" | F1-ghat-priests.png |
| F2 | PASS | "Waiting for the priest to accept", code 2944, "Expires in 4:58" | F2-waiting.png |
| F3 | PASS | Priest countdown 4:51, about 10 s after the devotee's 4:58. Accepted | F3-priest-alert.png |
| F4 | PASS | Only "1. Start Puja" and "2. Complete". The dialog asks for the booking code "before you begin", with a "Verify and start" button | F4-steps.png |
| F5 | PASS | Wrong code rejected; right code → "Puja is in progress" | F5-* |
| F6 | PARTIAL | 12 s after the start, the devotee still showed "Waiting for the priest to accept". It refreshes slowly (see C9: about 75 s) | F6-devotee-started.png |
| F7 | PASS | Complete needs no second code. Devotee: "Ritual completed. Thank you." | F7-* |
| G1 | PASS | After 5 min: devotee "The priest did not respond in time…", server CANCELLED, no priest alert | G1-* |
| G2 | PASS | Cancelled while waiting: "This request was cancelled.", no priest alert | G2-* |
| G3 | PARTIAL | 28 Nov 07:00 shown in the priest alert and on the devotee confirmation (fixed). The priest's active booking card doesn't show the date/time. Cancelled afterwards | G3-* |
| G4 | PARTIAL | With the scheduled booking accepted, a "Now" request was created (different time, so no 409 clash). The priest gets no alert while busy, so the request just waits to expire. Cancelled from the devotee | G4-* |
| G5 | PASS | Break → devotee sees "0 available now"; back to Available → "1 available now" | G5-* |
| G6–G8 | NOT TESTED (needs admin) | No CMS admin available | — |
| H1 | PASS (minor leftovers) | Telugu and Hindi in both apps: UI text translated. Still in English: CMS/priest data only ("pinda pradanam", priest custom offerings "test"/"Sri Venkateswara…", user profile values), "9:00 AM – 10:00 AM … IST" in the Hindi schedule hint, and the Profile "🌐 Change Language" label (priest, Hindi) | H1-* |
| H2 | FAIL (minor) | At 1.3×, two-line bottom labels in both apps ("Pushkaralu", "Duty Radar", "My Offerings"), and the LIVE banner wraps. No clipping beyond that. Reset to 1.0 | H2-* |

## Retest of the 25 Nov bugs

| Bug | Status |
|---|---|
| 1 Slots `null` | **Fixed** (deployed) |
| 2 Service matching | **Still open** on staging (fix not deployed / migration not run) |
| 3 Code field keeps the old code | **Fixed** |
| 4 No location feedback | **Fixed** |
| 5 / 9 Decline doesn't reach the devotee | **Still open.** The app decline doesn't reach the server; the server path works |
| 6 "Unable to load booking" at launch | Not reproduced (the user stayed logged in; the logged-out offline case was not retested) |
| Normal request expiry (11) | **Fixed** (deployed) |
| 10 Priest crash loop | Not seen: 3 force-stops while on duty, no crash |
| 12 Expiry wording (ghat) | **Fixed**: "The priest did not respond in time" |
| Scheduled date/time on devotee | **Fixed** |
| 15 Accept from Pushkaralu tab | **Fixed**: priest is taken to Duty Radar |

## Bugs found in this run

1. **D1: Priest decline still doesn't reach the server (high).**
   - Steps: send a home-visit request, then on the priest alert tap Decline and "Tap again to decline".
   - Expected: the booking is CANCELLED and the devotee sees "declined".
   - Actual: the booking stays REQUESTED, the alert reappears on the next poll, and the request only ends at expiry.
   - The server accepts the identical call made by hand (200). The `onReject` code captures the ID correctly, so the request is probably failing or not being sent. Its `Result` is ignored, so nothing is shown or logged. Next step: log or show the failure in `onReject` (`PriestApp.kt` ~603).
   - No crash line.
2. **D2: An expired home-visit request is shown as "The priest declined. Choose another priest or time."** It should say the request expired and nothing is owed (as the ghat flow now does).
3. **D3: Devotee status refreshes slowly.** It took about 75 s to show a decline and more than 12 s to show "Started". Once, after returning from the background, it showed "Could not refresh booking status. Check your connection." until the next screen change.
4. **D4: The priest Pushkaralu tab can disappear and show "Live Studio" instead.**
   - Seen after the font-scale change and again after a cold start; it lasted at least 30 s, while `/priests/events/config` returned the ACTIVE event.
   - `PriestApp.kt:141` fetches the config once and then waits 10 minutes. If the first call fails, nothing retries. It should retry on failure.
5. **D5: The priest alert shows an empty "Sankalpam:" line** when the devotee has no sankalpam purpose. Hide the line when it is empty.
6. **D6: The priest's active booking card doesn't show the scheduled date/time** for a scheduled ghat booking. It is shown only in the alert.
7. **D7: A request to a busy priest gets no feedback** (G4). The priest gets no alert while busy, and the devotee waits until expiry.
8. **Bug 2 is still open on staging** (service matching), as listed above.

## Fix applied after the run: priest decline (D1)

Made after the run ended, in the working tree. Not committed. It compiles (`:apps:priest-app:compileDebugKotlinAndroid`), but it has **not been retested on a device**, because the run's APK predates it.

**Root cause (app side).** The backend allows a priest to move a booking from REQUESTED to CANCELLED (`backend/internal/lifecycle/fsm.go`, which allows the priest role), and the same call made by hand returned 200. The priest app's `onReject` in `PriestApp.kt` had three problems:
1. It sent `updateBookingStatus(..., "CANCELLED")` in a fire-and-forget coroutine and ignored the `Result`, so a failure was never shown or logged.
2. It closed the alert and cleared `activeBookingId` straight away, before the server answered.
3. The 5-second poll then saw the booking still REQUESTED, with an ID different from the now-empty `activeBookingId`, and showed the same alert again. This repeated until the server expired the request.

A related problem was in `RapidoIncomingAlertModal.kt`. When the countdown reached zero on a request without `request_expires_at`, it called `onReject()`, so the app **declined automatically after 45 s**. That conflicted with the server's own 3-minute expiry.

**Changes:**
- `apps/priest-app/src/commonMain/kotlin/com/purohit/priest/PriestApp.kt`
  - A new `dismissedRequestIds` set. The poll skips REQUESTED bookings in this set, so a request that was declined or has lapsed is not shown again while the server catches up.
  - `onReject` now sets `incomingBusy`, which disables the buttons, and waits for the server's reply:
    - On success: the ID goes into `dismissedRequestIds` and the alert closes.
    - On failure that means the booking is already gone (a terminal state, `request_expired` or HTTP 404): the alert closes.
    - On any other failure: the alert stays open with "Could not decline this booking. Check your connection and try again." (English, Telugu or Hindi), and the priest can retry.
  - `onExpired` also adds the ID to `dismissedRequestIds` before it closes the alert.
- `apps/priest-app/src/commonMain/kotlin/com/purohit/priest/ui/dispatch/RapidoIncomingAlertModal.kt`
  - When the countdown reaches zero, the alert always calls `onExpired()`, which just closes it. It never declines automatically now; the server expires unanswered requests itself.
  - The default `onExpired` changed from `onReject` to a no-op.

**Retest needed:**
1. Rebuild and install the priest APK.
2. Repeat C9. Expected: after "Tap again to decline", the booking is CANCELLED on the server, the alert does not come back, and the devotee sees the decline.
3. Leave one request unanswered. Expected: the alert closes at zero without declining, and the server expires the booking after about 3 minutes.
4. Decline with the network off. Expected: an error is shown and the alert stays open.

## Other open items (not from this run)

- **Verification email delivery (Brevo SMTP).** On the Render deployment, the backend loaded the Brevo SMTP settings, but a verification-email request returned HTTP 502 "email delivery failed". Brevo showed no sending log for that attempt, and the logs don't say whether the connection, authentication or Brevo account setup failed. Next step: add server-side diagnostics that record which SMTP stage failed (dial, TLS, AUTH, MAIL FROM/RCPT, DATA) without logging the key, then retry.

## Suggested fix order

1. Retest the D1 decline fix on a device, then commit.
2. Deploy the bug 2 service-matching fix to staging and run its migration.
3. D2: show expiry wording for expired home-visit requests (match the ghat flow).
4. D4: retry the event-config fetch with backoff when it fails.
5. D3: speed up devotee booking-status refresh, and clear the "Could not refresh" message on the next success.
6. D5, D6, D7: smaller UI items.
7. Brevo SMTP diagnostics, then retest verification email with an account that has an email.
8. Run D1, E2 and G6–G8 with a CMS admin.

## Verification email

`/api/v1/user/email` exists. `GET` for the test devotee returned 200 `{"email":"","email_verified":false,"role":"devotee"}`. The account has no email, so the POST was **skipped** as instructed; no address was added.

## Not tested, and why

- D1, E2, G6, G7, G8: these need a CMS admin, who was not available.
- B2 as written: the Puja Store tab is no longer in the devotee nav, so location was checked on the Pushkaralu tab.
- Bug 6, logged-out offline launch: not set up, because it needs a logout or a network cut.

## Cleanup and test data

- All bookings from this run are COMPLETED or CANCELLED. No open bookings remain. The G3 scheduled booking was cancelled via the devotee API.
- Slots added: 29 Nov 09:00–18:00 (used and completed), plus 30 Nov 09:00–10:00, 11:00–12:00 and 13:00–14:00 (API fixtures).
- `font_scale` is 1.0 and `location_mode` is 3. Both apps are back in English.
- The priest is still Online and on duty at ghat suskfnk, as at the start.
