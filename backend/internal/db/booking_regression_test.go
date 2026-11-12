package db

import (
	"strings"
	"testing"

	"github.com/purohit-platform/backend/internal/models"
)

// TestAppLayerInvalidBookingReferencesRejected proves that CreateBooking does not silently
// default invalid pooja references to another pooja, and correctly fails on FK constraint.
func TestAppLayerInvalidBookingReferencesRejected(t *testing.T) {
	storage, _, _ := createIsolatedAuthTestDB(t)

	// Register a devotee first
	devoteeID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876543001",
		FullName:      "Devotee Valid",
		ConsentAgreed: true,
		Password:      "SecurePassword123!",
	})
	if err != nil {
		t.Fatalf("failed registering devotee: %v", err)
	}

	// 1. Attempt CreateBooking with invalid non-existent PoojaID
	invalidPoojaID := "00000000-0000-0000-0000-000000000999"
	_, _, err = storage.CreateBooking(models.BookingCreateRequest{
		UserID:       devoteeID,
		PoojaID:      invalidPoojaID,
		EventAddress: "Hyderabad",
		TotalAmount:  3500.0,
	})
	if err == nil {
		t.Fatalf("expected CreateBooking to fail with invalid PoojaID, but it succeeded")
	}
	if !strings.Contains(err.Error(), "foreign key") && !strings.Contains(err.Error(), "bookings_pooja_id_fkey") {
		t.Errorf("expected foreign key error, got: %v", err)
	}

	// 2. Query a valid pooja from catalog
	catalog, err := storage.GetCatalog("", "", "en")
	if err != nil || len(catalog) == 0 {
		t.Fatalf("failed to get catalog: %v", err)
	}
	validPoojaID, ok := catalog[0]["id"].(string)
	if !ok || validPoojaID == "" {
		t.Fatalf("failed to extract valid pooja ID from catalog")
	}

	// 3. Attempt CreateBooking with valid PoojaID -> must succeed
	bookingID, _, err := storage.CreateBooking(models.BookingCreateRequest{
		UserID:       devoteeID,
		PoojaID:      validPoojaID,
		EventAddress: "Hyderabad",
		TotalAmount:  3500.0,
	})
	if err != nil {
		t.Fatalf("CreateBooking with valid PoojaID failed: %v", err)
	}
	if bookingID == "" {
		t.Errorf("expected booking with valid ID")
	}
}

// TestAppLayerUnknownPersonalAndScheduleFieldsRemainUnknown proves that when personal or
// schedule details are omitted or unknown, CreateBooking, GetBookingDetails, and GetBookings
// do not fabricate fake default values.
func TestAppLayerUnknownPersonalAndScheduleFieldsRemainUnknown(t *testing.T) {
	storage, _, _ := createIsolatedAuthTestDB(t)

	// Register devotee without gotram/nakshatram/rashi
	devoteeID, err := storage.RegisterDevotee(models.DevoteeRegistrationRequest{
		Phone:         "+919876543002",
		FullName:      "Devotee Anamika",
		ConsentAgreed: true,
		Password:      "SecurePassword123!",
	})
	if err != nil {
		t.Fatalf("failed registering devotee: %v", err)
	}

	catalog, err := storage.GetCatalog("", "", "en")
	if err != nil || len(catalog) == 0 {
		t.Fatalf("failed to get catalog: %v", err)
	}
	validPoojaID := catalog[0]["id"].(string)

	// Create booking leaving personal and schedule fields blank
	bookingID, _, err := storage.CreateBooking(models.BookingCreateRequest{
		UserID:            devoteeID,
		PoojaID:           validPoojaID,
		EventAddress:      "Banjara Hills, Hyderabad",
		TotalAmount:       3000.0,
		ScheduledDate:     "",
		SlotTime:          "",
		DevoteeGotram:     "",
		DevoteeNakshatram: "",
		DevoteeRashi:      "",
		SankalpamPurpose:  "",
	})
	if err != nil {
		t.Fatalf("CreateBooking failed: %v", err)
	}

	// 1. Verify GetBookingDetails does not fabricate defaults
	details, err := storage.GetBookingDetails(bookingID)
	if err != nil {
		t.Fatalf("GetBookingDetails failed: %v", err)
	}
	if details.DevoteeGotram != "" {
		t.Errorf("expected empty DevoteeGotram from GetBookingDetails, got fabricated: %q", details.DevoteeGotram)
	}
	if details.DevoteeNakshatram != "" {
		t.Errorf("expected empty DevoteeNakshatram from GetBookingDetails, got fabricated: %q", details.DevoteeNakshatram)
	}
	if details.DevoteeRashi != "" {
		t.Errorf("expected empty DevoteeRashi from GetBookingDetails, got fabricated: %q", details.DevoteeRashi)
	}
	if details.ScheduledDate != "" {
		t.Errorf("expected empty ScheduledDate from GetBookingDetails, got fabricated: %q", details.ScheduledDate)
	}
	if details.SlotTime != "" {
		t.Errorf("expected empty SlotTime from GetBookingDetails, got fabricated: %q", details.SlotTime)
	}
	if details.SankalpamPurpose != "" {
		t.Errorf("expected empty SankalpamPurpose from GetBookingDetails, got fabricated: %q", details.SankalpamPurpose)
	}

	// 2. Verify GetBookings does not fabricate defaults
	bookings, err := storage.GetBookings("", devoteeID, "")
	if err != nil {
		t.Fatalf("GetBookings failed: %v", err)
	}
	if len(bookings) == 0 {
		t.Fatalf("expected at least 1 booking returned")
	}
	b := bookings[0]
	if b.DevoteeGotram != "" {
		t.Errorf("expected empty DevoteeGotram from GetBookings, got fabricated: %q", b.DevoteeGotram)
	}
	if b.ScheduledDate != "" {
		t.Errorf("expected empty ScheduledDate from GetBookings, got fabricated: %q", b.ScheduledDate)
	}
	if b.SlotTime != "" {
		t.Errorf("expected empty SlotTime from GetBookings, got fabricated: %q", b.SlotTime)
	}
	if b.SankalpamPurpose != "" {
		t.Errorf("expected empty SankalpamPurpose from GetBookings, got fabricated: %q", b.SankalpamPurpose)
	}
}
