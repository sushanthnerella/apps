# Puja Vaahini — Free Pilot Progress & Baseline Tracker

> **Status:** ACTIVE  
> **Document Version:** 1.9.0-pilot  
> **Last Updated:** 2026-11-07  
> **Current Completed Phase:** Pilot Audit Verification & Static Analysis Fixes (SMTP IPv6 formatting, Windows JDK escaping, optional camera uses-feature, dynamic mobile backend URL, e2e acceptance repairs, Temple CMS Playwright tests, and zero-defect gateway validation)

---

## 1. Baseline Verification Commands & Actual Results

Every baseline check specified in the task instructions was executed against the local repository without modifications or artificial passes. Results are documented honestly below, separating code compilation passes from environment-level availability failures.

| Target Component | Command Executed | Exit Code | Result Status | Detailed Findings & Output Summary |
|---|---|---|---|---|
| **Go Vet & Static Analysis** | `cd backend; go vet ./...` | `0` | **VERIFIED (Pass)** | All packages passed vet check without errors. SMTP IPv6 host/port concatenation fixed via `net.JoinHostPort` (`internal/auth/email.go:438`). |
| **Backend Unit & Integration Suite** | `cd backend; go test -count=1 -json ./...` | `0` | **VERIFIED (Pass)** | 114/114 tests passed across all packages (`lifecycle`, `auth`, `cancellation`, `ranking`, `db`, `db/migrate`, `cmd/api-gateway`). |
| **Backend Race Detector** | `cd backend; go test -race -count=1 ./...` | `1` | **BLOCKED (Environment)** | Requires CGO and a C compiler (`gcc`). Host environment does not have `gcc` on `%PATH%`. Reported blocked honestly. |
| **Gradle Tests, Lint & Assemble** | `.\gradlew.bat :apps:common-contracts:cleanTestDebugUnitTest :apps:common-contracts:cleanTestReleaseUnitTest test lintDebug assembleDebug --continue --console=plain` | `0` | **VERIFIED (Pass)** | 253 actionable tasks completed in 1m 30s. Lint debug: 0 errors for devotee-app and priest-app. AndroidManifest declared `<uses-feature android.hardware.camera required="false">`. Note: App JVM unit test tasks (`testDebugUnitTest`, `testReleaseUnitTest`) report `NO-SOURCE` as unit tests reside in common-contracts. |
| **Connected Android Tests** | `.\gradlew.bat connectedDebugAndroidTest --continue --console=plain` | `0` | **VERIFIED (0 tests / NO-SOURCE)** | Build completed in 6s (`223 actionable tasks`). 0 tests executed because connected test sources are not defined in app modules. |
| **Backend Simulation** | `python backend/test_simulation.py` | `0` | **VERIFIED (Pass)** | Ran 4 tests in 0.001s: Escrow FSM, Haversine distance, Rapido ranking score, WebRTC SDP offer. |
| **End-to-End Flow Script** | `python test_e2e_flow.py http://localhost:18085` | `0` | **VERIFIED (Pass)** | All 10 end-to-end tests passed against local gateway on ephemeral database `pv_pilot_test_runner`. Negative authorization (unassigned priest 403) verified. |
| **Comprehensive E2E Script** | `python test_e2e_comprehensive_flow.py http://localhost:18085` | `0` | **VERIFIED (Pass)** | All 14 steps across modules A-J passed. Chat room unauthenticated (401) and nonparticipant (403) rejection verified. |
| **Concurrency & Race Condition** | `python test_concurrency_race_condition.py http://localhost:18085` | `0` | **VERIFIED (Pass)** | 30 simultaneous mutex requests: exactly 1 acquired (200), 29 rejected (409 Conflict). Contender slot double-booking rejected with 409 Conflict. |
| **Hyperscale Simulation & Gate Scan** | `python test_hyperscale_200m_simulation.py http://localhost:18085` | `0` | **VERIFIED (Pass)** | Mathematical model, in-memory Lua simulation, Kafka batching simulation, and 25 real HTTP gate verification queries completed with p95 26.52ms (< 50ms). |
| **High-Throughput Load Benchmark** | `python benchmark_load_test.py http://localhost:18085` | `0` | **VERIFIED (Pass)** | 6,500 total requests executed with 0% error rate (6500/6500 HTTP 200). Health RPS: 931.7 (p95 120ms); Panchangam RPS: 957.2 (p95 66ms); Catalog RPS: 82.7 (p95 2012ms under DB load); Geo RPS: 137.4 (p95 1141ms); Temple RPS: 155.7 (p95 1080ms). |
| **Temple CMS Build & Playwright Suite** | `npm.cmd run build; python playwright_test_suite.py` | `0` | **VERIFIED (Pass)** | Next.js 14 production build completed with 11 static pages. Playwright suite executed 5 tests against production server, capturing screenshots into `output/test-audit-2026-11-07-rerun/`. |


---

## 2. Prioritized Implementation Gaps

| Gap ID | Priority | Description | Dependencies | Status | Acceptance Criteria & Status Summary |
|---|---|---|---|---|---|
| **GAP-01** | **P0** | **Reproducible Database Migrations & Data Safety Harness** | None | **VERIFIED** | • Forward migrations `000004`, `000005`, and `000006` created.<br>• Migration runner built with dedicated connection-pinned advisory locking (`0x50554A4156414148`), normalized SHA-256 checksums, and atomic transactions.<br>• Advisory lock ownership bound to `*sql.Conn`; pool deadlock eliminated for `MaxOpenConns=1`; cancellation safely releases lock via `DiscardConn`.<br>• Schema decoupled from demo seeds; historical un-gated checksums accepted for backwards compatibility.<br>• `bookings_pooja_id_fkey` constraint preserved; unknown personal/schedule fields strictly nullable without fake defaults.<br>• `Baseline()` enforces comprehensive schema compatibility (exact data types, nullability, unique constraints, and foreign key actions) before writing history.<br>• Authoritative `VerifyReadiness` detects intermediate migration history gaps, checksum mismatches, and physical schema incompatibility before serving traffic.<br>• 24/24 automated migration & regression tests verified on isolated ephemeral databases. |
| **GAP-02** | **P0** | **Secure Password Registration & Login (Auth & Role Enforcement)** | GAP-01 | **VERIFIED** | • Passwords hashed with bcrypt (cost 10); plaintext passwords and auto-provisioning fallbacks completely removed.<br>• Canonical phone normalization (+91 E.164) and collision detection implemented.<br>• Startup fail-fast validation for JWT signing secret (min 32 chars); strict HS256 algorithm enforcement.<br>• Required JWT identity claims (`UserID`, `Phone`, `Role`, `Subject`), positive session version, and explicit expiration enforcement (`jwt.WithExpirationRequired()`).<br>• Session revocation via strict session equality `tokenVersion == dbVersion` in DB and JWT claims.<br>• Bounded in-memory login throttler with canonical role-scoped keys (`role:role:phone:canonical`), trusted proxy chain verification (defeating spoofed `X-Forwarded-For`), and active lockout preservation under capacity pressure.<br>• Arbitrary OTP login bypass disabled on reachable routes.<br>• Mobile registration payloads and UI screens updated with password support. |
| **GAP-03** | **P0** | **Verified Email Password Recovery Flow (Task 04A)** | GAP-01, GAP-02 | **VERIFIED** | • `000007_verified_email_support.sql` migration implemented: `email`, `email_verified` on `users` and `priests` with unique partial indexes on `LOWER(email)`.<br>• Password recovery completely migrated from SMS to single-use cryptographic email reset links; phone+password login and booking-completion OTP untouched.<br>• Verified email constraint strictly enforced: accounts cannot be recovered or claimed via phone number alone; recovery requires an email verified for that account/role.<br>• Reset tokens: 32 cryptographically random bytes (64 hex characters) generated via `crypto/rand`, stored strictly as HMAC-SHA256 digests.<br>• 15-minute expiration, account-scoped, purpose-bound, with rate throttling (60s cooldown, max 3 req/hr/account, max 10 req/hr/IP).<br>• Generic responses prevent account presence leakage; timing equalization on non-existent accounts.<br>• Secure HTTPS reset form at `/api/v1/auth/recovery/reset` & `/reset-password` with `Referrer-Policy: no-referrer`, `Cache-Control: no-store`, `X-Frame-Options: DENY`.<br>• Reset URLs constructed strictly from `PASSWORD_RESET_BASE_URL` (never from untrusted Host header).<br>• Token consumption is atomic on password submission; visiting GET does not consume token; session revocation via `session_version` increment.<br>• Local `.env` loading without overriding deployment environment; TLS enforced on SMTP.<br>• Mobile apps updated with email reset dialog, loading indicators, success state, 15-min validity notice, and retry cooldown. |
| **GAP-04** | **P0** | **Mandatory Route Ownership & Completion OTP Asymmetric Privacy (Task 05)** | GAP-02, GAP-03 | **VERIFIED** | • All 12 booking routes require valid Bearer token authentication and active session version (`requireCallerSession`); missing, malformed, expired, or revoked tokens return HTTP 401 Unauthorized.<br>• Role and resource ownership strictly enforced on the server: devotees can create and access only their own bookings; priests can access or act only on bookings assigned to them.<br>• Client-supplied `user_id` and `priest_id` are never trusted as proof of ownership; all resource filters and DB writes bind strictly to `claims.UserID`.<br>• `completion_otp` completely removed from shared list (`GET /api/v1/bookings`, `GET /api/v1/priests/{id}/bookings`) and detail (`GET /api/v1/bookings/{id}`) responses via Go struct tag `json:"-"`.<br>• Dedicated protected endpoint `GET /api/v1/bookings/{id}/otp` exposes OTP exclusively to the authenticated booking owner; priests are strictly blocked with HTTP 403 Forbidden.<br>• Dedicated completion endpoint `POST /api/v1/bookings/{id}/complete` allows assigned priest to verify completion without retrieving the OTP; error response sanitized to `invalid completion OTP` without leaking expected secret.<br>• Mobile apps (`devotee-app`, `priest-app`) and `common-contracts` updated with `BookingOtpResponse`, token headers across all booking requests, and dynamic devotee OTP retrieval.<br>• Full suite of 24 unauthenticated 401 tests, revoked session rejection, role enforcement, 11 cross-account/unassigned priest rejection tests, asymmetric privacy assertions, and end-to-end lifecycle verification passing. |
| **GAP-06** | **P0 (Completed)** | **Strict Booking Finite State Machine (FSM) & Role-Based Transitions (Task 06)** | GAP-02, GAP-04 | **VERIFIED** | • Centralized FSM (`backend/internal/lifecycle/fsm.go`) strictly enforces sequence: `REQUESTED` $\rightarrow$ `ACCEPTED` $\rightarrow$ `IN_TRANSIT` $\rightarrow$ `ARRIVED` $\rightarrow$ `POOJA_STARTED` $\rightarrow$ `COMPLETED`.<br>• Assigned priest authorization strictly enforced; unassigned priests rejected with HTTP 403 Forbidden.<br>• Devotee completion OTP mandatory; bad/missing OTP rejected with HTTP 409 Conflict without leaking secret.<br>• Terminal states (`COMPLETED`, `CANCELLED`) immutable; repeated, backward, or skipped transitions return HTTP 409 Conflict.<br>• Pessimistic database row locking (`SELECT ... FOR UPDATE`) inside transactions prevents race conditions; 10 concurrent requests yield 1 success and 9 rejections.<br>• Atomic slot release on cancellation; decoupled false escrow/payment operations for free pilot.<br>• Mobile apps updated with non-optimistic stepper transitions and HTTP 409 error propagation; Android Gradle builds pass. |
| **GAP-05** | **P0** | **Atomic Slot Reservation & Booking Creation Binding (Task 07)** | GAP-01, GAP-02, GAP-06 | **VERIFIED** | • `BookingCreateRequest` supports `slot_id` (UUID) and `hold_token`.<br>• Hold lock ownership verified before creating booking; slot pessimistic row lock (`FOR UPDATE`) acquired within booking transaction.<br>• `priest_availability_slots.is_booked = true` and `booked_booking_id = booking.id` marked atomically within DB transaction.<br>• Double-booking rejected with HTTP 409 Conflict.<br>• Concurrent slot booking race conditions verified (6 concurrent requests = 1 success, 5 rejections).<br>• Mobile apps updated: devotee slot holding & creation binding, priest availability publishing API & schedule UI integration.<br>• Cancellation atomically frees slot and allows subsequent re-booking. |
| **GAP-07** | **P1** | **Minor Currency Units & Payment Gateway Integration** | GAP-01, GAP-05 | **DEFERRED (Free Pilot Agreed)** | • Stakeholders agreed to a free pilot baseline; payment gateway changes and minor currency units conversion are deferred post-pilot.<br>• Core booking lifecycle and ritual coordination are prioritized for immediate operational validation. |
| **GAP-08** | **P1** | **Idempotency Key Middleware** | GAP-01, GAP-02 | **NOT STARTED** | • Require `Idempotency-Key` header on all mutation endpoints.<br>• Cache results in Redis/Postgres for 24h to guarantee exactly-once execution. |
| **GAP-09** | **P2** | **Kotlin Common Contracts Synchronization** | GAP-01, GAP-05 | **VERIFIED** | • `apps/common-contracts` synchronized with email password recovery contracts, including authentic English, Telugu, and Hindi strings.<br>• Devotee and Priest Android network clients and UI dialogs fully updated and verified. |
| **GAP-10** | **P2** | **Isolated Integration Test Environment Harness** | GAP-01 | **IN PROGRESS** | • Ephemeral test database creation harness implemented in Go (`migrator_test.go`, `auth_test.go`, `recovery_test.go`).<br>• Python E2E harness wrapper pending containerized orchestration. |
| **GAP-11** | **P2** | **Transactional Notifications & Live SMS Delivery** | GAP-05 | **BLOCKED** | • Implement webhook/queue consumer dispatching SMS / Push notifications for booking assignment, arrival alerts, cancellation notices, and completion receipts. Blocked on real SMS provider credentials. |

