package db

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/db/migrate"
	"github.com/purohit-platform/backend/internal/lifecycle"
	"github.com/purohit-platform/backend/internal/models"
	"github.com/redis/go-redis/v9"
)

// Re-export lifecycle errors for backward compatibility and clean DB-level callers
var (
	ErrInvalidTransition  = lifecycle.ErrInvalidTransition
	ErrRepeatedTransition = lifecycle.ErrRepeatedTransition
	ErrSlotUnavailable    = lifecycle.ErrSlotUnavailable
)

type Storage struct {
	DB    *sql.DB
	Redis *redis.Client
}

// SanitizeRedisEndpoint strips sensitive user/password credentials from connection strings for safe logging
func SanitizeRedisEndpoint(raw string) string {
	if strings.HasPrefix(raw, "redis://") || strings.HasPrefix(raw, "rediss://") {
		if u, err := url.Parse(raw); err == nil {
			return u.Redacted()
		}
	}
	return raw
}

func InitStorage(pgConnStr, redisAddr string) (*Storage, error) {
	db, err := sql.Open("postgres", pgConnStr)
	if err != nil {
		return nil, fmt.Errorf("postgres connect error: %w", err)
	}

	// Bound connections per instance so a small Railway database is not overwhelmed.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("postgres ping error: %w", err)
	}

	var redisOpts *redis.Options
	if strings.HasPrefix(redisAddr, "redis://") || strings.HasPrefix(redisAddr, "rediss://") {
		opts, err := redis.ParseURL(redisAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
		}
		// If rediss:// is specified, enforce TLS config
		if strings.HasPrefix(redisAddr, "rediss://") && opts.TLSConfig == nil {
			opts.TLSConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
		}
		redisOpts = opts
	} else {
		redisOpts = &redis.Options{
			Addr:     redisAddr,
			Password: os.Getenv("REDIS_PASSWORD"),
		}
	}

	rdb := redis.NewClient(redisOpts)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sanitized := SanitizeRedisEndpoint(redisAddr)
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️ Redis ping warning for %s: %v (continuing with DB fallback)", sanitized, err)
	} else {
		log.Printf("✅ Connected to Redis at %s", sanitized)
	}

	log.Printf("✅ Connected to PostgreSQL database")
	return &Storage{DB: db, Redis: rdb}, nil
}

// PingDB checks PostgreSQL connectivity with context timeout
func (s *Storage) PingDB(ctx context.Context) error {
	if s.DB == nil {
		return errors.New("database not initialized")
	}
	return s.DB.PingContext(ctx)
}

// PingRedis checks Redis connectivity with context timeout
func (s *Storage) PingRedis(ctx context.Context) error {
	if s.Redis == nil {
		return errors.New("redis not initialized")
	}
	return s.Redis.Ping(ctx).Err()
}

// VerifySchemaReadiness ensures the database has applied all required migrations,
// has no intermediate gaps or checksum mismatches, and conforms to the physical schema requirements.
func (s *Storage) VerifySchemaReadiness(minVersion string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	migrationsDir, err := migrate.ResolveMigrationsDir()
	if err != nil {
		return fmt.Errorf("schema readiness check failed to locate migrations: %w", err)
	}

	migrator := migrate.New(s.DB, migrationsDir, "")
	if err := migrator.VerifyReadiness(ctx, minVersion); err != nil {
		return err
	}

	log.Printf("✅ Database schema readiness verified at version %s (all intermediate migrations present, checksums valid, physical schema compatible)", minVersion)
	return nil
}

// Login authenticates with password and verifies account credentials
func (s *Storage) Login(req models.LoginRequest) (*models.AuthResponse, error) {
	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role != "devotee" && role != "priest" {
		return nil, fmt.Errorf("role is required and must be either 'devotee' or 'priest'")
	}

	phone := strings.TrimSpace(req.Phone)
	if phone == "" {
		return nil, fmt.Errorf("phone number is required")
	}

	if strings.TrimSpace(req.Password) == "" {
		return nil, fmt.Errorf("password is required. OTP authentication is disabled for pilot security")
	}

	// Normalize phone number and generate lookup variants for database query
	canonicalPhone, err := auth.NormalizePhone(phone)
	if err != nil {
		auth.EqualizeTiming(req.Password)
		return nil, auth.ErrInvalidCredentials
	}

	variants, _ := auth.PhoneLookupVariants(phone)

	var userProfile models.UserProfile
	var storedHash string
	var sessionVersion int

	if role == "priest" {
		query := `
			SELECT id, phone, full_name, sampradaya, veda_shakha, 'Hyderabad', consent_agreed, created_at, COALESCE(password_hash, ''), COALESCE(session_version, 1)
			FROM priests
			WHERE phone = ANY($1)
		`
		rows, err := s.DB.Query(query, pq.Array(variants))
		if err != nil {
			return nil, fmt.Errorf("database query error: %w", err)
		}
		defer rows.Close()

		var matchCount int
		for rows.Next() {
			matchCount++
			if matchCount > 1 {
				return nil, fmt.Errorf("ambiguous account match: multiple priest accounts found for phone number %s", auth.MaskPhone(phone))
			}
			err := rows.Scan(
				&userProfile.ID, &userProfile.Phone, &userProfile.FullName,
				&userProfile.Gotram, &userProfile.Nakshatram, &userProfile.City,
				&userProfile.ConsentAgreed, &userProfile.CreatedAt, &storedHash, &sessionVersion,
			)
			if err != nil {
				return nil, fmt.Errorf("account scan error: %w", err)
			}
		}
		if matchCount == 0 {
			auth.EqualizeTiming(req.Password)
			return nil, auth.ErrInvalidCredentials
		}
		userProfile.Role = "priest"
	} else {
		query := `
			SELECT id, phone, full_name, COALESCE(gotram, ''), COALESCE(nakshatram, ''), COALESCE(rashi, ''), city, consent_agreed, created_at, COALESCE(password_hash, ''), COALESCE(session_version, 1)
			FROM users
			WHERE phone = ANY($1)
		`
		rows, err := s.DB.Query(query, pq.Array(variants))
		if err != nil {
			return nil, fmt.Errorf("database query error: %w", err)
		}
		defer rows.Close()

		var matchCount int
		for rows.Next() {
			matchCount++
			if matchCount > 1 {
				return nil, fmt.Errorf("ambiguous account match: multiple devotee accounts found for phone number %s", auth.MaskPhone(phone))
			}
			err := rows.Scan(
				&userProfile.ID, &userProfile.Phone, &userProfile.FullName,
				&userProfile.Gotram, &userProfile.Nakshatram, &userProfile.Rashi,
				&userProfile.City, &userProfile.ConsentAgreed, &userProfile.CreatedAt, &storedHash, &sessionVersion,
			)
			if err != nil {
				return nil, fmt.Errorf("account scan error: %w", err)
			}
		}
		if matchCount == 0 {
			auth.EqualizeTiming(req.Password)
			return nil, auth.ErrInvalidCredentials
		}
		userProfile.Role = "devotee"
	}

	// Check if account has legacy/missing/malformed password
	match, needsRecovery := auth.VerifyPassword(storedHash, req.Password)
	if needsRecovery {
		return nil, auth.ErrRecoveryRequired
	}
	if !match {
		return nil, auth.ErrInvalidCredentials
	}

	token, err := auth.GenerateToken(userProfile.ID, canonicalPhone, userProfile.Role, userProfile.FullName, sessionVersion)
	if err != nil {
		return nil, fmt.Errorf("JWT generation error: %w", err)
	}
	return &models.AuthResponse{
		Status:      "SUCCESS",
		AccessToken: token,
		TokenType:   "Bearer",
		User:        userProfile,
		Message:     "Authenticated successfully with password.",
	}, nil
}

