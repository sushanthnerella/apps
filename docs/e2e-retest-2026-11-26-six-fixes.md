# Retest of six fixes — 26 November 2026 (APKs from 9e37ec9)

## Environment

- Emulator `emulator-5554`, both apps installed from 9e37ec9 (not rebuilt). GPS fix 81.78 16.99.
- Backend `https://pooja-vahini-staging.onrender.com`, `/health` UP, `2.5.0-railway-staging`.
- **Staging was not redeployed.** 9e37ec9 is on `origin/fix/e2e-2026-11-25` (pushed 21:51 IST). From 21:52 to 22:26 IST, `GET /api/v1/bookings?priest_id=…` never returned `scheduled_at`, including for EVENT_SCHEDULED bookings. Staging still runs the old busy check.
- The host lost DNS for about 2 minutes around 22:00. This did not affect the emulator.
- Screenshots: `output/e2e-2026-11-26-retest6/`.
- Starting state: priest Offline, on duty at suskfnk (AVAILABLE), no custom offerings, no open bookings. font_scale 1.0, location_mode 3, both apps in English.

## Results

| # | Fix | Result | Notes | Screenshots |
|---|---|---|---|---|
| 1 | Accepted scheduled ritual >2 h ahead no longer blocks a "Now" request | **FAIL (blocked: staging not redeployed)** | Accepted an EVENT_SCHEDULED booking for 27 Nov 06:00 IST (26babb7a) in the priest app. A "Now" request to the same priest returned `409 {"error":"priest_busy"}`, which is the old behaviour. The priest app could not be tested either: it reads `scheduled_at` from the bookings list, staging does not send it, and with no value the app treats the booking as ongoing. The second check passed: with an ACCEPTED "Now" ritual (b76d69ab), a second "Now" request returned `409 priest_busy` | I1-scheduled-accepted.png |
| 2 | Ghat list "N available now" refreshes while the tab is open | **PASS** | The Pushkaralu tab stayed open while I set duty through the API. BREAK → "0 available now" after about 7 s. AVAILABLE → "1 available now" after about 11 s, measured with UI dumps that take about 4 s each. I did not leave or reopen the tab | I2-BREAK-auto.png, I2-AVAILABLE-auto.png |
| 3 | Offline cold start (saved token) shows a spinner, not the login form | **PASS** | Wi-Fi and data off, cold start: splash, then the header with a spinner, then "Something went wrong / Could not reach Puja Vaahini. Retrying when you are back online…". Behind the dialog there is only the spinner, with no login form. With the network back on, the app opened Services within 15 s and was still signed in. Minor: after I tapped OK, the dialog was still showing 2 s later. Either the tap missed or the next retry showed it again | I3-offline-1s.png, I3-offline-6s.png, I3-after-ok.png, I3-online.png |
| 4 | Ghat step view heading says "Puja is in progress" after the start | **PASS** | Ghat "Now" booking b76d69ab: the priest accepted and entered the code. "Verify and start" gave "Puja is in progress" (server POOJA_STARTED). In the devotee app, Bookings → View booking showed the heading "Puja is in progress" with Requested ✓ Accepted ✓ Started ✓. Before the start, the same screen said "Booking confirmed". I then completed the booking | I4-devotee-view.png, I4-devotee-started.png |
| 5 | My Offerings: Remove asks for confirmation | **PASS** | Added "E2E Retest6" (₹1100, 45 min) in the app. Remove showed "Remove this offering?" with Keep and Remove. Keep closed the dialog and the offering stayed. Remove → "No services yet", and the server returns `[]` | I5-published.png, I5-confirm-dialog.png, I5-after-keep.png, I5-after-remove.png |
| 6 | Home-visit decline: tap, wait about 10 s, tap again | **PASS** | Incoming request eee49c9a. First tap showed "Tap again to decline", which was still there after 10 s. The second tap declined it: the alert closed and the server shows CANCELLED | I6-alert.png, I6-first-tap.png, I6-after-10s.png, I6-second-tap.png |

## New problems / observations

1. **My Offerings fails to load when an offering has `samagri_checklist: null` (low).** I first created an offering through the API without a checklist. Staging stored `"samagri_checklist": null`, and My Offerings then showed "Could not load services. Please try again." Retry did not help. `PriestCustomServicePayload.samagri_checklist` is a non-null `List<String>`, so one such row breaks the whole list. The app always sends a checklist, but rows created by the CMS or a script could trigger this. I deleted that row, and the screen loaded again (I5-load-error.png).
2. Field taps in the "Create Custom Offering" dialog are unreliable while the keyboard is open, because the layout shifts. This is a test-tool quirk, not a confirmed bug. Pressing Back with the keyboard closed dismisses the whole dialog and loses the input.
3. The validation message "Enter a title, valid prices (₹) and a duration in minutes." appears when "Samagri Kit Included Cost" is empty. You have to type 0, because an empty field is not treated as 0. This is cosmetic.

No lines in `adb logcat -b crash`.

## Test data and cleanup

- Offerings: e4791d52 (API, deleted through the API) and "E2E Retest6" (app, removed in the app). The priest has no offerings.
- Bookings:
  - eee49c9a: home visit, created through the API, declined, CANCELLED. Its `pooja_id` was empty because my script failed to read one, and the server still accepted the booking.
  - b76d69ab: ghat "Now", COMPLETED.
  - 26babb7a: ghat scheduled for 27 Nov 06:00, cancelled by the devotee, CANCELLED.
  - Two "Now" requests were refused as `priest_busy` and created no booking.
  - No open bookings for either account.
- The priest's duty status was toggled through the API (BREAK, then back to AVAILABLE) and ends as AVAILABLE at suskfnk. The priest was set back to **Offline** in the app.
- Wi-Fi and data on, font_scale 1.0, location_mode 3, both apps in English. Nothing committed and no code changed.

## To do

- Redeploy staging from 9e37ec9 and retest issue 1: the server accepts a "Now" request while a scheduled ritual is accepted, and the priest app shows the alert.