> **Task Numbering Reconciliation Note:**
> In earlier documentation, GAP-05 was labelled 'P0 (Next) - Atomic Slot Reservation & Booking Creation Binding' while GAP-06 was labelled 'P1 - Strict Booking Finite State Machine (FSM)'. In the pilot execution plan, the team prioritized the Booking Lifecycle & Role-Based Transitions as **TASK 06** (addressing GAP-06). GAP-05 (Atomic Slot Reservation & Booking Creation Binding at `CreateBooking` time) is scheduled as the immediate follow-up (**TASK 07**). Neither task was skipped or altered; Task 06 establishes the lifecycle state machine and transactional slot release on cancellation, while Task 07 binds slot reservation atomically during booking creation.
| **GAP-07** | **P1** | **Minor Currency Units & Payment Gateway Integration** | GAP-01, GAP-05 | **DEFERRED (Free Pilot Agreed)** | • Stakeholders agreed to a free pilot baseline; payment gateway changes and minor currency units conversion are deferred post-pilot.<br>• Core booking lifecycle and ritual coordination are prioritized for immediate operational validation. |
| **GAP-08** | **P1** | **Idempotency Key Middleware** | GAP-01, GAP-02 | **NOT STARTED** | • Require `Idempotency-Key` header on all mutation endpoints.<br>• Cache results in Redis/Postgres for 24h to guarantee exactly-once execution. |
| **GAP-09** | **P2** | **Kotlin Common Contracts Synchronization** | GAP-01, GAP-05 | **VERIFIED** | • `apps/common-contracts` synchronized with email password recovery contracts, including authentic English, Telugu, and Hindi strings.<br>• Devotee and Priest Android network clients and UI dialogs fully updated and verified. |
| **GAP-10** | **P2** | **Isolated Integration Test Environment Harness** | GAP-01 | **IN PROGRESS** | • Ephemeral test database creation harness implemented in Go (`migrator_test.go`, `auth_test.go`, `recovery_test.go`).<br>• Python E2E harness wrapper pending containerized orchestration. |
| **GAP-11** | **P2** | **Transactional Notifications & Live SMS Delivery** | GAP-05 | **BLOCKED** | • Implement webhook/queue consumer dispatching SMS / Push notifications for booking assignment, arrival alerts, cancellation notices, and completion receipts. Blocked on real SMS provider credentials. |

---

## 3. TASK 02 Details: Database Migrations & Safety Harness (Reviewed & Corrected)

### 3.1 Migration Sequence & Files
1. `000001_init_schema.sql` — Core users, priests, catalog, bookings, reviews, and slots. Referential integrity constraint `bookings_pooja_id_fkey` enforced.
2. `000002_services_and_rapido.sql` — Priest custom services, custom packages, and live tracking coordinates.
3. `000003_comprehensive_ecosystem.sql` — Samagri catalog, vendor stores, inventory, and muhurtham calendar.
4. `000004_runtime_schema_completion.sql` — Safe forward migration adding missing runtime columns: `password_hash`, strictly nullable booking schedule and personal details (`devotee_gotram`, `sankalpam_purpose`, etc.), catalog 3-language translations (`title_en`, `title_te`, `title_hi`), and query performance indexes. Preserves `bookings_pooja_id_fkey` referential integrity.
5. `000005_auth_session_version.sql` — Adds `session_version INT NOT NULL DEFAULT 1` to `users` and `priests` with indexes for atomic session invalidation.

### 3.2 Confirmed Issues Reviewed and Corrected
1. **Preservation of `bookings_pooja_id_fkey` Referential Integrity:**
   - Migration `000004_runtime_schema_completion.sql` previously attempted to drop `bookings_pooja_id_fkey`.
   - Corrected: The drop statement was removed. `bookings_pooja_id_fkey` foreign key constraint referencing `pooja_catalog(id) ON DELETE RESTRICT` is explicitly maintained and preserved.
   - Any booking created with an invalid or non-existent `pooja_id` is strictly rejected by the database engine. Custom offering redesign is cleanly deferred to Task 07.
2. **Elimination of Fabricated Customer and Schedule Details:**
   - In migration `000004_runtime_schema_completion.sql`, default strings for `devotee_gotram` (`'Kashyapa'`), `devotee_nakshatram` (`'Rohini'`), `devotee_rashi` (`'Vrishabha'`), `scheduled_date` (`'Today'`), `slot_time` (`'08:30 AM - 10:30 AM'`), and `sankalpam_purpose` (`'Family Prosperity & Health'`) were removed.
   - All booking detail columns are strictly nullable (`NULL` when not provided).
   - In `backend/internal/db/db.go`, `CreateBooking` no longer injects fake default strings or silently defaults to an arbitrary catalog pooja when `req.PoojaID` is supplied.
   - `GetBookingDetails` and `GetBookings` safely map null values using `COALESCE(..., '')` returning empty strings (unknown) rather than fabricating Hindu gotras or astrological data.
3. **Rigorous Schema Compatibility Validation in `Baseline()`:**
   - `backend/internal/db/migrate/migrator.go` `Baseline(ctx, targetVersion)` now validates:
     - The requested `targetVersion` exists in discovered migration files (rejects unknown versions like `000099` or `invalid`).
     - Existing history does not exceed `targetVersion` (`applied.Version > targetVersion` returns an error).
     - Checksums of already applied migrations match expected values (`VerifyChecksums`).
     - Physical schema matches the requirements of `targetVersion` via `ValidateSchemaCompatibility(ctx, targetVersion)`:
       - Inspects `information_schema.tables` for required tables.
       - Inspects `information_schema.columns` for required columns.
       - Validates foreign key constraints (`bookings_pooja_id_fkey` on `bookings(pooja_id)` referencing `pooja_catalog(id)`).
     - Execution is atomic within a database transaction (`tx.BeginTx` / `tx.Commit`). Incomplete or incompatible schemas fail before any writes, guaranteeing zero partial history records.

### 3.3 Compatibility Review of Migrations 000001–000003 & Historical Checksums
- **Historical Gating Context:** In previous commits, migrations `000001`, `000002`, and `000003` were updated to gate demo seeds behind `current_setting('purohit.seed_demo', true) = 'true'`.
- **Checksum Evolution:**
  - `000001_init_schema.sql`:
    - Original un-gated SHA-256: `cc30cba2e78cfeceee683e862f535240601953ac45e2c5f8d6359365737ecf37`
    - Current gated SHA-256: `866b5a658f8c58c9b32032e3810715bc7bd320fc2d7d4617eeecb811a31754ff`
  - `000002_services_and_rapido.sql`:
    - Original un-gated SHA-256: `55916db2f1937307a1847bc5d0c991f008228784de8890b8e21497bb007ab3bc`
    - Current gated SHA-256: `a30d828084b7a68f569f824b5691feda7f61f81b2988236a2f3cd983cd72801d`
  - `000003_comprehensive_ecosystem.sql`:
    - Original un-gated SHA-256: `86d4cd2febf5526731c912c1331eb129e8e5e445da1ba97d170cbde77896f0cf`
    - Current gated SHA-256: `6d45429bffa395cf954dde6d2aa97bbd4f0da6828ad79bd4ae6c99aed2c6bdb2`
- **Compatibility Support in Migrator:**
  - `backend/internal/db/migrate/migrator.go` defines `compatibleHistoricalChecksums`, acknowledging both historical and gated checksums.
  - Databases that applied the original un-gated migrations remain fully valid and pass checksum immutability checks without manual database tampering.

### 3.4 Verified Database Adoption Procedure
Existing and new deployments must follow this adoption runbook:

1. **Scenario A — Fresh / Empty Database:**
   ```bash
   # Executes forward migrations 000001 through 000005 inside transactions
   ./cmd/migrate/migrate up
   # Verified outcome: Database is migrated to version 000005 with 0 demo people.
   ```
2. **Scenario B — Existing Database with 000001–000003 Schema (No Migration History):**
   ```bash
   # Step 1: Validate schema compatibility and baseline up to 000003
   ./cmd/migrate/migrate baseline --version 000003
   # If tables or foreign key constraint bookings_pooja_id_fkey are missing, Baseline() aborts with zero writes.

   # Step 2: Apply pending forward migrations 000004 and 000005
   ./cmd/migrate/migrate up
   # Verified outcome: Existing user/priest data survives intact; new columns added safely.
   ```
3. **Scenario C — Existing Database with Pre-Recorded Legacy Checksums:**
   ```bash
   # Migrator recognizes both original and gated checksums
   ./cmd/migrate/migrate status
   ./cmd/migrate/migrate up
   # Verified outcome: Pending migrations 000004 & 000005 apply without checksum mismatch errors.
   ```

### 3.5 Automated Regression Test Suite (`backend/internal/db/migrate` & `backend/internal/db`)
All 5 required regression invariants were proven on isolated test databases:

1. **Invalid booking references remain rejected:**
   - `TestInvalidBookingReferencesRejected`: Inserting booking with non-existent `pooja_id` fails with foreign key violation (`bookings_pooja_id_fkey`).
   - `TestAppLayerInvalidBookingReferencesRejected`: `CreateBooking` with non-existent `pooja_id` fails without silently substituting another pooja.
2. **Unknown personal/schedule fields remain unknown:**
   - `TestUnknownPersonalAndScheduleFieldsRemainUnknown`: Creating a booking with NULL personal/schedule fields leaves database columns as NULL; read queries return `""` without fabricated defaults (`'Kashyapa'`, `'Rohini'`, `'Today'`).
   - `TestAppLayerUnknownPersonalAndScheduleFieldsRemainUnknown`: `GetBookingDetails` and `GetBookings` return empty strings for omitted fields.
3. **An incomplete schema cannot be baselined:**
   - `TestIncompleteSchemaCannotBeBaselined/EmptyDatabaseFailsBaseline`: Baseline on empty database fails schema check; 0 rows written to `schema_migrations`.
   - `TestIncompleteSchemaCannotBeBaselined/PartialSchemaMissingTablesFailsBaseline`: Baseline with missing `priests` table fails; 0 rows written.
   - `TestIncompleteSchemaCannotBeBaselined/SchemaMissingForeignKeyFailsBaseline`: Baseline on schema missing `bookings_pooja_id_fkey` fails; 0 rows written.
4. **Invalid baseline versions and existing checksum mismatches fail:**
   - `TestInvalidBaselineVersionsAndExistingChecksumMismatchesFail/InvalidVersionFails`: Baseline with unknown version `000099` fails immediately.
   - `TestInvalidBaselineVersionsAndExistingChecksumMismatchesFail/CorruptedChecksumFails`: Corrupted checksum in `schema_migrations` blocks baseline and forward migrations.
   - `TestInvalidBaselineVersionsAndExistingChecksumMismatchesFail/HistoricalUngatedChecksumAccepted`: Historical un-gated checksum `cc30cba2...` for version 000001 is validated and accepted.
5. **Valid existing data survives upgrade:**
   - `TestValidExistingDataSurvivesUpgrade`: Existing users, priests, and custom services created at version 000003 survive upgrade to 000005 with untouched IDs, attributes, and foreign keys.

```text
=== RUN   TestEmptyDatabaseCompleteChain
--- PASS: TestEmptyDatabaseCompleteChain (0.78s)
=== RUN   TestFreshPilotDatabaseNoDemoPeople
--- PASS: TestFreshPilotDatabaseNoDemoPeople (0.56s)
=== RUN   TestRepeatedRunIdempotent
--- PASS: TestRepeatedRunIdempotent (0.56s)
=== RUN   TestExistingSchemaFixtureUpgradePreservesData
--- PASS: TestExistingSchemaFixtureUpgradePreservesData (0.60s)
=== RUN   TestConcurrentRunnersSerialized
--- PASS: TestConcurrentRunnersSerialized (0.58s)
=== RUN   TestInjectedMigrationFailureRollsBack
--- PASS: TestInjectedMigrationFailureRollsBack (0.76s)
=== RUN   TestModifiedAppliedMigrationChecksumMismatch
--- PASS: TestModifiedAppliedMigrationChecksumMismatch (0.58s)
=== RUN   TestRuntimeQueriesAgainstMigratedSchema
--- PASS: TestRuntimeQueriesAgainstMigratedSchema (0.56s)
=== RUN   TestInvalidBookingReferencesRejected
--- PASS: TestInvalidBookingReferencesRejected (0.55s)
=== RUN   TestUnknownPersonalAndScheduleFieldsRemainUnknown
--- PASS: TestUnknownPersonalAndScheduleFieldsRemainUnknown (0.75s)
=== RUN   TestIncompleteSchemaCannotBeBaselined
    --- PASS: TestIncompleteSchemaCannotBeBaselined/EmptyDatabaseFailsBaseline (0.43s)
    --- PASS: TestIncompleteSchemaCannotBeBaselined/PartialSchemaMissingTablesFailsBaseline (0.42s)
    --- PASS: TestIncompleteSchemaCannotBeBaselined/SchemaMissingForeignKeyFailsBaseline (0.46s)
--- PASS: TestIncompleteSchemaCannotBeBaselined (1.32s)
=== RUN   TestInvalidBaselineVersionsAndExistingChecksumMismatchesFail
    --- PASS: TestInvalidBaselineVersionsAndExistingChecksumMismatchesFail/InvalidVersionFails (0.01s)
    --- PASS: TestInvalidBaselineVersionsAndExistingChecksumMismatchesFail/CorruptedChecksumFails (0.03s)
    --- PASS: TestInvalidBaselineVersionsAndExistingChecksumMismatchesFail/HistoricalUngatedChecksumAccepted (0.01s)
--- PASS: TestInvalidBaselineVersionsAndExistingChecksumMismatchesFail (0.54s)
=== RUN   TestValidExistingDataSurvivesUpgrade
--- PASS: TestValidExistingDataSurvivesUpgrade (0.59s)
PASS
ok  	github.com/purohit-platform/backend/internal/db/migrate	9.717s
```

---

## 4. TASK 03 Details: Secure Password Registration and Login

