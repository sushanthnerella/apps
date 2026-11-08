package migrate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// MigrationAdvisoryLockID is a dedicated 64-bit advisory lock key ("PUJAVAAM" in hex)
const MigrationAdvisoryLockID int64 = 0x50554A4156414148

// Migration represents a discovered SQL migration file on disk
type Migration struct {
	Version  string
	Name     string
	Path     string
	Checksum string
	SQL      string
}

// AppliedMigration represents a row recorded in the schema_migrations table
type AppliedMigration struct {
	Version         string
	Name            string
	Checksum        string
	AppliedAt       time.Time
	ExecutionTimeMs int
}

// StatusReport details the current state of applied vs pending migrations
type StatusReport struct {
	AppliedCount int
	PendingCount int
	CurrentVer   string
	Details      []MigrationStatusItem
}

// MigrationStatusItem represents status for a single migration file
type MigrationStatusItem struct {
	Version   string
	Name      string
	Checksum  string
	Applied   bool
	AppliedAt *time.Time
}

// Migrator manages idempotent, transactional PostgreSQL schema migrations
type Migrator struct {
	db            *sql.DB
	migrationsDir string
	seedsDir      string
}

// New creates a Migrator configured with the given database handle and directory paths
func New(db *sql.DB, migrationsDir string, seedsDir string) *Migrator {
	return &Migrator{
		db:            db,
		migrationsDir: migrationsDir,
		seedsDir:      seedsDir,
	}
}

// ComputeChecksum normalizes line endings and returns a SHA-256 hex string
func ComputeChecksum(data []byte) string {
	normalized := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	hash := sha256.Sum256(normalized)
	return hex.EncodeToString(hash[:])
}

// queryExecer abstracts *sql.DB and *sql.Conn for unified query execution
type queryExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// EnsureSchemaHistoryTable ensures the schema_migrations audit table exists
func (m *Migrator) EnsureSchemaHistoryTable(ctx context.Context) error {
	return m.ensureSchemaHistoryTable(ctx, m.db)
}

