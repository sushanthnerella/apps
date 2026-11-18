# Devotee and Poojari redesign audit

This inventory describes the current source and the verification performed so far. “Partial” means themed or redesigned elements exist, but the complete screen has not been visually and functionally checked in all three themes, all supported languages, and at enlarged text. No screen is marked complete on that basis. Theme selection and restart persistence were exercised on Android for both apps; this does not establish full screen coverage.

## Devotee

| Screen or state | Status across three themes | Existing actions and verification | Remaining work |
| --- | --- | --- | --- |
| Login and registration | Partial | Sign-in was used for an earlier isolated journey. Phone/password keyboard types, password visibility on registration, localized login errors, announced errors, and flexible button height were added in the latest pass. | Android emulator check: login displayed and blank submission showed a phone error; registration navigation, scrolling, and blank submission error were verified. Still check complete registration and login with an isolated backend, all themes and translations, enlarged text, focus, screen reader, and iOS. |
| Appearance | Partial | Select each theme and restart tested on Android. | Verify all languages, screen reader, large text, and iOS persistence. |
| Catalog and required items | Partial | Service selection, adding, and dispatch tested in isolated journey. | Check every catalog state, translations, and scaling. |
| Priest discovery, date, slot and review | Partial | Address, date, priest, slot, and review tested for one isolated booking. | Check empty/error states, language coverage and alternative screen sizes. |
| Booking tracking and status, cancellation | Partial | Acceptance through completion displayed in isolated and Render emulator journeys. A server-backed booking history screen was added, with retry and localized terminal statuses; Android build and unit test passed. | Android 15 emulator verified history after restart, active booking reopening, empty and offline/retry states, cancellation and slot release; labels checked in English, Telugu and Hindi, all three themes selected, and the list scrolled at 2x font scale. Still verify screen-reader and keyboard behavior, full theme/language coverage and other device sizes. The backend currently returns at most 25 bookings. |
| Chat | Partial | Messages sent in both directions in isolated testing. | Localize remaining strings and verify screen reader, keyboard and all themes. |
| Temple directory and store | Partial | Existing directory and store actions retained; no complete UI journey verified. | Check API, empty/error states, translations and layouts. |
| Profile | Partial | Existing profile controls retained; no complete authenticated UI audit. | Check translations, scaling and every action. |
| E-pooja viewer | Partial | Existing unavailable state shown; video functionality is not implemented by this screen. | Verify localized unavailable state and accessibility. |
| Offline, loading and error states | Partial | Some states exercised; loading logo and theme-aware components exist. | Audit all error paths and reduced motion with assistive technology. |

## Poojari

| Screen or state | Status across three themes | Existing actions and verification | Remaining work |
| --- | --- | --- | --- |
| Login and registration | Partial | Sign-in used in isolated journey; UI registration not fully tested. | Check translations, validation, scaling and focus. |
| Appearance | Partial | Three themes and restart persistence tested on Android. | Verify all languages, screen reader, large text and iOS. |
| Duty dashboard, location and booking actions | Partial | Going online with actual emulator location and booking status actions tested in isolation. | Check offline and location failures, language coverage, all themes and scaling. |
| Incoming booking request | Partial | Acceptance, conflict error and keyboard dismissal exercised in isolation. | Check rejection, countdown, focus, translations, all themes and scaling. |
| Schedule | Partial | Publishing a slot through UI was previously exercised; refresh ordering has unit coverage. | Recheck all schedule operations and languages, error states and scaling. |
| Chat | Partial | Messages sent in both directions in isolation. | Localize remaining strings and test screen reader and keyboard. |
| Custom services | Partial | Existing controls retained; full service editing journey not verified. | Check save/error states, translations, themes and accessibility. |
| Materials checklist | Partial | Existing checklist controls retained; full UI journey not verified. | Replace remaining English strings, audit colors, error states and touch targets. |
| Colleague referral | Partial | Existing referral controls retained; complete referral not verified. | Localize remaining English strings and verify empty/error states. |
| Live studio and muhurtham | Partial | Existing screens retained; muhurtham includes an unavailable state. | Verify actual capabilities and localized states without suggesting unsupported functionality. |
| Profile | Partial | Existing controls retained; full UI audit not performed. | Verify actions, translations and scaling. |
| Offline, loading and error states | Partial | Some states exercised and reduced-motion support added. | Audit every path, colors and announcements. |

## Hard-coded values and open risks

1. **Store Vendors Null Issue (RESOLVED):** Fixed backend vendors_geo.go to initialize empty slice []models.Vendor{} instead of nil, ensuring [] JSON serialization. Updated PurohitApiClient.kt to safely parse 
ull as emptyList().
2. **Fabricated Address Fallback (RESOLVED):** Removed hardcoded 'Plot 42, Sri Rama Nagar...' fallback in db.go; CreateBooking now strictly requires a non-blank event_address with unit and regression tests. Removed fabricated 'Hyderabad' default on devotee registration.
3. **60-Minute Delivery Claim (RESOLVED):** Removed all claims of 60-minute delivery from EnStrings.kt, TeStrings.kt, HiStrings.kt, and PujaStoreScreen.kt. Replaced with honest pilot notice: 'Verified local vendor stores and samagri delivery are not yet active in this location for the free pilot. Ritual materials can be arranged directly with your assigned Purohit.'
4. **Intermittent Registration / Login Timeouts (RESOLVED):** Added 45s connect/socket timeouts and 60s request timeout plugin in Ktor / OkHttp engine across both apps to accommodate Render free-tier cold-starts.
5. **Push Notifications Limitation:** Render staging logs confirm LoggingNoOpNotifier (FCM service account is unconfigured). Real push notifications cannot be delivered over the air; internal-test users must be informed that alerts are in-app only.
6. **Password Recovery Limitation:** Server returns 503 email_service_unavailable because SMTP is unconfigured. Both Devotee and Priest apps honestly display that password recovery via email is currently unconfigured.
7. **Pilot Dakshina & Payments:** No online payment gateway is enabled for this pilot; priest charges and materials fees are strictly arranged directly, and cancellation modals state 100% free pilot cancellation with zero penalty.
8. **Automated Verification:** Verified against Render staging and Android emulator (Codex_Puja_Test_35): Devotee registration, duplicate account 409 rejection, wrong password rejection, valid login, store honest empty state, 3 themes (Royal Heritage, Calm Modern, Temple Earth), 3 languages (English, Telugu, Hindi), process restart / logout persistence, Priest duty toggle and schedule workbench, two-way chat, Swiggy cancellation modal with backend DB confirmation and slot release, sacred OTP completion with wrong OTP 409 rejection and valid OTP 200 completion, booking history listing, and 2.0x font scaling accessibility.