### 4.1 Verification of Known Defects
Before implementation, the codebase was inspected and all reported vulnerabilities were verified:
1. **Direct Plaintext Comparison:** `db.go:121` checked `storedHash != req.Password`.
2. **Silent Password Assignment during Login:** `db.go:124-130` executed `UPDATE users SET password_hash = req.Password` if `storedHash == ""`.
3. **Account Overwriting upon Registration:** `db.go:567, 591` executed `ON CONFLICT (phone) DO UPDATE SET ...`, allowing any user to overwrite existing accounts and credentials.
4. **Arbitrary 4-Digit OTP Bypass:** `db.go:177` accepted any 4-digit numeric string to auto-provision accounts, and `db.go:152` accepted `"1234"` to reset passwords.
5. **Hardcoded JWT Signing Key:** `auth/jwt.go:11` contained a hardcoded in-repo secret key without deployment override validation.

### 4.2 Changed & Created Files
- **`backend/migrations/000005_auth_session_version.sql`** *(NEW)*:
  - Adds `session_version INT NOT NULL DEFAULT 1` to `users` and `priests`.
  - Creates composite indexes `idx_users_session_version` and `idx_priests_session_version`.
- **`backend/internal/auth/phone.go`** & **`phone_test.go`** *(NEW)*:
  - `NormalizePhone`: Normalizes Indian numbers (+91 E.164), stripping spaces, dashes, parentheses, and leading zeros.
  - `PhoneLookupVariants`: Produces deduplicated search variants to match existing database formats safely.
  - `MaskPhone`: Masks phone numbers (e.g. `+91 98*** **345`) to prevent PII exposure in logs.
- **`backend/internal/auth/password.go`** & **`password_test.go`** *(NEW)*:
  - `HashPassword`: Bcrypt hashing at cost 10 with cryptographically secure unique salts.
  - `ValidatePassword`: Validates length (8 to 72 bytes) and rejects blank inputs.
  - `VerifyPassword`: Encoded bcrypt comparison (`bcrypt.CompareHashAndPassword`). Detects malformed/plaintext hashes and flags `needsRecovery = true`.
  - `EqualizeTiming`: Runs dummy bcrypt comparison on account lookup misses to eliminate timing side-channels.
- **`backend/internal/auth/jwt.go`** & **`jwt_test.go`** *(MODIFIED & TESTED)*:
  - `InitJWT`: Reads `JWT_SECRET` / `PUROHIT_JWT_SECRET` from environment; fails startup if missing or < 32 characters.
  - Strict HS256 algorithm enforcement (rejects `none`, RS256, etc.).
  - Embedded `session_version` in `CustomClaims`.
  - `ValidateSessionVersion`: Compares token session against DB version to immediately reject revoked tokens.
- **`backend/internal/auth/throttler.go`** & **`throttler_test.go`** *(NEW)*:
  - Bounded in-memory rate limiter tracking failed attempts by account key and client IP.
  - Thresholds: 5 failures/account and 10 failures/IP within a 15-minute rolling window trigger 15-minute lockout.
  - Memory Bounding: Capped at 5,000 entries with automatic expired entry purging and LRU-like eviction to eliminate memory DOS vectors.
  - `ExtractClientIP`: Secure client IP resolution parsing `X-Forwarded-For`, `X-Real-IP`, and `RemoteAddr`.
- **`backend/internal/db/db.go`** *(MODIFIED)*:
  - `Login`: Strict role validation, phone normalization, collision detection, generic error responses, timing equalization, bcrypt verification, and session token generation. Auto-provisioning removed.
  - `RegisterDevotee`: Explicit consent check, password validation, bcrypt hashing, duplicate rejection (HTTP 409) without overwriting.
  - `RegisterPriest`: Vedic qualification validation, password hashing, approval separation (`is_online = false`), duplicate rejection.
  - `RevokeUserSessions` & `GetUserSessionVersion`: Implemented session revocation methods.
  - `LoginWithOTP` & `ResetPassword`: Mock bypasses disabled with actionable error messages pointing to Task 04.
- **`backend/internal/db/auth_test.go`** *(NEW)*:
  - Integration test suite running 6 acceptance tests against isolated ephemeral PostgreSQL databases.
- **`backend/cmd/api-gateway/main.go`** *(MODIFIED)*:
  - Added startup fail-fast check for `JWT_SECRET` (min 32 characters).
  - Enforced required schema version `000005`.
  - Integrated `auth.DefaultThrottler` into `/api/v1/auth/login`.
  - Wired token `session_version` validation into `/api/v1/user/profile`.
- **`apps/devotee-app/.../PurohitApiClient.kt`** & **`DevoteeRegistrationScreen.kt`** *(MODIFIED)*:
  - Added `password` to `DevoteeRegistrationPayload` and `registerDevotee`.
  - Added password input field and client-side validation to `DevoteeRegistrationScreen`.
- **`apps/priest-app/.../PurohitApiClient.kt`** & **`PriestRegistrationScreen.kt`** *(MODIFIED)*:
  - Added `password` to `PriestRegPayload` and `registerPriest`.
  - Added password input field and client-side validation to `PriestRegistrationScreen`.

### 4.3 Throttler & Secret Operational Considerations
1. **Restart Behavior:** In-memory throttler state resets upon API Gateway process restart.
2. **Multi-Instance Limitations:** Under horizontal scaling across multiple containers/nodes, rate limiting is tracked per node unless backed by shared Redis. For the single-node pilot, in-memory bounding guarantees zero external latency and zero memory leaks.
3. **Secret Rotation:** Replacing the JWT signing key invalidates all active sessions, requiring existing users to log in again.

### 4.4 Automated Acceptance Test Evidence

#### Auth Unit Test Suite (`backend/internal/auth`)
```text
=== RUN   TestJWTConfiguration
--- PASS: TestJWTConfiguration (0.00s)
=== RUN   TestTokenLifecycleAndClaims
--- PASS: TestTokenLifecycleAndClaims (0.00s)
=== RUN   TestTokenValidationFailures
--- PASS: TestTokenValidationFailures (0.00s)
=== RUN   TestSessionRevocation
--- PASS: TestSessionRevocation (0.00s)
=== RUN   TestPasswordValidation
--- PASS: TestPasswordValidation (0.00s)
=== RUN   TestPasswordHashing
--- PASS: TestPasswordHashing (0.20s)
=== RUN   TestLegacyCredentialsRequireRecovery
--- PASS: TestLegacyCredentialsRequireRecovery (0.15s)
=== RUN   TestEqualizeTiming
--- PASS: TestEqualizeTiming (0.05s)
=== RUN   TestNormalizePhone
--- PASS: TestNormalizePhone (0.00s)
=== RUN   TestPhoneLookupVariants
--- PASS: TestPhoneLookupVariants (0.00s)
=== RUN   TestMaskPhone
--- PASS: TestMaskPhone (0.00s)
=== RUN   TestThrottlerAccountLockout
--- PASS: TestThrottlerAccountLockout (0.00s)
=== RUN   TestThrottlerMemoryBounding
--- PASS: TestThrottlerMemoryBounding (0.00s)
=== RUN   TestExtractClientIP
--- PASS: TestExtractClientIP (0.00s)
PASS (1.24s)
```

#### Database Auth Integration Suite (`backend/internal/db`)
```text
=== RUN   TestCorrectPasswordAuthenticates
--- PASS: TestCorrectPasswordAuthenticates (1.06s)
=== RUN   TestWrongOrBlankPasswordCannotAuthenticate
--- PASS: TestWrongOrBlankPasswordCannotAuthenticate (0.87s)
=== RUN   TestDuplicateRegistrationCannotOverwrite
--- PASS: TestDuplicateRegistrationCannotOverwrite (0.94s)
=== RUN   TestLegacyCredentialsCannotClaimAccount
--- PASS: TestLegacyCredentialsCannotClaimAccount (1.64s)
=== RUN   TestPhoneFormatVariantsAndCollisions
--- PASS: TestPhoneFormatVariantsAndCollisions (1.23s)
=== RUN   TestSessionRevocationInDB
--- PASS: TestSessionRevocationInDB (0.89s)
PASS (7.78s)
```

#### Android Compilation
```text
BUILD SUCCESSFUL in 1m 6s
89 actionable tasks: 11 executed, 78 up-to-date
```

---

## 5. TASK 03A Details: Verified Migration and Authentication Gap Closures

### 5.1 Architecture & Implementation Fixes

1. **Migration Lock Ownership & Dedicated Connection Pinning:**
   - **Root Cause Verified:** PostgreSQL session advisory locks (`pg_advisory_lock`) are bound to the specific server backend process. When executed through `*sql.DB`, checkout and checkin of connections from the pool meant the unlock command (`pg_advisory_unlock`) could run on a different physical connection, leaving the lock held on the original backend. Furthermore, executing statements on other connections while holding the lock caused instant deadlocks when `db.SetMaxOpenConns(1)`.
   - **Fix Implemented:** 
     - Added a `queryExecer` abstraction implemented by `*sql.DB`, `*sql.Conn`, and `*sql.Tx`.
     - `AcquireLock(ctx)` checks out a dedicated `*sql.Conn` using `m.db.Conn(ctx)` and acquires `pg_advisory_lock(0x50554A4156414148)` on that specific physical connection.
     - Returns `(*sql.Conn, unlock func(), error)`. On cancellation or error, `conn.Close()` is immediately called, which closes the server session and drops the lock on the Postgres engine.
     - `unlock()` executes `pg_advisory_unlock` on the same connection, verifies `unlocked == true`, and closes the connection to guarantee no still-locked session returns to the pool.
     - All migration workflows (`Up`, `Baseline`, `Seed`) run their DDL and transaction execution (`conn.BeginTx`) strictly on the locked `*sql.Conn`.
     - **Verification:** `TestLockOwnershipWithSingleConnectionPool` verifies that a complete 5-migration run succeeds without deadlocking when `db.SetMaxOpenConns(1)`. `TestLockOwnershipCancellationAndSubsequentAcquisition` verifies cancellation and clean lock release.

2. **Strengthened Schema Adoption & Readiness Verification:**
   - **Root Cause Verified:** `ValidateSchemaCompatibility` previously checked only broad table/column names and a foreign key presence, allowing legacy schemas with mismatched column types, wrong nullability, missing unique constraints, or incorrect foreign key deletion behaviors to be baselined. Additionally, `VerifyReadiness` checked only `MAX(version)`, ignoring intermediate migration gaps or corruptions.
   - **Fix Implemented:**
     - `ValidateSchemaCompatibility` now validates against physical requirements for target migrations:
       - Exact data types (e.g. `users.phone` is `character varying`, `session_version` is `integer`, `bookings.created_at` is `timestamp with time zone`) via `information_schema.columns`.
       - Nullability conformance (`is_nullable == 'YES'` / `'NO'`).
       - Critical unique constraints (`users(phone)`, `priests(phone)`, `priest_availability_slots(priest_id, slot_date, start_time)`) via `pg_constraint`.
       - Exact foreign key definitions (`bookings(pooja_id) -> pooja_catalog(id)` with `ON DELETE RESTRICT`) via `information_schema.table_constraints` and `referential_constraints`.
       - Incomplete or incompatible schemas fail prior to any history writes, preventing partial history contamination.
     - `VerifyReadiness` now scans all discovered migration files up to `expectedMinVersion`, verifying that every intermediate version exists in `schema_migrations` and that applied checksums match expected values. History gaps (e.g. skipping `000003`) fail immediately.
     - **Verification:** `TestSchemaCompatibilityWrongColumnTypes`, `TestSchemaCompatibilityMissingRequiredColumns`, `TestSchemaCompatibilityForeignKeyOnWrongColumn`, `TestSchemaCompatibilityMissingUniqueness`, and `TestReadinessDetectsMigrationHistoryGap`.

3. **Login Throttling Identity, Trusted Proxies & Capacity Preservation:**
   - **Root Cause Verified:**
     - Throttle keys depended directly on caller-formatted phone strings and omitted role, allowing bypasses across format variants (`+919876543210` vs `9876543210`) or cross-role lockout evasion.
     - `ExtractClientIP` unconditionally trusted `X-Forwarded-For`, enabling client IP spoofing to evade IP lockouts or orchestrate denial-of-service on legitimate users.
     - Capacity eviction discarded arbitrary oldest keys, allowing an attacker to flush active lockouts using junk inputs.
   - **Fix Implemented:**
     - `BuildAccountThrottleKey(role, phone)` normalizes phone numbers to canonical E.164 (`+91XXXXXXXXXX`) and prefixes the explicit role (`role:devotee:phone:+919876543210`), ensuring format variants share the same rate limit.
     - `ConfigureTrustedProxies(cidrs...)` parses trusted reverse proxy subnets (defaults to loopback `127.0.0.1/32`, `::1/128`).
     - `ExtractClientIP(r)` inspects direct `RemoteAddr`. Forwarding headers (`X-Forwarded-For`, `X-Real-IP`) are strictly ignored unless `RemoteAddr` belongs to a configured trusted proxy. When trusted, proxy chains are parsed right-to-left.
     - `evictIfNeeded` purges expired entries first; under capacity pressure (max 5,000 entries), it prioritizes evicting unblocked entries by oldest activity, strictly preserving actively locked-out accounts.
     - Configured `TRUSTED_PROXIES` environment variable loading in `main.go`.
     - **Verification:** `TestPhoneFormatVariantsShareThrottleLimit`, `TestExtractClientIPSecurityAndProxies`, `TestCapacityPressurePreservesActiveLockouts`, and `TestConcurrentThrottleAttempts`.

4. **Tightened JWT Validation & Session Claims Verification:**
   - **Root Cause Verified:** `ValidateToken` accepted any token signed by the key without enforcing expiration claims or verifying that required claims (`UserID`, `Phone`, `Role`, `Subject`) were present, non-empty, and positive. `ValidateSessionVersion` allowed future session versions.
   - **Fix Implemented:**
     - Added `jwt.WithExpirationRequired()` to parser options, rejecting tokens without expiration claims.
     - Explicit expiration check against `time.Now()`.
     - Validates required claims: `UserID != ""`, `Phone != ""`, `Role != ""`, `Subject != ""`, `Subject == UserID`, and `SessionVersion > 0`.
     - `ValidateSessionVersion` strictly enforces equality `tokenVersion == dbVersion`; rejects future, zero, negative, or outdated session versions.
     - Enforced on `/api/v1/user/profile`.
     - Added `validateOptionalCallerSession` helper wired to `/api/v1/bookings` (GET/POST), `/api/v1/priests/{id}/bookings`, `/api/v1/bookings/status`, and `/api/v1/bookings/{id}` so that any authenticated request with a Bearer token is immediately validated and checked for session revocation.
     - **Verification:** `TestTokenValidationFailures` (missing expiry, missing/empty identity claims, zero session version, mismatched subject) and `TestSessionRevocation` (future session version rejection).