// Note: Password recovery and email verification methods are modularized in internal/db/recovery.go

// LoginWithOTP is disabled for pilot security to prevent arbitrary OTP account claiming
func (s *Storage) LoginWithOTP(phone, otp, role string) (*models.AuthResponse, error) {
	return nil, fmt.Errorf("direct OTP login is currently disabled for security. Please authenticate using your password.")
}

// GetUserProfile retrieves profile by user ID and role
func (s *Storage) GetUserProfile(userID, role string) (*models.UserProfile, error) {
	var profile models.UserProfile
	if role == "priest" {
		query := `SELECT id, phone, full_name, sampradaya, veda_shakha, 'Hyderabad', consent_agreed, created_at, COALESCE(email, ''), email_verified FROM priests WHERE id = $1`
		err := s.DB.QueryRow(query, userID).Scan(
			&profile.ID, &profile.Phone, &profile.FullName,
			&profile.Gotram, &profile.Nakshatram, &profile.City,
			&profile.ConsentAgreed, &profile.CreatedAt,
			&profile.Email, &profile.EmailVerified,
		)
		profile.Role = "priest"
		return &profile, err
	}

	query := `SELECT id, phone, full_name, COALESCE(gotram, ''), COALESCE(nakshatram, ''), COALESCE(rashi, ''), city, consent_agreed, created_at, COALESCE(email, ''), email_verified FROM users WHERE id = $1`
	err := s.DB.QueryRow(query, userID).Scan(
		&profile.ID, &profile.Phone, &profile.FullName,
		&profile.Gotram, &profile.Nakshatram, &profile.Rashi,
		&profile.City, &profile.ConsentAgreed, &profile.CreatedAt,
		&profile.Email, &profile.EmailVerified,
	)
	profile.Role = "devotee"
	return &profile, err
}

// GetCatalog loads pooja catalog packages with optional search, category and language filters
func (s *Storage) GetCatalog(searchQuery, categoryFilter string, lang string) ([]map[string]interface{}, error) {
	query := `
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
		WHERE 1=1
	`
	var args []interface{}
	idx := 1

	if searchQuery != "" {
		query += fmt.Sprintf(" AND (title ILIKE $%d OR title_te ILIKE $%d OR title_hi ILIKE $%d OR description ILIKE $%d)", idx, idx, idx, idx)
		args = append(args, "%"+searchQuery+"%")
		idx++
	}

	if categoryFilter != "" && categoryFilter != "All" {
		query += fmt.Sprintf(" AND category ILIKE $%d", idx)
		args = append(args, "%"+categoryFilter+"%")
		idx++
	}

	query += " ORDER BY base_price ASC"

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var catalog []map[string]interface{}
	for rows.Next() {
		var id, rawTitle, titleEn, titleTe, titleHi, category, icon, rawDesc, descEn, descTe, descHi string
		var basePrice, samagriKitPrice float64
		var durationMinutes int
		var isOnlineEligible bool
		if err := rows.Scan(
			&id, &rawTitle, &titleEn, &titleTe, &titleHi,
			&category, &basePrice, &samagriKitPrice, &durationMinutes, &isOnlineEligible, &icon,
			&rawDesc, &descEn, &descTe, &descHi,
		); err != nil {
			return nil, err
		}

		displayTitle := rawTitle
		displayDesc := rawDesc
		switch strings.ToLower(lang) {
		case "te", "telugu":
			if titleTe != "" {
				displayTitle = titleTe
			}
			if descTe != "" {
				displayDesc = descTe
			}
		case "hi", "hindi":
			if titleHi != "" {
				displayTitle = titleHi
			}
			if descHi != "" {
				displayDesc = descHi
			}
		case "en", "english":
			if titleEn != "" {
				displayTitle = titleEn
			}
			if descEn != "" {
				displayDesc = descEn
			}
		}

		catalog = append(catalog, map[string]interface{}{
			"id":                 id,
			"title":              displayTitle,
			"title_en":           titleEn,
			"title_te":           titleTe,
			"title_hi":           titleHi,
			"category":           category,
			"base_price":         basePrice,
			"samagri_kit_price":  samagriKitPrice,
			"duration_minutes":   durationMinutes,
			"is_online_eligible": isOnlineEligible,
			"icon":               icon,
			"description":        displayDesc,
			"description_en":     descEn,
			"description_te":     descTe,
			"description_hi":     descHi,
		})
	}

	// Also fetch active priest custom services (e.g., "shani japam") and append to catalog
	csQuery := `
		SELECT cs.id, cs.title, 
		       COALESCE(cs.title_en, cs.title) as t_en, 
		       COALESCE(cs.title_te, cs.title) as t_te, 
		       COALESCE(cs.title_hi, cs.title) as t_hi, 
		       COALESCE(NULLIF(cs.category, ''), 'Purohit Special') as category, 
		       cs.dakshina_fee, COALESCE(cs.samagri_kit_price, 0), cs.duration_minutes, cs.is_online_eligible, '🪔' as icon, 
		       COALESCE(NULLIF(cs.description, ''), 'Custom ritual by registered Vedic Purohit') as description, 
		       COALESCE(cs.description_en, cs.description, '') as d_en, 
		       COALESCE(cs.description_te, cs.description, '') as d_te, 
		       COALESCE(cs.description_hi, cs.description, '') as d_hi,
		       COALESCE(p.full_name, 'Vedic Purohit') as priest_name
		FROM priest_custom_services cs
		LEFT JOIN priests p ON cs.priest_id = p.id
		WHERE cs.is_active = TRUE
	`
	var csArgs []interface{}
	csIdx := 1
	if searchQuery != "" {
		csQuery += fmt.Sprintf(" AND (cs.title ILIKE $%d OR cs.description ILIKE $%d)", csIdx, csIdx)
		csArgs = append(csArgs, "%"+searchQuery+"%")
		csIdx++
	}
	if categoryFilter != "" && categoryFilter != "All" {
		csQuery += fmt.Sprintf(" AND cs.category ILIKE $%d", csIdx)
		csArgs = append(csArgs, "%"+categoryFilter+"%")
		csIdx++
	}
	csQuery += " ORDER BY cs.created_at DESC"

	if csRows, err := s.DB.Query(csQuery, csArgs...); err == nil {
		defer csRows.Close()
		for csRows.Next() {
			var id, rawTitle, titleEn, titleTe, titleHi, category, icon, rawDesc, descEn, descTe, descHi, priestName string
			var dakshinaFee, samagriKitPrice float64
			var durationMinutes int
			var isOnlineEligible bool
			if err := csRows.Scan(
				&id, &rawTitle, &titleEn, &titleTe, &titleHi,
				&category, &dakshinaFee, &samagriKitPrice, &durationMinutes, &isOnlineEligible, &icon,
				&rawDesc, &descEn, &descTe, &descHi, &priestName,
			); err == nil {
				displayTitle := rawTitle
				displayDesc := rawDesc
				if priestName != "" {
					displayDesc = fmt.Sprintf("Offered by %s. %s", priestName, rawDesc)
				}
				catalog = append(catalog, map[string]interface{}{
					"id":                 id,
					"title":              displayTitle,
					"title_en":           titleEn,
					"title_te":           titleTe,
					"title_hi":           titleHi,
					"category":           category,
					"base_price":         dakshinaFee,
					"samagri_kit_price":  samagriKitPrice,
					"duration_minutes":   durationMinutes,
					"is_online_eligible": isOnlineEligible,
					"icon":               icon,
					"description":        displayDesc,
					"description_en":     descEn,
					"description_te":     descTe,
					"description_hi":     descHi,
				})
			}
		}
	}
	return catalog, nil
}

