package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/db/migrate"
)

// CreateIsolatedAuthTestDBForTest creates an isolated PostgreSQL test database with all migrations applied.
func CreateIsolatedAuthTestDBForTest(t *testing.T) (*Storage, *sql.DB, string) {
	auth.SetJWTSecretForTesting("purohit_platform_test_jwt_secret_key_32bytes_long!")

	rootConnStr := os.Getenv("TEST_PG_ROOT_CONN_STR")
	if rootConnStr == "" {
		rootConnStr = "postgres://postgres:123456@localhost:5432/postgres?sslmode=disable"
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
	testDBName := fmt.Sprintf("pv_gw_test_%d_%s", time.Now().Unix(), hex.EncodeToString(randBytes))

	_, err = rootDB.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, testDBName))
	if err != nil {
		rootDB.Close()
		t.Fatalf("failed to create isolated test database %s: %v", testDBName, err)
	}

	testConnStr := fmt.Sprintf("postgres://postgres:123456@localhost:5432/%s?sslmode=disable", testDBName)
	testDB, err := sql.Open("postgres", testConnStr)
	if err != nil {
		_, _ = rootDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, testDBName))
		rootDB.Close()
		t.Fatalf("failed to connect to test db: %v", err)
	}

	candidates := []string{
		"migrations",
		"backend/migrations",
		"../migrations",
		"../../migrations",
	}
	var migDir string
	for _, c := range candidates {
		if stat, err := os.Stat(c); err == nil && stat.IsDir() {
			migDir, _ = filepath.Abs(c)
			break
		}
	}

	migrator := migrate.New(testDB, migDir, "")
	if _, err := migrator.Up(context.Background(), false); err != nil {
		testDB.Close()
		_, _ = rootDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, testDBName))
		rootDB.Close()
		t.Fatalf("failed to migrate test db: %v", err)
	}

	storage := &Storage{DB: testDB, Redis: nil}

	t.Cleanup(func() {
		testDB.Close()
		_, _ = rootDB.Exec(fmt.Sprintf(`
			SELECT pg_terminate_backend(pid) FROM pg_stat_activity 
			WHERE datname = '%s' AND pid <> pg_backend_pid();
		`, testDBName))
		_, _ = rootDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, testDBName))
		rootDB.Close()
	})

	return storage, testDB, testConnStr
}
