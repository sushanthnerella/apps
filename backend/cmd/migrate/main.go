package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/db/migrate"
)

func findMigrationsDir(userSpecified string) string {
	if userSpecified != "" {
		return userSpecified
	}
	candidates := []string{
		"migrations",
		"./migrations",
		"../migrations",
		"backend/migrations",
		"../../migrations",
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "migrations"
}

func findSeedsDir(userSpecified string) string {
	if userSpecified != "" {
		return userSpecified
	}
	candidates := []string{
		"seeds",
		"./seeds",
		"../seeds",
		"backend/seeds",
		"../../seeds",
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "seeds"
}

func main() {
	var (
		pgConnStr    string
		migrationDir string
		seedsDir     string
		seedDemo     bool
		targetVer    string
		seedFile     string
		adminEmail   string
		adminPass    string
	)

	flag.StringVar(&pgConnStr, "db", "", "PostgreSQL connection string")
	flag.StringVar(&migrationDir, "dir", "", "Path to migrations directory")
	flag.StringVar(&seedsDir, "seeds-dir", "", "Path to seeds directory")
	flag.BoolVar(&seedDemo, "seed-demo", false, "Include demo seeds in fresh migration run (dev only)")
	flag.StringVar(&targetVer, "version", "000003", "Target version for baseline command")
	flag.StringVar(&seedFile, "file", "", "Explicit seed SQL file to execute")
	flag.StringVar(&adminEmail, "email", "", "Admin email for create-admin")
	flag.StringVar(&adminPass, "password", "", "Admin password for create-admin")
	flag.Parse()

	args := flag.Args()
	command := "up"
	if len(args) > 0 {
		command = args[0]
	}

	if pgConnStr == "" {
		pgConnStr = os.Getenv("PG_CONN_STR")
	}
	if pgConnStr == "" {
		pgConnStr = os.Getenv("DATABASE_URL")
	}
	if pgConnStr == "" {
		log.Fatal("PG_CONN_STR or DATABASE_URL is required")
	}

	dir := findMigrationsDir(migrationDir)
	sDir := findSeedsDir(seedsDir)

	sqlDB, err := sql.Open("postgres", pgConnStr)
	if err != nil {
		log.Fatalf("❌ Database connection error: %v", err)
	}
	defer sqlDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		log.Fatalf("❌ Database ping failed: %v", err)
	}

	migrator := migrate.New(sqlDB, dir, sDir)

	switch command {
	case "up":
		fmt.Printf("🕉️  Running Puja Vaahini migrations from: %s\n", dir)
		applied, err := migrator.Up(ctx, seedDemo)
		if err != nil {
			log.Fatalf("❌ Migration failed: %v", err)
		}
		if applied == 0 {
			fmt.Println("✨ Database schema is already up to date. No pending migrations.")
		} else {
			fmt.Printf("🚀 Successfully applied %d migration(s).\n", applied)
		}

	case "status":
		report, err := migrator.Status(ctx)
		if err != nil {
			log.Fatalf("❌ Failed to query migration status: %v", err)
		}
		report.PrintStatusTable(os.Stdout)

	case "baseline", "adopt":
		fmt.Printf("🏷️  Baselining existing schema up to version %s from: %s\n", targetVer, dir)
		count, err := migrator.Baseline(ctx, targetVer)
		if err != nil {
			log.Fatalf("❌ Baseline failed: %v", err)
		}
		fmt.Printf("✅ Recorded %d migration(s) as baseline history.\n", count)

	case "seed":
		fmt.Println("🌱 Applying development seed data...")
		if err := migrator.Seed(ctx, seedFile); err != nil {
			log.Fatalf("❌ Seeding failed: %v", err)
		}
		fmt.Println("✅ Development seed data applied successfully.")

	case "check":
		expectedMin := "000004"
		if len(args) > 1 {
			expectedMin = args[1]
		}
		if err := migrator.VerifyReadiness(ctx, expectedMin); err != nil {
			log.Fatalf("❌ Schema readiness check failed: %v", err)
		}
		fmt.Printf("✅ Schema is ready at or above version %s.\n", expectedMin)

	case "create-admin":
		if adminEmail == "" || adminPass == "" {
			log.Fatalf("❌ Usage: migrate -email <admin@domain> -password <secret> create-admin")
		}
		storage := &db.Storage{DB: sqlDB}
		admin, err := storage.CreateAdminUser(adminEmail, adminPass)
		if err != nil {
			log.Fatalf("❌ Failed to create admin user: %v", err)
		}
		fmt.Printf("✅ Admin user created successfully:\n  ID: %s\n  Email: %s\n  Role: %s\n", admin.ID, admin.Email, admin.Role)

	default:
		fmt.Printf("Usage: migrate [up|status|baseline|seed|check|create-admin] [flags]\n")
		os.Exit(1)
	}
}
