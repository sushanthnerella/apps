package lifecycle_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/purohit-platform/backend/internal/lifecycle"
)

func TestCanonicalStateSequence(t *testing.T) {
	priestID := "priest-uuid-123"
	devoteeID := "devotee-uuid-456"
	validOTP := "7429"

	steps := []struct {
		name string
		req  lifecycle.TransitionRequest
	}{
		{
			name: "REQUESTED -> ACCEPTED by assigned priest",
			req: lifecycle.TransitionRequest{
				CurrentStatus:    "REQUESTED",
				TargetStatus:     "ACCEPTED",
				CallerRole:       "priest",
				CallerID:         priestID,
				AssignedPriestID: priestID,
				OwnerDevoteeID:   devoteeID,
			},
		},
		{
			name: "ACCEPTED -> IN_TRANSIT by assigned priest",
			req: lifecycle.TransitionRequest{
				CurrentStatus:    "ACCEPTED",
				TargetStatus:     "IN_TRANSIT",
				CallerRole:       "priest",
				CallerID:         priestID,
				AssignedPriestID: priestID,
				OwnerDevoteeID:   devoteeID,
			},
		},
		{
			name: "IN_TRANSIT -> ARRIVED by assigned priest",
			req: lifecycle.TransitionRequest{
				CurrentStatus:    "IN_TRANSIT",
				TargetStatus:     "ARRIVED",
				CallerRole:       "priest",
				CallerID:         priestID,
				AssignedPriestID: priestID,
				OwnerDevoteeID:   devoteeID,
			},
		},
		{
			name: "ARRIVED -> POOJA_STARTED by assigned priest",
			req: lifecycle.TransitionRequest{
				CurrentStatus:    "ARRIVED",
				TargetStatus:     "POOJA_STARTED",
				CallerRole:       "priest",
				CallerID:         priestID,
				AssignedPriestID: priestID,
				OwnerDevoteeID:   devoteeID,
			},
		},
		{
			name: "POOJA_STARTED -> COMPLETED by assigned priest with valid OTP",
			req: lifecycle.TransitionRequest{
				CurrentStatus:    "POOJA_STARTED",
				TargetStatus:     "COMPLETED",
				CallerRole:       "priest",
				CallerID:         priestID,
				AssignedPriestID: priestID,
				OwnerDevoteeID:   devoteeID,
				ProvidedOTP:      validOTP,
				ExpectedOTP:      validOTP,
			},
		},
	}

	for _, tc := range steps {
		t.Run(tc.name, func(t *testing.T) {
			if err := lifecycle.ValidateTransition(tc.req); err != nil {
				t.Fatalf("expected transition to succeed, got error: %v", err)
			}
		})
	}
}

func TestSkippedTransitionsRejected(t *testing.T) {
	priestID := "priest-uuid-123"
	devoteeID := "devotee-uuid-456"

	skippedCases := []struct {
		from string
		to   string
	}{
		{"REQUESTED", "IN_TRANSIT"},
		{"REQUESTED", "ARRIVED"},
		{"REQUESTED", "POOJA_STARTED"},
		{"REQUESTED", "COMPLETED"},
		{"ACCEPTED", "ARRIVED"},
		{"ACCEPTED", "POOJA_STARTED"},
		{"ACCEPTED", "COMPLETED"},
		{"IN_TRANSIT", "POOJA_STARTED"},
		{"IN_TRANSIT", "COMPLETED"},
		{"ARRIVED", "COMPLETED"},
	}

	for _, tc := range skippedCases {
		t.Run(tc.from+"->"+tc.to, func(t *testing.T) {
			req := lifecycle.TransitionRequest{
				CurrentStatus:    tc.from,
				TargetStatus:     tc.to,
				CallerRole:       "priest",
				CallerID:         priestID,
				AssignedPriestID: priestID,
				OwnerDevoteeID:   devoteeID,
				ProvidedOTP:      "1234",
				ExpectedOTP:      "1234",
			}
			err := lifecycle.ValidateTransition(req)
			if err == nil {
				t.Fatalf("expected error for skipped transition %s -> %s, but got nil", tc.from, tc.to)
			}
			if !errors.Is(err, lifecycle.ErrInvalidTransition) {
				t.Fatalf("expected ErrInvalidTransition, got: %v", err)
			}
		})
	}
}