// RegisterDevotee securely creates a new devotee profile, hashes the password, and prevents account overwrite
func (s *Storage) RegisterDevotee(req models.DevoteeRegistrationRequest) (string, error) {
	fullName := strings.TrimSpace(req.FullName)
	if fullName == "" {
		return "", fmt.Errorf("full name is required")
	}

	if !req.ConsentAgreed {
		return "", fmt.Errorf("devotee terms and ritual consent are mandatory")
	}

	if err := auth.ValidatePassword(req.Password); err != nil {
		return "", err
	}

	pwdHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return "", err
	}

	canonicalPhone, err := auth.NormalizePhone(req.Phone)
	if err != nil {
		return "", err
	}

	variants, _ := auth.PhoneLookupVariants(req.Phone)

	// Check for duplicates in users (duplicate registration cannot overwrite profile or password)
	var existingUserCount int
	err = s.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE phone = ANY($1)`, pq.Array(variants)).Scan(&existingUserCount)
	if err != nil {
		return "", fmt.Errorf("failed to check existing user accounts: %w", err)
	}
	if existingUserCount > 0 {
		return "", fmt.Errorf("an account with phone number %s already exists; please log in", auth.MaskPhone(req.Phone))
	}

	// Check for collision in priests
	var existingPriestCount int
	err = s.DB.QueryRow(`SELECT COUNT(*) FROM priests WHERE phone = ANY($1)`, pq.Array(variants)).Scan(&existingPriestCount)
	if err != nil {
		return "", fmt.Errorf("failed to check existing priest accounts: %w", err)
	}
	if existingPriestCount > 0 {
		return "", fmt.Errorf("phone number %s is already registered as a priest; cannot create devotee profile", auth.MaskPhone(req.Phone))
	}

	city := strings.TrimSpace(req.City)
	if city == "" {
		city = "Hyderabad"
	}

	var id string
	query := `
		INSERT INTO users (phone, full_name, password_hash, gotram, nakshatram, rashi, city, consent_agreed, consent_timestamp, session_version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP, 1)
		RETURNING id;
	`
	err = s.DB.QueryRow(query, canonicalPhone, fullName, pwdHash, req.Gotram, req.Nakshatram, req.Rashi, city, req.ConsentAgreed).Scan(&id)
	return id, err
}

// RegisterPriest securely creates a new priest profile, hashes the password, and keeps duty/approval separate
func (s *Storage) RegisterPriest(req models.PriestRegistrationRequest) (string, error) {
	fullName := strings.TrimSpace(req.FullName)
	if fullName == "" {
		return "", fmt.Errorf("full name is required")
	}

	sampradaya := strings.TrimSpace(req.Sampradaya)
	if sampradaya == "" {
		return "", fmt.Errorf("sampradaya is required")
	}

	if !req.ConsentAgreed {
		return "", fmt.Errorf("purohit code of conduct consent is mandatory")
	}

	if err := auth.ValidatePassword(req.Password); err != nil {
		return "", err
	}

	pwdHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return "", err
	}

	canonicalPhone, err := auth.NormalizePhone(req.Phone)
	if err != nil {
		return "", err
	}

	variants, _ := auth.PhoneLookupVariants(req.Phone)

	// Check for duplicates in priests
	var existingPriestCount int
	err = s.DB.QueryRow(`SELECT COUNT(*) FROM priests WHERE phone = ANY($1)`, pq.Array(variants)).Scan(&existingPriestCount)
	if err != nil {
		return "", fmt.Errorf("failed to check existing priest accounts: %w", err)
	}
	if existingPriestCount > 0 {
		return "", fmt.Errorf("a priest profile with phone number %s already exists; please log in", auth.MaskPhone(req.Phone))
	}

	// Check for collision in users
	var existingUserCount int
	err = s.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE phone = ANY($1)`, pq.Array(variants)).Scan(&existingUserCount)
	if err != nil {
		return "", fmt.Errorf("failed to check existing user accounts: %w", err)
	}
	if existingUserCount > 0 {
		return "", fmt.Errorf("phone number %s is already registered as a devotee; cannot create priest profile", auth.MaskPhone(req.Phone))
	}

	vedaShakha := strings.TrimSpace(req.VedaShakha)
	if vedaShakha == "" {
		vedaShakha = "Rigveda"
	}
	expYears := req.ExperienceYears
	if expYears < 0 {
		expYears = 5
	}

	var id string
	query := `
		INSERT INTO priests (
			phone, full_name, password_hash, sampradaya, veda_shakha, experience_years, 
			latitude, longitude, consent_agreed, is_online, services_offered, consent_timestamp, session_version
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, false, 
			ARRAY['Gruhapravesam (House Warming)', 'Sri Satyanarayana Swamy Vratam', 'Maha Ganapathi Homa / Yagam', 'Navagraha Shanti Homa', 'Mrityunjaya Homa / Shanti', 'Namakaranam & Ayushya Homa'], 
			CURRENT_TIMESTAMP, 1
		)
		RETURNING id;
	`
	err = s.DB.QueryRow(query, canonicalPhone, fullName, pwdHash, sampradaya, vedaShakha, expYears, req.Latitude, req.Longitude, req.ConsentAgreed).Scan(&id)
	if err != nil {
		return "", err
	}

	if s.Redis != nil {
		ctx := context.Background()
		s.Redis.GeoAdd(ctx, "priests:geo", &redis.GeoLocation{
			Name:      id,
			Latitude:  req.Latitude,
			Longitude: req.Longitude,
		})
	}

	return id, nil
}

