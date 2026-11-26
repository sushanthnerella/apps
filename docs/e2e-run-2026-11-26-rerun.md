# E2E run — 26 November 2026 (rerun, APKs from 77838cd + uncommitted client fix)

## Environment

- Host: Windows. One emulator: `emulator-5554` (sdk_gphone64_x86_64, 1080x2400), running both apps. DNS worked at the start, so no AVD restart was needed.
- APKs: devotee and priest debug builds from HEAD 77838cd plus the uncommitted `PurohitApiClient.kt` change (null list → empty). Both installed 26 Nov 21:00 IST. Not rebuilt.
- Backend: `https://pooja-vahini-staging.onrender.com`. `/health` UP, `2.5.0-railway-staging`, Postgres and Redis connected.
- Deploy state: `DELETE /api/v1/priests/custom-services/{id}` **is deployed** (no auth → 401; an unknown path → 404). At the start, `GET /priests/custom-services?priest_id=<test priest>` returned `null`. At the end of the run it returned `[]`, so the uncommitted backend change (nil → `[]`) appears to have been deployed during the run.
- GPS fix: Rajahmundry (`adb emu geo fix 81.78 16.99`).
- Screenshots: `output/e2e-2026-11-26-rerun/`.
- No CMS admin, so every [CMS] step is NOT TESTED.

## Pre-run check

- No open bookings for the devotee or the priest. The priest had no custom offerings (`null`), so "E2E Offering 1126" from the earlier run is already gone.
- The priest was **Offline** and on duty at ghat suskfnk (Available).

## Results

