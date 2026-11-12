package migrate

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func getRootTestConnStr() string {
	connStr := os.Getenv("PG_TEST_CONN_STR")
	if connStr != "" {
		return connStr
	}
	connStr = os.Getenv("PG_CONN_STR")
	if connStr != "" {
		return connStr
	}
	return ""
}

func getMigrationsDir(t *testing.T) string {
	candidates := []string{
		"../../../migrations",
		"../../migrations",
		"../migrations",
		"migrations",
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err == nil {
			if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
				return abs
			}
		}
	}
	t.Fatalf("could not locate migrations directory from candidates: %v", candidates)
	return ""
}

func getSeedsDir(t *testing.T) string {
	candidates := []string{
		"../../../seeds",
		"../../seeds",
		"../seeds",
		"seeds",
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err == nil {
			if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
				return abs
			}
		}
	}
	t.Fatalf("could not locate seeds directory from candidates: %v", candidates)
	return ""
}

// createIsolatedTestDB creates an ephemeral database and registers a cleanup handler to drop it
func createIsolatedTestDB(t *testing.T) (*sql.DB, string) {
	rootConnStr := getRootTestConnStr()
	if rootConnStr == "" {
		t.Skip("Explicit PostgreSQL test connection is required")
	}
	rootDB, err := sql.Open("postgres", rootConnStr)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to root postgres (%v)", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rootDB.PingContext(ctx); err != nil {
		rootDB.Close()
		t.Skipf("Skipping test: postgres server unreachable (%v)", err)
	}

	randBytes := make([]byte, 4)
	_, _ = rand.Read(randBytes)
	testDBName := fmt.Sprintf("pv_test_%d_%s", time.Now().Unix(), hex.EncodeToString(randBytes))

	_, err = rootDB.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, testDBName))
	if err != nil {
		rootDB.Close()
		t.Fatalf("failed to create isolated test database %s: %v", testDBName, err)
	}

	testURL, err := url.Parse(rootConnStr)
	if err != nil || testURL.Host == "" {
		t.Fatal("test connection must be a PostgreSQL URL")
	}
	testURL.Path = "/" + testDBName
	testConnStr := testURL.String()
	testDB, err := sql.Open("postgres", testConnStr)
	if err != nil {
		rootDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, testDBName))
		rootDB.Close()
		t.Fatalf("failed to connect to test db: %v", err)
	}

	t.Cleanup(func() {
		testDB.Close()
		// Disconnect all sessions before dropping
		_, _ = rootDB.Exec(fmt.Sprintf(`
			SELECT pg_terminate_backend(pid) 
			FROM pg_stat_activity 
			WHERE datname = '%s' AND pid <> pg_backend_pid()
		`, testDBName))
		_, _ = rootDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, testDBName))
		rootDB.Close()
	})

	return testDB, testDBName
}

// 1. Acceptance Check: Empty test database -> complete migration chain succeeds
func TestEmptyDatabaseCompleteChain(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)

	migrator := New(db, migDir, seedsDir)
	ctx := context.Background()

	applied, err := migrator.Up(ctx, false)
	if err != nil {
		t.Fatalf("Up failed on empty database: %v", err)
	}
	if applied < 4 {
		t.Errorf("expected at least 4 migrations applied, got %d", applied)
	}

	report, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Status check failed: %v", err)
	}
	if report.PendingCount != 0 {
		t.Errorf("expected 0 pending migrations, got %d", report.PendingCount)
	}
	if report.AppliedCount != applied {
		t.Errorf("expected %d applied, got %d", applied, report.AppliedCount)
	}
}

// 2. Acceptance Check: Fresh pilot database -> no demo people
func TestFreshPilotDatabaseNoDemoPeople(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)

	migrator := New(db, migDir, seedsDir)
	ctx := context.Background()

	// Default pilot migration run (seedDemo = false)
	_, err := migrator.Up(ctx, false)
	if err != nil {
		t.Fatalf("Up failed: %v", err)
	}

	var userCount, priestCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount); err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM priests").Scan(&priestCount); err != nil {
		t.Fatalf("failed to count priests: %v", err)
	}

	if userCount != 0 {
		t.Errorf("fresh pilot database has %d users, expected 0 demo users", userCount)
	}
	if priestCount != 0 {
		t.Errorf("fresh pilot database has %d priests, expected 0 demo priests", priestCount)
	}

	// Verify standard pooja catalog is present
	var catalogCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pooja_catalog").Scan(&catalogCount); err != nil {
		t.Fatalf("failed to count catalog: %v", err)
	}
	if catalogCount == 0 {
		t.Errorf("expected standard pooja catalog to be seeded, got 0")
	}
}

// 3. Acceptance Check: Repeated run -> no duplicate changes or seed records
func TestRepeatedRunIdempotent(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)

	migrator := New(db, migDir, seedsDir)
	ctx := context.Background()

	applied1, err := migrator.Up(ctx, false)
	if err != nil {
		t.Fatalf("first Up failed: %v", err)
	}
	if applied1 == 0 {
		t.Fatalf("expected >0 migrations applied on first run")
	}

	applied2, err := migrator.Up(ctx, false)
	if err != nil {
		t.Fatalf("second Up failed: %v", err)
	}
	if applied2 != 0 {
		t.Errorf("expected 0 migrations applied on repeated run, got %d", applied2)
	}
}

