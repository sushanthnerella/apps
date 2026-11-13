package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/purohit-platform/backend/internal/models"
)

// GetTemples retrieves all active shrines and temples
func (s *Storage) GetTemples() ([]models.Temple, error) {
	query := `
		SELECT id, name, slug, deity, description, city, state, latitude, longitude, 
		       COALESCE(banner_image_url, ''), timings, contact_phone, is_active 
		FROM temples 
		WHERE is_active = TRUE 
		ORDER BY name ASC
	`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query temples: %w", err)
	}
	defer rows.Close()

	var temples []models.Temple
	for rows.Next() {
		var t models.Temple
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Slug, &t.Deity, &t.Description, &t.City,
			&t.State, &t.Latitude, &t.Longitude, &t.BannerImageURL,
			&t.Timings, &t.ContactPhone, &t.IsActive,
		); err != nil {
			continue
		}
		temples = append(temples, t)
	}
	return temples, nil
}

// GetTempleVipPujas retrieves VIP offerings for a given temple
func (s *Storage) GetTempleVipPujas(templeID string) ([]models.TempleVipPuja, error) {
	query := `
		SELECT id, temple_id, title, description, price, max_daily_slots, 
		       duration_minutes, is_online_video_call, is_active 
		FROM temple_vip_pujas 
		WHERE temple_id = $1 AND is_active = TRUE 
		ORDER BY price ASC
	`
	rows, err := s.DB.Query(query, templeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query temple vip pujas: %w", err)
	}
	defer rows.Close()

	var pujas []models.TempleVipPuja
	for rows.Next() {
		var p models.TempleVipPuja
		if err := rows.Scan(
			&p.ID, &p.TempleID, &p.Title, &p.Description, &p.Price,
			&p.MaxDailySlots, &p.DurationMinutes, &p.IsOnlineVideoCall, &p.IsActive,
		); err != nil {
			continue
		}
		pujas = append(pujas, p)
	}
	return pujas, nil
}

// BookTempleTicket issues a VIP pass or Live E-Pooja token with a unique verifiable QR token
func (s *Storage) BookTempleTicket(req models.BookTempleTicketRequest) (*models.TempleVipTicket, error) {
	// Generate unique cryptographic QR token
	tokenBytes := make([]byte, 16)
	_, _ = rand.Read(tokenBytes)
	qrToken := fmt.Sprintf("PV-TEMPLE-%s-%s", req.VisitDate, hex.EncodeToString(tokenBytes))

	if req.AmountPaidMinor == 0 && req.AmountPaid > 0 {
		req.AmountPaidMinor = int64(math.Round(req.AmountPaid * 100))
	}
	if req.AmountPaidMinor > 0 && req.AmountPaid == 0 {
		req.AmountPaid = float64(req.AmountPaidMinor) / 100.0
	}

	query := `
		INSERT INTO temple_vip_tickets (
			temple_id, vip_puja_id, user_id, devotee_name, devotee_gotram, 
			devotee_nakshatram, devotee_phone, visit_date, slot_time, 
			ticket_type, qr_token, status, amount_paid, amount_paid_minor, payment_mode
		) 
		VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10, $11, 'CONFIRMED', $12, $13, $14) 
		RETURNING id, created_at
	`
	ticket := models.TempleVipTicket{
		TempleID:          req.TempleID,
		VipPujaID:         req.VipPujaID,
		UserID:            req.UserID,
		DevoteeName:       req.DevoteeName,
		DevoteeGotram:     req.DevoteeGotram,
		DevoteeNakshatram: req.DevoteeNakshatram,
		DevoteePhone:      req.DevoteePhone,
		VisitDate:         req.VisitDate,
		SlotTime:          req.SlotTime,
		TicketType:        req.TicketType,
		QRToken:           qrToken,
		Status:            "CONFIRMED",
		AmountPaidMinor:   req.AmountPaidMinor,
		AmountPaid:        req.AmountPaid,
		Currency:          "INR",
		PaymentMode:       req.PaymentMode,
	}

	err := s.DB.QueryRow(
		query, req.TempleID, req.VipPujaID, req.UserID, req.DevoteeName,
		req.DevoteeGotram, req.DevoteeNakshatram, req.DevoteePhone,
		req.VisitDate, req.SlotTime, req.TicketType, qrToken,
		req.AmountPaid, req.AmountPaidMinor, req.PaymentMode,
	).Scan(&ticket.ID, &ticket.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to issue temple ticket: %w", err)
	}
	return &ticket, nil
}

// VerifyTempleTicket handles fast atomic gate verification and updates ticket status to CHECKED_IN
func (s *Storage) VerifyTempleTicket(qrToken string) (*models.TempleVipTicket, error) {
	var ticket models.TempleVipTicket
	var checkInTS *time.Time
	var visitDate time.Time

	atomicQuery := `
		UPDATE temple_vip_tickets 
		SET status = 'CHECKED_IN', check_in_timestamp = now()
		WHERE qr_token = $1 AND status <> 'CHECKED_IN'
		RETURNING id, temple_id, vip_puja_id, COALESCE(user_id::text, ''), devotee_name, 
		       COALESCE(devotee_gotram, ''), COALESCE(devotee_nakshatram, ''), devotee_phone, 
		       visit_date, slot_time, ticket_type, qr_token, status, 
		       COALESCE(amount_paid_minor, ROUND(amount_paid * 100)), amount_paid, 
		       payment_mode, check_in_timestamp, created_at
	`
	err := s.DB.QueryRow(atomicQuery, qrToken).Scan(
		&ticket.ID, &ticket.TempleID, &ticket.VipPujaID, &ticket.UserID, &ticket.DevoteeName,
		&ticket.DevoteeGotram, &ticket.DevoteeNakshatram, &ticket.DevoteePhone,
		&visitDate, &ticket.SlotTime, &ticket.TicketType, &ticket.QRToken,
		&ticket.Status, &ticket.AmountPaidMinor, &ticket.AmountPaid,
		&ticket.PaymentMode, &checkInTS, &ticket.CreatedAt,
	)
	if err == nil {
		ticket.VisitDate = visitDate.Format("2006-01-02")
		ticket.CheckInTimestamp = checkInTS
		ticket.Currency = "INR"
		return &ticket, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		// Zero rows updated. Look up the row once to return "already checked in" or "not found".
		var existingStatus string
		var existingTS *time.Time
		lookupErr := s.DB.QueryRow(`SELECT status, check_in_timestamp FROM temple_vip_tickets WHERE qr_token = $1`, qrToken).Scan(&existingStatus, &existingTS)
		if lookupErr == nil {
			if existingStatus == "CHECKED_IN" {
				return nil, fmt.Errorf("ticket was ALREADY CHECKED IN at %v", existingTS)
			}
			return nil, fmt.Errorf("ticket cannot be checked in with status '%s'", existingStatus)
		}
		return nil, fmt.Errorf("ticket not found or invalid QR token")
	}

	return nil, fmt.Errorf("failed to verify ticket: %w", err)
}