func TestBackwardTransitionsRejected(t *testing.T) {
	priestID := "priest-uuid-123"
	devoteeID := "devotee-uuid-456"

	backwardCases := []struct {
		from string
		to   string
	}{
		{"ACCEPTED", "REQUESTED"},
		{"IN_TRANSIT", "ACCEPTED"},
		{"ARRIVED", "IN_TRANSIT"},
		{"ARRIVED", "ACCEPTED"},
		{"POOJA_STARTED", "ARRIVED"},
		{"POOJA_STARTED", "IN_TRANSIT"},
		{"POOJA_STARTED", "ACCEPTED"},
	}

	for _, tc := range backwardCases {
		t.Run(tc.from+"->"+tc.to, func(t *testing.T) {
			req := lifecycle.TransitionRequest{
				CurrentStatus:    tc.from,
				TargetStatus:     tc.to,
				CallerRole:       "priest",
				CallerID:         priestID,
				AssignedPriestID: priestID,
				OwnerDevoteeID:   devoteeID,
			}
			err := lifecycle.ValidateTransition(req)
			if err == nil {
				t.Fatalf("expected error for backward transition %s -> %s, but got nil", tc.from, tc.to)
			}
			if !errors.Is(err, lifecycle.ErrInvalidTransition) {
				t.Fatalf("expected ErrInvalidTransition, got: %v", err)
			}
		})
	}
}

func TestRepeatedTransitionsRejected(t *testing.T) {
	priestID := "priest-uuid-123"
	devoteeID := "devotee-uuid-456"

	states := []string{"REQUESTED", "ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED"}

	for _, state := range states {
		t.Run(state+"->"+state, func(t *testing.T) {
			req := lifecycle.TransitionRequest{
				CurrentStatus:    state,
				TargetStatus:     state,
				CallerRole:       "priest",
				CallerID:         priestID,
				AssignedPriestID: priestID,
				OwnerDevoteeID:   devoteeID,
			}
			err := lifecycle.ValidateTransition(req)
			if err == nil {
				t.Fatalf("expected error for repeated transition in %s, got nil", state)
			}
			if !errors.Is(err, lifecycle.ErrRepeatedTransition) {
				t.Fatalf("expected ErrRepeatedTransition, got: %v", err)
			}
		})
	}
}

func TestTerminalStatesAreImmutable(t *testing.T) {
	priestID := "priest-uuid-123"
	devoteeID := "devotee-uuid-456"

	terminalStates := []string{"COMPLETED", "CANCELLED"}
	targets := []string{"REQUESTED", "ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED", "COMPLETED", "CANCELLED"}

	for _, term := range terminalStates {
		for _, target := range targets {
			t.Run(term+"->"+target, func(t *testing.T) {
				req := lifecycle.TransitionRequest{
					CurrentStatus:    term,
					TargetStatus:     target,
					CallerRole:       "priest",
					CallerID:         priestID,
					AssignedPriestID: priestID,
					OwnerDevoteeID:   devoteeID,
				}
				err := lifecycle.ValidateTransition(req)
				if err == nil {
					t.Fatalf("expected error transitioning from terminal state %s to %s, got nil", term, target)
				}
				if !errors.Is(err, lifecycle.ErrTerminalState) {
					t.Fatalf("expected ErrTerminalState, got: %v", err)
				}
			})
		}
	}
}

func TestDevoteeCannotTriggerPriestTransitions(t *testing.T) {
	priestID := "priest-uuid-123"
	devoteeID := "devotee-uuid-456"

	priestTransitions := []struct {
		from string
		to   string
	}{
		{"REQUESTED", "ACCEPTED"},
		{"ACCEPTED", "IN_TRANSIT"},
		{"IN_TRANSIT", "ARRIVED"},
		{"ARRIVED", "POOJA_STARTED"},
		{"POOJA_STARTED", "COMPLETED"},
	}

	for _, tc := range priestTransitions {
		t.Run("Devotee_"+tc.from+"->"+tc.to, func(t *testing.T) {
			req := lifecycle.TransitionRequest{
				CurrentStatus:    tc.from,
				TargetStatus:     tc.to,
				CallerRole:       "devotee",
				CallerID:         devoteeID,
				AssignedPriestID: priestID,
				OwnerDevoteeID:   devoteeID,
				ProvidedOTP:      "1234",
				ExpectedOTP:      "1234",
			}
			err := lifecycle.ValidateTransition(req)
			if err == nil {
				t.Fatalf("expected devotee transition %s -> %s to be forbidden, but got nil", tc.from, tc.to)
			}
			if !errors.Is(err, lifecycle.ErrUnauthorizedRole) {
				t.Fatalf("expected ErrUnauthorizedRole, got: %v", err)
			}
		})
	}
}

