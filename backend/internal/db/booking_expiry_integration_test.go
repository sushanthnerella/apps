package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/purohit-platform/backend/internal/models"
)

func TestBookingExpiryAndScheduledClashesPostgres(t *testing.T) {
	s, conn, _ := createIsolatedAuthTestDB(t)
	ctx := context.Background()
	var user, priest, event, ghat, ritual, service string
	queries := []struct {
		sql  string
		args []any
		out  *string
	}{
		{`INSERT INTO users(phone,full_name) VALUES ('+910000000001','Test') RETURNING id::text`, nil, &user},
		{`INSERT INTO priests(phone,full_name,sampradaya) VALUES ('+910000000002','Test','Smartha') RETURNING id::text`, nil, &priest},
		{`INSERT INTO events(slug,name,status,booking_enabled,starts_at,ends_at) VALUES ('test','Test','ACTIVE',true,now()-interval '1 day',now()+interval '2 days') RETURNING id::text`, nil, &event},
	}
	for _, q := range queries {
		if err := conn.QueryRowContext(ctx, q.sql, q.args...).Scan(q.out); err != nil {
			t.Fatal(err)
		}
	}
	if err := conn.QueryRowContext(ctx, `INSERT INTO ghats(event_id,name) VALUES ($1,'Test') RETURNING id::text`, event).Scan(&ghat); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRowContext(ctx, `INSERT INTO event_rituals(event_id,name,duration_minutes) VALUES ($1,'Test',60) RETURNING id::text`, event).Scan(&ritual); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO event_participations(event_id,priest_id,status) VALUES ($1,$2,'APPROVED')`, event, priest); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRowContext(ctx, `INSERT INTO event_priest_services(event_id,priest_id,ritual_id,price_minor) VALUES ($1,$2,$3,100) RETURNING id::text`, event, priest, ritual).Scan(&service); err != nil {
		t.Fatal(err)
	}

	var expired, future, slot string
	if err := conn.QueryRowContext(ctx, `INSERT INTO bookings(user_id,priest_id,total_amount,event_address,request_expires_at) VALUES ($1,$2,1,'Test',now()-interval '1 minute') RETURNING id::text`, user, priest).Scan(&expired); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRowContext(ctx, `INSERT INTO bookings(user_id,priest_id,total_amount,event_address,request_expires_at) VALUES ($1,$2,1,'Test',now()+interval '1 minute') RETURNING id::text`, user, priest).Scan(&future); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRowContext(ctx, `INSERT INTO priest_availability_slots(priest_id,slot_date,start_time,end_time,is_booked,booked_booking_id) VALUES ($1,current_date,'09:00','10:00',true,$2) RETURNING id::text`, priest, expired).Scan(&slot); err != nil {
		t.Fatal(err)
	}
	n, err := s.ExpireEventRequests(ctx)
	if err != nil || n != 1 {
		t.Fatalf("expire: count=%d err=%v", n, err)
	}
	var status, reason string
	var booked bool
	var linked bool
	if err := conn.QueryRowContext(ctx, `SELECT status,cancellation_reason FROM bookings WHERE id=$1`, expired).Scan(&status, &reason); err != nil {
		t.Fatal(err)
	}
	if status != "CANCELLED" || reason != "request_expired" {
		t.Fatalf("expired booking: %s %s", status, reason)
	}
	if err := conn.QueryRowContext(ctx, `SELECT is_booked,booked_booking_id IS NOT NULL FROM priest_availability_slots WHERE id=$1`, slot).Scan(&booked, &linked); err != nil {
		t.Fatal(err)
	}
	if booked || linked {
		t.Fatal("expired booking did not release its slot")
	}
	if err := conn.QueryRowContext(ctx, `SELECT status FROM bookings WHERE id=$1`, future).Scan(&status); err != nil || status != "REQUESTED" {
		t.Fatalf("future booking: status=%s err=%v", status, err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	scheduled := now.Add(3 * time.Hour)
	if _, err := conn.ExecContext(ctx, `INSERT INTO bookings(user_id,priest_id,total_amount,event_address,event_id,status,booking_type,scheduled_at,agreement_snapshot) VALUES ($1,$2,1,'Test',$3,'ACCEPTED','EVENT_SCHEDULED',$4,'{"duration_minutes":60}')`, user, priest, event, scheduled); err != nil {
		t.Fatal(err)
	}
	req := models.CreateEventBookingRequest{PriestID: priest, GhatID: ghat, ServiceID: service, When: "SCHEDULED", ScheduledAt: &scheduled, FamilySize: 1}
	if _, err := s.CreateEventBooking(ctx, user, event, req, now); !errors.Is(err, ErrPriestBusy) {
		t.Fatalf("overlap: expected busy, got %v", err)
	}
	adjacent := scheduled.Add(time.Hour)
	req.ScheduledAt = &adjacent
	if _, err := s.CreateEventBooking(ctx, user, event, req, now); err != nil {
		t.Fatalf("adjacent booking should succeed: %v", err)
	}
}
