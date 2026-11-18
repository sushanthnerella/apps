# Prompt for Gemini: close all remaining Puja Vaahini defects

## Role
You are a senior Go backend engineer, Kotlin Multiplatform (Compose) engineer and Next.js engineer working in the repository at `E:\New folder (5)`. Your job is to fix every defect listed below in one pass, with tests, without changing unrelated behaviour.

## Context
- Backend: Go, single API gateway in `backend/cmd/api-gateway/`, shared code in `backend/internal/`. PostgreSQL 16 + PostGIS, Redis. Migrations in `backend/migrations/` (latest is `000009`; the gateway refuses to start below its required schema version, so bump that check when you add a migration).
- Mobile: `apps/devotee-app`, `apps/priest-app`, shared `apps/common-contracts`.
- CMS: `temple-cms/` (Next.js 14 App Router).
- Existing auth: HS256 JWT via `backend/internal/auth/jwt.go`, `ValidateToken` checks `session_version`. Helper `requireCallerSession` already exists in the gateway; reuse it.
- Booking state machine: `backend/internal/lifecycle/fsm.go`. Do not weaken it.
- Line numbers below are from the current working tree and may drift by a few lines; locate by the function/route named.

## Hard rules
1. Fix root causes in shared code; do not patch each caller separately when one shared fix covers them.
2. Do not add new dependencies unless stated. Use the Go standard library and existing packages.
3. Do not touch video calling / WebRTC / coturn / TURN (`/ws/webrtc`, `cmd/webrtc-signaling`, coturn config, `BroadcasterStudioScreen`, `EPoojaViewerScreen`). Out of scope.
4. Do not touch `apps/vendor-app`. Out of scope (deferred).
5. No fake fallbacks: never return invented priests, names, gotram, or a "verified" result on error.
6. Every error response to clients is generic; log the internal error server-side only.
7. Every fix gets at least one automated test that fails before the fix and passes after.
8. Keep existing passing tests green: `cd backend && go test ./...`, `./gradlew :apps:common-contracts:allTests :apps:devotee-app:assembleDebug :apps:priest-app:assembleDebug`, `cd temple-cms && npm run build`.

## Issues to fix

### 1. Unauthenticated write and admin routes (security, critical)
File: `backend/cmd/api-gateway/main.go`
These routes accept requests with no authentication:

| Route | Approx. line | Required access |
|---|---|---|
| `PUT /api/v1/services/translations` | 665 | admin |
| `POST /api/v1/services/auto-translate` | 683 | admin |
| `POST /api/v1/poojas/{id}/samagri` | 703 | admin |
| `DELETE /api/v1/poojas/samagri/{item_id}` | 728 | admin |
| `POST /api/v1/priests/custom-services` | 632 | priest; `priest_id` taken from token, never from body |
| `POST /api/v1/store/orders` | 781 | devotee; `user_id` from token |
| `PUT/POST /api/v1/store/orders/{id}/status` | 800 | admin (vendor role does not exist yet) |
| `POST /api/v1/temples/tickets/book` | 848 | devotee; `user_id` from token |
| `POST /api/v1/temples/tickets/verify` | 863 | admin (gate staff) |
| `GET /api/v1/dispatch/radar` | 264 | admin |
| `POST /api/v1/dispatch/match` | 276 | devotee |

Fix: add one middleware, e.g. `requireRole(roles ...string)`, that wraps `requireCallerSession`, and apply it to each route. Any client-supplied identity field that conflicts with the token returns 403.
Tests: for each route, no token gives 401, wrong role gives 403, correct role gives 2xx.

### 2. Forgeable admin token (security, critical)
File: `backend/cmd/api-gateway/booking_handlers.go` lines ~106–118.
The fallback accepts any HS256 token with `role=admin` without checking expiry, issuer or session version, and nothing issues admin tokens.
Fix:
- Delete the fallback.
- Add an `admin` role to `ValidateToken` in `backend/internal/auth/jwt.go` with the same `exp` and `session_version` checks as other roles.
- Add a migration creating `admin_users` (id, email, password_hash bcrypt, role, session_version, created_at, disabled_at). Login is `POST /api/v1/auth/admin/login` (email + password), with the same throttling as existing login.
- Add a CLI flag or script in `backend/cmd/migrate` (or a new `cmd/create-admin`) to create the first admin. No hardcoded default admin.