func (m *Migrator) ensureSchemaHistoryTable(ctx context.Context, exec queryExecer) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(50) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			checksum VARCHAR(64) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			execution_time_ms INTEGER NOT NULL
		);
	`
	_, err := exec.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}
	return nil
}

// DiscardConn safely closes the underlying physical driver connection and ensures
// that database/sql discards it rather than returning a potentially locked session to the pool.
func DiscardConn(conn *sql.Conn) {
	if conn == nil {
		return
	}
	_ = conn.Raw(func(driverConn any) error {
		if dc, ok := driverConn.(driver.Conn); ok {
			_ = dc.Close()
		} else if closer, ok := driverConn.(io.Closer); ok {
			_ = closer.Close()
		}
		return driver.ErrBadConn
	})
	_ = conn.Close()
}

// AcquireLock acquires a dedicated sql.Conn and obtains a session-scoped PostgreSQL advisory lock.
// It returns the dedicated *sql.Conn and an unlock function that verifies the unlock and closes the connection.
// If lock acquisition fails or is cancelled, the physical connection is discarded immediately to prevent returning
// a still-locked session to the pool.
func (m *Migrator) AcquireLock(ctx context.Context) (*sql.Conn, func(), error) {
	conn, err := m.db.Conn(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to obtain dedicated connection for migration lock: %w", err)
	}

	_, err = conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", MigrationAdvisoryLockID)
	if err != nil {
		DiscardConn(conn)
		return nil, nil, fmt.Errorf("failed to acquire migration advisory lock: %w", err)
	}

	var once sync.Once
	unlock := func() {
		once.Do(func() {
			unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var unlocked bool
			err := conn.QueryRowContext(unlockCtx, "SELECT pg_advisory_unlock($1)", MigrationAdvisoryLockID).Scan(&unlocked)
			if err != nil || !unlocked {
				log.Printf("⚠️ Warning: pg_advisory_unlock failed (unlocked=%v, err=%v). Discarding physical connection to destroy session lock.", unlocked, err)
				DiscardConn(conn)
				return
			}
			_ = conn.Close()
		})
	}

	return conn, unlock, nil
}

// LoadMigrationFiles scans and parses all .sql files in migrationsDir in alphabetical version order
func (m *Migrator) LoadMigrationFiles() ([]Migration, error) {
	entries, err := os.ReadDir(m.migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory %s: %w", m.migrationsDir, err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		name := entry.Name()
		parts := strings.SplitN(name, "_", 2)
		if len(parts) < 2 {
			continue // skip non-numbered files
		}
		version := parts[0]

		path := filepath.Join(m.migrationsDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", path, err)
		}

		migrations = append(migrations, Migration{
			Version:  version,
			Name:     name,
			Path:     path,
			Checksum: ComputeChecksum(data),
			SQL:      string(data),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// GetAppliedMigrations returns a map and slice of all applied migrations from schema_migrations
func (m *Migrator) GetAppliedMigrations(ctx context.Context) (map[string]AppliedMigration, []AppliedMigration, error) {
	return m.getAppliedMigrations(ctx, m.db)
}

func (m *Migrator) getAppliedMigrations(ctx context.Context, exec queryExecer) (map[string]AppliedMigration, []AppliedMigration, error) {
	if err := m.ensureSchemaHistoryTable(ctx, exec); err != nil {
		return nil, nil, err
	}

	rows, err := exec.QueryContext(ctx, "SELECT version, name, checksum, applied_at, execution_time_ms FROM schema_migrations ORDER BY version ASC")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query schema_migrations: %w", err)
	}
	defer rows.Close()

	appliedMap := make(map[string]AppliedMigration)
	var appliedList []AppliedMigration
	for rows.Next() {
		var am AppliedMigration
		if err := rows.Scan(&am.Version, &am.Name, &am.Checksum, &am.AppliedAt, &am.ExecutionTimeMs); err != nil {
			return nil, nil, fmt.Errorf("failed to scan applied migration row: %w", err)
		}
		appliedMap[am.Version] = am
		appliedList = append(appliedList, am)
	}

	return appliedMap, appliedList, nil
}

// compatibleHistoricalChecksums maps migration versions to recognized valid SHA-256 checksums
// (e.g. original un-gated seeds vs gated seeds).
var compatibleHistoricalChecksums = map[string][]string{
	"000001": {
		"866b5a658f8c58c9b32032e3810715bc7bd320fc2d7d4617eeecb811a31754ff", // gated demo seeds
		"cc30cba2e78cfeceee683e862f535240601953ac45e2c5f8d6359365737ecf37", // original un-gated
	},
	"000002": {
		"a30d828084b7a68f569f824b5691feda7f61f81b2988236a2f3cd983cd72801d", // gated demo seeds
		"55916db2f1937307a1847bc5d0c991f008228784de8890b8e21497bb007ab3bc", // original un-gated
	},
	"000003": {
		"6d45429bffa395cf954dde6d2aa97bbd4f0da6828ad79bd4ae6c99aed2c6bdb2", // gated demo seeds
		"86d4cd2febf5526731c912c1331eb129e8e5e445da1ba97d170cbde77896f0cf", // original un-gated
	},
}

// isCompatibleChecksum returns true if the recorded checksum matches fileChecksum or is an approved historical variant
func isCompatibleChecksum(version, recordedChecksum, fileChecksum string) bool {
	if recordedChecksum == fileChecksum {
		return true
	}
	allowed, ok := compatibleHistoricalChecksums[version]
	if !ok {
		return false
	}
	recMatch := false
	fileMatch := false
	for _, c := range allowed {
		if recordedChecksum == c {
			recMatch = true
		}
		if fileChecksum == c {
			fileMatch = true
		}
	}
	return recMatch && fileMatch
}

// VerifyChecksums checks that already-applied migrations have not been altered on disk
func (m *Migrator) VerifyChecksums(files []Migration, applied map[string]AppliedMigration) error {
	for _, file := range files {
		if rec, exists := applied[file.Version]; exists {
			if !isCompatibleChecksum(file.Version, rec.Checksum, file.Checksum) {
				return fmt.Errorf("checksum mismatch for migration %s (version %s): database recorded %s, current file is %s. Applied migrations must be immutable",
					file.Name, file.Version, rec.Checksum, file.Checksum)
			}
		}
	}
	return nil
}

// Up runs all pending forward migrations in sequential order inside transactions on a locked connection
func (m *Migrator) Up(ctx context.Context, seedDemo bool) (int, error) {
	conn, unlock, err := m.AcquireLock(ctx)
	if err != nil {
		return 0, err
	}
	defer unlock()

	files, err := m.LoadMigrationFiles()
	if err != nil {
		return 0, err
	}

	appliedMap, _, err := m.getAppliedMigrations(ctx, conn)
	if err != nil {
		return 0, err
	}

	if err := m.VerifyChecksums(files, appliedMap); err != nil {
		return 0, err
	}

	appliedCount := 0
	for _, mig := range files {
		if _, exists := appliedMap[mig.Version]; exists {
			continue // already applied
		}

		start := time.Now()
		// Begin transaction on the same locked connection to prevent deadlocks when pool has 1 connection
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return appliedCount, fmt.Errorf("failed to begin transaction for migration %s: %w", mig.Name, err)
		}

		// Configure session demo flag: default false for pilot production
		seedSetting := "false"
		if seedDemo {
			seedSetting = "true"
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL purohit.seed_demo = '%s'", seedSetting)); err != nil {
			_ = tx.Rollback()
			return appliedCount, fmt.Errorf("failed to set demo flag for migration %s: %w", mig.Name, err)
		}

		// Execute migration DDL / DML
		if _, err := tx.ExecContext(ctx, mig.SQL); err != nil {
			_ = tx.Rollback()
			return appliedCount, fmt.Errorf("migration %s failed: %w", mig.Name, err)
		}

		execMs := int(time.Since(start).Milliseconds())

		// Record migration in schema_migrations within the same transaction
		recordQuery := `
			INSERT INTO schema_migrations (version, name, checksum, execution_time_ms)
			VALUES ($1, $2, $3, $4)
		`
		if _, err := tx.ExecContext(ctx, recordQuery, mig.Version, mig.Name, mig.Checksum, execMs); err != nil {
			_ = tx.Rollback()
			return appliedCount, fmt.Errorf("failed to record migration %s in history: %w", mig.Name, err)
		}

		if err := tx.Commit(); err != nil {
			return appliedCount, fmt.Errorf("failed to commit migration %s: %w", mig.Name, err)
		}

		appliedCount++
		log.Printf("✅ Applied migration %s (%d ms)", mig.Name, execMs)
	}

	return appliedCount, nil
}

// Status returns a detailed report of applied and pending migrations
func (m *Migrator) Status(ctx context.Context) (*StatusReport, error) {
	files, err := m.LoadMigrationFiles()
	if err != nil {
		return nil, err
	}

	appliedMap, appliedList, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	report := &StatusReport{
		AppliedCount: len(appliedList),
		Details:      make([]MigrationStatusItem, 0, len(files)),
	}
	if len(appliedList) > 0 {
		report.CurrentVer = appliedList[len(appliedList)-1].Version
	}

	for _, file := range files {
		item := MigrationStatusItem{
			Version:  file.Version,
			Name:     file.Name,
			Checksum: file.Checksum,
		}
		if rec, exists := appliedMap[file.Version]; exists {
			item.Applied = true
			t := rec.AppliedAt
			item.AppliedAt = &t
		} else {
			report.PendingCount++
		}
		report.Details = append(report.Details, item)
	}

	return report, nil
}

// ValidateSchemaCompatibility checks that the physical database schema contains all required
// tables, columns (with exact types and nullability), unique constraints, and exact foreign key definitions.
func (m *Migrator) ValidateSchemaCompatibility(ctx context.Context, targetVersion string) error {
	return m.validateSchemaCompatibility(ctx, targetVersion, m.db)
}

type colMeta struct {
	dataType   string
	isNullable bool
}

type fkMeta struct {
	srcTable     string
	srcCols      string
	refTable     string
	refCols      string
	deleteAction string
}

func (m *Migrator) validateSchemaCompatibility(ctx context.Context, targetVersion string, exec queryExecer) error {
	// 1. Gather all existing public tables
	tableRows, err := exec.QueryContext(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'")
	if err != nil {
		return fmt.Errorf("failed to inspect database tables: %w", err)
	}
	defer tableRows.Close()

	existingTables := make(map[string]bool)
	for tableRows.Next() {
		var tbl string
		if err := tableRows.Scan(&tbl); err != nil {
			return fmt.Errorf("failed scanning table name: %w", err)
		}
		existingTables[tbl] = true
	}

	// 2. Gather all existing public columns: table -> col -> colMeta
	colRows, err := exec.QueryContext(ctx, "SELECT table_name, column_name, data_type, is_nullable FROM information_schema.columns WHERE table_schema = 'public'")
	if err != nil {
		return fmt.Errorf("failed to inspect database columns: %w", err)
	}
	defer colRows.Close()

	existingColumns := make(map[string]map[string]colMeta)
	for colRows.Next() {
		var tbl, col, dt, nullable string
		if err := colRows.Scan(&tbl, &col, &dt, &nullable); err != nil {
			return fmt.Errorf("failed scanning column: %w", err)
		}
		if existingColumns[tbl] == nil {
			existingColumns[tbl] = make(map[string]colMeta)
		}
		existingColumns[tbl][col] = colMeta{
			dataType:   dt,
			isNullable: strings.ToUpper(nullable) == "YES",
		}
	}

	// 3. Gather unique and primary key constraints: table -> set of column strings (e.g. "phone" or "priest_id,slot_date,start_time")
	uniqueRows, err := exec.QueryContext(ctx, `
		SELECT 
			t.relname AS table_name,
			ARRAY_TO_STRING(ARRAY_AGG(a.attname ORDER BY u.attposition), ',') AS columns
		FROM pg_constraint c
		JOIN pg_class t ON c.conrelid = t.oid
		JOIN pg_namespace n ON t.relnamespace = n.oid
		CROSS JOIN LATERAL UNNEST(c.conkey) WITH ORDINALITY AS u(attnum, attposition)
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = u.attnum
		WHERE n.nspname = 'public' AND c.contype IN ('u', 'p')
		GROUP BY t.relname, c.conname, c.contype;
	`)
	if err != nil {
		return fmt.Errorf("failed to inspect unique constraints: %w", err)
	}
	defer uniqueRows.Close()

	existingUniques := make(map[string]map[string]bool)
	for uniqueRows.Next() {
		var tbl, cols string
		if err := uniqueRows.Scan(&tbl, &cols); err != nil {
			return fmt.Errorf("failed scanning unique constraint: %w", err)
		}
		if existingUniques[tbl] == nil {
			existingUniques[tbl] = make(map[string]bool)
		}
		existingUniques[tbl][cols] = true
	}

	// 4. Gather foreign key constraints with exact source, destination, and deletion actions
	fkRows, err := exec.QueryContext(ctx, `
		SELECT 
			src_tbl.relname AS src_table,
			ARRAY_TO_STRING(ARRAY_AGG(src_col.attname ORDER BY u.attposition), ',') AS src_cols,
			ref_tbl.relname AS ref_table,
			ARRAY_TO_STRING(ARRAY_AGG(ref_col.attname ORDER BY u.attposition), ',') AS ref_cols,
			c.confdeltype::text AS delete_action
		FROM pg_constraint c
		JOIN pg_class src_tbl ON c.conrelid = src_tbl.oid
		JOIN pg_class ref_tbl ON c.confrelid = ref_tbl.oid
		JOIN pg_namespace n ON src_tbl.relnamespace = n.oid
		CROSS JOIN LATERAL UNNEST(c.conkey) WITH ORDINALITY AS u(attnum, attposition)
		JOIN pg_attribute src_col ON src_col.attrelid = src_tbl.oid AND src_col.attnum = u.attnum
		CROSS JOIN LATERAL UNNEST(c.confkey) WITH ORDINALITY AS fu(attnum, attposition)
		JOIN pg_attribute ref_col ON ref_col.attrelid = ref_tbl.oid AND ref_col.attnum = fu.attnum AND fu.attposition = u.attposition
		WHERE n.nspname = 'public' AND c.contype = 'f'
		GROUP BY c.conname, src_tbl.relname, ref_tbl.relname, c.confdeltype;
	`)
	if err != nil {
		return fmt.Errorf("failed to inspect foreign key constraints: %w", err)
	}
	defer fkRows.Close()

	var existingFKs []fkMeta
	for fkRows.Next() {
		var fk fkMeta
		if err := fkRows.Scan(&fk.srcTable, &fk.srcCols, &fk.refTable, &fk.refCols, &fk.deleteAction); err != nil {
			return fmt.Errorf("failed scanning foreign key constraint: %w", err)
		}
		existingFKs = append(existingFKs, fk)
	}

	hasFK := func(srcTbl, srcCols, refTbl, refCols string, allowedActions ...string) bool {
		for _, fk := range existingFKs {
			if fk.srcTable == srcTbl && fk.srcCols == srcCols && fk.refTable == refTbl && fk.refCols == refCols {
				for _, a := range allowedActions {
					if fk.deleteAction == a {
						return true
					}
				}
			}
		}
		return false
	}

	// 5. Version-by-version schema verification
	if targetVersion >= "000001" {
		// Required tables
		reqTables := []string{"users", "priests", "pooja_catalog", "bookings", "webrtc_rooms"}
		for _, tbl := range reqTables {
			if !existingTables[tbl] {
				return fmt.Errorf("incompatible schema for baseline version %s: missing table '%s'", targetVersion, tbl)
			}
		}

		// Required columns with data types and nullability
		type colCheck struct {
			tbl      string
			col      string
			dataType string
			notNull  bool
		}
		reqCols := []colCheck{
			{"users", "id", "uuid", true},
			{"users", "phone", "character varying", true},
			{"users", "full_name", "character varying", true},
			{"priests", "id", "uuid", true},
			{"priests", "phone", "character varying", true},
			{"priests", "full_name", "character varying", true},
			{"priests", "sampradaya", "character varying", true},
			{"pooja_catalog", "id", "uuid", true},
			{"pooja_catalog", "title", "character varying", true},
			{"pooja_catalog", "base_price", "numeric", true},
			{"bookings", "id", "uuid", true},
			{"bookings", "user_id", "uuid", false},
			{"bookings", "priest_id", "uuid", false},
			{"bookings", "pooja_id", "uuid", false},
			{"bookings", "status", "character varying", true},
			{"bookings", "total_amount", "numeric", true},
		}

		for _, rc := range reqCols {
			meta, ok := existingColumns[rc.tbl][rc.col]
			if !ok {
				return fmt.Errorf("incompatible schema for baseline version %s: table '%s' missing required column '%s'", targetVersion, rc.tbl, rc.col)
			}
			if meta.dataType != rc.dataType {
				return fmt.Errorf("incompatible schema for baseline version %s: table '%s' column '%s' has type '%s', expected '%s'", targetVersion, rc.tbl, rc.col, meta.dataType, rc.dataType)
			}
			if rc.notNull && meta.isNullable {
				return fmt.Errorf("incompatible schema for baseline version %s: table '%s' column '%s' must not be nullable", targetVersion, rc.tbl, rc.col)
			}
		}

		// Required unique constraints
		if !existingUniques["users"]["phone"] {
			return fmt.Errorf("incompatible schema for baseline version %s: table 'users' is missing unique constraint on column(s) 'phone'", targetVersion)
		}
		if !existingUniques["priests"]["phone"] {
			return fmt.Errorf("incompatible schema for baseline version %s: table 'priests' is missing unique constraint on column(s) 'phone'", targetVersion)
		}

		// Exact foreign key relationships
		if !hasFK("bookings", "pooja_id", "pooja_catalog", "id", "r", "a") {
			return fmt.Errorf("incompatible schema for baseline version %s: missing foreign key constraint from bookings(pooja_id) to pooja_catalog(id) with ON DELETE RESTRICT", targetVersion)
		}
		if !hasFK("bookings", "user_id", "users", "id", "c") {
			return fmt.Errorf("incompatible schema for baseline version %s: missing foreign key constraint from bookings(user_id) to users(id) with ON DELETE CASCADE", targetVersion)
		}
		if !hasFK("bookings", "priest_id", "priests", "id", "n") {
			return fmt.Errorf("incompatible schema for baseline version %s: missing foreign key constraint from bookings(priest_id) to priests(id) with ON DELETE SET NULL", targetVersion)
		}
	}

	if targetVersion >= "000002" {
		if meta, ok := existingColumns["priests"]["services_offered"]; !ok || meta.dataType != "ARRAY" {
			return fmt.Errorf("incompatible schema for baseline version %s: table 'priests' missing or invalid type for column 'services_offered'", targetVersion)
		}
		if meta, ok := existingColumns["priests"]["eta_minutes"]; !ok || meta.dataType != "integer" {
			return fmt.Errorf("incompatible schema for baseline version %s: table 'priests' missing or invalid type for column 'eta_minutes'", targetVersion)
		}
		if meta, ok := existingColumns["pooja_catalog"]["description"]; !ok || meta.dataType != "text" {
			return fmt.Errorf("incompatible schema for baseline version %s: table 'pooja_catalog' missing or invalid type for column 'description'", targetVersion)
		}
		if meta, ok := existingColumns["pooja_catalog"]["icon"]; !ok || meta.dataType != "character varying" {
			return fmt.Errorf("incompatible schema for baseline version %s: table 'pooja_catalog' missing or invalid type for column 'icon'", targetVersion)
		}
	}

	if targetVersion >= "000003" {
		reqTables := []string{"priest_availability_slots", "priest_custom_services", "pooja_samagri_items", "booking_materials"}
		for _, tbl := range reqTables {
			if !existingTables[tbl] {
				return fmt.Errorf("incompatible schema for baseline version %s: missing table '%s'", targetVersion, tbl)
			}
		}

		if meta, ok := existingColumns["priest_availability_slots"]["slot_date"]; !ok || meta.dataType != "date" || meta.isNullable {
			return fmt.Errorf("incompatible schema for baseline version %s: priest_availability_slots.slot_date must be non-nullable date", targetVersion)
		}
		if meta, ok := existingColumns["priest_availability_slots"]["start_time"]; !ok || meta.dataType != "time without time zone" || meta.isNullable {
			return fmt.Errorf("incompatible schema for baseline version %s: priest_availability_slots.start_time must be non-nullable time", targetVersion)
		}
		if meta, ok := existingColumns["priest_custom_services"]["dakshina_fee"]; !ok || meta.dataType != "numeric" || meta.isNullable {
			return fmt.Errorf("incompatible schema for baseline version %s: priest_custom_services.dakshina_fee must be non-nullable numeric", targetVersion)
		}

		if !existingUniques["priest_availability_slots"]["priest_id,slot_date,start_time"] {
			return fmt.Errorf("incompatible schema for baseline version %s: priest_availability_slots missing unique constraint on (priest_id, slot_date, start_time)", targetVersion)
		}

		if !hasFK("priest_availability_slots", "priest_id", "priests", "id", "c") {
			return fmt.Errorf("incompatible schema for baseline version %s: missing foreign key from priest_availability_slots(priest_id) to priests(id)", targetVersion)
		}
		if !hasFK("priest_custom_services", "priest_id", "priests", "id", "c") {
			return fmt.Errorf("incompatible schema for baseline version %s: missing foreign key from priest_custom_services(priest_id) to priests(id)", targetVersion)
		}
	}

	if targetVersion >= "000004" {
		if meta, ok := existingColumns["users"]["password_hash"]; !ok || (meta.dataType != "character varying" && meta.dataType != "text") {
			return fmt.Errorf("incompatible schema for baseline version %s: users missing password_hash", targetVersion)
		}
		if meta, ok := existingColumns["priests"]["password_hash"]; !ok || (meta.dataType != "character varying" && meta.dataType != "text") {
			return fmt.Errorf("incompatible schema for baseline version %s: priests missing password_hash", targetVersion)
		}
		// Confirm booking personal/schedule details are strictly nullable
		for _, col := range []string{"scheduled_date", "slot_time", "devotee_gotram", "sankalpam_purpose"} {
			if meta, ok := existingColumns["bookings"][col]; !ok {
				return fmt.Errorf("incompatible schema for baseline version %s: bookings missing column '%s'", targetVersion, col)
			} else if !meta.isNullable {
				return fmt.Errorf("incompatible schema for baseline version %s: bookings.%s must be nullable", targetVersion, col)
			}
		}

		if !hasFK("bookings", "pooja_id", "pooja_catalog", "id", "r", "a") {
			return fmt.Errorf("incompatible schema for baseline version %s: bookings_pooja_id_fkey must be preserved", targetVersion)
		}
	}

	if targetVersion >= "000005" {
		if meta, ok := existingColumns["users"]["session_version"]; !ok || meta.dataType != "integer" || meta.isNullable {
			return fmt.Errorf("incompatible schema for baseline version %s: users.session_version must be non-nullable integer", targetVersion)
		}
		if meta, ok := existingColumns["priests"]["session_version"]; !ok || meta.dataType != "integer" || meta.isNullable {
			return fmt.Errorf("incompatible schema for baseline version %s: priests.session_version must be non-nullable integer", targetVersion)
		}
	}

	if targetVersion >= "000006" {
		if !existingTables["password_recovery_challenges"] {
			return fmt.Errorf("incompatible schema for baseline version %s: missing table 'password_recovery_challenges'", targetVersion)
		}
		if meta, ok := existingColumns["password_recovery_challenges"]["account_id"]; !ok || meta.dataType != "uuid" || meta.isNullable {
			return fmt.Errorf("incompatible schema for baseline version %s: password_recovery_challenges.account_id must be non-nullable uuid", targetVersion)
		}
		if meta, ok := existingColumns["password_recovery_challenges"]["code_hash"]; !ok || meta.dataType != "character varying" || meta.isNullable {
			return fmt.Errorf("incompatible schema for baseline version %s: password_recovery_challenges.code_hash must be non-nullable character varying", targetVersion)
		}
		if meta, ok := existingColumns["password_recovery_challenges"]["expires_at"]; !ok || meta.dataType != "timestamp with time zone" || meta.isNullable {
			return fmt.Errorf("incompatible schema for baseline version %s: password_recovery_challenges.expires_at must be non-nullable timestamp with time zone", targetVersion)
		}
	}

	if targetVersion >= "000007" {
		if meta, ok := existingColumns["users"]["email"]; !ok || meta.dataType != "character varying" || !meta.isNullable {
			return fmt.Errorf("incompatible schema for baseline version %s: users.email must be nullable character varying", targetVersion)
		}
		if meta, ok := existingColumns["users"]["email_verified"]; !ok || meta.dataType != "boolean" || meta.isNullable {
			return fmt.Errorf("incompatible schema for baseline version %s: users.email_verified must be non-nullable boolean", targetVersion)
		}
		if meta, ok := existingColumns["priests"]["email"]; !ok || meta.dataType != "character varying" || !meta.isNullable {
			return fmt.Errorf("incompatible schema for baseline version %s: priests.email must be nullable character varying", targetVersion)
		}
		if meta, ok := existingColumns["priests"]["email_verified"]; !ok || meta.dataType != "boolean" || meta.isNullable {
			return fmt.Errorf("incompatible schema for baseline version %s: priests.email_verified must be non-nullable boolean", targetVersion)
		}
		if meta, ok := existingColumns["password_recovery_challenges"]["email"]; !ok || meta.dataType != "character varying" || !meta.isNullable {
			return fmt.Errorf("incompatible schema for baseline version %s: password_recovery_challenges.email must be nullable character varying", targetVersion)
		}
	}

	return nil
}

// Baseline records existing migrations up to targetVersion without executing their DDL.
// It verifies that targetVersion is a known version, that already applied migrations match checksums,
// that no migrations higher than targetVersion are applied, and that the database schema is compatible
// with targetVersion before atomically recording the baselined migrations in schema_migrations.
func (m *Migrator) Baseline(ctx context.Context, targetVersion string) (int, error) {
	conn, unlock, err := m.AcquireLock(ctx)
	if err != nil {
		return 0, err
	}
	defer unlock()

	if err := m.ensureSchemaHistoryTable(ctx, conn); err != nil {
		return 0, err
	}

	files, err := m.LoadMigrationFiles()
	if err != nil {
		return 0, err
	}

	// 1. Validate that targetVersion exists in discovered migration files
	targetFound := false
	for _, f := range files {
		if f.Version == targetVersion {
			targetFound = true
			break
		}
	}
	if !targetFound {
		return 0, fmt.Errorf("invalid baseline version %s: version not found in migration files", targetVersion)
	}

	// 2. Validate existing history & checksums
	appliedMap, appliedList, err := m.getAppliedMigrations(ctx, conn)
	if err != nil {
		return 0, err
	}

	if err := m.VerifyChecksums(files, appliedMap); err != nil {
		return 0, fmt.Errorf("cannot baseline: %w", err)
	}

	for _, applied := range appliedList {
		if applied.Version > targetVersion {
			return 0, fmt.Errorf("cannot baseline to version %s: higher version %s is already recorded as applied", targetVersion, applied.Version)
		}
	}

	// 3. Validate schema compatibility against physical database using conn
	if err := m.validateSchemaCompatibility(ctx, targetVersion, conn); err != nil {
		return 0, fmt.Errorf("schema compatibility check failed for baseline: %w", err)
	}

	// 4. Atomically insert baseline records in a single transaction on conn
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin baseline transaction: %w", err)
	}
	defer tx.Rollback()

	count := 0
	for _, file := range files {
		if file.Version > targetVersion {
			break
		}
		if _, exists := appliedMap[file.Version]; exists {
			continue
		}

		query := `
			INSERT INTO schema_migrations (version, name, checksum, execution_time_ms)
			VALUES ($1, $2, $3, 0)
			ON CONFLICT (version) DO NOTHING
		`
		if _, err := tx.ExecContext(ctx, query, file.Version, file.Name, file.Checksum); err != nil {
			return 0, fmt.Errorf("failed to baseline migration %s: %w", file.Name, err)
		}
		count++
		log.Printf("🏷️ Baselined migration %s (version %s)", file.Name, file.Version)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit baseline transaction: %w", err)
	}

	return count, nil
}

// Seed executes development seed scripts on a locked connection
func (m *Migrator) Seed(ctx context.Context, seedFilePath string) error {
	conn, unlock, err := m.AcquireLock(ctx)
	if err != nil {
		return err
	}
	defer unlock()

	if seedFilePath == "" {
		seedFilePath = filepath.Join(m.seedsDir, "demo_seeds.sql")
	}

	data, err := os.ReadFile(seedFilePath)
	if err != nil {
		return fmt.Errorf("failed to read seed file %s: %w", seedFilePath, err)
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin seed transaction: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "SET LOCAL purohit.seed_demo = 'true'"); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to set demo flag: %w", err)
	}

	if _, err := tx.ExecContext(ctx, string(data)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to execute seed script %s: %w", seedFilePath, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit seed transaction: %w", err)
	}

	log.Printf("🌱 Successfully applied seed data from %s", seedFilePath)
	return nil
}

// VerifyReadiness checks whether the database is at or above the expected minimum version
// and verifies that there are no missing intermediate migrations or incompatible history.
func (m *Migrator) VerifyReadiness(ctx context.Context, expectedMinVersion string) error {
	var tableExists bool
	err := m.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = 'schema_migrations'
		)
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check schema_migrations existence: %w", err)
	}

	if !tableExists {
		return fmt.Errorf("schema_migrations table does not exist. Database has not been migrated yet")
	}

	files, err := m.LoadMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to load migration files for readiness check: %w", err)
	}

	appliedMap, appliedList, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to load applied migrations for readiness check: %w", err)
	}

	if len(appliedList) == 0 {
		return fmt.Errorf("no migrations recorded in schema_migrations. Database has not been migrated yet")
	}

	// 1. Verify that every intermediate migration up to expectedMinVersion has been applied (detect gaps)
	for _, file := range files {
		if file.Version <= expectedMinVersion {
			if _, applied := appliedMap[file.Version]; !applied {
				return fmt.Errorf("database schema readiness failure: intermediate migration %s (%s) is missing from schema_migrations history (history gap)", file.Version, file.Name)
			}
		}
	}

	// 2. Verify checksum immutability of all applied migrations
	if err := m.VerifyChecksums(files, appliedMap); err != nil {
		return fmt.Errorf("database schema readiness failure: %w", err)
	}

	// 3. Verify maximum applied version meets or exceeds expectedMinVersion
	latestVersion := appliedList[len(appliedList)-1].Version
	if latestVersion < expectedMinVersion {
		return fmt.Errorf("database schema version %s is below required minimum %s. Unapplied migrations exist", latestVersion, expectedMinVersion)
	}

	// 4. Verify physical schema compatibility (tables, columns, types, nullability, unique constraints, foreign keys)
	if err := m.ValidateSchemaCompatibility(ctx, expectedMinVersion); err != nil {
		return fmt.Errorf("database physical schema incompatibility for version %s: %w", expectedMinVersion, err)
	}

	return nil
}

// ResolveMigrationsDir resolves the path to the migrations directory reliably across different execution contexts
func ResolveMigrationsDir() (string, error) {
	if custom := os.Getenv("MIGRATIONS_DIR"); custom != "" {
		if info, err := os.Stat(custom); err == nil && info.IsDir() {
			return custom, nil
		}
	}
	candidates := []string{
		"backend/migrations",
		"migrations",
		"../migrations",
		"../../backend/migrations",
		"../../migrations",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			entries, err := os.ReadDir(c)
			if err == nil && len(entries) > 0 {
				for _, e := range entries {
					if strings.HasSuffix(e.Name(), ".sql") {
						return filepath.Abs(c)
					}
				}
			}
		}
	}
	return "", fmt.Errorf("could not locate migrations directory; please set MIGRATIONS_DIR environment variable")
}

// PrintStatusTable formats the status report into an ASCII table
func (r *StatusReport) PrintStatusTable(w io.Writer) {
	fmt.Fprintf(w, "====================================================================================\n")
	fmt.Fprintf(w, "PUJA VAAHINI DATABASE MIGRATION STATUS\n")
	fmt.Fprintf(w, "Current Applied Version: %s | Applied: %d | Pending: %d\n", r.CurrentVer, r.AppliedCount, r.PendingCount)
	fmt.Fprintf(w, "====================================================================================\n")
	fmt.Fprintf(w, "%-10s %-45s %-10s %-20s\n", "VERSION", "MIGRATION NAME", "STATUS", "APPLIED AT")
	fmt.Fprintf(w, "------------------------------------------------------------------------------------\n")
	for _, item := range r.Details {
		status := "PENDING"
		appliedAtStr := "-"
		if item.Applied {
			status = "APPLIED"
			if item.AppliedAt != nil {
				appliedAtStr = item.AppliedAt.Format("2006-01-02 15:04:05")
			}
		}
		fmt.Fprintf(w, "%-10s %-45s %-10s %-20s\n", item.Version, item.Name, status, appliedAtStr)
	}
	fmt.Fprintf(w, "====================================================================================\n")
}
