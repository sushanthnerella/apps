package lifecycle

import (
	"errors"
	"fmt"
	"strings"
)

// BookingStatus represents canonical booking lifecycle states.
type BookingStatus string

const (
	StatusRequested    BookingStatus = "REQUESTED"
	StatusDispatched   BookingStatus = "DISPATCHED"
	StatusAccepted     BookingStatus = "ACCEPTED"
	StatusInTransit    BookingStatus = "IN_TRANSIT"
	StatusArrived      BookingStatus = "ARRIVED"
	StatusPoojaStarted BookingStatus = "POOJA_STARTED"
	StatusCompleted    BookingStatus = "COMPLETED"
	StatusCancelled    BookingStatus = "CANCELLED"
)

// Role represents actors interacting with the booking lifecycle.
type Role string

const (
	RoleDevotee Role = "devotee"
	RolePriest  Role = "priest"
	RoleAdmin   Role = "admin"
	RoleSystem  Role = "system"
)

// Sentinel Errors
var (
	ErrInvalidTransition        = errors.New("invalid state transition")
	ErrRepeatedTransition       = errors.New("repeated state transition")
	ErrTerminalState            = errors.New("booking is in terminal state")
	ErrUnauthorizedRole         = errors.New("unauthorized role for state transition")
	ErrUnauthorizedPriest       = errors.New("access denied: you are not assigned to this booking")
	ErrUnauthorizedDevotee      = errors.New("access denied: you are not the owner of this booking")
	ErrUnauthorizedCancellation = errors.New("caller is not authorized to cancel this booking in its current state")
	ErrMissingOTP               = errors.New("completion OTP is required")
	ErrInvalidOTP               = errors.New("invalid completion OTP")
	ErrSlotUnavailable          = errors.New("priest availability slot is already booked")
	ErrBookingNotFound          = errors.New("booking not found")
)

// TransitionRequest encapsulates the context required to evaluate a state transition.
type TransitionRequest struct {
	CurrentStatus    string
	TargetStatus     string
	CallerRole       string
	CallerID         string
	AssignedPriestID string
	OwnerDevoteeID   string
	ProvidedOTP      string
	ExpectedOTP      string
}

// TransitionRule defines the criteria for moving from one state to another.
type TransitionRule struct {
	FromStatus   BookingStatus
	ToStatus     BookingStatus
	AllowedRoles []Role
	RequiresOTP  bool
}

// allowedTransitions defines the strict state transition table.
var allowedTransitions = map[BookingStatus]map[BookingStatus]TransitionRule{
	StatusRequested: {
		StatusAccepted: TransitionRule{
			FromStatus:   StatusRequested,
			ToStatus:     StatusAccepted,
			AllowedRoles: []Role{RolePriest, RoleAdmin, RoleSystem},
		},
		StatusCancelled: TransitionRule{
			FromStatus:   StatusRequested,
			ToStatus:     StatusCancelled,
			AllowedRoles: []Role{RoleDevotee, RolePriest, RoleAdmin, RoleSystem},
		},
	},
	StatusDispatched: {
		StatusAccepted: TransitionRule{
			FromStatus:   StatusDispatched,
			ToStatus:     StatusAccepted,
			AllowedRoles: []Role{RolePriest, RoleAdmin, RoleSystem},
		},
		StatusCancelled: TransitionRule{
			FromStatus:   StatusDispatched,
			ToStatus:     StatusCancelled,
			AllowedRoles: []Role{RoleDevotee, RolePriest, RoleAdmin, RoleSystem},
		},
	},
	StatusAccepted: {
		StatusInTransit: TransitionRule{
			FromStatus:   StatusAccepted,
			ToStatus:     StatusInTransit,
			AllowedRoles: []Role{RolePriest, RoleAdmin, RoleSystem},
		},
		StatusCancelled: TransitionRule{
			FromStatus:   StatusAccepted,
			ToStatus:     StatusCancelled,
			AllowedRoles: []Role{RoleDevotee, RolePriest, RoleAdmin, RoleSystem},
		},
	},
	StatusInTransit: {
		StatusArrived: TransitionRule{
			FromStatus:   StatusInTransit,
			ToStatus:     StatusArrived,
			AllowedRoles: []Role{RolePriest, RoleAdmin, RoleSystem},
		},
		StatusCancelled: TransitionRule{
			FromStatus:   StatusInTransit,
			ToStatus:     StatusCancelled,
			AllowedRoles: []Role{RoleDevotee, RoleAdmin, RoleSystem},
		},
	},
	StatusArrived: {
		StatusPoojaStarted: TransitionRule{
			FromStatus:   StatusArrived,
			ToStatus:     StatusPoojaStarted,
			AllowedRoles: []Role{RolePriest, RoleAdmin, RoleSystem},
		},
		StatusCancelled: TransitionRule{
			FromStatus:   StatusArrived,
			ToStatus:     StatusCancelled,
			AllowedRoles: []Role{RoleAdmin, RoleSystem},
		},
	},
	StatusPoojaStarted: {
		StatusCompleted: TransitionRule{
			FromStatus:   StatusPoojaStarted,
			ToStatus:     StatusCompleted,
			AllowedRoles: []Role{RolePriest, RoleAdmin, RoleSystem},
			RequiresOTP:  true,
		},
		StatusCancelled: TransitionRule{
			FromStatus:   StatusPoojaStarted,
			ToStatus:     StatusCancelled,
			AllowedRoles: []Role{RoleAdmin, RoleSystem},
		},
	},
	StatusCompleted: {}, // Terminal: no outbound transitions
	StatusCancelled: {}, // Terminal: no outbound transitions
}

