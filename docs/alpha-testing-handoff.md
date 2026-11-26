# Alpha testing handoff for Claude

Status from the November 15–17, 2026 reports. Start with the remaining tests below. Use the latest working tree and preserve existing changes.

## Verified

- **Regular booking on Render staging:** devotee selects a service, address, priest and slot, then sends the request. The assigned priest accepts, marks transit and arrival, starts the ritual, enters the completion code, and completes the booking. The devotee sees completion and booking history. One concurrent slot hold check produced one success and one conflict. Chat worked both ways and cancellation released the slot. These reports cover ordinary bookings.
- **Pushkaralu CMS in a local browser:** admin login; create event; Draft → Scheduled → Active; add and hide/show ghats and rituals; approve a priest; see ritual and participation audit entries; toggle booking and card issuance; emergency stop turns off both switches and preserves an existing request.
- **Pushkaralu devotee on Android emulator:** sign in, open live banner, select ghat and priest, see ritual and price, open scheduled date/time choices, submit an immediate request, and see the four-digit code and countdown. Scheduled submission was not exercised.
- **Pushkaralu priest on Android emulator:** the earlier tab crash was fixed and retested. The tab loaded an approved application and price. Tapping Save prices, navigating away and back, and reopening after restart produced no new crash entry. Application and duty fixtures were prepared through API/SQL for the earlier smoke test.
- **Backend automated coverage:** event config and authorization, overlap rules and concurrent accepts, regular slot regression, audit isolation, and booking lifecycle checks. The backend report records 429 passing tests across the full run and reruns. CMS production build, Android compilation, shared tests, and Go vet also passed.
- **Claude's build report:** priest and devotee release lint, release builds for all three Android apps, priest and devotee debug builds, and priest unit tests passed. Fragment 1.6.2 pins are present in both apps. Runtime permission prompts were not verified in that report.

## Remaining tests, in order

1. **Run the real priest setup from the UI.** In a fresh local fixture, schedule an event in CMS. Sign in as a fresh priest, open Pushkaralu, and apply before activation. Confirm the application appears in CMS. Approve it there, reopen or refresh the priest tab, save a changed price, and verify it persists after reopening. Activate the event and enable card issuance. Verify the card appears. Start, change, and end ghat duty from the app. Check that availability in the devotee app follows the duty state and location freshness.
2. **Run an immediate booking across both apps.** Devotee chooses the ghat and priest and sends the request. Priest sees and accepts it. Devotee sees confirmation. Priest marks arrival, verifies that a wrong code cannot start the ritual, then enters the correct code and completes it. Confirm both apps show completion and the agreed price. For a second request, verify cancellation or expiry is reflected accurately.
3. **Submit a scheduled booking from the app.** Pick a future time, send and accept it, and verify the date/time on both apps. Check that an overlapping request is rejected and a non-overlapping one succeeds. The backend already covers overlap rules; this run checks the app wiring.
4. **Retest Android permission prompts on the rebuilt APKs.** Use fresh app data on the test emulator. Check relevant permission allow/deny paths and verify location and background duty updates. Exercise Firebase push and email verification/reset separately if configured for alpha. iOS and broad language/accessibility coverage remain open.

## Environment and reporting

Use isolated local accounts and PostgreSQL for the full Pushkaralu run. Start the updated Go API and separate Next.js CMS against the same backend and apply migrations through 000018. The last verified Render service ran Go backend commit a16c68c and returned 404 for the Pushkaralu config endpoint. Recheck the deployed revision before trying the hosted flow.

For each test, record the environment/build, steps, expected vs. actual result, PASS/FAIL/BLOCKED, and screenshot/log for any failure. Use the app and CMS UI for the steps under test. Label API/SQL fixture setup separately. Restore any temporary local endpoint or network settings afterward.

## Source reports

- docs/emulator-cms-check-2026-11-16.md (includes November 17 priest recheck)
- docs/pushkaralu-fixes-2026-11-16.md
- Handoff workspace output/booking-release-20261115/RETEST.md
- Handoff workspace output/flows-20261115/RESULTS.md
- Claude's Fragment dependency and release lint report, pasted in the chat
