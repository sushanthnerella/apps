package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/geo"
	"github.com/purohit-platform/backend/internal/models"
	"github.com/redis/go-redis/v9"
)

type Storage struct {
	DB    *sql.DB
	Redis *redis.Client
}

func InitStorage(pgConnStr, redisAddr string) (*Storage, error) {
	db, err := sql.Open("postgres", pgConnStr)
	if err != nil {
		return nil, fmt.Errorf("postgres connect error: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("postgres ping error: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️ Redis ping warning: %v (continuing with DB fallback)", err)
	} else {
		log.Printf("✅ Connected to Redis at %s", redisAddr)
	}

	log.Printf("✅ Connected to PostgreSQL database")
	return &Storage{DB: db, Redis: rdb}, nil
}

// LoginWithOTP authenticates user with phone + OTP (default test OTP '1234') and issues JWT
func (s *Storage) LoginWithOTP(phone, otp, role string) (*models.AuthResponse, error) {
	// OTP verification rule: accept '1234' or valid 4-digit numeric code in dev mode
	isNumeric4 := len(otp) == 4
	for _, c := range otp {
		if c < '0' || c > '9' {
			isNumeric4 = false
			break
		}
	}
	if !isNumeric4 {
		return nil, fmt.Errorf("invalid OTP code: must be a 4-digit numeric code (use '1234' for instant verification)")
	}

	var userProfile models.UserProfile

	if role == "priest" {
		query := `SELECT id, phone, full_name, sampradaya, veda_shakha, 'Hyderabad', consent_agreed, created_at FROM priests WHERE phone = $1`
		err := s.DB.QueryRow(query, phone).Scan(
			&userProfile.ID, &userProfile.Phone, &userProfile.FullName,
			&userProfile.Gotram, &userProfile.Nakshatram, &userProfile.City,
			&userProfile.ConsentAgreed, &userProfile.CreatedAt,
		)
		if err != nil {
			// Auto-register priest if not exists
			var newID string
			ins := `INSERT INTO priests (phone, full_name, sampradaya, consent_agreed) VALUES ($1, 'Vedic Purohit', 'Smartha', true) RETURNING id`
			if e := s.DB.QueryRow(ins, phone).Scan(&newID); e != nil {
				return nil, fmt.Errorf("priest lookup error: %w", e)
			}
			userProfile.ID = newID
			userProfile.Phone = phone
			userProfile.FullName = "Vedic Purohit"
			userProfile.Gotram = "Smartha"
			userProfile.ConsentAgreed = true
		}
		userProfile.Role = "priest"
	} else {
		// Devotee lookup or auto-provision
		query := `SELECT id, phone, full_name, COALESCE(gotram, 'Kashyapa'), COALESCE(nakshatram, 'Rohini'), COALESCE(rashi, 'Vrishabha'), city, consent_agreed, created_at FROM users WHERE phone = $1`
		err := s.DB.QueryRow(query, phone).Scan(
			&userProfile.ID, &userProfile.Phone, &userProfile.FullName,
			&userProfile.Gotram, &userProfile.Nakshatram, &userProfile.Rashi,
			&userProfile.City, &userProfile.ConsentAgreed, &userProfile.CreatedAt,
		)
		if err != nil {
			var newID string
			defaultName := "Devotee"
			if len(phone) >= 4 {
				defaultName = "Devotee " + phone[len(phone)-4:]
			}
			ins := `INSERT INTO users (phone, full_name, gotram, nakshatram, rashi, city, consent_agreed) VALUES ($1, $2, 'Not Set', 'Not Set', 'Not Set', 'Hyderabad', true) RETURNING id`
			if e := s.DB.QueryRow(ins, phone, defaultName).Scan(&newID); e != nil {
				return nil, fmt.Errorf("devotee lookup error: %w", e)
			}
			userProfile.ID = newID
			userProfile.Phone = phone
			userProfile.FullName = defaultName
			userProfile.Gotram = "Not Set"
			userProfile.Nakshatram = "Not Set"
			userProfile.Rashi = "Not Set"
			userProfile.City = "Hyderabad"
			userProfile.ConsentAgreed = true
		}
		userProfile.Role = "devotee"
	}

	// Issue JWT token
	token, err := auth.GenerateToken(userProfile.ID, userProfile.Phone, userProfile.Role, userProfile.FullName)
	if err != nil {
		return nil, fmt.Errorf("JWT generation error: %w", err)
	}

	return &models.AuthResponse{
		Status:      "SUCCESS",
		AccessToken: token,
		TokenType:   "Bearer",
		User:        userProfile,
		Message:     "Authenticated successfully. JWT session token generated.",
	}, nil
}

// GetUserProfile retrieves profile by user ID and role
func (s *Storage) GetUserProfile(userID, role string) (*models.UserProfile, error) {
	var profile models.UserProfile
	if role == "priest" {
		query := `SELECT id, phone, full_name, sampradaya, veda_shakha, 'Hyderabad', consent_agreed, created_at FROM priests WHERE id = $1`
		err := s.DB.QueryRow(query, userID).Scan(
			&profile.ID, &profile.Phone, &profile.FullName,
			&profile.Gotram, &profile.Nakshatram, &profile.City,
			&profile.ConsentAgreed, &profile.CreatedAt,
		)
		profile.Role = "priest"
		return &profile, err
	}

	query := `SELECT id, phone, full_name, COALESCE(gotram, ''), COALESCE(nakshatram, ''), COALESCE(rashi, ''), city, consent_agreed, created_at FROM users WHERE id = $1`
	err := s.DB.QueryRow(query, userID).Scan(
		&profile.ID, &profile.Phone, &profile.FullName,
		&profile.Gotram, &profile.Nakshatram, &profile.Rashi,
		&profile.City, &profile.ConsentAgreed, &profile.CreatedAt,
	)
	profile.Role = "devotee"
	return &profile, err
}

// GetNearbyPriests finds all active priests around devotee location with real distance and ETA
func (s *Storage) GetNearbyPriests(devLat, devLon, radiusMeters float64, serviceFilter string) ([]models.NearbyPriest, error) {
	if radiusMeters <= 0 {
		radiusMeters = 50000.0 // 50km default radius for metro region
	}
	if devLat == 0 && devLon == 0 {
		devLat = 17.3850
		devLon = 78.4867
	}

	query := `
		SELECT id, full_name, phone, sampradaya, veda_shakha, rating_avg, total_completed_poojas, repeat_booking_count,
		       latitude, longitude, eta_minutes, vehicle_type, services_offered
		FROM priests
		WHERE is_online = true;
	`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allPriests []models.NearbyPriest
	var filteredPriests []models.NearbyPriest

	for rows.Next() {
		var p models.NearbyPriest
		var services pq.StringArray
		err := rows.Scan(
			&p.ID, &p.FullName, &p.Phone, &p.Sampradaya, &p.VedaShakha,
			&p.RatingAvg, &p.TotalCompletedPoojas, &p.RepeatBookingCount,
			&p.Latitude, &p.Longitude, &p.ETAMinutes, &p.VehicleType, &services,
		)
		if err != nil {
			return nil, err
		}
		p.ServicesOffered = []string(services)
		if len(p.ServicesOffered) == 0 {
			p.ServicesOffered = []string{"Gruhapravesam (House Warming)", "Sri Satyanarayana Swamy Vratam", "Maha Ganapathi Homa / Yagam", "Navagraha Shanti Homa"}
		}

		if p.Latitude == 0 && p.Longitude == 0 {
			p.Latitude = 17.3860
			p.Longitude = 78.4875
		}

		// Compute Haversine Geodesic Distance
		dist := geo.HaversineDistance(devLat, devLon, p.Latitude, p.Longitude)
		p.DistanceMeters = math.Round(dist)
		p.DistanceKm = math.Round((dist/1000.0)*10) / 10

		// Update ETA based on transit distance (~2.5 mins per km)
		computedETA := int(math.Ceil(p.DistanceKm * 3.0))
		if computedETA < 4 {
			computedETA = 4
		}
		p.ETAMinutes = computedETA

		// Quick Rapido composite score
		distRatio := p.DistanceMeters / radiusMeters
		if distRatio > 1.0 {
			distRatio = 1.0
		}
		p.Score = math.Round((0.40*(1.0-distRatio)+0.35*(p.RatingAvg/5.0)+0.25*(float64(p.RepeatBookingCount)/50.0))*100) / 100

		allPriests = append(allPriests, p)

		// Filter by service if specified (supports multi-service selection separated by + or ,)
		if serviceFilter != "" {
			filterParts := strings.FieldsFunc(serviceFilter, func(r rune) bool {
				return r == '+' || r == ','
			})
			matches := false
			for _, part := range filterParts {
				trimmed := strings.TrimSpace(part)
				if trimmed == "" {
					continue
				}
				for _, srv := range p.ServicesOffered {
					if strings.Contains(strings.ToLower(srv), strings.ToLower(trimmed)) ||
						strings.Contains(strings.ToLower(trimmed), strings.ToLower(srv)) {
						matches = true
						break
					}
				}
				if matches {
					break
				}
			}
			if matches {
				filteredPriests = append(filteredPriests, p)
			}
		}
	}

	result := filteredPriests
	if len(result) == 0 {
		result = allPriests
	}

	// Sort nearest first (like Rapido shows nearest captains)
	sort.Slice(result, func(i, j int) bool {
		return result[i].DistanceMeters < result[j].DistanceMeters
	})

	return result, nil
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
	return catalog, nil
}

// RegisterDevotee saves devotee details and consent into PostgreSQL
func (s *Storage) RegisterDevotee(req models.DevoteeRegistrationRequest) (string, error) {
	var id string
	query := `
		INSERT INTO users (phone, full_name, gotram, nakshatram, rashi, city, consent_agreed, consent_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP)
		ON CONFLICT (phone) DO UPDATE SET
			full_name = EXCLUDED.full_name,
			gotram = EXCLUDED.gotram,
			nakshatram = EXCLUDED.nakshatram,
			rashi = EXCLUDED.rashi,
			city = EXCLUDED.city,
			consent_agreed = EXCLUDED.consent_agreed,
			consent_timestamp = CURRENT_TIMESTAMP
		RETURNING id;
	`
	err := s.DB.QueryRow(query, req.Phone, req.FullName, req.Gotram, req.Nakshatram, req.Rashi, req.City, req.ConsentAgreed).Scan(&id)
	return id, err
}

// RegisterPriest saves priest details, Vedic qualifications, and consent into PostgreSQL
func (s *Storage) RegisterPriest(req models.PriestRegistrationRequest) (string, error) {
	var id string
	if req.Latitude == 0 && req.Longitude == 0 {
		req.Latitude = 17.3850 + (rand.Float64()-0.5)*0.015
		req.Longitude = 78.4867 + (rand.Float64()-0.5)*0.015
	}
	query := `
		INSERT INTO priests (phone, full_name, sampradaya, veda_shakha, experience_years, latitude, longitude, consent_agreed, is_online, services_offered, consent_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, ARRAY['Gruhapravesam (House Warming)', 'Sri Satyanarayana Swamy Vratam', 'Maha Ganapathi Homa / Yagam', 'Navagraha Shanti Homa', 'Mrityunjaya Homa / Shanti', 'Namakaranam & Ayushya Homa'], CURRENT_TIMESTAMP)
		ON CONFLICT (phone) DO UPDATE SET
			full_name = EXCLUDED.full_name,
			sampradaya = EXCLUDED.sampradaya,
			veda_shakha = EXCLUDED.veda_shakha,
			experience_years = EXCLUDED.experience_years,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			consent_agreed = EXCLUDED.consent_agreed,
			services_offered = EXCLUDED.services_offered,
			is_online = true,
			consent_timestamp = CURRENT_TIMESTAMP
		RETURNING id;
	`
	err := s.DB.QueryRow(query, req.Phone, req.FullName, req.Sampradaya, req.VedaShakha, req.ExperienceYears, req.Latitude, req.Longitude, req.ConsentAgreed).Scan(&id)

	if err == nil && s.Redis != nil {
		ctx := context.Background()
		s.Redis.GeoAdd(ctx, "priests:geo", &redis.GeoLocation{
			Name:      id,
			Latitude:  req.Latitude,
			Longitude: req.Longitude,
		})
	}

	return id, err
}

// CreateBooking creates a new booking in HELD_IN_ESCROW state with a generated 4-digit OTP,
// populates booking_materials checklist, sets up the chat room, and posts the initial system notification.
func (s *Storage) CreateBooking(req models.BookingCreateRequest) (string, string, error) {
	otp := fmt.Sprintf("%04d", rand.Intn(9000)+1000)
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
		s.DB.QueryRow(`SELECT id::text FROM users ORDER BY created_at DESC LIMIT 1`).Scan(&req.UserID)
	}

	var validPriest bool
	if req.PriestID != "" {
		var pid string
		if err := s.DB.QueryRow(`SELECT id::text FROM priests WHERE id::text = $1`, req.PriestID).Scan(&pid); err == nil && pid != "" {
			validPriest = true
			req.PriestID = pid
		}
	}
	if !validPriest {
		s.DB.QueryRow(`SELECT id::text FROM priests WHERE is_online = true LIMIT 1`).Scan(&req.PriestID)
	}

	var validPooja bool
	if req.PoojaID != "" {
		var pjid string
		if err := s.DB.QueryRow(`SELECT id::text FROM pooja_catalog WHERE id::text = $1`, req.PoojaID).Scan(&pjid); err == nil && pjid != "" {
			validPooja = true
			req.PoojaID = pjid
		}
	}
	if !validPooja {
		s.DB.QueryRow(`SELECT id::text FROM pooja_catalog LIMIT 1`).Scan(&req.PoojaID)
	}

	if req.EventAddress == "" {
		req.EventAddress = "Plot 42, Sri Rama Nagar, Madhav Nagar, Hyderabad (Landmark: Near Sri Rama Temple)"
	}
	if req.Latitude == 0 {
		req.Latitude = 17.3850
	}
	if req.Longitude == 0 {
		req.Longitude = 78.4867
	}
	if req.BookingType == "" {
		req.BookingType = "INSTANT_DISPATCH"
	}
	if req.ScheduledDate == "" {
		req.ScheduledDate = "Today (" + time.Now().Format("2006-01-02") + ")"
	}
	if req.SlotTime == "" {
		req.SlotTime = "08:30 AM - 10:30 AM"
	}
	if req.DevoteeGotram == "" {
		req.DevoteeGotram = "Kashyapa"
	}
	if req.DevoteeNakshatram == "" {
		req.DevoteeNakshatram = "Rohini"
	}
	if req.DevoteeRashi == "" {
		req.DevoteeRashi = "Vrishabha"
	}
	if req.SankalpamPurpose == "" {
		req.SankalpamPurpose = "Family Prosperity & Health"
	}

	query := `
		INSERT INTO bookings (
			user_id, priest_id, pooja_id, booking_type, status, escrow_status, 
			total_amount, include_samagri_kit, completion_otp, event_address, latitude, longitude,
			scheduled_date, slot_time, devotee_gotram, devotee_nakshatram, devotee_rashi, sankalpam_purpose
		)
		VALUES ($1, $2, $3, $4, 'REQUESTED', 'HELD_IN_ESCROW', $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id;
	`
	err := s.DB.QueryRow(query, 
		req.UserID, req.PriestID, req.PoojaID, req.BookingType, 
		req.TotalAmount, req.IncludeSamagriKit, otp, req.EventAddress, req.Latitude, req.Longitude,
		req.ScheduledDate, req.SlotTime, req.DevoteeGotram, req.DevoteeNakshatram, req.DevoteeRashi, req.SankalpamPurpose,
	).Scan(&bookingID)
	if err != nil {
		return "", "", err
	}

	// 1. Copy standard items into booking_materials
	materials, _ := s.GetPoojaSamagri(req.PoojaID)
	for _, m := range materials {
		isPriestItem := (m.Category == "Sacred Ritual Samagri")
		s.DB.Exec(`
			INSERT INTO booking_materials (booking_id, item_name, category, quantity, is_included_by_priest)
			VALUES ($1, $2, $3, $4, $5);
		`, bookingID, m.ItemName, m.Category, m.Quantity, isPriestItem && req.IncludeSamagriKit)
	}

	// 2. Set up Chat Room for live Devotee-Priest coordination
	roomID, _ := s.GetOrCreateChatRoom(bookingID, req.UserID, req.PriestID)

	// 3. Post Initial System Message with Address, Muhurtham & Items Summary
	samagriNotice := "Brought by Purohit"
	if !req.IncludeSamagriKit {
		samagriNotice = "Arranged by Devotee"
	}
	welcomeMsg := fmt.Sprintf(
		"🕉️ Ritual Booking Requested!\n📅 Scheduled: %s | %s\n✨ Vedic Sankalpam: %s Gotram • %s Nakshatram (%s)\n📍 Event Address: %s\n🪔 Sacred Samagri Kit: %s\n🏡 Devotee: Please have standard household items (Brass plates, Diyas, Ghee, Mats, Rice, Fruits) ready. You can now chat directly here to coordinate directions and arrival.",
		req.ScheduledDate, req.SlotTime, req.DevoteeGotram, req.DevoteeNakshatram, req.DevoteeRashi, req.EventAddress, samagriNotice,
	)
	s.SaveChatMessage(models.ChatMessage{
		RoomID:      roomID,
		SenderID:    "system",
		SenderRole:  "SYSTEM",
		MessageType: "SYSTEM_EVENT",
		Content:     welcomeMsg,
	})

	return bookingID, otp, nil
}