// RevokeUserSessions increments session_version in database, immediately invalidating active tokens
func (s *Storage) RevokeUserSessions(userID, role string) error {
	table := "users"
	if role == "priest" {
		table = "priests"
	}
	query := fmt.Sprintf("UPDATE %s SET session_version = session_version + 1 WHERE id = $1", table)
	res, err := s.DB.Exec(query, userID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("account not found with id %s", userID)
	}
	return nil
}

// GetUserSessionVersion retrieves the current session_version for an account
func (s *Storage) GetUserSessionVersion(userID, role string) (int, error) {
	table := "users"
	if role == "priest" {
		table = "priests"
	} else if role == "admin" {
		table = "admin_users"
	}
	var version int
	var query string
	if role == "admin" {
		query = fmt.Sprintf("SELECT COALESCE(session_version, 1) FROM %s WHERE id = $1 AND disabled_at IS NULL", table)
	} else {
		query = fmt.Sprintf("SELECT COALESCE(session_version, 1) FROM %s WHERE id = $1", table)
	}
	err := s.DB.QueryRow(query, userID).Scan(&version)
	return version, err
}

// CreateBooking creates a new booking in HELD_IN_ESCROW state with a generated 4-digit OTP,
// binds the requested slot if slot_id is provided, populates booking_materials checklist,
// sets up the chat room, and posts the initial system notification inside an atomic transaction.
func (s *Storage) CreateBooking(req models.BookingCreateRequest) (string, string, error) {
	randomOTP, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", "", fmt.Errorf("generate completion code: %w", err)
	}
	otp := fmt.Sprintf("%04d", randomOTP.Int64())
	var bookingID string

	var validUser bool
	if req.UserID != "" {
		var uid string
		if err := s.DB.QueryRow(`SELECT id::text FROM users WHERE id::text = $1`, req.UserID).Scan(&uid); err == nil && uid != "" {
			validUser = true
			req.UserID = uid
		}
	}
	if !validUser {
		return "", "", fmt.Errorf("invalid or non-existent user_id: %s", req.UserID)
	}

	if req.EventAddress == "" {
		req.EventAddress = "Plot 42, Sri Rama Nagar, Madhav Nagar, Hyderabad (Landmark: Near Sri Rama Temple)"
	}
	if req.BookingType == "" {
		req.BookingType = "INSTANT_DISPATCH"
	}

	ctx := context.Background()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// If a specific slot_id is requested, verify hold and bind slot atomically within tx
	if req.SlotID != "" {
		slotDate, slotTime, slotErr := s.VerifyAndConsumeSlotHold(ctx, tx, req.SlotID, req.UserID, "", req.PriestID)
		if slotErr != nil {
			return "", "", slotErr
		}
		if req.ScheduledDate == "" {
			req.ScheduledDate = slotDate
		}
		if req.SlotTime == "" {
			req.SlotTime = slotTime
		}
	}

	// Nullable personal and schedule fields are preserved as provided without fabricating false defaults
	var schedDate, slotTime, gotram, nakshatram, rashi, purpose *string
	if req.ScheduledDate != "" {
		schedDate = &req.ScheduledDate
	}
	if req.SlotTime != "" {
		slotTime = &req.SlotTime
	}
	if req.DevoteeGotram != "" {
		gotram = &req.DevoteeGotram
	}
	if req.DevoteeNakshatram != "" {
		nakshatram = &req.DevoteeNakshatram
	}
	if req.DevoteeRashi != "" {
		rashi = &req.DevoteeRashi
	}
	if req.SankalpamPurpose != "" {
		purpose = &req.SankalpamPurpose
	}

	var priestIDVal *string
	if req.PriestID != "" {
		priestIDVal = &req.PriestID
	}
	var poojaIDVal *string
	if req.PoojaID != "" {
		poojaIDVal = &req.PoojaID
	}

	if req.TotalAmountMinor == 0 && req.TotalAmount > 0 {
		req.TotalAmountMinor = int64(math.Round(req.TotalAmount * 100))
	}
	if req.TotalAmountMinor > 0 && req.TotalAmount == 0 {
		req.TotalAmount = float64(req.TotalAmountMinor) / 100.0
	}

	query := `
		INSERT INTO bookings (
			user_id, priest_id, pooja_id, booking_type, status, escrow_status, 
			total_amount, total_amount_minor, include_samagri_kit, completion_otp, event_address, latitude, longitude,
			scheduled_date, slot_time, devotee_gotram, devotee_nakshatram, devotee_rashi, sankalpam_purpose
		)
		VALUES ($1, $2, $3, $4, 'REQUESTED', 'HELD_IN_ESCROW', $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id;
	`
	err = tx.QueryRowContext(ctx, query,
		req.UserID, priestIDVal, poojaIDVal, req.BookingType,
		req.TotalAmount, req.TotalAmountMinor, req.IncludeSamagriKit, otp, req.EventAddress, req.Latitude, req.Longitude,
		schedDate, slotTime, gotram, nakshatram, rashi, purpose,
	).Scan(&bookingID)
	if err != nil {
		_ = tx.Rollback()
		return "", "", err
	}

	// If slot was bound, associate booked_booking_id to the created bookingID
	if req.SlotID != "" {
		updateSlotBooking := `UPDATE priest_availability_slots SET booked_booking_id = $1::uuid WHERE id = $2::uuid`
		if _, err := tx.ExecContext(ctx, updateSlotBooking, bookingID, req.SlotID); err != nil {
			_ = tx.Rollback()
			return "", "", fmt.Errorf("failed to link slot to booking: %w", err)
		}
	}

	// 1. Copy standard items into booking_materials inside same transaction
	materials, err := s.GetPoojaSamagriTx(ctx, tx, req.PoojaID)
	if err != nil {
		_ = tx.Rollback()
		return "", "", fmt.Errorf("failed to fetch samagri items: %w", err)
	}
	for _, m := range materials {
		isPriestItem := (m.Category == "Sacred Ritual Samagri")
		_, err := tx.ExecContext(ctx, `
			INSERT INTO booking_materials (booking_id, item_name, category, quantity, is_included_by_priest)
			VALUES ($1, $2, $3, $4, $5);
		`, bookingID, m.ItemName, m.Category, m.Quantity, isPriestItem && req.IncludeSamagriKit)
		if err != nil {
			_ = tx.Rollback()
			return "", "", fmt.Errorf("failed to insert booking material: %w", err)
		}
	}

	// 2. Set up Chat Room inside same transaction
	roomID, err := s.GetOrCreateChatRoomTx(ctx, tx, bookingID, req.UserID, req.PriestID)
	if err != nil {
		_ = tx.Rollback()
		return "", "", fmt.Errorf("failed to create chat room: %w", err)
	}

	// 3. Post Initial System Message inside same transaction
	samagriNotice := "Brought by Purohit"
	if !req.IncludeSamagriKit {
		samagriNotice = "Arranged by Devotee"
	}
	welcomeMsg := fmt.Sprintf(
		"🕉️ Ritual Booking Requested!\n📅 Scheduled: %s | %s\n✨ Vedic Sankalpam: %s Gotram • %s Nakshatram (%s)\n📍 Event Address: %s\n🪔 Sacred Samagri Kit: %s\n🏡 Devotee: Please have standard household items (Brass plates, Diyas, Ghee, Mats, Rice, Fruits) ready. You can now chat directly here to coordinate directions and arrival.",
		req.ScheduledDate, req.SlotTime, req.DevoteeGotram, req.DevoteeNakshatram, req.DevoteeRashi, req.EventAddress, samagriNotice,
	)
	_, err = s.SaveChatMessageTx(ctx, tx, models.ChatMessage{
		RoomID:      roomID,
		SenderID:    "00000000-0000-0000-0000-000000000000",
		SenderRole:  "SYSTEM",
		MessageType: "SYSTEM_EVENT",
		Content:     welcomeMsg,
	})
	if err != nil {
		_ = tx.Rollback()
		return "", "", fmt.Errorf("failed to save initial system message: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", "", fmt.Errorf("failed to commit booking transaction: %w", err)
	}

	return bookingID, otp, nil
}