func TestUnassignedPriestRejected(t *testing.T) {
	assignedPriestID := "priest-assigned"
	unauthorizedPriestID := "priest-imposter"
	devoteeID := "devotee-uuid-456"

	req := lifecycle.TransitionRequest{
		CurrentStatus:    "ACCEPTED",
		TargetStatus:     "IN_TRANSIT",
		CallerRole:       "priest",
		CallerID:         unauthorizedPriestID,
		AssignedPriestID: assignedPriestID,
		OwnerDevoteeID:   devoteeID,
	}

	err := lifecycle.ValidateTransition(req)
	if err == nil {
		t.Fatalf("expected unassigned priest transition to fail, got nil")
	}
	if !errors.Is(err, lifecycle.ErrUnauthorizedPriest) {
		t.Fatalf("expected ErrUnauthorizedPriest, got: %v", err)
	}
}

func TestCompletionOTPValidation(t *testing.T) {
	priestID := "priest-uuid-123"
	devoteeID := "devotee-uuid-456"
	expectedOTP := "8841"

	t.Run("Missing OTP", func(t *testing.T) {
		req := lifecycle.TransitionRequest{
			CurrentStatus:    "POOJA_STARTED",
			TargetStatus:     "COMPLETED",
			CallerRole:       "priest",
			CallerID:         priestID,
			AssignedPriestID: priestID,
			OwnerDevoteeID:   devoteeID,
			ProvidedOTP:      "",
			ExpectedOTP:      expectedOTP,
		}
		err := lifecycle.ValidateTransition(req)
		if err == nil {
			t.Fatalf("expected missing OTP error, got nil")
		}
		if !errors.Is(err, lifecycle.ErrMissingOTP) {
			t.Fatalf("expected ErrMissingOTP, got: %v", err)
		}
	})

	t.Run("Incorrect OTP", func(t *testing.T) {
		req := lifecycle.TransitionRequest{
			CurrentStatus:    "POOJA_STARTED",
			TargetStatus:     "COMPLETED",
			CallerRole:       "priest",
			CallerID:         priestID,
			AssignedPriestID: priestID,
			OwnerDevoteeID:   devoteeID,
			ProvidedOTP:      "0000",
			ExpectedOTP:      expectedOTP,
		}
		err := lifecycle.ValidateTransition(req)
		if err == nil {
			t.Fatalf("expected invalid OTP error, got nil")
		}
		if !errors.Is(err, lifecycle.ErrInvalidOTP) {
			t.Fatalf("expected ErrInvalidOTP, got: %v", err)
		}
		// Invariant: Error string must never leak the secret expected OTP!
		if strings.Contains(err.Error(), expectedOTP) {
			t.Fatalf("SECURITY VIOLATION: error message leaked expected OTP '%s'", expectedOTP)
		}
	})

	t.Run("Correct OTP", func(t *testing.T) {
		req := lifecycle.TransitionRequest{
			CurrentStatus:    "POOJA_STARTED",
			TargetStatus:     "COMPLETED",
			CallerRole:       "priest",
			CallerID:         priestID,
			AssignedPriestID: priestID,
			OwnerDevoteeID:   devoteeID,
			ProvidedOTP:      expectedOTP,
			ExpectedOTP:      expectedOTP,
		}
		if err := lifecycle.ValidateTransition(req); err != nil {
			t.Fatalf("expected valid OTP transition to succeed, got: %v", err)
		}
	})
}

