# Pooja Vahini redesign: goals and status

Status as of 14 November 2026. This is a handoff for the existing Devotee and Poojari apps in `E:\New folder (5)`. The redesign is **in progress**: work listed as implemented below is not a claim that every screen has passed review in every theme, language, or accessibility setting.

## Goal

Redesign the existing apps using the supplied temple-inspired concept while retaining one Devotee app and one Poojari app. Both apps should provide an easy-to-find, device-persistent theme selector with three choices:

1. **Royal Heritage:** maroon, ivory, and muted gold.
2. **Calm Modern:** deep teal, warm white, and saffron.
3. **Temple Earth:** forest green, sandstone, and terracotta.

Use the supplied golden rising-sun logo, keep the existing flows and data, and make the Devotee experience welcoming and booking-focused while prioritizing requests, availability, and schedule for Poojaris. Support the existing languages, readable and scalable text, accessible controls, and lightweight motion that respects reduced-motion settings. Display real data and honest loading, empty, error, and success states; never present invented production data as real.

Preserve existing work and functionality. Do not publish the apps or deploy the backend. Backend changes should be limited to necessary fixes, with tests and an explanation.

## Implemented work

- Theme palettes, an Appearance selector, previews, and local theme persistence were added to both apps. Theme selection and persistence after restart were exercised on Android for each app.
- The supplied sun image is used in app branding and loading UI. Theme-aware styling and layouts have been introduced across existing screens, but screen-by-screen completion remains open.
- Accessibility and layout improvements include larger targets in parts of the booking flow, a scrollable incoming-request dialog, clearer error feedback, and improvements for enlarged text. Reduced-motion checks were added for selected animations; a full accessibility review is still needed.
- Some English, Telugu, and Hindi text was added or improved. Translation coverage is not yet complete.
- Several fabricated catalog, service, colleague, and materials fallbacks were removed. A complete audit is still needed, including the backend address fallback described below.
- Focused fixes and tests were added for chat request serialization, schedule refresh ordering, duty/location updates, and nearby-priest responses without a phone number.
- A backend fix makes acceptance verify a booking's already-reserved slot instead of guessing its time. An isolated regression and targeted slot-binding, cancellation-release, and concurrent-reservation tests passed. Unslotted bookings still follow the legacy path.
- Both Android debug APK build tasks and configured Android unit-test tasks succeeded on the last run; Gradle reported most tasks up to date.

## What has been tested

- In a prior isolated Android emulator journey, a Devotee selected a service, address, date, priest, and matching slot and submitted a booking. The Poojari accepted it and advanced it through arrival and completion; the Devotee saw the resulting status. The specific slot used was 17:30–19:30.
- Chat messages were sent in both directions in isolated testing. Poojari online status and emulator location were exercised, and a schedule slot was previously published through the UI.
- All three themes were selected and their persistence across app restarts checked on Android. This was not a visual audit of every screen.
- The full UI booking journey was not rerun after the reserved-slot backend fix. Existing end-to-end scripts have previously failed on a missing `Content-Language` response and a missing idempotency key, respectively; they must not be counted as passing suites.

## Work still to finish

1. **Complete and review every screen** in both apps under all three themes, including authentication, registration, catalog, discovery, booking and status, schedule, chat, profile, settings, available feature screens, and loading, empty, success, and error states. The detailed inventory and current status are in [redesign-audit-2026-11-14.md](redesign-audit-2026-11-14.md).
2. **Finish translations** for all existing supported languages. Remaining English literals include parts of Poojari materials, referral, and chat. Verify text wrapping and meaning, including errors and confirmations.
3. **Finish accessibility and motion review:** screen-reader labels and announcements, focus order, contrast, touch targets, keyboard behavior, small and large screens, enlarged text, and reduced-motion behavior. Verify each theme rather than assuming theme tokens alone make a screen accessible.
4. **Audit hard-coded values.** Replace user-facing production data with values from the existing API, user input, or device settings, and move translatable text and reusable design values to appropriate resources or tokens. Retain legitimate constants where appropriate. In particular, remove the fabricated address fallback in backend `CreateBooking` with validation and compatibility tests; audit the remaining fixed appointment-time assumptions for unslotted bookings.
5. **Verify the remaining journeys and states** in an isolated environment: registration, authentication errors, booking with multiple appointment times, cancellation, schedule operations and errors, duty/location failures, chat, materials, referral, profile, services, store, temple, and any available video functionality. Check real loading and error handling without introducing sample production data.
6. **Rerun the full Devotee booking journey after the backend fix**, complete relevant test suites, fix failures, and rebuild both Android APKs after any further changes. Review screens visually on available devices or emulators. Build and test iOS when an appropriate environment is available.

## Current Android debug APKs

- Devotee: `E:\New folder (5)\apps\devotee-app\build\outputs\apk\debug\devotee-app-debug.apk`
- Poojari: `E:\New folder (5)\apps\priest-app\build\outputs\apk\debug\priest-app-debug.apk`

These are debug builds from the last successful Android build, not published releases. The backend fix is source code and is not deployed by building these APKs.