| Step | Result | Notes | Screenshot |
|---|---|---|---|
| A1 | PASS | Already signed in from before the run. Opens on Services | A1-devotee-launch.png |
| A2 | PASS | 3 force-stops, still signed in, no "revoked" dialog, no crash | A2-devotee-relaunch3.png |
| A3 | PASS | Priest opens on Duty Radar, in English (already signed in) | A3-priest-launch.png |
| A4 | PASS | Force-stop, still signed in | A4-priest-relaunch.png |
| B1 | PASS | Online, "Location updated. Devotees nearby can find you." Server `is_online:true` | B1-priest-online.png |
| B2 | PASS (adapted) | There is no Puja Store tab. Pushkaralu "Sort by nearest": 0.0 km, no error | B2-D2-sort-nearest.png |
| B3 | PASS | `location_mode 2`, cold start: location still found, 0.0 km. Restored to 3 | B3-location-mode2.png |
| C1 | PASS (adapted) | 27–30 Nov already had overlapping slots, so I filled the gap on 27 Nov: "Slot published: 11:00 AM – 12:00 PM IST" | C1-slot-published.png |
| C2 | PASS | Namakaranam, address, 27 Nov, 11–12 slot, priest, review, "Request sent to priest" | C2-*.png |
| C3 | PASS | Devotee name, gotram/nakshatram, address, ₹3216, 2:47 countdown | C3-priest-alert.png |
| C4 | PASS | Only the next step enabled each time (1→2→3→4). Checked the `enabled` flags in the UI dump | C4-step0.png, C4-after-*.png |
| C5 | PASS | Accepted → On the way → Arrived → Puja in progress, each shown 4–5 s after returning to the devotee app. "Cancel booking" shown while Accepted/On the way and hidden from Arrived. **No "Could not refresh" error on any of the 4 returns** | C5-devotee-*.png |
| C6 | PASS | Wrong code: "Incorrect code. Check the devotee's code and try again.", booking stays POOJA_STARTED. Field empty when reopened. Right code → COMPLETED | C6-wrong-code.png, C6-completed.png |
| C7 | PASS | "Puja completed" | C7-devotee-completed.png |
| C8 | PASS | Second booking (2–6 PM) accepted. Devotee cancelled while Accepted (reason required) → CANCELLED on both sides. Priest back to "Ready" | C8-*.png |
| C9 | PASS | Priest declined (the "Tap again to decline" confirm disarms after a few seconds; it worked with two quick taps) → CANCELLED. Devotee: "The priest declined. Choose another priest or time." 7 s after returning to the app | C9-*.png |
| D1 [CMS] | NOT TESTED | No admin. API: event "godavari pushkarlu" ACTIVE, `booking_enabled:true` | — |
| D2 | PASS | LIVE · Day 9 of 24, ghat list, 0.0 km | D2-pushkaralu-tab.png |
| E1 | PASS (existing) | Already approved. "Prices saved" for pinda pradanam ₹501 | E1-prices-saved.png |
| E2 [CMS] | NOT TESTED | Needs an admin (the priest is already approved) | — |
| E3 | PASS | Approved, Active card PV-7nfttixrznsk7hepdor5hpz7, 2026-11-18 – 2026-12-11 | E3-priest-pushkaralu.png |
| E4 | PASS (existing) | On duty at suskfnk. Available / Performing / Break chips | E4-duty.png |
| F1 | PASS | Priest listed, "from ₹501" | F1-ghat-priests.png |
| F2 | PASS | "Waiting for the priest to accept", code shown, "Expires in 4:55" | F2-waiting.png |
| F3 | PASS | Priest showed 4:48 about 7 s later, which matches. "Sankalpam: pinda pradanam". Accepted | F3-priest-alert.png |
| F4 | PASS | Only "1. Start Puja" and "2. Complete". Dialog asks for the booking code "before you begin", with "Verify and start" | F4-steps.png, F4-dialog.png |
| F5 | PASS | Wrong code: "Incorrect code…", server ARRIVED. Right code: "Puja is in progress", POOJA_STARTED | F5-wrong.png, F5-started.png |
| F6 | **PASS** (previously PARTIAL) | Ghat screen: Requested ✓ Accepted ✓ Started ✓. **Bookings → View booking also opens the same step view**. Small issue: the heading still says "Booking confirmed" while the ritual is in progress | F6-devotee-ghat-view.png, F6-from-bookings.png |
| F7 | **PASS** (previously FAIL) | Priest app force-stopped and relaunched while POOJA_STARTED. "2. Complete" completed the booking at once, with **no code dialog** (COMPLETED). Devotee: "Ritual completed. Thank you." | F7-after-restart.png, F7-priest-complete.png, F7-devotee-completed.png |
| G1 | PASS | Now request left unanswered. Priest alert closed by about 4:50. Server CANCELLED, escrow PAY_AT_GHAT. Devotee: "The priest did not respond in time. Please choose another priest." | G1-*.png |
| G2 | PASS | Cancelled while waiting (confirm dialog): "This request was cancelled.", server CANCELLED, priest shows "Ready" | G2-*.png |
| G3 | PASS | Scheduled 28 Nov 06:00 AM shown in the priest alert, on the priest card and on the devotee screen. Accepted, then cancelled from the devotee app → CANCELLED | G3-*.png |
| G4 | PASS (see note) | With the 28 Nov 06:00 scheduled booking accepted, a "Now" request was refused: "This priest already has a ritual at that time. Choose another time or priest." Note: the open booking was 2 days later, so "at that time" isn't accurate. The busy check seems to block on any open booking, not just one at an overlapping time | G4-now-while-scheduled.png |
| G5 | PASS (minor) | Break → devotee "0 available now". Back to Available → API `available_count:1` at once, but the devotee tab kept showing "0 available now" until I left the tab and opened it again | G5-*.png |
| G6–G8 [CMS] | NOT TESTED | No admin | — |
| H1 | PASS (partial coverage) | Both apps switch to Telugu and Hindi from the header (tabs, headings, buttons translated). I saw no new wrong-language text. My Pushkaralu screen taps landed on other tabs, so the Pushkaralu and booking-card screens were not checked in Telugu/Hindi this run. Both apps were set back to English | H1-*.png |
| H2 | PASS | 1.3×: "Pushkaralu" stays on one line. "Duty / Radar" and "My / Offerings" wrap, as designed. Reset to 1.0 | H2-*.png |

## Targeted fix checks (77838cd + uncommitted fix)