// 4. Acceptance Check: Existing-schema fixture -> upgrade preserves representative records and IDs
func TestExistingSchemaFixtureUpgradePreservesData(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)
	ctx := context.Background()

	// Simulate existing installation by manually executing migrations 000001, 000002, 000003
	files, err := os.ReadDir(migDir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}
	for _, f := range files {
		if f.Name() < "000004" && strings.HasSuffix(f.Name(), ".sql") {
			content, _ := os.ReadFile(filepath.Join(migDir, f.Name()))
			_, err := db.ExecContext(ctx, string(content))
			if err != nil {
				t.Fatalf("failed to apply fixture %s: %v", f.Name(), err)
			}
		}
	}

	// Insert representative existing customer and priest records
	existingUserID := "11111111-1111-1111-1111-111111111111"
	existingPriestID := "22222222-2222-2222-2222-222222222222"
	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, phone, full_name, gotram, nakshatram, rashi, city, consent_agreed)
		VALUES ($1, '+919999988888', 'Existing Devotee Ananya', 'Kashyapa', 'Rohini', 'Vrishabha', 'Hyderabad', true);
	`, existingUserID)
	if err != nil {
		t.Fatalf("failed to insert existing user fixture: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO priests (id, phone, full_name, sampradaya, veda_shakha, is_online, consent_agreed)
		VALUES ($1, '+919999977777', 'Existing Priest Narayana', 'Smartha', 'Rigveda', true, true);
	`, existingPriestID)
	if err != nil {
		t.Fatalf("failed to insert existing priest fixture: %v", err)
	}

	migrator := New(db, migDir, seedsDir)

	// Baseline up to version 000003
	baselined, err := migrator.Baseline(ctx, "000003")
	if err != nil {
		t.Fatalf("Baseline failed: %v", err)
	}
	if baselined != 3 {
		t.Errorf("expected 3 migrations baselined, got %d", baselined)
	}

	// Run forward migrations (should apply all migrations after 000003)
	applied, err := migrator.Up(ctx, false)
	if err != nil {
		t.Fatalf("Up after baseline failed: %v", err)
	}
	if applied < 1 {
		t.Errorf("expected at least 1 migration applied after baseline, got %d", applied)
	}

	// Verify representative records still exist with intact IDs
	var userName, userPhone, userPwdHash string
	err = db.QueryRowContext(ctx, "SELECT full_name, phone, password_hash FROM users WHERE id = $1", existingUserID).Scan(&userName, &userPhone, &userPwdHash)
	if err != nil {
		t.Fatalf("existing user record was corrupted or deleted: %v", err)
	}
	if userName != "Existing Devotee Ananya" || userPhone != "+919999988888" {
		t.Errorf("user data altered: got %s, %s", userName, userPhone)
	}

	var priestName, priestPhone, priestPwdHash string
	err = db.QueryRowContext(ctx, "SELECT full_name, phone, password_hash FROM priests WHERE id = $1", existingPriestID).Scan(&priestName, &priestPhone, &priestPwdHash)
	if err != nil {
		t.Fatalf("existing priest record was corrupted or deleted: %v", err)
	}
	if priestName != "Existing Priest Narayana" || priestPhone != "+919999977777" {
		t.Errorf("priest data altered: got %s, %s", priestName, priestPhone)
	}
}

// 5. Acceptance Check: Two concurrent runners -> serialized execution via PostgreSQL advisory lock
func TestConcurrentRunnersSerialized(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)

	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	appliedCh := make(chan int, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m := New(db, migDir, seedsDir)
			applied, err := m.Up(context.Background(), false)
			if err != nil {
				errCh <- err
			} else {
				appliedCh <- applied
			}
		}()
	}

	wg.Wait()
	close(errCh)
	close(appliedCh)

	for err := range errCh {
		t.Fatalf("concurrent migration run failed: %v", err)
	}

	totalApplied := 0
	for count := range appliedCh {
		totalApplied += count
	}

	if totalApplied < 4 {
		t.Errorf("expected at least 4 migrations applied in total across concurrent runners, got %d", totalApplied)
	}
}

