# Prompt for Gemini: close 4 remaining defects (round 2)

## Role
You are a senior Go, Next.js and Kotlin Multiplatform engineer working in `E:\New folder (5)`. Fix exactly the 4 defects below, at the root cause, with tests that fail before the fix and pass after. Do not change anything else.

## Hard rules
1. Read every file you change in full before editing. Trace every caller of any function you change.
2. No new dependencies, except `jose` in temple-cms if you choose that option in Defect 1.
3. Do not touch WebRTC/video, vendor-app, or push notifications.
4. Return generic error messages to clients; log details server-side only.
5. New migrations continue after `000014`. Update the gateway's required schema version in `backend/cmd/api-gateway/main.go` (~line 96).
6. All must stay green:
   - `cd backend && go vet ./... && go test -count=1 ./...`
   - `cd temple-cms && npm run build`
   - `./gradlew :apps:common-contracts:allTests :apps:devotee-app:assembleDebug :apps:priest-app:assembleDebug`
7. Do not commit, push, or leave helper scripts (`fix_*.py` etc.) in the repo. Delete the `output/` folder if you created it.

---

## Defect 1: CMS token readable by browser JavaScript; middleware does not validate it

**Files:**
- `temple-cms/app/api/session/route.ts` (~lines 19, 31)
- `temple-cms/lib/api.ts` (~lines 11-14)
- `temple-cms/middleware.ts`
- `temple-cms/app/login/page.tsx`
- `temple-cms/components/LogoutButton.tsx`

**Problem:**
- POST and GET `/api/session` return `{ token }` in the JSON body.
- `lib/api.ts` keeps the token in JS memory and sends it from the browser, which defeats the `httpOnly` cookie. Any XSS can steal the admin token.
- `middleware.ts` only checks that the cookie exists, so any value passes.

**Required fix (BFF proxy pattern):**
1. The token must never reach browser JS.
   - Remove `token` from every `/api/session` response body. POST returns `{ ok: true }`. GET returns `{ authenticated: true, role, exp }` only.
2. Create a server-side proxy route `temple-cms/app/api/backend/[...path]/route.ts`:
   - Handle GET, POST, PUT, DELETE.
   - Read the cookie server-side and forward to `${API_BASE_URL}/<path>` with `Authorization: Bearer <cookie>`. Pass through method, query string, body, `Content-Type` and `Idempotency-Key`.
   - Return the backend's status and body. On a backend 401, clear the cookie.
   - Use a server-only env var `API_BASE_URL`, not `NEXT_PUBLIC_*`. Update `.env.example` and `docker-compose.prod.yml`.
   - Allow only paths starting with `api/v1/`. Reject `..` and absolute URLs (no SSRF).
3. Rewrite `lib/api.ts`:
   - `apiFetch(path, init)` calls `/api/backend/${path}` with `credentials: 'same-origin'`.
   - No token variable anywhere in client code.
   - On 401, redirect to `/login`.
4. `middleware.ts` must reject tokens that are invalid or expired, not just missing ones. Either:
   - (a) verify the HS256 JWT signature and `exp` with `jose` (`jwtVerify`) using server-only `JWT_SECRET` (same secret as the backend), and require `role === 'admin'`; or
   - (b) if sharing the secret is unacceptable, decode and check `exp` and `role` in middleware, and rely on the proxy's backend 401 for signature checks.

   Pick (a) unless the secret cannot be shared, and state which you chose.

   On failure: delete the cookie and redirect to `/login?next=<path>`.
5. Login (`app/login/page.tsx`) posts credentials to `/api/session`, which calls backend `POST /api/v1/auth/admin/login` server-side and sets the cookie:
   - `httpOnly`
   - `secure` in production
   - `sameSite: 'strict'`
   - `path: '/'`
   - `maxAge` equal to the token's remaining lifetime
6. Logout calls `DELETE /api/session`, which clears the cookie.
7. CSRF: the proxy rejects state-changing methods when the `Origin` header is missing or not the CMS's own origin.

