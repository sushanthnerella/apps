package models

import "time"

// Event lifecycle states (PRD v2 EVT-002).
const (
	EventDraft     = "DRAFT"
	EventScheduled = "SCHEDULED"
	EventActive    = "ACTIVE"
	EventPaused    = "PAUSED"
	EventEnded     = "ENDED"
	EventArchived  = "ARCHIVED"
)

// EventTransitions lists the lifecycle moves an admin may make.
var EventTransitions = map[string][]string{
	EventDraft:     {EventScheduled},
	EventScheduled: {EventDraft, EventActive},
	EventActive:    {EventPaused, EventEnded},
	EventPaused:    {EventActive, EventEnded},
	EventEnded:     {EventArchived},
}

type Ghat struct {
	ID        string   `json:"id"`
	EventID   string   `json:"event_id"`
	Name      string   `json:"name"`
	NameTe    string   `json:"name_te"`
	NameHi    string   `json:"name_hi"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	SortOrder int      `json:"sort_order"`
	IsActive  bool     `json:"is_active"`
}

type Event struct {
	ID                  string    `json:"id"`
	Slug                string    `json:"slug"`
	Name                string    `json:"name"`
	NameTe              string    `json:"name_te"`
	NameHi              string    `json:"name_hi"`
	River               string    `json:"river"`
	Status              string    `json:"status"`
	BookingEnabled      bool      `json:"booking_enabled"`
	CardIssuanceEnabled bool      `json:"card_issuance_enabled"`
	GovtCardsEnabled    bool      `json:"govt_cards_enabled"`
	StartsAt            time.Time `json:"starts_at"`
	EndsAt              time.Time `json:"ends_at"`
	ConfigVersion       int64     `json:"config_version"`
	UpdatedAt           time.Time `json:"updated_at"`
	Ghats               []Ghat    `json:"ghats"`
}

type CreateEventRequest struct {
	Slug     string    `json:"slug"`
	Name     string    `json:"name"`
	NameTe   string    `json:"name_te"`
	NameHi   string    `json:"name_hi"`
	River    string    `json:"river"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}

// UpdateEventRequest is a partial update; nil fields are left unchanged.
type UpdateEventRequest struct {
	Status              *string `json:"status"`
	BookingEnabled      *bool   `json:"booking_enabled"`
	CardIssuanceEnabled *bool   `json:"card_issuance_enabled"`
}

type UpsertGhatRequest struct {
	Name      string   `json:"name"`
	NameTe    string   `json:"name_te"`
	NameHi    string   `json:"name_hi"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	SortOrder int      `json:"sort_order"`
	IsActive  *bool    `json:"is_active"`
}

type AuditEntry struct {
	ID          int64     `json:"id"`
	ActorID     *string   `json:"actor_id"`
	ActorRole   string    `json:"actor_role"`
	Action      string    `json:"action"`
	EntityType  string    `json:"entity_type"`
	EntityID    string    `json:"entity_id"`
	BeforeState []byte    `json:"-"`
	AfterState  []byte    `json:"-"`
	Before      any       `json:"before"`
	After       any       `json:"after"`
	CreatedAt   time.Time `json:"created_at"`
}