// 6. Acceptance Check: Injected migration failure -> atomic rollback, no partial schema or history entry
func TestInjectedMigrationFailureRollsBack(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)

	// Create temporary directory with standard migrations plus an invalid one
	tempDir := t.TempDir()
	files, _ := os.ReadDir(migDir)
	for _, f := range files {
		data, _ := os.ReadFile(filepath.Join(migDir, f.Name()))
		_ = os.WriteFile(filepath.Join(tempDir, f.Name()), data, 0644)
	}

	// Add failing migration with version after existing files
	failingVersion := "000008"
	failingSQL := `
		CREATE TABLE test_should_not_exist (id INT);
		INVALID SQL SYNTAX THAT CAUSES IMMEDIATE FAILURE;
	`
	_ = os.WriteFile(filepath.Join(tempDir, failingVersion+"_failing_migration.sql"), []byte(failingSQL), 0644)

	migrator := New(db, tempDir, seedsDir)
	ctx := context.Background()

	_, err := migrator.Up(ctx, false)
	if err == nil {
		t.Fatalf("expected migration Up to fail on syntax error, but succeeded")
	}

	// Verify failed table does not exist
	var tableExists bool
	_ = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = 'test_should_not_exist'
		)
	`).Scan(&tableExists)
	if tableExists {
		t.Errorf("test_should_not_exist table was created, transaction was not rolled back")
	}

	// Verify failing version was not recorded in schema_migrations
	var recExists bool
	_ = db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)", failingVersion).Scan(&recExists)
	if recExists {
		t.Errorf("failed migration %s was recorded in schema_migrations", failingVersion)
	}
}

// 7. Acceptance Check: Modified applied migration -> checksum mismatch is detected
func TestModifiedAppliedMigrationChecksumMismatch(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)

	tempDir := t.TempDir()
	files, _ := os.ReadDir(migDir)
	for _, f := range files {
		data, _ := os.ReadFile(filepath.Join(migDir, f.Name()))
		_ = os.WriteFile(filepath.Join(tempDir, f.Name()), data, 0644)
	}

	migrator := New(db, tempDir, seedsDir)
	ctx := context.Background()

	// Initial apply
	_, err := migrator.Up(ctx, false)
	if err != nil {
		t.Fatalf("initial Up failed: %v", err)
	}

	// Modify an applied migration on disk
	mig1Path := filepath.Join(tempDir, "000001_init_schema.sql")
	data, _ := os.ReadFile(mig1Path)
	data = append(data, []byte("\n-- MALICIOUS OR ACCIDENTAL EDIT\n")...)
	_ = os.WriteFile(mig1Path, data, 0644)

	// Second run should fail with checksum mismatch
	_, err = migrator.Up(ctx, false)
	if err == nil {
		t.Fatalf("expected checksum mismatch error, but Up succeeded")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("expected error to contain 'checksum mismatch', got: %v", err)
	}
}

// 8. Acceptance Check: Auth, catalog, and booking queries execute cleanly against resulting schema
func TestRuntimeQueriesAgainstMigratedSchema(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)

	migrator := New(db, migDir, seedsDir)
	ctx := context.Background()

	_, err := migrator.Up(ctx, true) // Apply migrations
	if err != nil {
		t.Fatalf("Up failed: %v", err)
	}

	// Apply demo seeds so users and priests exist for query verification
	if err := migrator.Seed(ctx, ""); err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	// 1. Auth Login Query
	var uID, uPhone, uName, uHash string
	err = db.QueryRowContext(ctx, `
		SELECT id, phone, full_name, COALESCE(password_hash, '') 
		FROM users 
		ORDER BY created_at DESC LIMIT 1
	`).Scan(&uID, &uPhone, &uName, &uHash)
	if err != nil {
		t.Errorf("Auth Login user query failed: %v", err)
	}

	// 2. Catalog Query with i18n columns
	rows, err := db.QueryContext(ctx, `
		SELECT id, title, 
		       COALESCE(title_en, title) as t_en, 
		       COALESCE(title_te, title) as t_te, 
		       COALESCE(title_hi, title) as t_hi, 
		       category, base_price, samagri_kit_price, duration_minutes, is_online_eligible, icon, 
		       description, 
		       COALESCE(description_en, description) as d_en, 
		       COALESCE(description_te, description) as d_te, 
		       COALESCE(description_hi, description) as d_hi
		FROM pooja_catalog
		ORDER BY base_price ASC
	`)
	if err != nil {
		t.Errorf("Catalog query failed: %v", err)
	} else {
		rows.Close()
	}

	// 3. Booking creation query with all detail fields
	var bookingID string
	err = db.QueryRowContext(ctx, `
		INSERT INTO bookings (
			user_id, priest_id, pooja_id, booking_type, status, escrow_status, 
			total_amount, include_samagri_kit, completion_otp, event_address, latitude, longitude,
			scheduled_date, slot_time, devotee_gotram, devotee_nakshatram, devotee_rashi, sankalpam_purpose
		)
		VALUES ($1, NULL, NULL, 'CALENDAR_SLOT', 'REQUESTED', 'HELD_IN_ESCROW', 3500.0, true, '1234', 'Plot 42, Hyderabad', 17.3850, 78.4867, 'Today', '09:00 AM', 'Kashyapa', 'Rohini', 'Vrishabha', 'Family Prosperity')
		RETURNING id
	`, uID).Scan(&bookingID)
	if err != nil {
		t.Errorf("Booking create query failed: %v", err)
	}

	// 4. Booking details retrieval query
	var bDate, bTime, bGotram, bPurpose string
	err = db.QueryRowContext(ctx, `
		SELECT scheduled_date, slot_time, devotee_gotram, sankalpam_purpose 
		FROM bookings 
		WHERE id = $1
	`, bookingID).Scan(&bDate, &bTime, &bGotram, &bPurpose)
	if err != nil {
		t.Errorf("Booking details query failed: %v", err)
	}
	if bDate != "Today" || bGotram != "Kashyapa" {
		t.Errorf("unexpected booking detail values: date=%s, gotram=%s", bDate, bGotram)
	}

	// 5. Schema readiness check
	if err := migrator.VerifyReadiness(ctx, "000004"); err != nil {
		t.Errorf("VerifyReadiness failed: %v", err)
	}
}

// 9. Regression Test: Invalid booking references remain rejected (preserves referential integrity)
func TestInvalidBookingReferencesRejected(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)

	migrator := New(db, migDir, seedsDir)
	ctx := context.Background()

	// Migrate database to current schema (version 000005)
	if _, err := migrator.Up(ctx, false); err != nil {
		t.Fatalf("Up failed: %v", err)
	}

	// Insert valid test devotee
	var userID string
	err := db.QueryRowContext(ctx, `
		INSERT INTO users (phone, full_name, consent_agreed)
		VALUES ('+919876511111', 'Test Devotee FK', true)
		RETURNING id
	`).Scan(&userID)
	if err != nil {
		t.Fatalf("failed inserting user: %v", err)
	}

	// Attempt to create booking with non-existent pooja_id
	invalidPoojaID := "00000000-0000-0000-0000-000000000999"
	_, err = db.ExecContext(ctx, `
		INSERT INTO bookings (
			user_id, pooja_id, total_amount, event_address
		)
		VALUES ($1, $2, 2500.0, 'Banjara Hills, Hyderabad')
	`, userID, invalidPoojaID)

	if err == nil {
		t.Fatalf("expected foreign key violation when inserting booking with invalid pooja_id, but insert succeeded")
	}
	if !strings.Contains(err.Error(), "foreign key") && !strings.Contains(err.Error(), "bookings_pooja_id_fkey") {
		t.Errorf("expected foreign key constraint violation error, got: %v", err)
	}

	// Verify that inserting with a valid pooja_id succeeds
	var validPoojaID string
	err = db.QueryRowContext(ctx, "SELECT id FROM pooja_catalog LIMIT 1").Scan(&validPoojaID)
	if err != nil {
		t.Fatalf("failed querying pooja catalog: %v", err)
	}

	var validBookingID string
	err = db.QueryRowContext(ctx, `
		INSERT INTO bookings (
			user_id, pooja_id, total_amount, event_address
		)
		VALUES ($1, $2, 2500.0, 'Banjara Hills, Hyderabad')
		RETURNING id
	`, userID, validPoojaID).Scan(&validBookingID)
	if err != nil {
		t.Fatalf("inserting booking with valid pooja_id failed: %v", err)
	}
	if validBookingID == "" {
		t.Errorf("expected valid booking ID to be returned")
	}
}

// 10. Regression Test: Unknown personal and schedule fields remain unknown (no fabricated defaults)
func TestUnknownPersonalAndScheduleFieldsRemainUnknown(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)

	migrator := New(db, migDir, seedsDir)
	ctx := context.Background()

	if _, err := migrator.Up(ctx, false); err != nil {
		t.Fatalf("Up failed: %v", err)
	}

	// Devotee with no gotram, nakshatram, or rashi
	var userID string
	err := db.QueryRowContext(ctx, `
		INSERT INTO users (phone, full_name, consent_agreed)
		VALUES ('+919876522222', 'Devotee Unknown Details', true)
		RETURNING id
	`).Scan(&userID)
	if err != nil {
		t.Fatalf("failed inserting user: %v", err)
	}

	var poojaID string
	err = db.QueryRowContext(ctx, "SELECT id FROM pooja_catalog LIMIT 1").Scan(&poojaID)
	if err != nil {
		t.Fatalf("failed querying pooja catalog: %v", err)
	}

	// Create booking with NULL schedule and personal fields
	var bookingID string
	err = db.QueryRowContext(ctx, `
		INSERT INTO bookings (
			user_id, pooja_id, total_amount, event_address,
			scheduled_date, slot_time, devotee_gotram, devotee_nakshatram, devotee_rashi, sankalpam_purpose
		)
		VALUES ($1, $2, 2500.0, 'Madhapur, Hyderabad', NULL, NULL, NULL, NULL, NULL, NULL)
		RETURNING id
	`, userID, poojaID).Scan(&bookingID)
	if err != nil {
		t.Fatalf("failed inserting booking with NULL fields: %v", err)
	}

	// Inspect raw column values in database
	var schedDate, slotTime, gotram, nakshatram, rashi, purpose sql.NullString
	err = db.QueryRowContext(ctx, `
		SELECT scheduled_date, slot_time, devotee_gotram, devotee_nakshatram, devotee_rashi, sankalpam_purpose
		FROM bookings
		WHERE id = $1
	`, bookingID).Scan(&schedDate, &slotTime, &gotram, &nakshatram, &rashi, &purpose)
	if err != nil {
		t.Fatalf("failed querying booking row: %v", err)
	}

	if schedDate.Valid {
		t.Errorf("scheduled_date should be NULL, got %q (fabricated default string)", schedDate.String)
	}
	if slotTime.Valid {
		t.Errorf("slot_time should be NULL, got %q (fabricated default string)", slotTime.String)
	}
	if gotram.Valid {
		t.Errorf("devotee_gotram should be NULL, got %q (fabricated default string)", gotram.String)
	}
	if nakshatram.Valid {
		t.Errorf("devotee_nakshatram should be NULL, got %q (fabricated default string)", nakshatram.String)
	}
	if rashi.Valid {
		t.Errorf("devotee_rashi should be NULL, got %q (fabricated default string)", rashi.String)
	}
	if purpose.Valid {
		t.Errorf("sankalpam_purpose should be NULL, got %q (fabricated default string)", purpose.String)
	}

	// Test read query with COALESCE returns empty string, not fabricated defaults
	var readGotram, readNakshatram, readRashi, readSchedDate, readSlotTime, readPurpose string
	err = db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(devotee_gotram, ''),
			COALESCE(devotee_nakshatram, ''),
			COALESCE(devotee_rashi, ''),
			COALESCE(scheduled_date, ''),
			COALESCE(slot_time, ''),
			COALESCE(sankalpam_purpose, '')
		FROM bookings
		WHERE id = $1
	`, bookingID).Scan(&readGotram, &readNakshatram, &readRashi, &readSchedDate, &readSlotTime, &readPurpose)
	if err != nil {
		t.Fatalf("failed reading booking via COALESCE: %v", err)
	}

	if readGotram != "" || readNakshatram != "" || readRashi != "" || readSchedDate != "" || readSlotTime != "" || readPurpose != "" {
		t.Errorf("expected empty strings for unknown fields, got: gotram=%q, date=%q, purpose=%q", readGotram, readSchedDate, readPurpose)
	}
}