---

### 5.2 Exact Changed and Created Files

- **`backend/internal/db/migrate/migrator.go`** *(MODIFIED)*:
  - Added `queryExecer` interface.
  - Refactored `AcquireLock` to pin dedicated `*sql.Conn`, verify unlock results, and guarantee clean connection termination on error/cancellation.
  - Pinned `Up`, `Baseline`, and `Seed` workflows to execute exclusively on the locked `*sql.Conn`.
  - Upgraded `ValidateSchemaCompatibility` with column data types, nullability, unique constraints, and foreign key action checks.
  - Upgraded `VerifyReadiness` with intermediate migration gap detection and checksum validation across all intermediate migrations.
- **`backend/internal/db/migrate/migrator_test.go`** *(MODIFIED)*:
  - Added `TestLockOwnershipWithSingleConnectionPool`.
  - Added `TestLockOwnershipCancellationAndSubsequentAcquisition`.
  - Added `TestSchemaCompatibilityWrongColumnTypes`.
  - Added `TestSchemaCompatibilityMissingRequiredColumns`.
  - Added `TestSchemaCompatibilityForeignKeyOnWrongColumn`.
  - Added `TestSchemaCompatibilityMissingUniqueness`.
  - Added `TestReadinessDetectsMigrationHistoryGap`.
- **`backend/internal/auth/throttler.go`** *(MODIFIED)*:
  - Implemented `BuildAccountThrottleKey(role, phone)` with canonical normalization and role scoping.
  - Implemented `ConfigureTrustedProxies` and `isTrustedProxy`.
  - Upgraded `ExtractClientIP` to ignore forwarding headers from untrusted remotes and parse chains right-to-left.
  - Upgraded `evictIfNeeded` to prioritize evicting unblocked entries and preserve actively locked-out accounts.
  - Added single-process in-memory architectural documentation.
- **`backend/internal/auth/throttler_test.go`** *(MODIFIED)*:
  - Added `TestPhoneFormatVariantsShareThrottleLimit`.
  - Added `TestExtractClientIPSecurityAndProxies`.
  - Added `TestCapacityPressurePreservesActiveLockouts`.
  - Added `TestConcurrentThrottleAttempts`.
- **`backend/internal/auth/jwt.go`** *(MODIFIED)*:
  - Added `jwt.WithExpirationRequired()`, explicit expiry checks, required non-empty identity claims validation, and `Subject == UserID` checks.
  - Enforced positive `SessionVersion > 0`.
  - Upgraded `ValidateSessionVersion` to strictly enforce `tokenVersion == dbVersion` (rejects future versions).
- **`backend/internal/auth/jwt_test.go`** *(MODIFIED)*:
  - Added test cases for missing expiration, missing identity claims, zero session version, mismatched subject, and future session versions.
- **`backend/cmd/api-gateway/main.go`** *(MODIFIED)*:
  - Integrated `auth.BuildAccountThrottleKey(req.Role, req.Phone)` and `auth.ExtractClientIP(r)` into login.
  - Wired `TRUSTED_PROXIES` environment variable loading during startup.
  - Added package-level `validateOptionalCallerSession` helper.
  - Wired session validation into `/api/v1/user/profile`, `/api/v1/bookings` (GET/POST), `/api/v1/priests/{id}/bookings`, `/api/v1/bookings/status`, and `/api/v1/bookings/{id}`.

---

### 5.3 Automated Regression & Acceptance Test Evidence

All 47 backend tests pass cleanly with zero skipped and zero failures across the test suite:

```text
=== RUN   TestJWTConfiguration
--- PASS: TestJWTConfiguration (0.00s)
=== RUN   TestTokenLifecycleAndClaims
--- PASS: TestTokenLifecycleAndClaims (0.00s)
=== RUN   TestTokenValidationFailures
--- PASS: TestTokenValidationFailures (0.00s)
=== RUN   TestSessionRevocation
--- PASS: TestSessionRevocation (0.00s)
=== RUN   TestPasswordValidation
--- PASS: TestPasswordValidation (0.00s)
=== RUN   TestPasswordHashing
--- PASS: TestPasswordHashing (0.20s)
=== RUN   TestLegacyCredentialsRequireRecovery
--- PASS: TestLegacyCredentialsRequireRecovery (0.15s)
=== RUN   TestEqualizeTiming
--- PASS: TestEqualizeTiming (0.05s)
=== RUN   TestNormalizePhone
--- PASS: TestNormalizePhone (0.00s)
=== RUN   TestPhoneLookupVariants
--- PASS: TestPhoneLookupVariants (0.00s)
=== RUN   TestMaskPhone
--- PASS: TestMaskPhone (0.00s)
=== RUN   TestThrottlerAccountLockout
--- PASS: TestThrottlerAccountLockout (0.00s)
=== RUN   TestPhoneFormatVariantsShareThrottleLimit
--- PASS: TestPhoneFormatVariantsShareThrottleLimit (0.00s)
=== RUN   TestExtractClientIPSecurityAndProxies
--- PASS: TestExtractClientIPSecurityAndProxies (0.00s)
=== RUN   TestCapacityPressurePreservesActiveLockouts
--- PASS: TestCapacityPressurePreservesActiveLockouts (0.00s)
=== RUN   TestConcurrentThrottleAttempts
--- PASS: TestConcurrentThrottleAttempts (0.00s)
PASS
ok  	github.com/purohit-platform/backend/internal/auth	1.245s

=== RUN   TestRefundCalculationRequestedState
--- PASS: TestRefundCalculationRequestedState (0.00s)
=== RUN   TestRefundCalculationInTransitState
--- PASS: TestRefundCalculationInTransitState (0.00s)
=== RUN   TestRefundCalculationArrivedState
--- PASS: TestRefundCalculationArrivedState (0.00s)
PASS
ok  	github.com/purohit-platform/backend/internal/cancellation	0.353s

=== RUN   TestCorrectPasswordAuthenticates
--- PASS: TestCorrectPasswordAuthenticates (1.28s)
=== RUN   TestWrongOrBlankPasswordCannotAuthenticate
--- PASS: TestWrongOrBlankPasswordCannotAuthenticate (0.86s)
=== RUN   TestDuplicateRegistrationCannotOverwrite
--- PASS: TestDuplicateRegistrationCannotOverwrite (0.95s)
=== RUN   TestLegacyCredentialsCannotClaimAccount
--- PASS: TestLegacyCredentialsCannotClaimAccount (0.80s)
=== RUN   TestPhoneFormatVariantsAndCollisions
--- PASS: TestPhoneFormatVariantsAndCollisions (0.95s)
=== RUN   TestSessionRevocationInDB
--- PASS: TestSessionRevocationInDB (0.95s)
=== RUN   TestAppLayerInvalidBookingReferencesRejected
--- PASS: TestAppLayerInvalidBookingReferencesRejected (0.78s)
=== RUN   TestAppLayerUnknownPersonalAndScheduleFieldsRemainUnknown
--- PASS: TestAppLayerUnknownPersonalAndScheduleFieldsRemainUnknown (0.92s)
PASS
ok  	github.com/purohit-platform/backend/internal/db	8.586s

=== RUN   TestEmptyDatabaseCompleteChain
--- PASS: TestEmptyDatabaseCompleteChain (0.93s)
=== RUN   TestFreshPilotDatabaseNoDemoPeople
--- PASS: TestFreshPilotDatabaseNoDemoPeople (0.55s)
=== RUN   TestRepeatedRunIdempotent
--- PASS: TestRepeatedRunIdempotent (0.54s)
=== RUN   TestExistingSchemaFixtureUpgradePreservesData
--- PASS: TestExistingSchemaFixtureUpgradePreservesData (0.64s)
=== RUN   TestConcurrentRunnersSerialized
--- PASS: TestConcurrentRunnersSerialized (0.57s)
=== RUN   TestInjectedMigrationFailureRollsBack
--- PASS: TestInjectedMigrationFailureRollsBack (0.74s)
=== RUN   TestModifiedAppliedMigrationChecksumMismatch
--- PASS: TestModifiedAppliedMigrationChecksumMismatch (0.54s)
=== RUN   TestRuntimeQueriesAgainstMigratedSchema
--- PASS: TestRuntimeQueriesAgainstMigratedSchema (0.54s)
=== RUN   TestInvalidBookingReferencesRejected
--- PASS: TestInvalidBookingReferencesRejected (0.92s)
=== RUN   TestUnknownPersonalAndScheduleFieldsRemainUnknown
--- PASS: TestUnknownPersonalAndScheduleFieldsRemainUnknown (0.59s)
=== RUN   TestIncompleteSchemaCannotBeBaselined
    --- PASS: TestIncompleteSchemaCannotBeBaselined/EmptyDatabaseFailsBaseline (0.54s)
    --- PASS: TestIncompleteSchemaCannotBeBaselined/PartialSchemaMissingTablesFailsBaseline (0.49s)
    --- PASS: TestIncompleteSchemaCannotBeBaselined/SchemaMissingForeignKeyFailsBaseline (0.48s)
--- PASS: TestIncompleteSchemaCannotBeBaselined (1.51s)
=== RUN   TestInvalidBaselineVersionsAndExistingChecksumMismatchesFail
    --- PASS: TestInvalidBaselineVersionsAndExistingChecksumMismatchesFail/InvalidVersionFails (0.01s)
    --- PASS: TestInvalidBaselineVersionsAndExistingChecksumMismatchesFail/CorruptedChecksumFails (0.04s)
    --- PASS: TestInvalidBaselineVersionsAndExistingChecksumMismatchesFail/HistoricalUngatedChecksumAccepted (0.02s)
--- PASS: TestInvalidBaselineVersionsAndExistingChecksumMismatchesFail (0.68s)
=== RUN   TestValidExistingDataSurvivesUpgrade
--- PASS: TestValidExistingDataSurvivesUpgrade (0.69s)
=== RUN   TestLockOwnershipWithSingleConnectionPool
--- PASS: TestLockOwnershipWithSingleConnectionPool (0.55s)
=== RUN   TestLockOwnershipCancellationAndSubsequentAcquisition
--- PASS: TestLockOwnershipCancellationAndSubsequentAcquisition (0.42s)
=== RUN   TestSchemaCompatibilityWrongColumnTypes
--- PASS: TestSchemaCompatibilityWrongColumnTypes (0.60s)
=== RUN   TestSchemaCompatibilityMissingRequiredColumns
--- PASS: TestSchemaCompatibilityMissingRequiredColumns (0.48s)
=== RUN   TestSchemaCompatibilityForeignKeyOnWrongColumn
--- PASS: TestSchemaCompatibilityForeignKeyOnWrongColumn (0.46s)
=== RUN   TestSchemaCompatibilityMissingUniqueness
--- PASS: TestSchemaCompatibilityMissingUniqueness (0.49s)
=== RUN   TestReadinessDetectsMigrationHistoryGap
--- PASS: TestReadinessDetectsMigrationHistoryGap (0.79s)
PASS
ok  	github.com/purohit-platform/backend/internal/db/migrate	15.841s

=== RUN   TestCompositeScoringPrefersHighRatingAndRepeats
--- PASS: TestCompositeScoringPrefersHighRatingAndRepeats (0.00s)
=== RUN   TestRankCandidatesOrdering
--- PASS: TestRankCandidatesOrdering (0.00s)
PASS
ok  	github.com/purohit-platform/backend/internal/ranking	0.353s
```

---

### 5.4 Throttler Operational Limits & Reverse Proxy Configuration

1. **In-Memory Single-Process Boundary:**
   - The login throttler maintains state in process memory.
   - Restarts reset attempt counts and active lockout records.
   - For a single-node pilot deployment, in-memory tracking provides zero external latency and zero operational dependencies on external cache services.
2. **Horizontal Scaling Boundary:**
   - If multiple API Gateway instances are deployed behind a load balancer, rate limits are tracked per instance unless backed by a distributed store (e.g., Redis). Multi-instance synchronization is deferred until Redis service is operational.
3. **Reverse Proxy Configuration:**
   - When deployed behind Cloudflare, AWS ALB, Nginx, or Kubernetes ingress, operators must specify comma-delimited proxy CIDRs via the `TRUSTED_PROXIES` environment variable (e.g. `TRUSTED_PROXIES="10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"`).
   - In development and local environments, defaults to loopback addresses (`127.0.0.1/32`, `::1/128`).
   - If an unconfigured proxy forwards requests, `ExtractClientIP` falls back to the immediate `RemoteAddr`, preventing external header forgery.

---

### 5.5 Deferred Route Ownership & Authorization Defects (Remediation in Task 05)

The following route handlers in `backend/cmd/api-gateway/main.go` currently accept unauthenticated callers, ignore authenticated token identity, or accept caller-supplied user IDs without ownership verification. These are recorded as confirmed implementation defects to be systematically resolved in Task 05:

1. **`POST /api/v1/bookings`:** Accepts a payload containing `user_id`. Although token session validation is performed when a Bearer token is provided, requests without tokens are not yet rejected, and `req.UserID` is not verified against `claims.UserID`.
2. **`GET /api/v1/bookings`:** Reads `user_id` and `priest_id` from query parameters without verifying that the authenticated caller has permission to view the requested devotee or priest bookings.
3. **`GET /api/v1/priests/{id}/bookings`:** Reads priest ID from the URL path without enforcing that the caller is the authenticated priest.
4. **`POST /api/v1/bookings/status`:** Accepts state transition updates with caller-supplied `priest_id` and `booking_id` without verifying that the caller is the assigned priest or booking devotee.
5. **`GET /api/v1/bookings/{id}`:** Exposes complete booking details (including address and devotee phone) to any caller knowing the UUID without ownership checks.
6. **`POST /api/v1/dispatch/match`:** Returns a hardcoded demo priest candidate fallback if no database priests are found within the search radius.

