# End-to-end test prompt (devotee + priest apps, normal and Pushkaralu bookings)

Copy everything below the line into the agent.

---

You are testing the Pooja Vahini Android apps end to end on an emulator against the staging backend. Work in the repository at `E:\New folder (5)`. Do not change source code unless a step says so; this is a test pass. Record every result honestly: PASS, FAIL (with the exact error text and a screenshot) or NOT TESTED (with the reason). Never mark a step PASS that you did not actually see work.

## Setup

1. Android SDK is at `%LOCALAPPDATA%\Android\Sdk`. Start the emulator `Pooja_Pilot_Test` (`emulator -avd Pooja_Pilot_Test`), wait for `adb shell getprop sys.boot_completed` to return `1`.
2. Set a GPS fix at Rajahmundry: `adb emu geo fix 81.78 16.99`.
3. Build and install both debug APKs:
   - `.\gradlew :apps:devotee-app:assembleDebug :apps:priest-app:assembleDebug`
   - `adb install -r apps\devotee-app\build\outputs\apk\debug\devotee-app-debug.apk`
   - `adb install -r apps\priest-app\build\outputs\apk\debug\priest-app-debug.apk`
4. Test accounts are in `.staging-test-devotee.env` (repo root, gitignored): `STAGING_TEST_DEVOTEE_PHONE/PASSWORD` and `STAGING_TEST_PRIEST_PHONE/PASSWORD`. Read them from the file; do not print the passwords in your report.
5. Both apps run on the same emulator. Switch with `adb shell monkey -p com.purohit.devotee 1` and `adb shell monkey -p com.purohit.priest 1`. Use `adb shell uiautomator dump` to find element bounds and `adb shell input tap/text` to act. Take a screenshot (`adb exec-out screencap -p > file.png`) at every checkpoint and save them under `output/e2e-<date>/`.
6. After every app switch, check `adb logcat -d -b crash` for new crashes and record them.

A CMS admin (a human) is available for steps marked **[CMS]**. When you reach one, stop and ask them to do it, then continue.

## Part A — Login and session

A1. Devotee app: log in with the devotee account. Expected: Services home screen.
A2. Force-stop the devotee app (`adb shell am force-stop com.purohit.devotee`) and relaunch, three times. Expected: still logged in every time, no "session revoked" dialog.
A3. Priest app: log in with the priest account. Expected: Duty Radar screen, in English (the emulator language is English).
A4. Force-stop and relaunch the priest app. Expected: still logged in.

## Part B — Location

B1. Priest app: turn the Online switch on. Expected: "Online", no location error.
B2. Devotee app: Puja Store tab. Expected: no "Could not get a current location" error (vendors may be empty on staging; that is fine).
B3. Indoor simulation: in the emulator's extended controls, or with `adb shell settings put secure location_mode 2` (battery saving, network only), repeat B2. Expected: a location is still found or a clear message is shown; record which. Restore with `location_mode 3`.

## Part C — Normal priest booking (home visit)

C1. Priest app, Schedule tab: publish an availability slot for tomorrow, 09:00–18:00. Expected: confirmation message.
C2. Devotee app, Services: add one puja (for example Namakaranam) and tap the dispatch button. Enter a test address, pick tomorrow's date, select the slot and the test priest, and send the request.
C3. Priest app: an incoming request alert appears with the devotee name, gotram/nakshatram, address and amount. Accept it.
C4. Priest app, active booking card. Check the step buttons:
   - Only "1. In Transit" is enabled after accept.
   - Tap it; now only "2. Arrived" is enabled and "1" is disabled.
   - Tap "2. Arrived"; only "3. Start Puja" is enabled.
   - Tap "3. Start Puja"; only "4. Complete" is enabled.
   Record FAIL if any earlier step is still tappable.
C5. Devotee app, Bookings → View booking, after each priest step. Expected: status text follows (Accepted, On the way, Arrived, Puja in progress). Check the "Cancel booking" button: visible while Accepted/On the way, hidden once Arrived or Puja in progress.
C6. Devotee: note the 4-digit completion code. Priest: tap "4. Complete", enter a wrong code first (expected: error, booking stays in progress), then the right code (expected: completed).
C7. Devotee: booking shows "Puja completed".
C8. Cancel path: create a second normal booking, have the priest accept, then cancel it from the devotee app while it is Accepted. Expected: cancelled on both sides.
C9. Reject path: create a third booking and have the priest decline it. Expected: devotee sees it was declined.