// 11. Regression Test: An incomplete schema cannot be baselined without writing partial history
func TestIncompleteSchemaCannotBeBaselined(t *testing.T) {
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)
	ctx := context.Background()

	// Subtest A: Completely empty database
	t.Run("EmptyDatabaseFailsBaseline", func(t *testing.T) {
		db, _ := createIsolatedTestDB(t)
		migrator := New(db, migDir, seedsDir)

		_, err := migrator.Baseline(ctx, "000003")
		if err == nil {
			t.Fatalf("expected baseline on empty database to fail schema compatibility check, but succeeded")
		}
		if !strings.Contains(err.Error(), "schema compatibility check failed") {
			t.Errorf("expected error to mention schema compatibility check failed, got: %v", err)
		}

		// Ensure 0 records were written to schema_migrations
		var historyCount int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&historyCount)
		if err != nil {
			t.Fatalf("failed to query schema_migrations: %v", err)
		}
		if historyCount != 0 {
			t.Errorf("expected 0 history records written, got %d (partial write occurred)", historyCount)
		}
	})

	// Subtest B: Partial schema missing required tables
	t.Run("PartialSchemaMissingTablesFailsBaseline", func(t *testing.T) {
		db, _ := createIsolatedTestDB(t)
		migrator := New(db, migDir, seedsDir)

		// Create only users table
		_, err := db.ExecContext(ctx, `
			CREATE TABLE users (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				phone VARCHAR(20) UNIQUE NOT NULL,
				full_name VARCHAR(100) NOT NULL
			);
		`)
		if err != nil {
			t.Fatalf("failed to create users table fixture: %v", err)
		}

		_, err = migrator.Baseline(ctx, "000001")
		if err == nil {
			t.Fatalf("expected baseline on partial schema to fail, but succeeded")
		}
		if !strings.Contains(err.Error(), "missing table") {
			t.Errorf("expected error to mention missing table, got: %v", err)
		}

		var historyCount int
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&historyCount)
		if historyCount != 0 {
			t.Errorf("partial history written: expected 0, got %d", historyCount)
		}
	})

	// Subtest C: Schema missing foreign key constraint bookings_pooja_id_fkey
	t.Run("SchemaMissingForeignKeyFailsBaseline", func(t *testing.T) {
		db, _ := createIsolatedTestDB(t)
		migrator := New(db, migDir, seedsDir)

		// Create tables for 000001 but without FK from bookings to pooja_catalog
		_, err := db.ExecContext(ctx, `
			CREATE TABLE users (id UUID PRIMARY KEY, phone VARCHAR(20) UNIQUE NOT NULL, full_name VARCHAR(100) NOT NULL);
			CREATE TABLE priests (id UUID PRIMARY KEY, phone VARCHAR(20) UNIQUE NOT NULL, full_name VARCHAR(100) NOT NULL, sampradaya VARCHAR(50) NOT NULL);
			CREATE TABLE pooja_catalog (id UUID PRIMARY KEY, title VARCHAR(150) NOT NULL, base_price NUMERIC(10, 2) NOT NULL);
			CREATE TABLE webrtc_rooms (id UUID PRIMARY KEY);
			CREATE TABLE bookings (
				id UUID PRIMARY KEY,
				user_id UUID REFERENCES users(id),
				priest_id UUID REFERENCES priests(id),
				pooja_id UUID, -- NO foreign key constraint to pooja_catalog
				status VARCHAR(30) NOT NULL DEFAULT 'REQUESTED',
				total_amount NUMERIC(10, 2) NOT NULL
			);
		`)
		if err != nil {
			t.Fatalf("failed to create fixture: %v", err)
		}

		_, err = migrator.Baseline(ctx, "000001")
		if err == nil {
			t.Fatalf("expected baseline to fail due to missing foreign key, but succeeded")
		}
		if !strings.Contains(err.Error(), "foreign key constraint") || !strings.Contains(err.Error(), "pooja_catalog") {
			t.Errorf("expected error mentioning foreign key constraint, got: %v", err)
		}

		var historyCount int
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&historyCount)
		if historyCount != 0 {
			t.Errorf("partial history written: expected 0, got %d", historyCount)
		}
	})
}