// GetBookingOwner retrieves user_id, priest_id, status, and completion_otp for authorization and privacy checks.
func (s *Storage) GetBookingOwner(bookingID string) (userID, priestID, status, completionOTP string, err error) {
	var uid string
	var pidVal sql.NullString
	var st, otp string
	query := `SELECT user_id::text, priest_id::text, status, completion_otp FROM bookings WHERE id = $1`
	err = s.DB.QueryRow(query, bookingID).Scan(&uid, &pidVal, &st, &otp)
	if err != nil {
		return "", "", "", "", err
	}
	if pidVal.Valid {
		priestID = pidVal.String
	}
	return uid, priestID, st, otp, nil
}

// VerifyBookingParticipant checks whether the caller is the owner devotee or assigned priest of a booking.
func (s *Storage) VerifyBookingParticipant(bookingID, callerID, callerRole string) (ownerUID, assignedPID string, err error) {
	ownerUID, assignedPID, _, _, err = s.GetBookingOwner(bookingID)
	if err != nil {
		return "", "", fmt.Errorf("booking not found: %w", err)
	}
	if callerRole == "devotee" || callerRole == "user" {
		if ownerUID != callerID {
			return "", "", fmt.Errorf("access denied: you are not the devotee for this booking")
		}
		return ownerUID, assignedPID, nil
	}
	if callerRole == "priest" {
		if assignedPID == "" || assignedPID != callerID {
			return "", "", fmt.Errorf("access denied: you are not the assigned priest for this booking")
		}
		return ownerUID, assignedPID, nil
	}
	return "", "", fmt.Errorf("access denied: unknown role %s", callerRole)
}