**Verification:**
- `grep -rn "token" temple-cms/lib temple-cms/app --include=*.tsx` shows no token handling in client components.
- `npm run build` passes.
- Manual check, steps written in the summary:
  - a forged cookie value `abc` is redirected to `/login`;
  - an expired token is redirected;
  - a valid admin token can load pages;
  - DevTools shows the cookie as HttpOnly, and no response body contains the JWT.

---

## Defect 2: Idempotency race and client keys not reused on retry

**Files:**
- `backend/cmd/api-gateway/auth_middleware.go` (~lines 86-150)
- `backend/internal/db/idempotency.go`
- `backend/migrations/000011_idempotency_keys.sql`
- devotee and priest `PurohitApiClient.kt` (~lines 698, 732, 797)
- callers in UI screens

**Problem:**
- The middleware reads the key, runs the handler, then upserts the result. Two concurrent requests with the same key both execute the handler, creating duplicate bookings or orders.
- In the apps, every call generates a fresh key by default, so a user retry or a network retry is never deduplicated.

**Required fix (backend, reserve-first):**
1. Add a new migration that changes `idempotency_keys`:
   - add `status TEXT NOT NULL DEFAULT 'IN_PROGRESS'` with values `IN_PROGRESS` / `COMPLETED`;
   - add `locked_until TIMESTAMPTZ`;
   - keep the primary key `(caller_id, key)`.
2. Middleware flow:
   1. Compute `request_hash` = SHA-256 of method + route template + canonical body.
   2. `INSERT ... (status='IN_PROGRESS', locked_until=now()+interval '60 seconds') ON CONFLICT (caller_id, key) DO NOTHING RETURNING ...`
   3. If the insert succeeded, run the handler. Capture status and body with a response recorder. Then `UPDATE ... SET status='COMPLETED', response_status, response_body`.
      - For 5xx responses, delete the row so the client can retry.
      - For 2xx and 4xx responses, store the result.
   4. If there was a conflict, `SELECT` the row:
      - a different `request_hash` returns 422;
      - `COMPLETED` replays the stored status and body, adding the header `Idempotent-Replayed: true`;
      - `IN_PROGRESS` with `locked_until > now()` returns 409 with a `Retry-After: 1` header;
      - `IN_PROGRESS` with an expired lock is taken over with `UPDATE ... SET locked_until=now()+60s WHERE locked_until < now() RETURNING`. If that returns zero rows, return 409.
   5. If the handler panics, delete the row (use `defer`/`recover`) and re-panic.
3. Keys expire after 24 h. Delete expired rows on insert conflict or in a periodic cleanup.

**Required fix (apps):**
4. Remove the default `= newIdempotencyKey()` parameters.
   - Every mutating client method (create booking, store order, ticket book, slot hold, cancel) requires an `idempotencyKey: String` argument.
5. In each calling screen or state holder, create the key once per user intent:
   - in `remember { mutableStateOf(newIdempotencyKey()) }` or the ViewModel, when the confirmation sheet opens;
   - reuse it for every retry of that action, including automatic network retries inside the client;
   - generate a new key only after a definitive success, or when the user changes the request inputs.
6. Handle 409 with `Retry-After` by waiting and retrying with the same key.

**Tests:**
- Go: `TestIdempotencyConcurrentSameKey` fires 20 goroutines with the same key and body at `POST /api/v1/bookings` (or a test handler that counts invocations). Assert the handler ran exactly once, all responses are either the same success or 409, and the database has exactly 1 booking.
- Go: a different body with the same key returns 422.
- Go: a 5xx response allows a retry with the same key to execute again.
- Kotlin (common-contracts test, or an app unit test if one exists): a retry helper reuses the same key across attempts.

---

## Defect 3: Store orders still use client-sent float amounts

**Files:**
- `backend/internal/db/vendors.go` (~lines 81-98)
- `backend/internal/models/entities.go` (store order request/response types)
- `backend/migrations/000012_money_minor_units.sql`
- `apps/common-contracts/.../StoreModels.kt`
- the devotee app store screens