// 12. Regression Test: Invalid baseline versions and existing checksum mismatches fail
func TestInvalidBaselineVersionsAndExistingChecksumMismatchesFail(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)

	migrator := New(db, migDir, seedsDir)
	ctx := context.Background()

	// Apply migrations 000001 to 000003 manually as schema fixture
	files, err := os.ReadDir(migDir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}
	for _, f := range files {
		if f.Name() < "000004" && strings.HasSuffix(f.Name(), ".sql") {
			content, _ := os.ReadFile(filepath.Join(migDir, f.Name()))
			_, err := db.ExecContext(ctx, string(content))
			if err != nil {
				t.Fatalf("failed applying fixture %s: %v", f.Name(), err)
			}
		}
	}

	// Subtest A: Invalid baseline version strings
	t.Run("InvalidVersionFails", func(t *testing.T) {
		_, err := migrator.Baseline(ctx, "000099")
		if err == nil {
			t.Errorf("expected Baseline with version '000099' to fail, but succeeded")
		}
		if !strings.Contains(err.Error(), "version not found in migration files") {
			t.Errorf("expected error mentioning version not found, got: %v", err)
		}

		_, err = migrator.Baseline(ctx, "nonexistent")
		if err == nil {
			t.Errorf("expected Baseline with version 'nonexistent' to fail, but succeeded")
		}
	})

	// Subtest B: Checksum mismatch on already-recorded migration
	t.Run("CorruptedChecksumFails", func(t *testing.T) {
		// Validly baseline 000001 first
		_, err := migrator.Baseline(ctx, "000001")
		if err != nil {
			t.Fatalf("baseline 000001 failed: %v", err)
		}

		// Corrupt the recorded checksum for 000001
		corruptedHash := "deadbeef00000000000000000000000000000000000000000000000000000000"
		_, err = db.ExecContext(ctx, "UPDATE schema_migrations SET checksum = $1 WHERE version = '000001'", corruptedHash)
		if err != nil {
			t.Fatalf("failed updating checksum: %v", err)
		}

		// Attempting further baseline or Up must fail with checksum mismatch
		_, err = migrator.Baseline(ctx, "000003")
		if err == nil {
			t.Fatalf("expected Baseline to fail on checksum mismatch, but succeeded")
		}
		if !strings.Contains(err.Error(), "checksum mismatch") {
			t.Errorf("expected error containing 'checksum mismatch', got: %v", err)
		}

		_, err = migrator.Up(ctx, false)
		if err == nil {
			t.Fatalf("expected Up to fail on checksum mismatch, but succeeded")
		}
		if !strings.Contains(err.Error(), "checksum mismatch") {
			t.Errorf("expected error containing 'checksum mismatch', got: %v", err)
		}
	})

	// Subtest C: Historical un-gated checksum is accepted
	t.Run("HistoricalUngatedChecksumAccepted", func(t *testing.T) {
		// Set original un-gated checksum for 000001
		origUngatedChecksum000001 := "cc30cba2e78cfeceee683e862f535240601953ac45e2c5f8d6359365737ecf37"
		_, err = db.ExecContext(ctx, "UPDATE schema_migrations SET checksum = $1 WHERE version = '000001'", origUngatedChecksum000001)
		if err != nil {
			t.Fatalf("failed updating to historical checksum: %v", err)
		}

		// Now Baseline up to 000003 should succeed because 000001 checksum matches approved historical list
		baselined, err := migrator.Baseline(ctx, "000003")
		if err != nil {
			t.Fatalf("Baseline failed with historical checksum: %v", err)
		}
		if baselined != 2 { // 000002 and 000003 baselined (000001 was already present)
			t.Errorf("expected 2 newly baselined migrations, got %d", baselined)
		}
	})
}