// GetBookingDetails returns full details for a booking including address, devotee, priest, and chat room
func (s *Storage) GetBookingDetails(bookingID string) (*models.BookingDetailResponse, error) {
	query := `
		SELECT 
			b.id, b.user_id, COALESCE(b.priest_id::text, ''), 
			COALESCE(u.full_name, ''), COALESCE(u.phone, ''),
			COALESCE(p.full_name, ''), COALESCE(p.phone, ''),
			COALESCE(pc.title, ''),
			b.booking_type, b.status, b.escrow_status, b.total_amount, b.include_samagri_kit,
			b.completion_otp, COALESCE(b.event_address, ''), b.latitude, b.longitude,
			COALESCE(b.scheduled_date, ''), COALESCE(b.slot_time, ''),
			COALESCE(b.devotee_gotram, u.gotram, ''), COALESCE(b.devotee_nakshatram, u.nakshatram, ''),
			COALESCE(b.devotee_rashi, u.rashi, ''), COALESCE(b.sankalpam_purpose, ''),
			COALESCE(cr.id::text, ''), b.created_at
		FROM bookings b
		LEFT JOIN users u ON b.user_id = u.id
		LEFT JOIN priests p ON b.priest_id = p.id
		LEFT JOIN pooja_catalog pc ON b.pooja_id = pc.id
		LEFT JOIN chat_rooms cr ON cr.booking_id = b.id
		WHERE b.id = $1;
	`
	var res models.BookingDetailResponse
	err := s.DB.QueryRow(query, bookingID).Scan(
		&res.ID, &res.UserID, &res.PriestID,
		&res.DevoteeName, &res.DevoteePhone,
		&res.PriestName, &res.PriestPhone,
		&res.PoojaTitle,
		&res.BookingType, &res.Status, &res.EscrowStatus, &res.TotalAmount, &res.IncludeSamagriKit,
		&res.CompletionOTP, &res.EventAddress, &res.Latitude, &res.Longitude,
		&res.ScheduledDate, &res.SlotTime,
		&res.DevoteeGotram, &res.DevoteeNakshatram, &res.DevoteeRashi, &res.SankalpamPurpose,
		&res.ChatRoomID, &res.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get booking details: %w", err)
	}
	return &res, nil
}

// GetBookings retrieves bookings filtered by priest_id, user_id, or status
func (s *Storage) GetBookings(priestID, userID, status string) ([]models.BookingDetailResponse, error) {
	query := `
		SELECT 
			b.id, b.user_id, COALESCE(b.priest_id::text, ''), 
			COALESCE(u.full_name, ''), COALESCE(u.phone, ''),
			COALESCE(p.full_name, ''), COALESCE(p.phone, ''),
			COALESCE(pc.title, cs.title, ''),
			b.booking_type, b.status, b.escrow_status, b.total_amount, b.include_samagri_kit,
			b.completion_otp, COALESCE(b.event_address, ''),
			b.latitude, b.longitude,
			COALESCE(b.scheduled_date, ''), COALESCE(b.slot_time, ''),
			COALESCE(b.devotee_gotram, u.gotram, ''), COALESCE(b.devotee_nakshatram, u.nakshatram, ''),
			COALESCE(b.devotee_rashi, u.rashi, ''), COALESCE(b.sankalpam_purpose, ''),
			COALESCE(cr.id::text, ''), b.created_at
		FROM bookings b
		JOIN users u ON b.user_id = u.id
		LEFT JOIN priests p ON b.priest_id = p.id
		LEFT JOIN pooja_catalog pc ON b.pooja_id = pc.id
		LEFT JOIN priest_custom_services cs ON b.pooja_id = cs.id
		LEFT JOIN chat_rooms cr ON cr.booking_id = b.id
		WHERE 1=1
	`
	var args []interface{}
	idx := 1

	if priestID != "" && priestID != "all" {
		query += fmt.Sprintf(" AND b.priest_id = $%d", idx)
		args = append(args, priestID)
		idx++
	}
	if userID != "" && userID != "all" {
		query += fmt.Sprintf(" AND b.user_id = $%d", idx)
		args = append(args, userID)
		idx++
	}
	if status != "" && status != "ALL" {
		if status == "ACTIVE" {
			query += " AND b.status IN ('REQUESTED', 'ACCEPTED', 'IN_TRANSIT', 'ARRIVED', 'POOJA_STARTED')"
		} else {
			query += fmt.Sprintf(" AND b.status = $%d", idx)
			args = append(args, status)
			idx++
		}
	}

	query += " ORDER BY (b.status = 'REQUESTED') DESC, b.created_at DESC LIMIT 25;"

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []models.BookingDetailResponse
	for rows.Next() {
		var res models.BookingDetailResponse
		if err := rows.Scan(
			&res.ID, &res.UserID, &res.PriestID,
			&res.DevoteeName, &res.DevoteePhone,
			&res.PriestName, &res.PriestPhone,
			&res.PoojaTitle,
			&res.BookingType, &res.Status, &res.EscrowStatus, &res.TotalAmount, &res.IncludeSamagriKit,
			&res.CompletionOTP, &res.EventAddress, &res.Latitude, &res.Longitude,
			&res.ScheduledDate, &res.SlotTime,
			&res.DevoteeGotram, &res.DevoteeNakshatram, &res.DevoteeRashi, &res.SankalpamPurpose,
			&res.ChatRoomID, &res.CreatedAt,
		); err != nil {
			continue
		}
		bookings = append(bookings, res)
	}
	return bookings, nil
}

