# Pushkaralu review fixes — 2026-11-16

Fixed the three reviewed issues:

- Event acceptance no longer allocates legacy fixed availability slots. Event conflicts use scheduled_at and the frozen ritual duration; accepts serialize per priest. Completed rituals no longer prevent subsequent requests. Ordinary booking slot behavior remains covered by its existing regression.
- Added authenticated GET /api/v1/priests/events/config, preferring the live event then the earliest scheduled event. The priest app uses it and shows the event tab for scheduled events. Public devotee config stays live-only.
- Event audit history includes ritual and participation changes and excludes records belonging to other events.

Regression tests first reproduced the missing endpoint, legacy slot allocation, rejected subsequent/scheduled bookings and missing audit entries. Final tests cover these plus overlapping/non-overlapping scheduled requests, concurrent accepts, authentication/roles, live-event priority and audit isolation.

Validation:

- Backend go test ./... passed with TEST_PG_ROOT_CONN_STR set to an isolated local PostgreSQL 18 instance; 47 older tests initially skipped because they require PG_TEST_CONN_STR too.
- Re-ran internal/db and internal/db/migrate with both test connection variables configured. Both packages passed with no skipped tests. Combined latest results: Counter({'pass': 429}).
- Fixed test-only timestamp collisions in mock SQL driver names and outdated migration-count/latest-version expectations discovered during those reruns.
- Focused gateway event and ordinary-slot regression tests passed again after adding concurrency and event-isolation coverage.
- go vet ./... passed.
- CMS npm run build passed, including type checks.
- Gradle shared contract unit tests and devotee/priest Android Kotlin compilation passed; changed code was recompiled.
- git diff --check passed.

No deployment or real notifications. Tests used temporary databases; the dedicated PostgreSQL instance was stopped afterward. Live emulator/browser interactions and iOS builds were not run. PostGIS is not installed in the local test instance; migrations use their existing fallback.
