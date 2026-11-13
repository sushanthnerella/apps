package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/db"
)

func main() {
	var (
		pgConnStr string
		email     string
		password  string
	)

	flag.StringVar(&pgConnStr, "db", "", "PostgreSQL connection string")
	flag.StringVar(&email, "email", "", "Admin email address")
	flag.StringVar(&password, "password", "", "Admin password (minimum 8 characters)")
	flag.Parse()

	if pgConnStr == "" {
		pgConnStr = os.Getenv("PG_CONN_STR")
	}
	if pgConnStr == "" {
		pgConnStr = os.Getenv("DATABASE_URL")
	}
	if pgConnStr == "" {
		log.Fatal("❌ Error: PG_CONN_STR or DATABASE_URL is required")
	}

	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		log.Fatal("❌ Error: valid -email is required")
	}
	if err := auth.ValidatePassword(password); err != nil {
		log.Fatalf("❌ Error: invalid password: %v", err)
	}

	sqlDB, err := sql.Open("postgres", pgConnStr)
	if err != nil {
		log.Fatalf("❌ Database connection error: %v", err)
	}
	defer sqlDB.Close()

	storage := &db.Storage{DB: sqlDB}
	admin, err := storage.CreateAdminUser(email, password)
	if err != nil {
		log.Fatalf("❌ Failed to create admin user: %v", err)
	}

	fmt.Printf("✅ Successfully created admin user: %s (ID: %s)\n", admin.Email, admin.ID)
}