| Item | Result | Evidence |
|---|---|---|
| My Offerings with an empty list | **PASS** | Server returned `null`. The screen shows "No services yet", with no "Could not load services" (S1-my-offerings-empty.png). Hindi empty state is also fine |
| Add an offering | PASS | "E2E Rerun 1126", ₹1100, 45 min. Listed with **"Samagri Kit: +₹0"** (the old 1500 default is fixed). Server `samagri_kit_price_minor:0` (S3-offering-published.png) |
| Remove an offering | **PASS** | "Remove" → back to "No services yet". Server returns `null`/`[]` for the priest afterwards. The DELETE endpoint is deployed on staging. UX note: Remove acts on one tap, with no confirmation (S4-remove-tap.png) |
| Offline cold start, devotee | **PASS (minor UX)** | Wi-Fi and data off: "Something went wrong / Could not reach Puja Vaahini. Retrying when you are back online…". After OK, the **login form** shows underneath while offline. With the network back on, it went to Services by itself within about 10 s and stayed signed in (R1-*.png) |
| Offline cold start, priest | **PASS** | "Connection Disconnected" shown. With the network back on, it **recovered to Duty Radar by itself within about 10 s**, still signed in (R2-*.png) |
| Priest "Try Again" | **PASS** | Offline: Try Again stays on "Connection Disconnected" (no login screen). Online: Try Again → Duty Radar, signed in (R3-*.png) |
| Only a 401 logs out | PASS (indirect) | No unexpected logouts in any of the offline tests. The API token I created at the start still worked after a later in-app login, so a new login does not revoke older sessions |
| Devotee tracking after background | **PASS** | 4 returns during C5 and 1 after the decline: never "Could not refresh booking status". Status was current within 4–7 s |
| Ghat Complete after priest restart | **PASS** | See F7 |
| EVENT_* from Bookings → step view | **PASS** | See F6 |
| Event config after login | **PASS** | Logged out, logged in inside the app, opened Pushkaralu: "Serve at godavari pushkarlu / Approved" loaded about 7 s after opening the tab. No "Video calls are not available" placeholder (X-priest-pushkaralu-after-login.png) |

## Bugs / observations in this run

No failures. Minor items:

1. **Busy check ignores time (low).** An accepted scheduled ghat booking on 28 Nov 06:00 blocks a "Now" request on 26 Nov with "already has a ritual at that time". The busy check probably looks for any open booking for the priest, not one at an overlapping time. Check that this is intended.
2. **Ghat availability count goes stale (low).** After the priest went back from Break to Available, the devotee ghat list kept "0 available now" until the tab was opened again. The count isn't refreshed while the list is on screen.
3. **Devotee offline start shows the login form (low, UX).** The session is kept and restored automatically, but while offline the user sees a login form behind the retry message and may try to sign in again.
4. **Ghat step view heading says "Booking confirmed" while Started ✓ (cosmetic).**
5. **My Offerings "Remove" has no confirmation (cosmetic/UX).**
6. **Home-visit decline confirm disarms quickly (cosmetic):** a second tap a few seconds later shows nothing, so the priest may think the decline worked. Already noted in the previous run.

No lines in `adb logcat -b crash` during the run.

## Not tested, and why

- [CMS] D1, E2, G6, G7, G8: no CMS admin.
- H1 Pushkaralu screens and priest booking card in Telugu/Hindi: my taps landed on other tabs. Only the main screens were checked.
- Email verification: neither account has an email.

## Cleanup and test data

- Bookings created: f68d1900 (home visit, COMPLETED, 27 Nov 11–12), e8fafca8 (C8, devotee-cancelled), cc4ae8f8 (C9, declined), e07beba4 (ghat Now, COMPLETED), 1e2762cd (G1, expired), 4865d985 (G2, cancelled), 6cc7631c (G3 scheduled, cancelled). One Now request was refused as busy (G4), so it created no booking. **No open bookings remain.**
- Slot added: 27 Nov 11:00 AM – 12:00 PM (now used by f68d1900).
- Offering "E2E Rerun 1126" (bc15c9d3…) created and removed. The priest has no active offerings.
- Priest signed out and back in inside the app once (event-config check). Set back to **Offline** (as found), still on duty at suskfnk (Available).
- `font_scale` 1.0, `location_mode` 3, Wi-Fi and data on, both apps in English.
- Nothing committed. No source files changed.