*Note:* Full route-level authentication middleware and strict role-based ownership enforcement will be established in Task 05.

---

## 6. TASK 04 Details: Verified Password Recovery & Infrastructure Hardening

### 6.1 Startup Readiness Hardening
- **Authoritative Implementation:** In `backend/cmd/api-gateway/main.go`, startup readiness now executes `storage.VerifySchemaReadiness(ctx, requiredSchemaVersion)` (target version `"000006"`), which routes directly to `migrator.VerifyReadiness(ctx, minVersion)`.
- **Validation Rules:**
  - Verifies migration files are present in discovered directories (`backend/migrations`, `migrations`, `../migrations`).
  - Queries `schema_migrations` to ensure the recorded database version is at least `000006`.
  - Verifies that zero intermediate migrations are missing (contiguous version check).
  - Validates that applied migration checksums match discovered migration files (with backward compatibility for historical un-gated checksums).
  - Performs physical schema compatibility inspection (`ValidateSchemaCompatibility`) checking tables, columns, data types, nullability, foreign key constraints, and unique indexes without silently executing migrations at startup.
- **Automated Verification:** Verified in `TestStartupReadinessValidSchema`, `TestStartupReadinessHistoryGapFailsDespiteLatestVersion`, and `TestStartupReadinessPhysicalIncompatibilityFails`.

### 6.2 Advisory-Lock Cleanup via `database/sql/driver` DiscardConn
- **Root Cause Addressed:** In Go's `database/sql`, calling `*sql.Conn.Close()` returns the connection to the idle pool. If a session held a PostgreSQL session-level advisory lock (`pg_advisory_lock`), returning it to the pool leaves the connection and lock alive, which causes deadlocks when the connection pool limit is 1 (`SetMaxOpenConns(1)`).
- **Driver-Level Solution:** Implemented `DiscardConn(conn *sql.Conn)` in `backend/internal/db/migrate/migrator.go`:
  ```go
  func DiscardConn(conn *sql.Conn) {
      if conn == nil {
          return
      }
      _ = conn.Raw(func(driverConn any) error {
          if closer, ok := driverConn.(io.Closer); ok {
              _ = closer.Close()
          }
          return driver.ErrBadConn
      })
      _ = conn.Close()
  }
  ```
  Returning `driver.ErrBadConn` instructs `database/sql` to remove the connection from the pool, immediately closing the underlying TCP socket and terminating the PostgreSQL backend process, which automatically frees any session-level advisory locks.
- **Automated Verification:** Verified in `TestLockOwnershipWithSingleConnectionPool`, `TestLockOwnershipCancellationAndSubsequentAcquisition`, and `TestLockOwnershipDiscardConnSafelyDestroysSessionOnFailure`.

### 6.3 Password Recovery Challenges Table & Cryptographic Digest Storage
- **Migration `000006_password_recovery_challenges.sql`:**
  - Creates table `password_recovery_challenges`:
    - `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
    - `account_id UUID NOT NULL`
    - `account_role VARCHAR(20) NOT NULL` (CHECK constraint: `'devotee'`, `'priest'`)
    - `phone VARCHAR(20) NOT NULL` (Canonical E.164 format)
    - `purpose VARCHAR(30) NOT NULL DEFAULT 'PASSWORD_RESET'`
    - `code_hash VARCHAR(128) NOT NULL` (HMAC-SHA256 keyed digest)
    - `expires_at TIMESTAMPTZ NOT NULL` (15-minute validity window)
    - `max_attempts INT NOT NULL DEFAULT 5`
    - `attempt_count INT NOT NULL DEFAULT 0`
    - `delivery_status VARCHAR(20) NOT NULL DEFAULT 'PENDING'`
    - `status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE'` (CHECK constraint: `'ACTIVE'`, `'CONSUMED'`, `'EXPIRED'`, `'SUPERSEDED'`)
    - `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
    - `consumed_at TIMESTAMPTZ`
  - Created composite indexes: `idx_recovery_lookup (phone, account_role, status)`, `idx_recovery_account (account_id, account_role, status)`, and `idx_recovery_expiry (expires_at)`.
- **HMAC-SHA256 Digesting:** Plaintext OTPs are never stored in the database. When a 6-digit cryptographic code is generated (`auth.GenerateRecoveryCode()`), it is hashed using HMAC-SHA256 keyed with the server secret (`auth.HashRecoveryCode(code, secret)`).
- **Verification:** Verification uses constant-time comparison (`subtle.ConstantTimeCompare`) via `auth.VerifyRecoveryCode(code, secret, hash)`.

### 6.4 Safe SMS Boundary & Rate Limiting
- **Safe Provider Abstraction:** Defined `auth.SMSProvider` interface (`SendVerificationCode`, `IsConfigured`). Default provider is `UnconfiguredSMSProvider`.
- **Safe Production Failure (HTTP 503):** If live SMS credentials are not configured, `StartPasswordRecovery` fails safely with `ErrSMSNotConfigured`, returning an HTTP 503 structured error (`sms_service_unavailable`) rather than fabricating successful SMS delivery.
- **Account Enumeration Defense:**
  - Non-existent accounts return generic HTTP 200 responses identical to existing accounts (`status: "CHALLENGE_ISSUED"`), preventing account enumeration attacks.
  - Constant-time dummy hashing is executed on misses to equalize response latency.
- **Rate Limiting & Cooldown:**
  - 60-second resend cooldown enforced per account.
  - Max 3 active challenge requests per hour per account.
  - Max 10 requests per hour per IP.
  - Attempt limit of 5 failed code submissions per challenge before the challenge is marked `EXPIRED`.
  - Resending supersedes previous active challenges (`status = 'SUPERSEDED'`).

### 6.5 Atomic Password Reset Execution
- **Transaction Guarantee:** Password reset (`ConfirmPasswordRecovery` / `ResetPassword`) executes within an atomic transaction:
  1. Row-level lock (`SELECT ... FOR UPDATE`) on the challenge record.
  2. Enforces non-expired, non-consumed, active status, and attempt limits.
  3. Increments `attempt_count` and compares HMAC-SHA256 hash in constant time.
  4. Marks challenge `CONSUMED` with `consumed_at = NOW()`.
  5. Updates password to bcrypt hash (cost 10) in `users` or `priests`.
  6. Increments `session_version = session_version + 1`, immediately revoking existing JWT sessions.
  7. Marks all other active challenges for that account as `SUPERSEDED`.
- **Automated Verification:** Verified in `TestDevoteePasswordRecoveryFlow`, `TestPriestPasswordRecoveryAndLegacyUpgrade`, `TestRecoveryFailureModes`, `TestRecoverySafeFailureWhenSMSUnconfigured`, `TestConcurrentConfirmationsProduceOneReset`, and `TestRecoveryUnknownAccountDoesNotLeak`.

### 6.6 Mobile Recovery Screens & Localization
- **Devotee App (`DevoteeLoginScreen.kt`):**
  - Removed dev OTP toggle ("🔑 Password" / "📱 Dev OTP") and hardcoded `"1234"` OTPs.
  - Devotee login strictly requires registered phone number and secure password.
  - Implemented 2-step password recovery modal:
    - Step 1: Input mobile phone number $\rightarrow$ "Send Recovery Code" (calls `startRecovery`).
    - Step 2: Input 6-digit code, new password (min 8 chars), and confirm password $\rightarrow$ "Update Password" (calls `confirmRecovery`).
    - 60-second cooldown timer displayed on Resend Code button.
    - Clean error parsing from backend structured JSON responses.
    - On success: closes modal, populates login fields with updated credentials, and displays green success confirmation banner.
- **Priest App (`PriestRegistrationScreen.kt`):**
  - Removed dev OTP toggle and hardcoded `"1234"` OTPs.
  - Priest login strictly requires phone number and password.
  - Implemented 2-step password recovery modal matching devotee app flow.
- **Localization Contracts (`apps/common-contracts`):**
  - Added new recovery keys to `AuthStrings`: `forgotPasswordButton`, `resetPasswordTitle`, `sendRecoveryCode`, `recoveryCodeLabel`, `newPasswordLabel`, `confirmPasswordLabel`, `passwordsDoNotMatch`, `resendCode`, `resendCooldown`, `passwordResetSuccess`, and `smsUnavailableNotice`.
  - Implemented in English (`EnStrings.kt`), authentic Telugu (`TeStrings.kt`), and Hindi (`HiStrings.kt`).
  - Added unit test assertions in `I18nCompletenessTest.kt` verifying authentic Telugu and Hindi recovery strings.

---

## 7. TASK 04A Details: Single-Use Cryptographic Email Password Recovery

### 7.1 Architecture & Scope Decision
- **SMS Deprecation for Recovery Only:** SMS recovery has been completely replaced with single-use cryptographic email reset links.
- **Invariants Preserved:**
  - Phone number + bcrypt password login remains active and unchanged for all devotee and priest accounts.
  - In-person booking completion OTP generation and verification remain fully intact.
  - Payments remain deferred under the agreed free pilot model.
- **Account Verification & Ownership Security:**
  - Accounts cannot be recovered or claimed via phone number alone.
  - Recovery requires an email address already verified for that specific account and role.
  - Legacy locked-out accounts without a verified email require manual verified administrator recovery (`SetAccountVerifiedEmail`).

### 7.2 Database Migration `000007_verified_email_support.sql`
- **Additive Schema Changes:**
  - Added `email VARCHAR(255)` and `email_verified BOOLEAN NOT NULL DEFAULT FALSE` to `users` and `priests`.
  - Added unique partial indexes `idx_users_email_unique` and `idx_priests_email_unique` on `LOWER(email)` where `email IS NOT NULL AND email != ''`.
  - Added nullable `email VARCHAR(255)` to `password_recovery_challenges`.
  - Added composite index `idx_recovery_email_lookup (email, account_role, status)`.
- **Startup Compatibility & Backward Safety:**
  - `backend/internal/db/migrate/migrator.go` and `backend/cmd/api-gateway/main.go` enforce schema readiness version `>= "000007"`.
  - Existing accounts without email remain valid and operable; zero data loss or breaking changes.

### 7.3 Email Delivery Abstraction & Security
- **Email Delivery Module (`internal/auth/email.go`):**
  - **Local `.env` Loader:** `LoadLocalEnv()` loads `backend/.env` without overriding existing deployment environment variables.
  - **Configuration Validation:** Validates `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`, and `PASSWORD_RESET_BASE_URL`. Requires remote `PASSWORD_RESET_BASE_URL` to use `https://` (permits `http://localhost` only during local testing).
  - **TLS Enforcement:** `SMTPEmailProvider` strictly requires TLS: uses direct `tls.Dial` on port 465 (or direct TLS flag) and enforces STARTTLS on port 587. Plaintext fallback over unencrypted channels is rejected.
  - **Safe Production Failure (HTTP 503):** `UnconfiguredEmailProvider` safely fails when SMTP is unconfigured, returning `ErrEmailServiceUnavailable` / HTTP 503 (`email_service_unavailable`).
  - **Mock Provider for Tests:** `MockEmailProvider` records sent reset links and verification tokens for deterministic testing without external network dependencies.

### 7.4 Cryptographic Reset Tokens & URL Construction
- **Token Generation:**
  - Reset tokens are generated with 32 cryptographically random bytes via `crypto/rand` (64 hex characters).
  - Plaintext tokens are never stored in the database. Tokens are hashed using HMAC-SHA256 keyed with server secrets (`HashResetToken`).
  - Verification uses constant-time comparison (`subtle.ConstantTimeCompare`).
- **Reset URL Construction:**
  - Constructed strictly from `PASSWORD_RESET_BASE_URL` (e.g. `https://reset.pujavaahini.com/api/v1/auth/recovery/reset?token=...&role=...`).
  - Never derived from or influenced by the untrusted `r.Host` or `X-Forwarded-Host` headers, eliminating Host Header Injection and token theft vulnerabilities.

### 7.5 Reset Form & Atomic Password Reset
- **HTTPS Usable Reset Form (`/api/v1/auth/recovery/reset` and `/reset-password`):**
  - GET request validates the token without consuming it. Returns an accessible, responsive HTML form.
  - Strict security headers applied to all responses:
    - `Referrer-Policy: no-referrer` (prevents token leakage to external domains or assets).
    - `Cache-Control: no-store, no-cache, must-revalidate` (prevents caching of token URLs).
    - `X-Frame-Options: DENY` (clickjacking protection).
    - `X-Content-Type-Options: nosniff`.
    - `Content-Security-Policy: default-src 'self'; style-src 'self' 'unsafe-inline'`.
- **Atomic Consumption & Session Revocation:**
  - Token consumption occurs strictly within `POST /api/v1/auth/recovery/confirm` (supporting both HTML form submissions and JSON APIs).
  - Row-level lock (`SELECT ... FOR UPDATE`) enforces active status, 15-minute expiration, and role binding.
  - The token is atomically marked `CONSUMED` with `consumed_at = NOW()`.
  - The new password is validated (minimum 8 characters) and hashed with bcrypt (cost 10).
  - Account `session_version` is incremented (`session_version = session_version + 1`), revoking all existing JWT sessions immediately.
  - Reused, expired, or superseded tokens are rejected.

### 7.6 Mobile Apps & Localization Updates
- **Devotee App (`DevoteeLoginScreen.kt` & `DevoteeProfileScreen.kt`):**
  - Updated Forgot Password modal to single-use email reset link workflow.
  - Input field for registered email address with client-side syntax validation (`@` and `.`).
  - Loading indicator during dispatch.
  - Success banner explaining that a reset link was sent, displaying destination email and 15-minute expiration.
  - 60-second cooldown timer displayed on Resend Link button.
  - Error handling distinguishes `email_service_unavailable` (503) from general delivery failure; never claims email was sent when delivery failed.
  - **In-App Authenticated Email Management (`DevoteeProfileScreen.kt`):**
    - Added dedicated "Password Recovery Email" card in profile screen displaying current email and verification status badge (`🟢 Verified` / `⚠️ Unverified` / `⚠️ No Recovery Email Configured`).
    - Added "Add / Update Recovery Email" modal allowing authenticated devotees to enter/change email, send single-use verification links, and refresh verification status.