**Problem:**
- The server sums float subtotals sent by the client and converts with `math.Round(x*100)`.
- Clients can set their own prices.
- The old float columns are still written alongside the new ones.

**Required fix:**
1. The server never trusts client prices. The request carries only `vendor_id` and items `[{inventory_item_id, quantity}]` (quantity is an integer between 1 and 100).
2. Inside one transaction:
   - `SELECT price_minor, in_stock, vendor_id FROM vendor_inventory WHERE id = ANY($1) FOR SHARE`;
   - reject an unknown item, an item from another vendor, or an out-of-stock item with 422;
   - compute `line_total_minor = price_minor * quantity` and `total_minor` as the sum, all `int64`;
   - check for overflow;
   - insert the order and items with the minor-unit columns only.
3. Remove float arithmetic from this path entirely: no `float64` and no `math.Round` in `vendors.go`.
   - If the old float columns are `NOT NULL`, add a migration that makes them nullable or drops them.
   - Grep every read of those columns and switch it to `*_minor`.
4. The response returns `*_minor` plus `currency: "INR"`. Keep deprecated float response fields only if existing app builds read them, computed as `minor/100` for display only.
5. Update `StoreModels.kt` and the store screens to send only ids and quantities, and to format money from minor units via the existing `MoneyFormatting.kt`.

**Tests:**
- An order with a client-supplied price field is ignored (or rejected), and the total equals the server price × quantity.
- Prices of 3 items × 333 paise give an exact total.
- An item from another vendor returns 422; an out-of-stock item returns 422.
- `grep -n "float64\|math.Round" backend/internal/db/vendors.go` finds nothing.

---

## Defect 4: Priest location stale-write race and missing capture time accepted

**Files:**
- `backend/cmd/api-gateway/location_routes.go` (~lines 40-46)
- the location DB function it calls
- `backend/migrations/000013_priest_location_tracking.sql`
- `apps/priest-app/.../DutyLocationService.kt`

**Problem:**
- The newer-than check is a separate read and then a write, so two concurrent updates can let an older point overwrite a newer one.
- When `captured_at` is missing, the check is skipped and the point is accepted.

**Required fix:**
1. Validation, returning 400 on failure:
   - `captured_at` is required, in RFC 3339 / ISO 8601 UTC;
   - it may not be more than 2 minutes in the future (clock skew) nor older than 30 minutes;
   - latitude is between -90 and 90, longitude between -180 and 180, and `accuracy_m` is between 0 and 10000.
2. Make the update atomic in one statement:
   ```sql
   UPDATE priests SET latitude=$2, longitude=$3, location_accuracy_m=$4, location_captured_at=$5, ...
   WHERE id=$1 AND (location_captured_at IS NULL OR location_captured_at < $5)
   RETURNING id
   ```
   - Zero rows returned means the point is stale: return 200 with `{"applied": false, "reason": "stale"}` so the client does not retry.
   - Keep any history insert (if a history table exists) independent: history may store all points, but the "latest" row only moves forward.
   - Also update the PostGIS geography column, if present, in the same statement.
3. Only accept the update while the priest is on duty. If off duty, return 409 and do not store the point.
4. Priest app `DutyLocationService.kt`:
   - always send `captured_at` from the `Location` fix time (`location.time`, converted to UTC ISO 8601), not the send time;
   - send `accuracy_m` from `location.accuracy`.

**Tests:**
- `TestLocationRejectsMissingCapturedAt` returns 400.
- `TestLocationRejectsFutureAndTooOld` returns 400.
- `TestLocationConcurrentStaleWrite`: send a point at T2, then concurrently 10 points at T1 < T2. The stored point remains T2.
- `TestLocationOffDutyRejected` returns 409.

---

## Deliverable
Finish with a table: defect # | files changed | test names | status (FIXED only if its tests pass; otherwise BLOCKED with the exact error line). Also include:
- the manual verification steps for Defect 1;
- the full output tail of each of the three test/build commands.
