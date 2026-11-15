package models

import "time"

const (
	ParticipationSubmitted        = "SUBMITTED"
	ParticipationChangesRequested = "CHANGES_REQUESTED"
	ParticipationApproved         = "APPROVED"
	ParticipationRejected         = "REJECTED"
	ParticipationSuspended        = "SUSPENDED"

	BookingTypeEventNow       = "EVENT_NOW"
	BookingTypeEventScheduled = "EVENT_SCHEDULED"
)

type EventRitual struct {
	ID              string `json:"id"`
	EventID         string `json:"event_id"`
	Name            string `json:"name"`
	NameTe          string `json:"name_te"`
	NameHi          string `json:"name_hi"`
	DurationMinutes int    `json:"duration_minutes"`
	SortOrder       int    `json:"sort_order"`
	IsActive        bool   `json:"is_active"`
}

type UpsertRitualRequest struct {
	Name            string `json:"name"`
	NameTe          string `json:"name_te"`
	NameHi          string `json:"name_hi"`
	DurationMinutes int    `json:"duration_minutes"`
	SortOrder       int    `json:"sort_order"`
	IsActive        *bool  `json:"is_active"`
}

type PriestService struct {
	ID              string `json:"id"`
	RitualID        string `json:"ritual_id"`
	RitualName      string `json:"ritual_name"`
	RitualNameTe    string `json:"ritual_name_te"`
	RitualNameHi    string `json:"ritual_name_hi"`
	DurationMinutes int    `json:"duration_minutes"`
	PriceMinor      int64  `json:"price_minor"`
}

type EventCard struct {
	CardCode  string    `json:"card_code"`
	Issuer    string    `json:"issuer"`
	Status    string    `json:"status"`
	IssuedAt  time.Time `json:"issued_at"`
	EventName string    `json:"event_name"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
}

// Participation is what the priest app shows on its Pushkaralu tab.
type Participation struct {
	EventID        string          `json:"event_id"`
	Status         string          `json:"status"` // NOT_APPLIED or a Participation* value
	ReviewerReason string          `json:"reviewer_reason"`
	SubmittedAt    *time.Time      `json:"submitted_at"`
	Card           *EventCard      `json:"card"`
	Rituals        []EventRitual   `json:"rituals"`
	Services       []PriestService `json:"services"`
	Duty           *DutyState      `json:"duty"`
}

type DutyState struct {
	GhatID     string    `json:"ghat_id"`
	DutyStatus string    `json:"duty_status"`
	StartedAt  time.Time `json:"started_at"`
}

type SetServicesRequest struct {
	Services []struct {
		RitualID   string `json:"ritual_id"`
		PriceMinor int64  `json:"price_minor"`
	} `json:"services"`
}

type DutyRequest struct {
	GhatID     string `json:"ghat_id"`
	DutyStatus string `json:"duty_status"`
}

type ReviewParticipationRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// ParticipationRow is the CMS review queue row.
type ParticipationRow struct {
	PriestID        string     `json:"priest_id"`
	PriestName      string     `json:"priest_name"`
	Sampradaya      string     `json:"sampradaya"`
	ExperienceYears int        `json:"experience_years"`
	Status          string     `json:"status"`
	ReviewerReason  string     `json:"reviewer_reason"`
	SubmittedAt     time.Time  `json:"submitted_at"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
	ServicesCount   int        `json:"services_count"`
}

// GhatPriest is one row on the devotee's "priests at ghat" list. Location is never exposed.
type GhatPriest struct {
	PriestID        string          `json:"priest_id"`
	FullName        string          `json:"full_name"`
	RatingAvg       float64         `json:"rating_avg"`
	CompletedPoojas int             `json:"completed_poojas"`
	DutyStatus      string          `json:"duty_status"`
	LocationAgeSec  int64           `json:"location_age_seconds"`
	Services        []PriestService `json:"services"`
}

type GhatAvailability struct {
	GhatID         string `json:"ghat_id"`
	AvailableCount int    `json:"available_count"`
}

type CreateEventBookingRequest struct {
	PriestID    string     `json:"priest_id"`
	GhatID      string     `json:"ghat_id"`
	ServiceID   string     `json:"service_id"`
	When        string     `json:"when"` // NOW or SCHEDULED
	ScheduledAt *time.Time `json:"scheduled_at"`
	FamilySize  int        `json:"family_size"`
}

type EventBookingResponse struct {
	BookingID        string    `json:"booking_id"`
	Status           string    `json:"status"`
	BookingCode      string    `json:"booking_code"`
	TotalAmountMinor int64     `json:"total_amount_minor"`
	RequestExpiresAt time.Time `json:"request_expires_at"`
	ChatRoomID       string    `json:"chat_room_id"`
}

// PublicCardStatus is all the public QR page reveals: no documents, contact or location.
type PublicCardStatus struct {
	Status     string    `json:"status"`
	Issuer     string    `json:"issuer"`
	PriestName string    `json:"priest_name"`
	EventName  string    `json:"event_name"`
	ValidFrom  time.Time `json:"valid_from"`
	ValidUntil time.Time `json:"valid_until"`
}