## Part D — CMS event setup

D1. **[CMS]** Ask the admin to confirm: an event is Active, it has at least one ghat with correct coordinates near Rajahmundry and at least one ritual, and booking is switched on.
D2. Devotee app: the Pushkaralu tab is visible with a LIVE banner and the ghat list. Tap "Sort by nearest". Expected: location used, distance looks sensible (a few km, not thousands). Record the distance shown.

## Part E — Priest Pushkaralu participation

E1. Priest app, Pushkaralu tab. If not yet applied: tap apply. Expected: "Submitted, under review". Set a price for each ritual and save. Expected: "Prices saved".
E2. **[CMS]** Ask the admin to approve the test priest on the Events page (if already approved, skip).
E3. Priest app: reopen the Pushkaralu tab. Expected: "Approved" and an ACTIVE event card with a card ID and valid dates.
E4. Choose the ghat and tap "Start duty". Expected: status chips appear (Available / Performing / Break) with Available selected.

## Part F — Pushkaralu booking (now)

F1. Devotee app, Pushkaralu → ghat. Expected: the test priest appears as available with the price set in E1.
F2. Request → ritual → "Now" → Send request. Expected: "Waiting for the priest to accept", a 4-digit booking code, and an expiry countdown. Record the countdown value.
F3. Priest app: alert appears. Record the priest's countdown value; it should roughly match the devotee's (if the backend fix is deployed). Accept.
F4. Priest active booking card: only "1. Start Puja" and "2. Complete" are shown (no travel/arrived steps). Tap Start Puja. Expected: dialog asks for the devotee's booking code "before you begin" with a "Verify and start" button.
F5. Enter a wrong code first (expected: error, not started), then the devotee's code. Expected: "Puja is in progress".
F6. Devotee: steps show Requested ✓ Accepted ✓ Started ✓.
F7. Priest: tap "2. Complete". Expected: completes without asking for a second code. Devotee: "Ritual completed. Thank you."

## Part G — Pushkaralu edge cases

G1. Expiry: send a "Now" request and do not accept on the priest side. Wait for the countdown to reach zero. Expected: devotee sees "request expired", priest alert closes, nothing is owed.
G2. Devotee cancel: send a request and cancel it from the devotee app while waiting. Expected: cancelled on both sides.
G3. Scheduled: send a "Schedule" request for a future time. Expected: priest receives it; accept; booking shows the scheduled time on both sides. Do not complete it; cancel it afterwards.
G4. Priest busy: with one accepted ghat booking still open, send a second "Now" request. Expected: the priest cannot accept two at once (clear "busy" message), or record what happens.
G5. Break: priest sets "Break". Expected: devotee no longer sees the priest as available at the ghat. Set back to Available.
G6. **[CMS]** Ask the admin to switch booking off for the event. Expected: devotee cannot send new requests (clear message). Ask them to switch it back on.
G7. **[CMS]** Ask the admin to press Emergency stop, then check that an already-accepted booking is still visible and completable on both sides, and that no new requests can be sent. Ask the admin to undo it afterwards.
G8. **[CMS]** Ask the admin to check that the bookings from Part F and G appear in the CMS with the right status.

## Part H — Languages and display

H1. In each app, switch language to Telugu and to Hindi from the header. Open the Pushkaralu screens and the booking card. Record any text that stays in the wrong language or is clipped.
H2. Set the emulator font size to the largest (`adb shell settings put system font_scale 1.3`), reopen the main screens, record clipping. Reset to `1.0`.

## Report

Write `docs/e2e-run-<YYYY-MM-DD>.md` with:
- Environment (emulator, APK build time, backend URL).
- A table: step ID, result (PASS / FAIL / NOT TESTED), notes, screenshot file.
- A list of every bug found, with steps to reproduce, expected vs actual, and the crash log line if any.
- What was not tested and why.

Do not commit, push, or change staging data beyond the test accounts and the bookings this run creates. Cancel any booking you leave open.