// IsTerminal returns true if the status is an immutable end state.
func IsTerminal(st BookingStatus) bool {
	return st == StatusCompleted || st == StatusCancelled
}

// NormalizeRole maps role aliases to canonical Role constants.
func NormalizeRole(role string) Role {
	clean := strings.ToLower(strings.TrimSpace(role))
	switch clean {
	case "devotee", "user":
		return RoleDevotee
	case "priest":
		return RolePriest
	case "admin":
		return RoleAdmin
	case "system", "":
		return RoleSystem
	default:
		return Role(clean)
	}
}

// NormalizeStatus converts raw status strings to canonical BookingStatus.
func NormalizeStatus(status string) BookingStatus {
	return BookingStatus(strings.ToUpper(strings.TrimSpace(status)))
}

func isRoleAllowed(role Role, allowed []Role) bool {
	for _, a := range allowed {
		if a == role {
			return true
		}
	}
	return false
}

// ValidateTransition strictly validates a proposed state transition against the FSM,
// caller role authorization, booking assignment/ownership, and completion OTP integrity.
func ValidateTransition(req TransitionRequest) error {
	from := NormalizeStatus(req.CurrentStatus)
	to := NormalizeStatus(req.TargetStatus)

	// 1. Terminal state protection: immutable once COMPLETED or CANCELLED
	if IsTerminal(from) {
		return fmt.Errorf("%w: cannot transition from terminal state '%s' to '%s'", ErrTerminalState, from, to)
	}

	// 2. Reject repeated/redundant transitions
	if from == to {
		return fmt.Errorf("%w: booking is already in state '%s'", ErrRepeatedTransition, from)
	}

	// 3. Look up allowed transition rules
	targets, fromOk := allowedTransitions[from]
	if !fromOk {
		return fmt.Errorf("%w: invalid current status '%s'", ErrInvalidTransition, from)
	}

	rule, toOk := targets[to]
	if !toOk {
		return fmt.Errorf("%w: transition from '%s' to '%s' is not allowed", ErrInvalidTransition, from, to)
	}

	// 4. Role Authorization
	normalizedRole := NormalizeRole(req.CallerRole)
	if !isRoleAllowed(normalizedRole, rule.AllowedRoles) {
		if to == StatusCancelled {
			return fmt.Errorf("%w: role '%s' cannot cancel booking in state '%s'", ErrUnauthorizedCancellation, req.CallerRole, from)
		}
		return fmt.Errorf("%w: role '%s' cannot perform transition from '%s' to '%s'", ErrUnauthorizedRole, req.CallerRole, from, to)
	}

	// 5. Ownership & Assignment Verification
	switch normalizedRole {
	case RolePriest:
		// If the booking already has an assigned priest, only that priest can advance it
		if req.AssignedPriestID != "" && req.CallerID != "" && req.CallerID != req.AssignedPriestID {
			return ErrUnauthorizedPriest
		}
	case RoleDevotee:
		// Devotees can only cancel bookings; priest transitions are forbidden
		if to != StatusCancelled {
			return fmt.Errorf("%w: devotees can only cancel bookings; priest transitions are forbidden", ErrUnauthorizedRole)
		}
		if req.OwnerDevoteeID != "" && req.CallerID != "" && req.CallerID != req.OwnerDevoteeID {
			return ErrUnauthorizedDevotee
		}
	}

	// 6. Completion OTP Verification
	if rule.RequiresOTP {
		trimmedOTP := strings.TrimSpace(req.ProvidedOTP)
		if trimmedOTP == "" {
			return ErrMissingOTP
		}
		if trimmedOTP != strings.TrimSpace(req.ExpectedOTP) {
			// Invariant: Never leak expected OTP in the error message
			return ErrInvalidOTP
		}
	}

	return nil
}