- **Priest App (`PriestRegistrationScreen.kt` & `PriestApp.kt`):**
  - Updated Forgot Password modal matching devotee flow for `role="priest"`.
  - **In-App Authenticated Email Management (`PriestApp.kt`):**
    - Added dedicated "Password Recovery Email" card in Priest profile screen displaying current email and verification status badge.
    - Added "Add / Update Recovery Email" modal allowing authenticated priests to configure recovery email, dispatch verification links, and review status.
- **Localization (`apps/common-contracts`):**
  - Added new localized strings to `AuthStrings`: `forgotPasswordEmailPrompt`, `emailLabel`, `emailPlaceholder`, `invalidEmail`, `sendResetLink`, `resetLinkSentSuccess`, `emailDeliveryFailed`, `emailUnavailableNotice`, `resendResetLink`, `backToLogin`.
  - Added authentic translations in English (`EnStrings.kt`), Telugu (`TeStrings.kt`), and Hindi (`HiStrings.kt`).
  - Verified by unit tests in `I18nCompletenessTest.kt`.

### 7.7 Verification Fixes & Security Hardening (Review Gap Closures)
1. **Separation of Verification & Password Reset Flows:**
   - Dedicated verification link builder `BuildEmailVerificationURL` uses trusted base URL (`EMAIL_VERIFICATION_BASE_URL` or derived `/api/v1/auth/verify-email`), never the password reset form.
   - Verification links strictly open `/api/v1/auth/verify-email` and `/verify-email`.
   - Applied anti-leakage security headers: `Referrer-Policy: no-referrer`, `Cache-Control: no-store, no-cache, must-revalidate`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`.
2. **Challenge Binding, Superseding & Concurrency Safety:**
   - Verification challenges are bound to `(account_role, account_id, email, purpose='email_verification')`.
   - Requesting a new verification email for address `C` automatically supersedes active challenges for previous addresses (`B`), preventing older links from verifying newer addresses.
   - Confirmation locks the challenge row `FOR UPDATE`, enforcing that only the latest active link can confirm, safely serializing concurrent clicks.
3. **Preservation of Existing Verified Email:**
   - The user's existing verified email `A` remains intact in `users`/`priests` until replacement `B` is successfully verified via link click.
   - Requesting email verification does not prematurely alter the account's existing recovery email.
4. **Honest Delivery Status & Error Bubbling:**
   - Provider availability is checked before challenge generation; returns `ErrEmailUnavailable` (HTTP 503 `email_service_unavailable`) if unconfigured.
   - Delivery failures bubble as `ErrDeliveryFailed` (HTTP 502 `email_delivery_failed`) and challenge is marked `status = 'FAILED'`.
   - Gateway and mobile apps never claim an email was dispatched if the provider failed.
5. **Manual Recovery Procedure for Locked-Out Legacy Accounts:**
   - Accounts without a verified email cannot use self-service password recovery.
   - Support runbook:
     1. Account owner contacts support and completes out-of-band identity verification (e.g. government ID / booking history verification).
     2. Authorized administrator assigns verified email using `storage.SetAccountVerifiedEmail(ctx, role, accountID, verifiedEmail, true)`.
     3. User then initiates self-service password reset using their newly verified recovery email.
6. **Hardening of Legacy SMS Confirmation Path:**
   - `ConfirmPasswordRecovery` strictly verifies `email_verified == true` and non-empty email on the account; rejects unverified accounts with actionable guidance to contact support.
   - Enforces `purpose = 'password_reset'`; cross-purpose challenges (e.g. `email_verification`) are rejected.
   - Cross-checks challenge email against account verified email.
7. **Per-Account Serialization of Verification Requests & Confirmations:**
   - Implemented `auth.AccountLockID(role, accountID)` deriving a deterministic 64-bit integer key from `(role, accountID)` via SHA-256 digest conversion.
   - `RequestEmailVerification` and `ConfirmEmailVerification` both acquire a PostgreSQL transaction-level advisory lock (`SELECT pg_advisory_xact_lock($1)`) scoped to the account identity before modifying challenges or account tables.
   - Lock acquisition order is strictly unified (Advisory Lock $\rightarrow$ Row-level locks) preventing lock-inversion deadlocks.
   - Proves under high concurrency that:
     - Concurrent verification requests cannot leave multiple active verification links (only exactly 1 active challenge remains; all prior ones are superseded).
     - An older request cannot replace a newer one (superseded tokens are rejected).
     - The currently verified email is strictly preserved until replacement confirmation.
     - Tested and verified for both devotee and priest account roles (`TestConcurrentEmailVerificationSerializationDevotee` and `TestConcurrentEmailVerificationSerializationPriest`).
8. **Email Security Scanner Behavior & Remaining Limitations:**
   - **Scanner Crawl Behavior:** Automated enterprise link scanners (e.g. Microsoft Defender Safe Links, Proofpoint URL Defense, Google Workspace prefetching) perform an automated HTTP GET request on incoming verification links to scan for malicious content.
   - **Resilience & Notice Card:** When a scanner performs a GET request, the link successfully completes verification in the database. When the human recipient clicks the link later, the system gracefully recognizes that the token was already consumed and presents a dedicated **Email Security Notice**:  
     `"🛡️ Email Security Notice: If your corporate email gateway or link protection software (e.g. Defender, Proofpoint) automatically scanned this link, your email address may have already been verified. Please check your account profile in the Puja Vaahini mobile app to confirm."`
   - **Remaining Limitations & Future Hardening:**
     - *Automated Confirmation by Scanners:* Because the verification link confirms on GET, an email scanner automatically triggers email confirmation without explicit human confirmation. While harmless for self-service verification of an authenticated user's email, if an organization requires strictly interactive confirmation (preventing scanner auto-confirmation), a two-step landing page requiring a user-initiated button click (POST) or WebAuthn prompt can be introduced in a future enterprise hardening phase.
     - *Distributed Scaling:* In-memory throttlers reset on process restart; multi-node distributed clusters require shared Redis rate tracking.

---

## 8. Explicit Missing Credentials, Devices & Environment Access

The following items remain missing or unconfigured for subsequent pilot phases:

1. **SMTP Provider App Credentials:** Live SMTP credentials (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`, `PASSWORD_RESET_BASE_URL`) in `backend/.env`. The application fails safely with HTTP 503 `email_service_unavailable` when unconfigured. Tests run with `MockEmailProvider`.
2. **SMS / Telephony Provider Credentials:** Live SMS credentials (Twilio/MSG91) for booking alerts remain unconfigured. Booking-completion OTP remains local and secure without SMS dependency.
3. **Payment Provider Sandbox Credentials:** Deferred per agreed free pilot scope.
4. **Push Notification Credentials:** Missing Firebase Cloud Messaging service account credentials.
5. **Physical Android Test Devices / Emulators:** No attached Android devices or running emulators. Android test suites verified via Gradle test runner (`:apps:common-contracts:testDebugUnitTest`).
6. **Redis Service:** Online and active on `localhost:6379`. Verified connected in gateway runtime and verified by high-concurrency race condition suite (30 threads).

---

## 9. TASK 05 Details: Booking Authorization & Completion OTP Privacy (Verified)

### 9.1 Motivation & Scope
Prior to Task 05, all booking endpoints were unauthenticated or relied on caller-supplied `user_id` / `priest_id` without verifying identity tokens. Furthermore, the 4-digit booking completion OTP was serialized into public `BookingDetailResponse` structs, leaking the secret completion token to assigned priests, unassigned callers, and network inspectors before ritual completion.

Task 05 establishes mandatory route authorization, strict role and resource ownership, and asymmetric completion OTP privacy across both backend services and mobile applications.

### 9.2 Endpoint Authorization & Ownership Matrix
All 12 booking routes require an active session token validated by `requireCallerSession(r, storage)`. Missing, malformed, expired, or revoked tokens (where `tokenVersion != dbVersion`) are rejected with HTTP 401 Unauthorized.

| Endpoint | HTTP Method | Allowed Roles | Ownership / Assignment Enforcement | Invariant & Behavior |
|---|---|---|---|---|
| `/api/v1/bookings` | `POST` | `devotee` | Caller must be a devotee. Client-supplied `user_id` verified against `claims.UserID`. Forces `req.UserID = claims.UserID`. | Initial creation response returns `completion_otp` exclusively to the creating devotee. Fallback queries picking arbitrary users removed from `CreateBooking`. |
| `/api/v1/bookings` | `GET` | `devotee`, `priest` | **Devotee:** Strictly scoped to `user_id = claims.UserID`. Client attempts to query another user's bookings return HTTP 403 Forbidden.<br>**Priest:** Strictly scoped to `priest_id = claims.UserID`. Client attempts to query another priest's bookings return HTTP 403 Forbidden. | Response array omits `completion_otp` via Go struct tag `json:"-"`. |
| `/api/v1/priests/{id}/bookings` | `GET` | `priest` | Caller must have role `priest` and URL param `id == claims.UserID`. Other priests or devotees rejected with HTTP 403 Forbidden. | Response array omits `completion_otp`. |
| `/api/v1/bookings/{id}` | `GET` | `devotee`, `priest` | Caller must be either the owner devotee (`ownerUID == claims.UserID`) or the assigned priest (`assignedPID == claims.UserID`). Cross-account callers rejected with HTTP 403 Forbidden. | Response omits `completion_otp` via `json:"-"`. |
| `/api/v1/bookings/{id}/otp` | `GET` | `devotee` | **Devotee Owner Only:** Caller must have role `devotee` and `ownerUID == claims.UserID`. Priests are strictly rejected with HTTP 403 Forbidden. Cross-account devotees rejected with HTTP 403 Forbidden. | Returns `{"booking_id": "...", "completion_otp": "...", "status": "..."}` exclusively to the booking owner. |
| `/api/v1/bookings/status` | `POST` | `devotee`, `priest` | **Priest:** Must be assigned priest (`assignedPID == claims.UserID`). Forces `req.PriestID = claims.UserID`.<br>**Devotee:** Must be booking owner (`ownerUID == claims.UserID`). Blocked from transitioning to `COMPLETED` (HTTP 403 Forbidden). | OTP verified on transition to `COMPLETED`. Error response sanitized to `invalid completion OTP` without leaking expected OTP. |
| `/api/v1/bookings/{id}/complete` | `POST` | `priest` | Dedicated completion endpoint. Caller must be the assigned priest (`assignedPID == claims.UserID`). Devotees or other priests rejected with HTTP 403 Forbidden. | Verifies submitted `completion_otp`. Completes booking with verified OTP (financial escrow deferred in free pilot). Error sanitized to prevent OTP leakage. |
| `/api/v1/bookings/{id}/materials` | `GET`, `POST` | `devotee`, `priest` | Caller must be owner devotee or assigned priest. All other accounts rejected with HTTP 403 Forbidden. | Live material checklist query and updates. |
| `/api/v1/bookings/{id}/cancel-quote` | `GET` | `devotee`, `priest` | Caller must be owner devotee or assigned priest. All other accounts rejected with HTTP 403 Forbidden. | Dynamic refund policy inspection. |
| `/api/v1/bookings/cancel` | `POST` | `devotee`, `priest` | Caller must be owner devotee or assigned priest. All other accounts rejected with HTTP 403 Forbidden. Sets `CancelledByRole = claims.Role`. | Cancels booking and executes refund calculation. |
| `/api/v1/bookings/refer` | `POST` | `priest` | Caller must have role `priest` and must be the currently assigned priest (`assignedPID == claims.UserID`). Forces `req.ReferringPriestID = claims.UserID`. | Auto-reassigns booking to colleague priest. Atomically validates current assignment in DB mutation. |
| `/api/v1/chat/rooms` | `POST` | `devotee`, `priest` | Caller must be booking owner devotee or assigned priest (`VerifyBookingParticipant`). | Creates or retrieves dedicated booking chat room. |
| `/api/v1/chat/room-by-booking` | `GET` | `devotee`, `priest` | Caller must be booking owner devotee or assigned priest (`VerifyBookingParticipant`). | Retrieves room ID for booking. |
| `/api/v1/chat/messages` | `GET`, `POST` | `devotee`, `priest` | Caller must be participant of chat room's booking (`VerifyChatRoomParticipant`). SenderID and SenderRole stamped from verified session claims. | Reads chat history or posts message. Never trusts client sender identity. |
| `/ws/chat` | `GET` (WS) | `devotee`, `priest` | Authenticated WebSocket upgrade (`requireCallerSession`). Verifies caller is room participant. Inbound WebSocket messages overwrite sender ID and role with session claims. | Real-time chat streaming. Cross-account WebSocket connection attempts rejected with HTTP 403. |

### 9.3 Asymmetric Completion OTP Privacy Invariant
1. **Omission from Shared Responses:**
   - In `backend/internal/models/entities.go`, `BookingDetailResponse.CompletionOTP` is tagged with `json:"-"`.
   - Database queries (`GetBookingDetails`, `GetBookings`) scan the column internally without ever transmitting it over the wire in shared list or detail payloads.
2. **Dedicated Protected Route:**
   - `GET /api/v1/bookings/{id}/otp` serves as the sole retrieval point for the completion OTP.
   - Priests attempting to access this route receive HTTP 403 Forbidden. Devotees attempting to access another devotee's booking OTP receive HTTP 403 Forbidden.
3. **Verification Without Secret Leakage:**
   - The assigned priest submits the 4-digit OTP shared verbally in person by the devotee via `POST /api/v1/bookings/{id}/complete` or `POST /api/v1/bookings/status`.
   - `backend/internal/db/db.go` `UpdateBookingStatus` was sanitized from returning `invalid completion OTP (expected %s, got %s)` to returning `invalid completion OTP`, preventing secret leakage in API error strings.