Tests: expired admin token, token after session_version bump, and token with no `exp` are all rejected.

### 3. Temple CMS has no login and hardcodes the API URL (security, critical)
Files: `temple-cms/app/*/page.tsx`, especially:
- `dispatch-radar/page.tsx:57`
- `gate-scanner/page.tsx:20`
- `samagri-catalog/page.tsx:40,61,88`
- `services-translation/page.tsx:31,57,85`

Fix:
- Create `temple-cms/lib/api.ts`: one `apiFetch(path, init)` that reads the base URL from `process.env.NEXT_PUBLIC_API_BASE_URL` and adds `Authorization: Bearer <token>`. On 401 it redirects to `/login`. Replace every raw `fetch('http://localhost:8085…')` with it.
- Add `app/login/page.tsx` calling `POST /api/v1/auth/admin/login`.
- Store the token in an httpOnly, Secure, SameSite=Strict cookie set by a Next.js route handler (`app/api/session/route.ts`), not localStorage.
- Add `middleware.ts` that redirects unauthenticated users to `/login` for all routes except `/login`.
- Add a logout action.
- Add `NEXT_PUBLIC_API_BASE_URL` to `docker-compose.prod.yml` for the temple-cms service and to an `.env.example`.

Tests: `npm run build` passes, and a grep for `localhost:8085` in `temple-cms/app` returns nothing.

### 4. Gate scanner shows "verified" on failure (correctness/safety)
File: `temple-cms/app/gate-scanner/page.tsx` lines ~32–33 and ~50–61.
On network error it shows `valid: true` / "Offline Ed25519 Verified" with an invented devotee. On success it fills in invented names and gotram when fields are missing.
Fix:
- On any error, show a clear "Unable to verify, do not admit, retry" state with `valid: false`.
- Render only fields the server returned, with no placeholder identities.
- Do not claim Ed25519 offline verification, because none is implemented.

### 5. Temple ticket double check-in race (correctness)
File: `backend/internal/db/temples.go`, `VerifyTempleTicket`, lines ~131–147.
It does SELECT, then a status check, then a separate UPDATE, so two concurrent scans can both succeed.
Fix: one atomic statement:
```sql
UPDATE temple_vip_tickets SET status='CHECKED_IN', checked_in_at=now()
WHERE qr_token=$1 AND status <> 'CHECKED_IN'
RETURNING ...
```
If zero rows are updated, look up the row once to return "already checked in" or "not found".
Test: 20 goroutines verifying the same ticket give exactly 1 success.

### 6. No idempotency on creates (correctness)
No `Idempotency-Key` support exists anywhere.
Fix:
- Add a migration `idempotency_keys` (key, caller_id, route, request_hash, response_status, response_body, created_at, primary key (caller_id, key)).
- Add Go middleware for the following POST routes: `POST /api/v1/bookings`, `/api/v1/store/orders`, `/api/v1/temples/tickets/book`, `/api/v1/priests/slots/hold`, `/api/v1/bookings/{id}/cancel`.
- Behaviour:
  - A missing header gives 400 on these routes.
  - The same key with the same body replays the stored response.
  - The same key with a different body gives 422.
  - Keys expire after 24 h; a cleanup job or delete-on-read is acceptable.
- Mobile: generate a UUID per user action in `PurohitApiClient` (devotee and priest) and send it as `Idempotency-Key`. Reuse the same key on retries of that action.

Test: two identical POST /bookings with the same key create one booking.

### 7. Money stored as float (correctness)
Files: `backend/internal/models/entities.go` lines ~162, 186, 215, 278, 322, plus every DB read and write of those fields. Postgres currently uses `NUMERIC(10,2)`.
Fix:
- Add a migration adding `*_minor BIGINT` columns (paise), backfilled with `ROUND(amount*100)`.
- Change Go fields to `int64` with JSON names `*_minor` plus `currency: "INR"`.
- Keep the old float JSON fields for one release as read-only aliases, marked deprecated, so current app builds keep working.
- Update `backend/internal/cancellation/rules.go` to integer arithmetic.
- Update `apps/common-contracts` DTOs and both apps' price display to format from minor units.