// 13. Regression Test: Valid existing data survives upgrade
func TestValidExistingDataSurvivesUpgrade(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)
	ctx := context.Background()

	// Apply schema migrations up to 000003
	files, err := os.ReadDir(migDir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}
	for _, f := range files {
		if f.Name() < "000004" && strings.HasSuffix(f.Name(), ".sql") {
			content, _ := os.ReadFile(filepath.Join(migDir, f.Name()))
			_, err := db.ExecContext(ctx, string(content))
			if err != nil {
				t.Fatalf("failed applying fixture %s: %v", f.Name(), err)
			}
		}
	}

	// Insert realistic production data
	customUserID := "33333333-3333-3333-3333-333333333333"
	customPriestID := "44444444-4444-4444-4444-444444444444"
	customServiceID := "55555555-5555-5555-5555-555555555555"

	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, phone, full_name, gotram, nakshatram, rashi, city, consent_agreed)
		VALUES ($1, '+919848011223', 'Smt. Lakshmi Devi', 'Kashyapa', 'Rohini', 'Vrishabha', 'Vijayawada', true);
	`, customUserID)
	if err != nil {
		t.Fatalf("failed to insert user fixture: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO priests (id, phone, full_name, sampradaya, veda_shakha, is_online, consent_agreed)
		VALUES ($1, '+919848099887', 'Brahmasri Venkata Somayajulu', 'Smartha', 'Yajurveda', true, true);
	`, customPriestID)
	if err != nil {
		t.Fatalf("failed to insert priest fixture: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO priest_custom_services (id, priest_id, title, category, dakshina_fee, duration_minutes)
		VALUES ($1, $2, 'Special Chandi Sapthashathi Parayanam', 'Vedic Anushthanam', 12116.0, 180);
	`, customServiceID, customPriestID)
	if err != nil {
		t.Fatalf("failed to insert custom service fixture: %v", err)
	}

	migrator := New(db, migDir, seedsDir)

	// Baseline up to version 000003
	_, err = migrator.Baseline(ctx, "000003")
	if err != nil {
		t.Fatalf("Baseline failed: %v", err)
	}

	// Apply forward migrations 000004, 000005, 000006, 000007
	applied, err := migrator.Up(ctx, false)
	if err != nil {
		t.Fatalf("Up failed: %v", err)
	}
	if applied != 13 {
		t.Errorf("expected 13 migrations applied (000004 through 000016), got %d", applied)
	}

	// Verify all pre-existing records survive intact
	var uName, uPhone, uGotram, uNakshatram, uRashi string
	var uPassHash string
	var uSessionVer int
	err = db.QueryRowContext(ctx, `
		SELECT full_name, phone, gotram, nakshatram, rashi, password_hash, session_version
		FROM users
		WHERE id = $1
	`, customUserID).Scan(&uName, &uPhone, &uGotram, &uNakshatram, &uRashi, &uPassHash, &uSessionVer)
	if err != nil {
		t.Fatalf("user record query failed: %v", err)
	}

	if uName != "Smt. Lakshmi Devi" || uPhone != "+919848011223" || uGotram != "Kashyapa" || uNakshatram != "Rohini" || uRashi != "Vrishabha" {
		t.Errorf("user record corrupted during upgrade: name=%s, gotram=%s, nakshatram=%s", uName, uGotram, uNakshatram)
	}
	if uSessionVer != 1 {
		t.Errorf("expected default session_version 1, got %d", uSessionVer)
	}
	if uPassHash != "" {
		t.Errorf("expected empty password_hash for existing user, got %q", uPassHash)
	}

	var pName, pPhone, pSampradaya string
	var pSessionVer int
	err = db.QueryRowContext(ctx, `
		SELECT full_name, phone, sampradaya, session_version
		FROM priests
		WHERE id = $1
	`, customPriestID).Scan(&pName, &pPhone, &pSampradaya, &pSessionVer)
	if err != nil {
		t.Fatalf("priest record query failed: %v", err)
	}
	if pName != "Brahmasri Venkata Somayajulu" || pPhone != "+919848099887" || pSampradaya != "Smartha" {
		t.Errorf("priest record corrupted during upgrade: %s, %s, %s", pName, pPhone, pSampradaya)
	}

	var sTitle string
	var sFee float64
	err = db.QueryRowContext(ctx, `
		SELECT title, dakshina_fee
		FROM priest_custom_services
		WHERE id = $1
	`, customServiceID).Scan(&sTitle, &sFee)
	if err != nil {
		t.Fatalf("custom service query failed: %v", err)
	}
	if sTitle != "Special Chandi Sapthashathi Parayanam" || sFee != 12116.0 {
		t.Errorf("custom service corrupted: title=%s, fee=%f", sTitle, sFee)
	}
}

// 14. Regression Test: Avoid deadlocks when the pool has only one available connection
func TestLockOwnershipWithSingleConnectionPool(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	db.SetMaxOpenConns(1) // Constrain connection pool to exactly 1 physical connection
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)

	migrator := New(db, migDir, seedsDir)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Up must complete without deadlock even when MaxOpenConns = 1
	applied, err := migrator.Up(ctx, false)
	if err != nil {
		t.Fatalf("Up failed with MaxOpenConns(1): %v", err)
	}
	if applied < 5 {
		t.Errorf("expected at least 5 migrations applied, got %d", applied)
	}
}

// 15. Regression Test: Lock cancellation cleanup and subsequent successful acquisition
func TestLockOwnershipCancellationAndSubsequentAcquisition(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)

	migrator := New(db, migDir, seedsDir)

	// Step 1: Pre-cancelled context should fail immediately and clean up
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, _, err := migrator.AcquireLock(cancelledCtx)
	if err == nil {
		t.Fatalf("expected AcquireLock with cancelled context to fail, but succeeded")
	}

	// Step 2: Subsequent lock acquisition must succeed without being blocked by leaked session
	validCtx, validCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer validCancel()

	conn, unlock, err := migrator.AcquireLock(validCtx)
	if err != nil {
		t.Fatalf("subsequent AcquireLock failed after cancelled attempt: %v", err)
	}
	if conn == nil {
		t.Fatalf("expected valid dedicated connection")
	}

	// Unlock should release cleanly
	unlock()
}

// 16. Regression Test: Wrong column types reject baseline
func TestSchemaCompatibilityWrongColumnTypes(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)
	ctx := context.Background()

	// Create users table with phone as INTEGER instead of VARCHAR
	_, err := db.ExecContext(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			phone INTEGER UNIQUE NOT NULL, -- WRONG TYPE: integer instead of character varying
			full_name VARCHAR(100) NOT NULL
		);
		CREATE TABLE priests (
			id UUID PRIMARY KEY,
			phone VARCHAR(20) UNIQUE NOT NULL,
			full_name VARCHAR(100) NOT NULL,
			sampradaya VARCHAR(50) NOT NULL
		);
		CREATE TABLE pooja_catalog (
			id UUID PRIMARY KEY,
			title VARCHAR(150) NOT NULL,
			base_price NUMERIC(10, 2) NOT NULL
		);
		CREATE TABLE webrtc_rooms (id UUID PRIMARY KEY);
		CREATE TABLE bookings (
			id UUID PRIMARY KEY,
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			priest_id UUID REFERENCES priests(id) ON DELETE SET NULL,
			pooja_id UUID REFERENCES pooja_catalog(id) ON DELETE RESTRICT,
			status VARCHAR(30) NOT NULL DEFAULT 'REQUESTED',
			total_amount NUMERIC(10, 2) NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("failed creating fixture: %v", err)
	}

	migrator := New(db, migDir, seedsDir)
	_, err = migrator.Baseline(ctx, "000001")
	if err == nil {
		t.Fatalf("expected baseline to fail on wrong column type for users.phone, but succeeded")
	}
	if !strings.Contains(err.Error(), "type 'integer', expected 'character varying'") {
		t.Errorf("expected error mentioning wrong column type, got: %v", err)
	}
}

// 17. Regression Test: Missing required columns reject baseline
func TestSchemaCompatibilityMissingRequiredColumns(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)
	ctx := context.Background()

	// Create users table missing full_name column
	_, err := db.ExecContext(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			phone VARCHAR(20) UNIQUE NOT NULL
			-- MISSING: full_name column
		);
		CREATE TABLE priests (
			id UUID PRIMARY KEY,
			phone VARCHAR(20) UNIQUE NOT NULL,
			full_name VARCHAR(100) NOT NULL,
			sampradaya VARCHAR(50) NOT NULL
		);
		CREATE TABLE pooja_catalog (
			id UUID PRIMARY KEY,
			title VARCHAR(150) NOT NULL,
			base_price NUMERIC(10, 2) NOT NULL
		);
		CREATE TABLE webrtc_rooms (id UUID PRIMARY KEY);
		CREATE TABLE bookings (
			id UUID PRIMARY KEY,
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			priest_id UUID REFERENCES priests(id) ON DELETE SET NULL,
			pooja_id UUID REFERENCES pooja_catalog(id) ON DELETE RESTRICT,
			status VARCHAR(30) NOT NULL DEFAULT 'REQUESTED',
			total_amount NUMERIC(10, 2) NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("failed creating fixture: %v", err)
	}

	migrator := New(db, migDir, seedsDir)
	_, err = migrator.Baseline(ctx, "000001")
	if err == nil {
		t.Fatalf("expected baseline to fail on missing column full_name, but succeeded")
	}
	if !strings.Contains(err.Error(), "missing required column 'full_name'") {
		t.Errorf("expected error mentioning missing column 'full_name', got: %v", err)
	}
}

// 18. Regression Test: Foreign key on wrong column rejects baseline
func TestSchemaCompatibilityForeignKeyOnWrongColumn(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)
	ctx := context.Background()

	// Create bookings where foreign key to pooja_catalog is mistakenly on priest_id instead of pooja_id
	_, err := db.ExecContext(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			phone VARCHAR(20) UNIQUE NOT NULL,
			full_name VARCHAR(100) NOT NULL
		);
		CREATE TABLE priests (
			id UUID PRIMARY KEY,
			phone VARCHAR(20) UNIQUE NOT NULL,
			full_name VARCHAR(100) NOT NULL,
			sampradaya VARCHAR(50) NOT NULL
		);
		CREATE TABLE pooja_catalog (
			id UUID PRIMARY KEY,
			title VARCHAR(150) NOT NULL,
			base_price NUMERIC(10, 2) NOT NULL
		);
		CREATE TABLE webrtc_rooms (id UUID PRIMARY KEY);
		CREATE TABLE bookings (
			id UUID PRIMARY KEY,
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			priest_id UUID REFERENCES pooja_catalog(id) ON DELETE RESTRICT, -- WRONG SOURCE COLUMN!
			pooja_id UUID, -- No constraint
			status VARCHAR(30) NOT NULL DEFAULT 'REQUESTED',
			total_amount NUMERIC(10, 2) NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("failed creating fixture: %v", err)
	}

	migrator := New(db, migDir, seedsDir)
	_, err = migrator.Baseline(ctx, "000001")
	if err == nil {
		t.Fatalf("expected baseline to fail on foreign key on wrong column, but succeeded")
	}
	if !strings.Contains(err.Error(), "missing foreign key constraint from bookings(pooja_id) to pooja_catalog(id)") {
		t.Errorf("expected error mentioning missing foreign key constraint from bookings(pooja_id) to pooja_catalog(id), got: %v", err)
	}
}

// 19. Regression Test: Missing uniqueness rejects baseline
func TestSchemaCompatibilityMissingUniqueness(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)
	ctx := context.Background()

	// Create users table WITHOUT unique constraint on phone
	_, err := db.ExecContext(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			phone VARCHAR(20) NOT NULL, -- MISSING UNIQUE CONSTRAINT!
			full_name VARCHAR(100) NOT NULL
		);
		CREATE TABLE priests (
			id UUID PRIMARY KEY,
			phone VARCHAR(20) UNIQUE NOT NULL,
			full_name VARCHAR(100) NOT NULL,
			sampradaya VARCHAR(50) NOT NULL
		);
		CREATE TABLE pooja_catalog (
			id UUID PRIMARY KEY,
			title VARCHAR(150) NOT NULL,
			base_price NUMERIC(10, 2) NOT NULL
		);
		CREATE TABLE webrtc_rooms (id UUID PRIMARY KEY);
		CREATE TABLE bookings (
			id UUID PRIMARY KEY,
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			priest_id UUID REFERENCES priests(id) ON DELETE SET NULL,
			pooja_id UUID REFERENCES pooja_catalog(id) ON DELETE RESTRICT,
			status VARCHAR(30) NOT NULL DEFAULT 'REQUESTED',
			total_amount NUMERIC(10, 2) NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("failed creating fixture: %v", err)
	}

	migrator := New(db, migDir, seedsDir)
	_, err = migrator.Baseline(ctx, "000001")
	if err == nil {
		t.Fatalf("expected baseline to fail on missing unique constraint, but succeeded")
	}
	if !strings.Contains(err.Error(), "missing unique constraint on column(s) 'phone'") {
		t.Errorf("expected error mentioning missing unique constraint, got: %v", err)
	}
}

// 20. Regression Test: Readiness check detects migration history gap
func TestReadinessDetectsMigrationHistoryGap(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	migDir := getMigrationsDir(t)
	seedsDir := getSeedsDir(t)
	ctx := context.Background()

	migrator := New(db, migDir, seedsDir)

	// Apply all migrations 000001 through 000005
	_, err := migrator.Up(ctx, false)
	if err != nil {
		t.Fatalf("Up failed: %v", err)
	}

	// Verify readiness initially passes
	if err := migrator.VerifyReadiness(ctx, "000005"); err != nil {
		t.Fatalf("expected initial VerifyReadiness to pass, got: %v", err)
	}

	// Simulate an intermediate migration history gap by deleting migration 000003 from schema_migrations
	_, err = db.ExecContext(ctx, "DELETE FROM schema_migrations WHERE version = '000003'")
	if err != nil {
		t.Fatalf("failed to simulate history gap: %v", err)
	}

	// VerifyReadiness must now fail with history gap error
	err = migrator.VerifyReadiness(ctx, "000005")
	if err == nil {
		t.Fatalf("expected VerifyReadiness to fail on missing intermediate migration 000003, but succeeded")
	}
	if !strings.Contains(err.Error(), "history gap") || !strings.Contains(err.Error(), "000003") {
		t.Errorf("expected error mentioning history gap and version 000003, got: %v", err)
	}
}

// 21. Regression Test: Failed unlock or cancelled session safely discards physical connection, freeing lock
func TestLockOwnershipDiscardConnSafelyDestroysSessionOnFailure(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	db.SetMaxOpenConns(1) // Single pooled connection
	ctx := context.Background()

	migrator := New(db, getMigrationsDir(t), getSeedsDir(t))

	// Acquire lock on dedicated connection
	conn1, _, err := migrator.AcquireLock(ctx)
	if err != nil {
		t.Fatalf("failed acquiring lock on conn1: %v", err)
	}

	// Simulate uncertain/failed release or aborted transaction:
	// DiscardConn destroys the physical driver socket and forces database/sql to purge it
	DiscardConn(conn1)

	// In a pool limited to 1 connection, AcquireLock must succeed because the dead connection was discarded
	// and a new physical connection was spawned without inheriting the old session's advisory lock
	conn2, unlock2, err := migrator.AcquireLock(ctx)
	if err != nil {
		t.Fatalf("expected subsequent AcquireLock on single-connection pool to succeed after DiscardConn, got: %v", err)
	}
	defer unlock2()

	// Verify clean unlock works
	unlock2()
	_ = conn2
}

// 22. Acceptance Test: Startup readiness verifies valid schema 000008 completely
func TestStartupReadinessValidSchema(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	ctx := context.Background()
	migrator := New(db, getMigrationsDir(t), getSeedsDir(t))

	// Up to latest version (includes 000008 priest location confirmation)
	applied, err := migrator.Up(ctx, false)
	if err != nil {
		t.Fatalf("Up failed: %v", err)
	}
	if applied < 8 {
		t.Fatalf("expected at least 8 migrations applied, got %d", applied)
	}

	if err := migrator.VerifyReadiness(ctx, "000008"); err != nil {
		t.Fatalf("expected VerifyReadiness to pass on valid schema, got: %v", err)
	}
}

// 23. Acceptance Test: Startup readiness rejects history gap even when latest version 000008 is present
func TestStartupReadinessHistoryGapFailsDespiteLatestVersion(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	ctx := context.Background()
	migrator := New(db, getMigrationsDir(t), getSeedsDir(t))

	_, err := migrator.Up(ctx, false)
	if err != nil {
		t.Fatalf("Up failed: %v", err)
	}

	// Delete intermediate migration 000004 from schema_migrations
	_, err = db.ExecContext(ctx, "DELETE FROM schema_migrations WHERE version = '000004'")
	if err != nil {
		t.Fatalf("failed deleting migration: %v", err)
	}

	// Latest version in schema_migrations is 000008, but intermediate version 000004 is missing
	var maxVer string
	_ = db.QueryRowContext(ctx, "SELECT MAX(version) FROM schema_migrations").Scan(&maxVer)
	if maxVer != "000016" {
		t.Fatalf("expected max version to be 000016, got %s", maxVer)
	}

	err = migrator.VerifyReadiness(ctx, "000008")
	if err == nil {
		t.Fatalf("expected VerifyReadiness to fail on missing intermediate migration 000004, but succeeded")
	}
	if !strings.Contains(err.Error(), "history gap") || !strings.Contains(err.Error(), "000004") {
		t.Errorf("expected error mentioning history gap and version 000004, got: %v", err)
	}
}

// 24. Acceptance Test: Startup readiness rejects physical schema incompatibility even when version matches
func TestStartupReadinessPhysicalIncompatibilityFails(t *testing.T) {
	db, _ := createIsolatedTestDB(t)
	ctx := context.Background()
	migrator := New(db, getMigrationsDir(t), getSeedsDir(t))

	_, err := migrator.Up(ctx, false)
	if err != nil {
		t.Fatalf("Up failed: %v", err)
	}

	// Drop required table password_recovery_challenges
	_, err = db.ExecContext(ctx, "DROP TABLE password_recovery_challenges CASCADE")
	if err != nil {
		t.Fatalf("failed dropping table: %v", err)
	}

	err = migrator.VerifyReadiness(ctx, "000008")
	if err == nil {
		t.Fatalf("expected VerifyReadiness to fail when required table is missing, but succeeded")
	}
	if !strings.Contains(err.Error(), "missing table 'password_recovery_challenges'") {
		t.Errorf("expected error mentioning missing table, got: %v", err)
	}
}