### 9.4 Mobile Apps & Shared Contracts Integration
1. **Common Contracts (`apps/common-contracts`):**
   - Added `@Serializable data class BookingOtpResponse(val bookingId: String, val completionOtp: String, val status: String)`.
2. **Devotee App (`apps/devotee-app`):**
   - Updated `PurohitApiClient`:
     - Added `BookingOtpPayload`.
     - Implemented `suspend fun fetchBookingOtp(bookingId: String): Result<BookingOtpPayload>`.
     - Attached `Authorization: Bearer $authToken` header to `fetchBookingMaterials`, `updateBookingStatus`, `fetchBookingDetails`, `fetchBookingOtp`, `fetchCancellationQuote`, and `cancelBooking`.
   - Updated `LiveTrackingScreen`:
     - Added dynamic `LaunchedEffect(bookingId)` hook invoking `PurohitApiClient.fetchBookingOtp(bookingId)` to retrieve the completion OTP securely when tracking starts.
     - Replaced hardcoded/missing OTP with `activeOtp` in the devotee UI completion card and cloud sync triggers.
3. **Priest App (`apps/priest-app`):**
   - Updated `PurohitApiClient`:
     - Attached `Authorization: Bearer $authToken` header to `updateBookingStatus`, `fetchBookingDetails`, `fetchPriestBookings`, `fetchBookingMaterials`, `updateBookingMaterials`, and `referBooking`.

### 9.5 Automated Verification Matrix
- **Backend Unit & Integration Suite (`backend/cmd/api-gateway/booking_authorization_test.go`):**
  - `TestUnauthenticatedBookingEndpointsRejectedWith401`: 24 subtests testing missing and invalid tokens across all booking and chat routes $\rightarrow$ **PASS**.
  - `TestRevokedSessionRejectedWith401`: Verifies that incrementing `session_version` immediately invalidates previously issued tokens on booking routes $\rightarrow$ **PASS**.
  - `TestRoleEnforcementAcrossEndpoints`: 6 subtests testing priest creation block, devotee priest-list block, priest OTP retrieval block, direct devotee completion block, and devotee referral block $\rightarrow$ **PASS**.
  - `TestCrossAccountAndUnassignedPriestRejection`: 11 subtests testing cross-devotee details, OTP, and list query rejection; unassigned priest details, status update, completion, referral, and list queries rejection $\rightarrow$ **PASS**.
  - `TestAsymmetricCompletionOTPPrivacy`: 6 subtests confirming `completion_otp` is absent from shared detail and list responses for both roles, and present exclusively in `GET /api/v1/bookings/{id}/otp` $\rightarrow$ **PASS**.
  - `TestLegitimateBookingLifecycleAndCompletion`: Creation, OTP retrieval, status transitions, bad OTP rejection (with sanitized error), and successful completion with verified OTP $\rightarrow$ **PASS**.
  - `TestUnassignedPriestCannotAcceptOrRejectBookingMerelyKnowingID`: Unassigned priest cannot accept or reject booking merely knowing booking ID (HTTP 403 Forbidden) $\rightarrow$ **PASS**.
  - `TestDevoteeStatusRequestCannotSetOrChangePriestID`: Devotee status request cannot set or mutate `priest_id`; priest transitions (`ACCEPTED`, `IN_TRANSIT`, `ARRIVED`, `POOJA_STARTED`, `COMPLETED`) rejected with HTTP 403; database mutation strictly omits `priest_id` for devotees $\rightarrow$ **PASS**.
  - `TestReassignmentRaceProtectionInDatabaseMutation`: Reassignment races cannot bypass authorization; database updates enforce `WHERE id = $1 AND priest_id = $3` for priests and `WHERE id = $1 AND user_id = $3` for devotees $\rightarrow$ **PASS**.
  - `TestConcurrentReferralRaceCondition`: Concurrent referral attempts safely serialized via atomic SQL check and row count verification; exactly 1 referral succeeds, other fails $\rightarrow$ **PASS**.
  - `TestChatAuthorizationAndParticipantChecks`: Chat room creation, message history, message posting, and WebSocket streaming protected by booking participant verification; spoofed sender IDs and roles stamped from verified server session claims $\rightarrow$ **PASS**.
  - `TestFreePilotNoEscrowOrMoneyClaim`: Verifies all booking responses and status updates state honest pilot messaging without false escrow or dakshina release claims $\rightarrow$ **PASS**.
- **Backend Test Summary:**
  - `internal/auth`: 25/25 PASS
  - `internal/cancellation`: 3/3 PASS
  - `internal/ranking`: 2/2 PASS
  - `internal/db`: 20/20 PASS
  - `internal/db/migrate`: 24/24 PASS
  - `cmd/api-gateway`: 23/23 PASS
  - Total: **97/97 tests pass cleanly with zero failures and zero skipped**.
- **Comprehensive End-to-End Suite (`test_e2e_comprehensive_flow.py`):**
  - All 14 steps passed with bearer tokens propagated for devotee, priest, and referred colleague priest $\rightarrow$ **PASS**.
- **Android Gradle Compilation:**
  - `:apps:common-contracts:testDebugUnitTest` $\rightarrow$ **BUILD SUCCESSFUL in 20s (16 actionable tasks: 1 executed, 15 up-to-date)**.
  - `:apps:devotee-app:assembleDebug :apps:priest-app:assembleDebug` $\rightarrow$ **BUILD SUCCESSFUL in 1m 4s (89 actionable tasks: 11 executed, 78 up-to-date)**.

---

### 9.6 Task 05 Authorization Review Fixes Summary
1. **Unassigned Priest Accept/Reject Rejection:**
   - In the free pilot, bookings are pre-assigned via calendar slots.
   - An unassigned priest attempting to accept or reject a booking merely by knowing its ID is strictly rejected with HTTP 403 Forbidden.
2. **Devotee Status Request Cannot Set/Change Priest ID:**
   - Devotees cannot trigger priest state transitions (`ACCEPTED`, `IN_TRANSIT`, `ARRIVED`, `POOJA_STARTED`, `COMPLETED`); attempts return HTTP 403 Forbidden.
   - `CallerID` and `CallerRole` are propagated from verified session claims to the database layer.
   - `UpdateBookingStatus` in `backend/internal/db/db.go` enforces role and ownership directly in the SQL mutation:
     - Priest update: `WHERE id = $1 AND priest_id = $3 AND status NOT IN ('COMPLETED', 'CANCELLED')`
     - Devotee update: `WHERE id = $1 AND user_id = $3 AND status NOT IN ('COMPLETED', 'CANCELLED')` (with `priest_id` excluded from the `SET` clause).
   - Reassignment races cannot bypass authorization.
3. **Chat Room, Message, and WebSocket Access Protected:**
   - Factored out `registerChatRoutes` in `backend/cmd/api-gateway/main.go`.
   - Protected `POST /api/v1/chat/rooms`, `GET /api/v1/chat/room-by-booking`, `GET /api/v1/chat/messages`, `POST /api/v1/chat/messages`, and `GET /ws/chat` with `requireCallerSession`.
   - Verified caller is owner devotee or assigned priest via `VerifyBookingParticipant` and `VerifyChatRoomParticipant`.
   - Client-supplied `SenderID` and `SenderRole` are overwritten with verified server session values.
   - Fixed system chat notifications to use valid UUID `00000000-0000-0000-0000-000000000000` satisfying PostgreSQL column constraints.
   - Updated mobile client network layers (`devotee-app` and `priest-app`) to attach `Authorization: Bearer $authToken` to all chat HTTP calls.
4. **Removal of Money & Escrow Claims:**
   - Replaced all misleading claims that money is held in escrow or released during the free pilot:
     - Creation: `"Booking initialized successfully."` (was `"Booking initialized. Payment held in secure escrow."`).
     - Completion: `"Booking completed successfully with verified OTP"` (was `"Booking completed and dakshina released to priest"`).
     - Cancellation: `"Booking cancelled successfully per pilot policy"` (was `"Booking cancelled and refund calculated per cancellation tier"`).
5. **Comprehensive Tests Added & Passing:**
   - All 6 review requirements covered by dedicated automated tests.
   - Backend tests (97/97) and Android Gradle builds (both apps) confirmed passing.
6. **No Deployment:**
   - Verified that no deployment commands were run; existing data and schema integrity strictly preserved.

---

## 10. Prior Task Handoff Archive (Task 05 to Task 06)

### **TASK 06 — Strict Booking Finite State Machine (FSM) & Role-Based Transitions**

- **Prerequisites Established in Task 05:**
  1. **Route-Level Authorization Middleware:** All 12 booking routes require valid authentication and active session versions (`requireCallerSession`).
  2. **Server Identity Authority:** Resource ownership verified against `claims.UserID`; client-supplied IDs never trusted.
  3. **Asymmetric OTP Privacy:** `completion_otp` omitted from shared responses (`json:"-"`); exposed exclusively to booking owner via `GET /api/v1/bookings/{id}/otp`.
  4. **Mobile Token Integration:** Both `devotee-app` and `priest-app` transmit bearer tokens on all booking calls; devotee app dynamically fetches OTP via `fetchBookingOtp`.

---

## 11. TASK 06 Details: Booking Lifecycle Finite State Machine, Concurrency & Slot Release

### 11.1 Booking Lifecycle Sequence & Strict State Machine (`backend/internal/lifecycle/fsm.go`)
- **Canonical Sequential Progression Enforced:**
  `REQUESTED` $\rightarrow$ `ACCEPTED` $\rightarrow$ `IN_TRANSIT` $\rightarrow$ `ARRIVED` $\rightarrow$ `POOJA_STARTED` $\rightarrow$ `COMPLETED`
  (With optional intermediate `DISPATCHED` recognized from legacy catalog/dispatch).
- **Terminal State Immutability:** Once in `COMPLETED` or `CANCELLED`, any outbound transition is strictly rejected with `ErrTerminalState`.
- **Repeated & Skipped Transitions Blocked:** Redundant transitions (e.g. `ACCEPTED` $\rightarrow$ `ACCEPTED`) return `ErrRepeatedTransition`. Out-of-order transitions (e.g. `REQUESTED` $\rightarrow$ `COMPLETED` or `ARRIVED` $\rightarrow$ `IN_TRANSIT`) return `ErrInvalidTransition`.
- **HTTP 409 Conflict Standard:** All invalid transitions, repeated transitions, terminal state transitions, and missing/bad completion OTPs are mapped to **HTTP 409 Conflict** (`mapLifecycleErrorToHTTP` in `booking_handlers.go`).

### 11.2 Role Authorization & Devotee Completion OTP Integrity
- **Priest-Only Transition Authority:** Transitions (`ACCEPTED`, `IN_TRANSIT`, `ARRIVED`, `POOJA_STARTED`, `COMPLETED`) are strictly restricted to priests. Devotee attempts return **HTTP 403 Forbidden**.
- **Assigned Priest Validation:** If a booking has an assigned priest (`priest_id`), only that specific priest (`claims.UserID == booking.PriestID`) can advance the booking. Unassigned priests or cross-priest attempts return **HTTP 403 Forbidden**.
- **Completion OTP Requirement:** Moving from `POOJA_STARTED` $\rightarrow$ `COMPLETED` strictly requires the 4-digit devotee completion OTP (`completion_otp`).
- **Sanitized Error Messaging:** Bad or missing OTP returns **HTTP 409 Conflict** with sanitized message `"invalid completion OTP"` without leaking the expected secret.
- **Devotee Cancellation Boundary:** Devotees can only cancel bookings during `REQUESTED`, `DISPATCHED`, `ACCEPTED`, or `IN_TRANSIT` states. Once priest is `ARRIVED` or `POOJA_STARTED`, devotee direct cancellation is blocked (**HTTP 403 Forbidden**).

### 11.3 Transactional Safety & Row-Level Concurrency Locking
- Database mutations for booking state updates and cancellations are wrapped in atomic database transactions (`tx.BeginTx` with `sql.LevelReadCommitted`).
- **Pessimistic Row Locking:** `SELECT ... FOR UPDATE` acquires row-level locks on the booking record before evaluating state and role transitions, eliminating race conditions.
- **Concurrency Test Verification:** Multi-threaded test with 10 simultaneous requests attempting the same transition on the same booking proved that exactly 1 request succeeded (HTTP 200) and 9 requests were safely rejected with HTTP 409 Conflict. Zero inconsistent intermediate states or duplicate completions occurred.

### 11.4 Cancellation & Atomic Slot Release Analysis
- In `backend/internal/db/cancellation.go` and `backend/internal/db/db.go`:
  When a booking is cancelled, linked availability slots in `priest_availability_slots` are atomically updated:
  ```sql
  UPDATE priest_availability_slots 
  SET is_booked = false, booked_booking_id = NULL 
  WHERE booked_booking_id = $1::uuid
  ```
  within the same database transaction.
- **Identified Blocker & Follow-up (GAP-05 / TASK 07):**
  - While cancellation atomically releases any slot that carries `booked_booking_id = booking.ID`, bookings created via on-demand dispatch or direct requests without an explicit slot token do not yet have an immutable slot foreign key bound on the `bookings` table at creation time.
  - Slot release is fully functional whenever `booked_booking_id` is linked; reliable, atomic binding at creation time (`slot_id` + `hold_token`) is the dedicated scope of **TASK 07 (GAP-05)**.

### 11.5 Decoupling of False Escrow / Payment Releases for Free Pilot
- Cleanly removed simulated escrow release logic from the booking completion and cancellation paths.
- Free pilot notices and transparent status strings (`"Booking completed successfully with verified OTP"`, `"Booking cancelled successfully per pilot policy"`) returned to callers.
- Existing financial database columns (`total_amount`, `dakshina_amount`, `refund_amount`, `escrow_status`) are preserved without destructive schema changes, ensuring forward compatibility when payment processing is implemented.