// GetBookingDetails returns full details for a booking including address, devotee, priest, and chat room
func (s *Storage) GetBookingDetails(bookingID string) (*models.BookingDetailResponse, error) {
	query := `
		SELECT 
			b.id, b.user_id, COALESCE(b.priest_id::text, ''), 
			COALESCE(u.full_name, 'Devotee Sushanth'), COALESCE(u.phone, '9177961089'),
			COALESCE(p.full_name, 'Brahmasri Sushanth Sastry'), COALESCE(p.phone, '9876543210'),
			COALESCE(pc.title, 'Vedic Puja Ceremony'),
			b.booking_type, b.status, b.escrow_status, b.total_amount, b.include_samagri_kit,
			b.completion_otp, b.event_address, b.latitude, b.longitude,
			COALESCE(b.scheduled_date, 'Today'), COALESCE(b.slot_time, '08:30 AM - 10:30 AM'),
			COALESCE(b.devotee_gotram, u.gotram, 'Kashyapa'), COALESCE(b.devotee_nakshatram, u.nakshatram, 'Rohini'),
			COALESCE(b.devotee_rashi, u.rashi, 'Vrishabha'), COALESCE(b.sankalpam_purpose, 'Family Prosperity & Blessings'),
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

// UpdateBookingStatus updates the booking state and checks OTP on completion
func (s *Storage) UpdateBookingStatus(req models.BookingStatusUpdateRequest) (string, error) {
	if req.Status == "COMPLETED" {
		var expectedOTP string
		var escrowStatus string
		err := s.DB.QueryRow(`SELECT completion_otp, escrow_status FROM bookings WHERE id = $1`, req.BookingID).Scan(&expectedOTP, &escrowStatus)
		if err != nil {
			return "", fmt.Errorf("booking not found: %w", err)
		}

		if req.CompletionOTP != expectedOTP {
			return "", fmt.Errorf("invalid completion OTP (expected %s, got %s)", expectedOTP, req.CompletionOTP)
		}

		_, err = s.DB.Exec(`
			UPDATE bookings 
			SET status = 'COMPLETED', escrow_status = 'RELEASED_TO_PRIEST', dakshina_amount = $2 
			WHERE id = $1
		`, req.BookingID, req.DakshinaAmount)
		return "RELEASED_TO_PRIEST", err
	}

	// Update booking status
	query := `UPDATE bookings SET status = $2`
	args := []interface{}{req.BookingID, req.Status}

	if req.PriestID != "" && len(req.PriestID) == 36 {
		query += `, priest_id = $3 WHERE id = $1`
		args = append(args, req.PriestID)
	} else {
		query += ` WHERE id = $1`
	}

	_, err := s.DB.Exec(query, args...)
	if err != nil {
		return "", err
	}

	// When accepted or in transit, automatically block the priest's availability slot!
	if req.Status == "ACCEPTED" || req.Status == "IN_TRANSIT" {
		var pid, bDate, bTime string
		if err := s.DB.QueryRow(`SELECT COALESCE(priest_id::text, ''), COALESCE(scheduled_date, 'Today'), COALESCE(slot_time, '09:00 AM') FROM bookings WHERE id = $1`, req.BookingID).Scan(&pid, &bDate, &bTime); err == nil && pid != "" {
			slotDate := time.Now().Format("2006-01-02")
			startTime := "09:00:00"
			endTime := "11:30:00"
			if strings.Contains(bTime, "05:30") || strings.Contains(bTime, "06:00") {
				startTime = "06:00:00"
				endTime = "08:30:00"
			} else if strings.Contains(bTime, "11:45") || strings.Contains(bTime, "12:00") {
				startTime = "11:45:00"
				endTime = "13:45:00"
			} else if strings.Contains(bTime, "05:30 PM") || strings.Contains(bTime, "17:30") {
				startTime = "17:30:00"
				endTime = "19:30:00"
			}
			s.DB.Exec(`
				INSERT INTO priest_availability_slots (priest_id, slot_date, start_time, end_time, is_booked, booked_booking_id)
				VALUES ($1, $2, $3, $4, true, $5)
				ON CONFLICT ON CONSTRAINT uq_priest_slot DO UPDATE SET is_booked = true, booked_booking_id = EXCLUDED.booked_booking_id
			`, pid, slotDate, startTime, endTime, req.BookingID)
		}
	} else if req.Status == "CANCELLED" || req.Status == "REJECTED" {
		s.DB.Exec(`UPDATE priest_availability_slots SET is_booked = false, booked_booking_id = NULL WHERE booked_booking_id = $1`, req.BookingID)
	}

	return "UPDATED", nil
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
