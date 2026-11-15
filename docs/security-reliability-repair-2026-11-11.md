# Security and reliability repair — 2026-11-11

This is a partial implementation of the requested issue list, not a production-readiness signoff. Existing uncommitted work was preserved. No live deployment, credential rotation, email delivery or production migration was performed.

## Implemented

- Devotee/priest sessions now use SessionVault. Android uses AES-GCM with Android Keystore keys and private preferences; iOS entry points initialize the existing Keychain vault. Startup validates the stored JWT through the profile endpoint (including server session-version revocation checks). Logout clears the vault and cached profile; profile HTTP 401 clears the token. Network failures retain the saved token.
- All three Android manifests disable cleartext traffic and backup. Contacts and priest background-location permissions were removed, including runtime contacts requests. Merged debug manifests were checked.
- Removed the priest server-address editor and all ngrok warning headers. API base addresses resolve dynamically through BackendConfig. HTTP-engine initialization failures now preserve and surface their cause.
- Both chat screens consume the gateway WebSocket and reconcile history on connection/events. Reconnect uses jittered exponential backoff, capped at 30 seconds, and is cancelled with the screen. Tracking backs off to 60 seconds after failures and stops at completed/cancelled/rejected states.
- Shared HTTP origin policy accepts exact ALLOWED_ORIGINS entries; missing Origin is allowed for native clients. Unknown browser origins are rejected. Gateway video signaling now checks JWT and chat-room membership before upgrade.
- Completion codes use crypto/rand. Five failed attempts permanently lock that booking's completion code; an assigned priest cannot bypass the cap by subsequently submitting the correct code. Attempts are persisted under the booking row lock; unauthorized callers cannot consume them. Support intervention is required after lockout; no self-service unlock UI was added.
- Public NearbyPriest/PriestCandidate JSON excludes phone numbers. Booking-specific authorized contact access remains separate.
- Nearby search performs radius selection in the database. PostGIS installations use ST_DWithin and a partial GiST expression index. Plain PostgreSQL uses latitude indexing and SQL spherical-distance filtering. Requested service filters no longer silently fall back to unrelated priests. Radius is bounded to 100 km. Redis GEOSEARCH is not implemented.
- Removed production database-password fallbacks and hardcoded Compose database/TURN/JWT secrets. Both Compose files require environment-supplied secrets; Redis authentication is configured, and database/Redis published ports bind to localhost. Test database connection URLs inherit explicit test configuration rather than embedding passwords.
- Extracted chat routes, realtime hubs, origin policy and discovery into focused files. Fixed room-map mutation under a shared lock and idempotent disconnect cleanup. Legacy dispatch-service, webrtc-signaling and mock_server.py now exit with a gateway migration message instead of serving hardcoded or duplicate APIs.
- Added distinct DevoteeApp/PriestApp/VendorApp Swift hosts and an XcodeGen project specification; removed armv7 requirement. Added the vendor Kotlin iOS entry point and Darwin engine dependency.
- Removed simulated vendor orders/inventory/store identity, live video states/viewer counts and Muhurtham results. These screens now show localized unavailable notices. The backend no longer fabricates fallback vendor inventory.
- Added Telugu/Hindi/English unavailable and booking-status strings, and corrected logout copy to describe password login.

## OTP and password recovery findings

OTP login is disabled in source. The normal SMS provider is unconfigured; no production SMS implementation was found. This does not establish the state of a separately deployed service.

Current mobile recovery requests an email link, with a separate resend action. The backend reset form consumes the token and updates the password. The reported double-start/no-reset problem does not describe the active email UI. Legacy SMS-oriented client helper methods remain unused; they should be removed in a later API cleanup rather than re-enabled.

## Verification

- Full `go test ./... -count=1` passed using isolated local PostgreSQL test databases, including the new completion-attempt and public-phone regression checks. Migration expectations were updated for 000009.
- Shared Kotlin debug unit tests passed.
- Devotee, priest and vendor Android debug APK assembly passed.
- Both Compose YAML files and the XcodeGen specification parsed successfully. Docker runtime validation was unavailable because Docker is not installed here.
- Logs: `output/security-backend-tests.log`, `output/security-build.log`.
- Apple builds, device restart/Keychain/Keystore tests, multi-device chat recovery and the PostGIS query/index execution plan were not run. Local PostgreSQL lacks PostGIS; its fallback path was tested. iOS hosts need macOS/Xcode validation.

## Required rollout configuration

1. Configure PG_CONN_STR (or DATABASE_URL), JWT_SECRET, REDIS_ADDR/REDIS_PASSWORD and exact ALLOWED_ORIGINS. Compose additionally requires POSTGRES_PASSWORD, TURN_USER, TURN_PASSWORD and PUBLIC_API_URL. Use independently generated secrets; rotate previously exposed real credentials outside this patch.
2. Run the existing migration tool to apply 000009 before starting the gateway. The gateway requires schema version 000009. Production should install PostGIS before applying the migration to create the GiST index. If PostGIS is installed afterwards, explicitly create that index using the expression in the migration; already-applied migrations are not rerun.
3. Serve the gateway behind HTTPS. Android builds now reject HTTP, including emulator/local HTTP endpoints. Browser reset origins and CMS origins must be listed in ALLOWED_ORIGINS.
4. On macOS, run `xcodegen generate` in apps/iosApp, open PujaVaahini.xcodeproj, select the role-specific scheme and configure signing.

## Still unfinished

- The vendor app has no authenticated vendor network workflow. It is intentionally unavailable instead of presenting demo sales. Real vendor account authentication/ownership, inventory CRUD and order processing must be implemented and tested before activation.
- No actual camera/WebRTC media pipeline or live E-Pooja stream is implemented. Signaling alone is not a video call. TURN deployment and a two-device media test are still required.
- Muhurtham requests/calculation/persistence are not implemented. The existing backend Panchangam calculation is approximate and must not be presented as an authoritative computation.
- Localization outside the changed notices, logout and booking status flows is incomplete.
- iOS host sources/specification exist, but no generated Xcode project or Apple build has been verified on this Windows machine.