func TestPilotCancellationPolicy(t *testing.T) {
	priestID := "priest-uuid-123"
	devoteeID := "devotee-uuid-456"

	t.Run("Devotee can cancel in REQUESTED", func(t *testing.T) {
		req := lifecycle.TransitionRequest{
			CurrentStatus:    "REQUESTED",
			TargetStatus:     "CANCELLED",
			CallerRole:       "devotee",
			CallerID:         devoteeID,
			OwnerDevoteeID:   devoteeID,
			AssignedPriestID: priestID,
		}
		if err := lifecycle.ValidateTransition(req); err != nil {
			t.Fatalf("expected devotee cancellation in REQUESTED to succeed, got: %v", err)
		}
	})

	t.Run("Devotee can cancel in ACCEPTED", func(t *testing.T) {
		req := lifecycle.TransitionRequest{
			CurrentStatus:    "ACCEPTED",
			TargetStatus:     "CANCELLED",
			CallerRole:       "devotee",
			CallerID:         devoteeID,
			OwnerDevoteeID:   devoteeID,
			AssignedPriestID: priestID,
		}
		if err := lifecycle.ValidateTransition(req); err != nil {
			t.Fatalf("expected devotee cancellation in ACCEPTED to succeed, got: %v", err)
		}
	})

	t.Run("Devotee can cancel in IN_TRANSIT", func(t *testing.T) {
		req := lifecycle.TransitionRequest{
			CurrentStatus:    "IN_TRANSIT",
			TargetStatus:     "CANCELLED",
			CallerRole:       "devotee",
			CallerID:         devoteeID,
			OwnerDevoteeID:   devoteeID,
			AssignedPriestID: priestID,
		}
		if err := lifecycle.ValidateTransition(req); err != nil {
			t.Fatalf("expected devotee cancellation in IN_TRANSIT to succeed, got: %v", err)
		}
	})

	t.Run("Devotee CANNOT cancel in ARRIVED", func(t *testing.T) {
		req := lifecycle.TransitionRequest{
			CurrentStatus:    "ARRIVED",
			TargetStatus:     "CANCELLED",
			CallerRole:       "devotee",
			CallerID:         devoteeID,
			OwnerDevoteeID:   devoteeID,
			AssignedPriestID: priestID,
		}
		err := lifecycle.ValidateTransition(req)
		if err == nil {
			t.Fatalf("expected devotee cancellation in ARRIVED to be rejected, got nil")
		}
		if !errors.Is(err, lifecycle.ErrUnauthorizedCancellation) {
			t.Fatalf("expected ErrUnauthorizedCancellation, got: %v", err)
		}
	})

	t.Run("Devotee CANNOT cancel in POOJA_STARTED", func(t *testing.T) {
		req := lifecycle.TransitionRequest{
			CurrentStatus:    "POOJA_STARTED",
			TargetStatus:     "CANCELLED",
			CallerRole:       "devotee",
			CallerID:         devoteeID,
			OwnerDevoteeID:   devoteeID,
			AssignedPriestID: priestID,
		}
		err := lifecycle.ValidateTransition(req)
		if err == nil {
			t.Fatalf("expected devotee cancellation in POOJA_STARTED to be rejected, got nil")
		}
		if !errors.Is(err, lifecycle.ErrUnauthorizedCancellation) {
			t.Fatalf("expected ErrUnauthorizedCancellation, got: %v", err)
		}
	})

	t.Run("Admin CAN cancel in ARRIVED and POOJA_STARTED", func(t *testing.T) {
		for _, st := range []string{"REQUESTED", "ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED"} {
			req := lifecycle.TransitionRequest{
				CurrentStatus:    st,
				TargetStatus:     "CANCELLED",
				CallerRole:       "admin",
				CallerID:         "admin-uuid",
				AssignedPriestID: priestID,
				OwnerDevoteeID:   devoteeID,
			}
			if err := lifecycle.ValidateTransition(req); err != nil {
				t.Fatalf("expected admin cancellation in %s to succeed, got: %v", st, err)
			}
		}
	})

	t.Run("Unauthorized Devotee cannot cancel another devotee's booking", func(t *testing.T) {
		req := lifecycle.TransitionRequest{
			CurrentStatus:    "ACCEPTED",
			TargetStatus:     "CANCELLED",
			CallerRole:       "devotee",
			CallerID:         "devotee-attacker",
			OwnerDevoteeID:   devoteeID,
			AssignedPriestID: priestID,
		}
		err := lifecycle.ValidateTransition(req)
		if err == nil {
			t.Fatalf("expected attacker cancellation to fail, got nil")
		}
		if !errors.Is(err, lifecycle.ErrUnauthorizedDevotee) {
			t.Fatalf("expected ErrUnauthorizedDevotee, got: %v", err)
		}
	})
}