// UpdateBookingStatus updates the booking state inside an atomic transaction,
// enforcing pessimistic row locks, state machine sequence validation, role authorization,
// slot conflict checks, slot release on cancellation, and rollback safety.
func (s *Storage) UpdateBookingStatus(req models.BookingStatusUpdateRequest) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Begin transaction with Read Committed isolation
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 2. Lock the booking row with SELECT ... FOR UPDATE
	var currentUID string
	var currentPIDVal sql.NullString
	var currentStatus string
	var expectedOTPVal sql.NullString
	var schedDateVal sql.NullString
	var slotTimeVal sql.NullString
	var currentEscrow string
	var currentDakshina sql.NullFloat64
	var otpAttempts int

	lockQuery := `
		SELECT 
			user_id::text, 
			COALESCE(priest_id::text, ''), 
			status, 
			COALESCE(completion_otp, ''), 
			COALESCE(scheduled_date, 'Today'), 
			COALESCE(slot_time, '09:00 AM'), 
			escrow_status,
			COALESCE(dakshina_amount, 0.0), completion_otp_attempts
		FROM bookings 
		WHERE id = $1 
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, lockQuery, req.BookingID).Scan(
		&currentUID,
		&currentPIDVal,
		&currentStatus,
		&expectedOTPVal,
		&schedDateVal,
		&slotTimeVal,
		&currentEscrow,
		&currentDakshina,
		&otpAttempts,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", lifecycle.ErrBookingNotFound
		}
		return "", fmt.Errorf("failed to lock booking row: %w", err)
	}

	currentPID := currentPIDVal.String
	expectedOTP := expectedOTPVal.String
	scheduledDate := schedDateVal.String
	slotTime := slotTimeVal.String

	// Resolve caller role and caller ID with test/fallback compatibility
	callerRole := req.CallerRole
	callerID := req.CallerID
	actingPriestID := currentPID

	if callerRole == "" {
		if req.PriestID != "" {
			callerRole = "priest"
			if callerID == "" {
				callerID = req.PriestID
			}
		} else {
			callerRole = "system"
		}
	}
	if req.Status == "ACCEPTED" && actingPriestID == "" {
		actingPriestID = callerID
	}

	// 3. Centralized FSM transition validation
	transReq := lifecycle.TransitionRequest{
		CurrentStatus:    currentStatus,
		TargetStatus:     req.Status,
		CallerRole:       callerRole,
		CallerID:         callerID,
		AssignedPriestID: actingPriestID,
		OwnerDevoteeID:   currentUID,
		ProvidedOTP:      req.CompletionOTP,
		ExpectedOTP:      expectedOTP,
	}

	validationErr := lifecycle.ValidateTransition(transReq)
	if req.Status == "COMPLETED" && (validationErr == nil || errors.Is(validationErr, lifecycle.ErrInvalidOTP) || errors.Is(validationErr, lifecycle.ErrMissingOTP)) {
		if otpAttempts >= 5 {
			return "", fmt.Errorf("%w: attempt limit reached; contact support", lifecycle.ErrInvalidOTP)
		}
		if validationErr != nil {
			if _, err := tx.ExecContext(ctx, "UPDATE bookings SET completion_otp_attempts = completion_otp_attempts + 1 WHERE id = $1", req.BookingID); err != nil {
				return "", err
			}
			if err := tx.Commit(); err != nil {
				return "", err
			}
			return "", validationErr
		}
	}
	if validationErr != nil {
		return "", validationErr
	}

	// 4. Slot Verification and Atomic Handling on Transitions
	if req.Status == "ACCEPTED" {
		slotPriestID := currentPID
		if slotPriestID == "" {
			slotPriestID = callerID
		}
		if slotPriestID == "" && req.PriestID != "" {
			slotPriestID = req.PriestID
		}

		if slotPriestID != "" {
			slotDate := time.Now().Format("2006-01-02")
			if scheduledDate != "" && scheduledDate != "Today" && len(scheduledDate) == 10 {
				slotDate = scheduledDate
			}
			startTime := "09:00:00"
			endTime := "11:30:00"
			if strings.Contains(slotTime, "05:30") || strings.Contains(slotTime, "06:00") {
				startTime = "06:00:00"
				endTime = "08:30:00"
			} else if strings.Contains(slotTime, "11:45") || strings.Contains(slotTime, "12:00") {
				startTime = "11:45:00"
				endTime = "13:45:00"
			} else if strings.Contains(slotTime, "05:30 PM") || strings.Contains(slotTime, "17:30") {
				startTime = "17:30:00"
				endTime = "19:30:00"
			}

			var slotID string
			var isBooked bool
			var bookedBookingID sql.NullString
			slotQuery := `
				SELECT id::text, is_booked, booked_booking_id::text 
				FROM priest_availability_slots 
				WHERE priest_id = $1::uuid AND slot_date = $2 AND start_time = $3::time
				FOR UPDATE
			`
			err = tx.QueryRowContext(ctx, slotQuery, slotPriestID, slotDate, startTime).Scan(&slotID, &isBooked, &bookedBookingID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					insertSlotQuery := `
						INSERT INTO priest_availability_slots (priest_id, slot_date, start_time, end_time, is_booked, booked_booking_id)
						VALUES ($1::uuid, $2, $3::time, $4::time, true, $5::uuid)
					`
					if _, err := tx.ExecContext(ctx, insertSlotQuery, slotPriestID, slotDate, startTime, endTime, req.BookingID); err != nil {
						return "", fmt.Errorf("failed to allocate priest availability slot: %w", err)
					}
				} else {
					return "", fmt.Errorf("failed to check priest availability slot: %w", err)
				}
			} else {
				if isBooked && bookedBookingID.Valid && bookedBookingID.String != req.BookingID {
					return "", fmt.Errorf("%w: priest slot on %s at %s is already booked by booking %s",
						lifecycle.ErrSlotUnavailable, slotDate, startTime, bookedBookingID.String)
				}
				updateSlotQuery := `
					UPDATE priest_availability_slots 
					SET is_booked = true, booked_booking_id = $1::uuid 
					WHERE id = $2::uuid
				`
				if _, err := tx.ExecContext(ctx, updateSlotQuery, req.BookingID, slotID); err != nil {
					return "", fmt.Errorf("failed to update priest availability slot: %w", err)
				}
			}
		}
	} else if req.Status == "CANCELLED" || req.Status == "REJECTED" {
		releaseSlotQuery := `
			UPDATE priest_availability_slots 
			SET is_booked = false, booked_booking_id = NULL 
			WHERE booked_booking_id = $1::uuid
		`
		if _, err := tx.ExecContext(ctx, releaseSlotQuery, req.BookingID); err != nil {
			return "", fmt.Errorf("failed to release priest availability slot: %w", err)
		}
	}

	// 5. Mutate Bookings Row
	var resultStatus string = "UPDATED"

	if req.Status == "COMPLETED" {
		resultStatus = "RELEASED_TO_PRIEST"
		updateBookingQuery := `
			UPDATE bookings 
			SET status = 'COMPLETED', 
			    escrow_status = 'RELEASED_TO_PRIEST', 
			    dakshina_amount = CASE WHEN $2 > 0 THEN $2 ELSE dakshina_amount END
			WHERE id = $1::uuid
		`
		res, err := tx.ExecContext(ctx, updateBookingQuery, req.BookingID, req.DakshinaAmount)
		if err != nil {
			return "", fmt.Errorf("failed updating booking to COMPLETED: %w", err)
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			return "", fmt.Errorf("booking completion failed: row disappeared")
		}
	} else if req.Status == "CANCELLED" {
		resultStatus = "REFUNDED_TO_DEVOTEE"
		updateBookingQuery := `
			UPDATE bookings 
			SET status = 'CANCELLED', 
			    escrow_status = 'REFUNDED_TO_DEVOTEE'
			WHERE id = $1::uuid
		`
		res, err := tx.ExecContext(ctx, updateBookingQuery, req.BookingID)
		if err != nil {
			return "", fmt.Errorf("failed updating booking to CANCELLED: %w", err)
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			return "", fmt.Errorf("booking cancellation failed: row disappeared")
		}
	} else {
		var updateBookingQuery string
		var args []interface{}
		if currentPID == "" && actingPriestID != "" {
			updateBookingQuery = `UPDATE bookings SET status = $2, priest_id = $3::uuid WHERE id = $1::uuid`
			args = []interface{}{req.BookingID, req.Status, actingPriestID}
		} else {
			updateBookingQuery = `UPDATE bookings SET status = $2 WHERE id = $1::uuid`
			args = []interface{}{req.BookingID, req.Status}
		}
		res, err := tx.ExecContext(ctx, updateBookingQuery, args...)
		if err != nil {
			return "", fmt.Errorf("failed updating booking status to %s: %w", req.Status, err)
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			return "", fmt.Errorf("booking status update failed: row disappeared")
		}
	}

	// 6. Commit Transaction
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit booking update transaction: %w", err)
	}

	return resultStatus, nil
}

// GetAllPriests returns all verified priests in the system
func (s *Storage) GetAllPriests() ([]models.NearbyPriest, error) {
	query := `
		SELECT 
			id, full_name, phone, sampradaya, COALESCE(veda_shakha, ''),
			COALESCE(rating_avg, 4.9), COALESCE(total_completed_poojas, 12), COALESCE(repeat_booking_count, 5),
			latitude, longitude, eta_minutes, vehicle_type, services_offered
		FROM priests
		ORDER BY is_online DESC, rating_avg DESC;
	`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var priests []models.NearbyPriest
	for rows.Next() {
		var p models.NearbyPriest
		var services pq.StringArray
		if err := rows.Scan(
			&p.ID, &p.FullName, &p.Phone, &p.Sampradaya, &p.VedaShakha,
			&p.RatingAvg, &p.TotalCompletedPoojas, &p.RepeatBookingCount,
			&p.Latitude, &p.Longitude, &p.ETAMinutes, &p.VehicleType, &services,
		); err != nil {
			continue
		}
		p.ServicesOffered = []string(services)
		priests = append(priests, p)
	}
	return priests, nil
}

// GetDispatchRadarData returns live geo-coordinates of online priests and active devotee bookings for the Temple CMS
func (s *Storage) GetDispatchRadarData() (*models.DispatchRadarResponse, error) {
	priestQuery := `
		SELECT 
			p.id, p.full_name, p.phone, p.sampradaya, COALESCE(p.veda_shakha, ''),
			COALESCE(p.rating_avg, 4.95), p.is_online, p.latitude, p.longitude, p.vehicle_type,
			p.services_offered, COALESCE(b.id::text, ''), COALESCE(b.status, 'AVAILABLE')
		FROM priests p
		LEFT JOIN bookings b ON b.priest_id = p.id AND b.status IN ('REQUESTED', 'ACCEPTED', 'IN_TRANSIT', 'ARRIVED', 'POOJA_STARTED')
		WHERE p.is_online = true OR b.id IS NOT NULL
		ORDER BY p.is_online DESC;
	`
	pRows, err := s.DB.Query(priestQuery)
	if err != nil {
		return nil, err
	}
	defer pRows.Close()

	var onlinePriests []models.DispatchRadarPriest
	for pRows.Next() {
		var p models.DispatchRadarPriest
		var services pq.StringArray
		if err := pRows.Scan(
			&p.ID, &p.FullName, &p.Phone, &p.Sampradaya, &p.VedaShakha,
			&p.RatingAvg, &p.IsOnline, &p.Latitude, &p.Longitude, &p.VehicleType,
			&services, &p.ActiveBookingID, &p.Status,
		); err != nil {
			continue
		}
		p.ServicesOffered = []string(services)
		onlinePriests = append(onlinePriests, p)
	}

	devoteeQuery := `
		SELECT 
			u.id, u.full_name, u.phone, COALESCE(u.gotram, 'Kashyapa'), COALESCE(u.nakshatram, 'Rohini'), COALESCE(u.rashi, 'Vrishabha'),
			COALESCE(b.latitude, 17.3850), COALESCE(b.longitude, 78.4867), u.city,
			(b.id IS NOT NULL) AS has_active_booking,
			COALESCE(b.id::text, ''), COALESCE(pc.title, 'Sri Satyanarayana Swamy Vratam'),
			COALESCE(b.status, 'COMPLETED'), COALESCE(b.scheduled_date, 'Today'), COALESCE(b.slot_time, '08:30 AM - 10:30 AM'),
			COALESCE(p.full_name, 'Brahmasri Sushanth Sastry'), COALESCE(b.event_address, 'Hyderabad')
		FROM users u
		LEFT JOIN bookings b ON b.user_id = u.id AND b.status IN ('REQUESTED', 'ACCEPTED', 'IN_TRANSIT', 'ARRIVED', 'POOJA_STARTED')
		LEFT JOIN pooja_catalog pc ON b.pooja_id = pc.id
		LEFT JOIN priests p ON b.priest_id = p.id
		ORDER BY has_active_booking DESC, u.created_at DESC
		LIMIT 15;
	`
	dRows, err := s.DB.Query(devoteeQuery)
	if err != nil {
		return nil, err
	}
	defer dRows.Close()

	var activeDevotees []models.DispatchRadarDevotee
	pendingCount := 0
	for dRows.Next() {
		var d models.DispatchRadarDevotee
		if err := dRows.Scan(
			&d.ID, &d.FullName, &d.Phone, &d.Gotram, &d.Nakshatram, &d.Rashi,
			&d.Latitude, &d.Longitude, &d.City,
			&d.HasActiveBooking,
			&d.ActiveBookingID, &d.ActivePoojaTitle,
			&d.Status, &d.ScheduledDate, &d.SlotTime,
			&d.AssignedPriest, &d.EventAddress,
		); err != nil {
			continue
		}
		if d.Status == "REQUESTED" {
			pendingCount++
		}
		activeDevotees = append(activeDevotees, d)
	}

	return &models.DispatchRadarResponse{
		TotalOnlinePriests:   len(onlinePriests),
		TotalActiveDevotees:  len(activeDevotees),
		PendingRequestsCount: pendingCount,
		OnlinePriests:        onlinePriests,
		ActiveDevotees:       activeDevotees,
	}, nil
}