Test: cancellation quote on 3716.00 INR with the Rs 150 fee returns exact integers.

### 8. CreateBooking ignores side-effect errors (correctness)
File: `backend/internal/db/db.go`
- Material inserts ignore errors (~745–752).
- The chat room and system message are created after commit with errors ignored (~759, ~770).

Fix: move the material inserts, chat room creation and system message inside the same transaction, check every error, and roll back on failure. `GetPoojaSamagri` must read using the transaction.
Test: force a material insert failure and assert that no booking row and no held slot remain.

### 9. `/dispatch/match` invents data (correctness)
File: `backend/cmd/api-gateway/main.go` ~280–287 and ~309–323.
- On a bad body it substitutes a demo request.
- With no candidates it returns a fake priest.
- `AcceptanceRate` is hardcoded to 0.96.
- If the radius is missing, candidates are wrongly filtered.

Fix:
- Return 400 on an invalid body.
- Return 200 with an empty list and a reason when there are no candidates.
- Read the acceptance rate from priest data, defaulting to a neutral 0.5 for new priests, documented in a comment.
- Require `search_radius_meters` or apply a documented default (5000).

### 10. Fixed-interval polling without backoff (performance/battery)
Files:
- `apps/devotee-app/.../ui/dispatch/RapidoNearbyPriestsScreen.kt:132` (2 s)
- `apps/priest-app/.../PriestApp.kt:185` (2.5 s)

Fix: reuse the jittered backoff helper already used by `LiveTrackingScreen.kt:37` (extract it to `common-contracts` if it is not shared yet). Use a base 5 s, a max 30 s, back off on errors, pause when the screen is not visible, and stop on dispose.

### 11. Priest foreground location service missing (feature gap)
File: `apps/priest-app/src/androidMain/AndroidManifest.xml:8-9` declares `FOREGROUND_SERVICE` and `FOREGROUND_SERVICE_LOCATION` but there is no `<service>`.
Fix:
- Implement `DutyLocationService` (Android foreground service, `foregroundServiceType="location"`) with a persistent notification "On duty – sharing location".
- It posts to `POST /api/v1/priests/me/location` every 120 s with `captured_at` (UTC ISO 8601) and `accuracy_m`.
- Start it on duty ON and stop it on duty OFF or logout.
- Request `POST_NOTIFICATIONS` on Android 13+.
- Backend: in the location handler, reject points whose `captured_at` is older than the stored latest point.

Tests: a Go test for the older-point rejection, and a manual test noted in the summary.

### 12. Push notifications not wired (feature gap, launch-blocking)
Only `UnconfiguredSMSProvider` exists (`backend/internal/auth/sms.go:24`).
Fix:
- Add `backend/internal/notify` with a `Push` interface and an FCM HTTP v1 implementation. Credentials come from the `FCM_SERVICE_ACCOUNT_JSON` env var; if it is unset, a logging no-op is used.
- Add a `device_tokens` table and `POST /api/v1/devices` (auth required).
- Send on: new booking request (priest), accepted/declined (devotee), cancellation (both).
- Lock-screen text must not include OTPs or precise location.
- Mobile: register the FCM token after login (Android).
- Allowed new dependency: Firebase Messaging for Android only.

## Deliverables
1. Code changes for issues 1–12.
2. New migrations numbered after `000009`, with the gateway's required schema version updated.
3. Tests as specified; all suites green.
4. `.env.example` files for backend and temple-cms listing every required env var (no real secrets).
5. A final summary table with: issue #, files changed, test name, status (FIXED / BLOCKED + reason). Do not mark anything FIXED without a passing test or, for device-only behaviour, explicit manual steps.

## Do not
- Do not commit secrets, change git history, or push.
- Do not mark the work done if any test fails. Report failures with the exact error line.
