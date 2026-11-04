package db

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

	query := `
		INSERT INTO temple_vip_tickets (
			temple_id, vip_puja_id, user_id, devotee_name, devotee_gotram, 
			devotee_nakshatram, devotee_phone, visit_date, slot_time, 
			ticket_type, qr_token, status, amount_paid, payment_mode
		) 
		VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10, $11, 'CONFIRMED', $12, $13) 
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
		AmountPaid:        req.AmountPaid,
		PaymentMode:       req.PaymentMode,
	}

	err := s.DB.QueryRow(
		query, req.TempleID, req.VipPujaID, req.UserID, req.DevoteeName,
		req.DevoteeGotram, req.DevoteeNakshatram, req.DevoteePhone,
		req.VisitDate, req.SlotTime, req.TicketType, qrToken,
		req.AmountPaid, req.PaymentMode,
	).Scan(&ticket.ID, &ticket.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to issue temple ticket: %w", err)
	}
	return &ticket, nil
}

// VerifyTempleTicket handles fast gate verification and updates ticket status to CHECKED_IN
func (s *Storage) VerifyTempleTicket(qrToken string) (*models.TempleVipTicket, error) {
	var ticket models.TempleVipTicket
	var checkInTS *time.Time
	var visitDate time.Time

	query := `
		SELECT id, temple_id, vip_puja_id, COALESCE(user_id::text, ''), devotee_name, 
		       COALESCE(devotee_gotram, ''), COALESCE(devotee_nakshatram, ''), devotee_phone, 
		       visit_date, slot_time, ticket_type, qr_token, status, amount_paid, 
		       payment_mode, check_in_timestamp, created_at 
		FROM temple_vip_tickets 
		WHERE qr_token = $1
	`
	err := s.DB.QueryRow(query, qrToken).Scan(
		&ticket.ID, &ticket.TempleID, &ticket.VipPujaID, &ticket.UserID, &ticket.DevoteeName,
		&ticket.DevoteeGotram, &ticket.DevoteeNakshatram, &ticket.DevoteePhone,
		&visitDate, &ticket.SlotTime, &ticket.TicketType, &ticket.QRToken,
		&ticket.Status, &ticket.AmountPaid, &ticket.PaymentMode, &checkInTS, &ticket.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("ticket not found or invalid QR token: %w", err)
	}
	ticket.VisitDate = visitDate.Format("2006-01-02")
	ticket.CheckInTimestamp = checkInTS

	if ticket.Status == "CHECKED_IN" {
		return &ticket, fmt.Errorf("ticket was ALREADY CHECKED IN at %v", checkInTS)
	}

	// Mark checked in
	now := time.Now()
	_, err = s.DB.Exec(`UPDATE temple_vip_tickets SET status = 'CHECKED_IN', check_in_timestamp = $2 WHERE id = $1`, ticket.ID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to record ticket check-in: %w", err)
	}
	ticket.Status = "CHECKED_IN"
	ticket.CheckInTimestamp = &now

	return &ticket, nil
}