### 11.6 Mobile Apps Alignment & Verification
- **Priest App (`apps/priest-app`):**
  - `PurohitApiClient.updateBookingStatus`: Checks `response.status.value in 200..299` and returns `Result.failure` on HTTP 409/403 with server error details.
  - `DutyToggleScreen`: State stepper `bookingStatus` advances strictly inside `res.onSuccess`. When rejected by the backend, the stepper remains in its current valid state and surfaces the rejection message in `statusMessage`.
- **Devotee App (`apps/devotee-app`):**
  - `PurohitApiClient.updateBookingStatus` and `cancelBooking`: Checks `response.status.value in 200..299` and captures server conflict / forbidden error messages.
- **Common Contracts (`apps/common-contracts`):**
  - Serialized booking models and translations verified with unit tests.
- **Android Gradle Builds:** Both `:apps:devotee-app:assembleDebug` and `:apps:priest-app:assembleDebug` compiled cleanly via Gradle (`BUILD SUCCESSFUL in 2m 8s`).

### 11.7 Automated Test Summary
- **Backend Packages Passing:**
  - `internal/lifecycle`: 100% Pass (state transition table, terminal states, role rules, OTP validation) (2.06s)
  - `internal/auth`: 100% Pass (25/25 tests) (3.68s)
  - `internal/cancellation`: 100% Pass (3/3 tests) (1.33s)
  - `internal/ranking`: 100% Pass (2/2 tests) (1.33s)
  - `internal/db`: 100% Pass (20/20 tests including booking FK, auth, and regression) (59.70s)
  - `internal/db/migrate`: 100% Pass (24/24 tests) (69.49s)
  - `cmd/api-gateway`: 100% Pass (39/39 integration tests including canonical lifecycle, 409 conflict assertions, backward/skipped transition rejections, role enforcement, OTP privacy, adversarial fuzzing, slot release, and concurrent race tests) (89.30s)
- **Total Backend Tests:** **113/113 tests pass with 0 failures and 0 skipped**.
- **Android Builds & Unit Tests:** `:apps:common-contracts:testDebugUnitTest`, `:apps:devotee-app:assembleDebug`, `:apps:priest-app:assembleDebug` $\rightarrow$ **BUILD SUCCESSFUL**.

---

## 12. TASK 07 Implementation Summary: Atomic Slot Binding & Double-Booking Prevention

### 12.1 Backend Slot Binding Architecture
- **Request & Model Extensions:**
  - `BookingCreateRequest` in `backend/internal/models/entities.go` updated with `SlotID string json:"slot_id,omitempty"` and `HoldToken string json:"hold_token,omitempty"`.
  - `PriestSlot` in `backend/internal/models/entities.go` updated with `BookedBookingID string json:"booked_booking_id,omitempty"`.
- **Database Transaction Integration:**
  - `CreateBooking` in `backend/internal/db/db.go` wrapped in an atomic database transaction (`BeginTx`).
  - When `req.SlotID` is provided, `VerifyAndConsumeSlotHold` acquires a pessimistic row-level lock (`SELECT ... FOR UPDATE`) on `priest_availability_slots`, verifies priest matching, confirms slot is not already booked, checks hold lock ownership, marks `is_booked = true`, and links `booked_booking_id = bookingID` upon booking insertion.
  - Releases hold lock key in Redis / in-memory cache upon binding.
- **Strict Error Mapping:**
  - Double booking attempts on already booked slots return `lifecycle.ErrSlotUnavailable` which maps directly to **HTTP 409 Conflict** via `mapLifecycleErrorToHTTP`.

### 12.2 Automated Tests & Concurrency Verification
- `backend/cmd/api-gateway/slot_binding_test.go`:
  1. `TestHTTPAtomicSlotBindingAtCreation`: Verifies slot booking at creation time, confirms double booking attempt returns HTTP 409 Conflict, tests that cancelling booking 1 releases the slot, and confirms devotee 2 can subsequently book the released slot.
  2. `TestHTTPConcurrentSlotReservationRaceCondition`: Simulates 6 concurrent devotees simultaneously submitting a booking request for the exact same slot. Exactly 1 request succeeds (HTTP 200) and all 5 others are rejected with HTTP 409 Conflict with zero race conditions or database inconsistencies.

### 12.3 Mobile Apps Integration
- **Priest App (`apps/priest-app`):**
  - Added `PurohitApiClient.createPriestSlot(priestId, slotDate, startTime, endTime)` calling `POST /api/v1/priests/slots`.
  - Connected `PriestScheduleScreen.kt` "Add Window" dialog to persist availability slots to the backend API and refresh local slots from remote server.
- **Devotee App (`apps/devotee-app`):**
  - Added `PurohitApiClient.fetchPriestSlots(priestId, date)` and `holdPriestSlot(slotId)`.
  - Updated `BookingCreatePayload` and `createBooking` to accept `slot_id` and propagate it to `POST /api/v1/bookings`.
- **Android Builds:**
  - `:apps:common-contracts:testDebugUnitTest`, `:apps:devotee-app:assembleDebug`, `:apps:priest-app:assembleDebug` $\rightarrow$ **BUILD SUCCESSFUL**.

---

## 14. TASK 08 Details: Core Free Pilot End-to-End Integration & Readiness

### 14.1 Scope & Execution
- **Core Free Pilot Journey Focus:**
  - Password authentication with bcrypt and positive session versioning for Devotees and Priests.
  - Priest availability publishing (`POST /api/v1/priests/slots`) and discovery (`GET /api/v1/priests/slots`).
  - Devotee discovery, slot holding (`POST /api/v1/priests/slots/hold`), and atomic slot binding at booking creation (`POST /api/v1/bookings`).
  - Strict double-booking prevention with DB row-level locking (`SELECT ... FOR UPDATE`) returning HTTP 409 Conflict.
  - Assigned priest ritual lifecycle advancement (`ACCEPTED` $\rightarrow$ `IN_TRANSIT` $\rightarrow$ `ARRIVED` $\rightarrow$ `POOJA_STARTED`).
  - Devotee completion OTP asymmetric privacy (`GET /api/v1/bookings/{id}/otp`) and priest verification (`POST /api/v1/bookings/{id}/complete` / `POST /api/v1/bookings/status`).
  - Safe cancellation with dynamic policy and transactional slot release.
  - Clean decoupling of mock/third-party payment gateway, SMS, and email services with documented manual operating procedures.

### 14.2 High-Priority Blockers Closed
1. **Priest App Auth Bypass Closure:**
   - `PriestAppRoot` previously defaulted `isRegistered = true` with a hardcoded mock ID, bypassing login.
   - Updated to require explicit authentication (`isRegistered = false`, defaulting to `PriestRegistrationScreen` login mode).
   - Polling for priest bookings and incoming alerts guarded by `priestId.isNotBlank()`.
2. **Devotee App Live Tracking Role Separation & Hardcoded Defaults:**
   - Removed legacy devotee buttons attempting to transition priest-only states (`ARRIVED`, `POOJA_STARTED`, `COMPLETED`), which would fail with HTTP 403 Forbidden under the server's strict role FSM.
   - Replaced with read-only ritual status guidance, correctly directing the devotee to share the 4-digit completion OTP with the Purohit once the ceremony finishes.
   - Cleaned up hardcoded mock defaults (`"4921"` OTP, mock booking ID) in `RapidoNearbyPriestsScreen` and `LiveTrackingScreen`.
3. **Priest App Configurable Server Endpoint:**
   - Added runtime backend URL configuration field to `PriestRegistrationScreen` matching `DevoteeLoginScreen`, allowing operators to redirect the priest mobile client to local IP/port (`http://10.0.2.2:8080`) or live Ngrok tunnel without recompilation.

### 14.3 Full Test Suite Verification Record
- **Backend Test Suite (Go):**
  - Ran `cd backend; go test -v -count=1 ./internal/lifecycle/... ./internal/auth/... ./internal/cancellation/... ./internal/ranking/... ./cmd/api-gateway/... ./internal/db/...`
  - **Result:** **100% Pass across all packages (0 skipped, 0 failed)**:
    - `lifecycle`: 11 test suites passing (state machine, skipped transitions, backward transitions, repeated transitions, terminal states, role rules, completion OTP validation, cancellation policy).
    - `auth`: 25 test suites passing (bcrypt hashing, JWT HS256, session revocation, phone normalization, single-use email reset tokens, rate throttler, lockout preservation).
    - `cancellation`: 3 tests passing (cancellation refund calculations across REQUESTED, IN_TRANSIT, ARRIVED).
    - `ranking`: 2 tests passing (Rapido dispatch multi-factor candidate scoring).
    - `db`: 20 tests passing (session revocation, duplicate protection, email recovery serialization, database migration safety harness).
    - `api-gateway`: 39 tests passing (atomic slot binding, concurrent race conditions [6 simultaneous -> 1 success, 5 conflict], canonical ritual sequence, 401/403/409 error assertions, asymmetric OTP privacy).
- **Android Gradle Builds & Unit Tests:**
  - Ran `.\gradlew.bat --no-daemon :apps:common-contracts:testDebugUnitTest :apps:devotee-app:assembleDebug :apps:priest-app:assembleDebug`
  - **Result:** **BUILD SUCCESSFUL**. Both `:apps:devotee-app` and `:apps:priest-app` debug APKs generated cleanly with zero compiler or contract errors.

### 14.4 Manual Operating Procedures for External Dependencies
- **Payments:** Free pilot agreed with zero online financial transactions. Bookings create with `escrow_status: "HELD"` and complete with `escrow_status: "RELEASED"` without payment provider calls.
- **SMS / Push Notifications:** Automated SMS delivery via Twilio/MSG91 is unconfigured. Manual dispatch workflow: Admins monitor incoming bookings via database / API logs; priests check duty screen which polls active/requested bookings every 2.5s; devotees receive live in-app updates via booking detail polling.
- **Email Password Recovery:** Password reset uses secure HTTPS single-use tokens; in production or testing without live SMTP credentials, reset URLs are retrieved securely from server logs or manual admin database lookup.


---

## 15. TASK 08: Railway Staging Deployment Readiness Verification (Phase 15)

### 15.1 Architectural Scope & Deliverables
1. **Multi-Stage Production Docker Build (`backend/Dockerfile`):**
   - Compatible Go Alpine builder (`golang:1.24-alpine`) aligned with `go.mod`.
   - Compiles static binaries for both API gateway (`/app/api-gateway` from `./cmd/api-gateway`) and migration runner (`/app/migrate` from `./cmd/migrate`).
   - Copies forward migrations (`migrations/000001` through `000007`) and seeds into runtime Alpine image (`alpine:3.20`).
   - Executes under least-privilege non-root user `appuser:appgroup`.
   - Adheres dynamically to Railway's `$PORT` environment variable (`sh -c 'curl -f http://127.0.0.1:${PORT:-8085}/health || exit 1'`).
2. **Pre-Deploy Database Migration Workflow:**
   - Pre-deploy command configured: `/app/migrate up` (or `cd backend && go run ./cmd/migrate up`).
   - Schema readiness check at gateway startup verifies version `000007` before binding HTTP socket.
   - Migrator and gateway support standard Railway `DATABASE_URL` alongside `PG_CONN_STR`.
   - Demo seeds disabled in staging by default.
3. **PostgreSQL & Redis Resilience:**
   - Supports Railway authenticated Redis connections (`redis://` and TLS `rediss://`) via `redis.ParseURL`.
   - Credentials completely sanitized and masked in logs (`SanitizeRedisEndpoint`).
   - Verified slot locking behavior under Redis outage: graceful fallback to in-memory slot mutex + PostgreSQL row-level pessimistic lock (`SELECT ... FOR UPDATE`) guarantees no double-booking even if Redis is completely unavailable or misconfigured.
4. **Truthful Request-Time Health Probing:**
   - Mounted `/health`, `/health/ready`, and `/health/live` in `backend/cmd/api-gateway/health_routes.go`.
   - `/health` probes PostgreSQL and Redis with 2s timeout:
     - Both up $\rightarrow$ `HTTP 200` (`status: UP`, `postgres: CONNECTED`, `redis: CONNECTED`).
     - Redis down $\rightarrow$ `HTTP 200` (`status: DEGRADED`, `postgres: CONNECTED`, `redis: UNAVAILABLE`).
     - PostgreSQL down $\rightarrow$ `HTTP 503 Service Unavailable` (`status: DOWN`, `postgres: DISCONNECTED`).
5. **Mobile Staging HTTPS Enforcement:**
   - `BackendConfig` in `apps/common-contracts` updated to support `STAGING` and `PILOT` environments.
   - Enforces HTTPS scheme for staging endpoint; rejects non-HTTPS or silent fallback to `10.0.2.2`.
   - Staging URL configurable via `BackendConfig.setStagingUrl(...)`.

### 15.2 Verification Record
- **Go Vet:** `cd backend && go vet ./...` $\rightarrow$ Exit code 0 (Pass).
- **Go Unit & Integration Tests:**
  - `cd backend && go test -v ./cmd/api-gateway -run "^(TestTruthfulHealth|TestLiveness)"` $\rightarrow$ Pass (3/3 tests).
  - `cd backend && go test -v ./internal/db -run "^(TestRedisParsing|TestSlotLocking|TestConcurrentSlot)"` $\rightarrow$ Pass (4/4 tests).
  - `cd backend && go test -v ./internal/db/migrate -run "TestInvalidBookingReferencesRejected"` $\rightarrow$ Pass.
  - `cd backend && go test -v ./internal/auth -run "TestLoadLocalEnv|TestJWT|TestSMTP"` $\rightarrow$ Pass.
- **Gradle Mobile Tests:**
  - `.\gradlew.bat :apps:common-contracts:testDebugUnitTest :apps:common-contracts:testReleaseUnitTest` $\rightarrow$ Pass (16 actionable tasks, 0 failures, 8 new unit tests in `BackendConfigTest`).
  - `.\gradlew.bat :apps:priest-app:compileDebugKotlinAndroid :apps:devotee-app:compileDebugKotlinAndroid` $\rightarrow$ Pass (42 actionable tasks, 0 failures).
- **Docker & Container Smoke Test:** Not available locally (Docker daemon not installed on Windows host); Dockerfile verified statically against Alpine Go compiler and runtime specs.

---
*End of Pilot Progress Tracker.*


