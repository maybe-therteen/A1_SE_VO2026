package domain

import (
	"errors"
	"testing"
	"time"
)

func TestSkiValidateAndState(t *testing.T) {
	ski := &Ski{
		ID:        1,
		Number:    "SKI-101",
		Type:      "Race",
		LengthCm:  175,
		Available: true,
	}

	if err := ski.Validate(); err != nil {
		t.Fatalf("expected valid ski, got error: %v", err)
	}

	if err := ski.MarkRented(); err != nil {
		t.Fatalf("expected ski to be rentable, got error: %v", err)
	}
	if ski.Available {
		t.Fatalf("expected ski to be unavailable after rental")
	}

	if err := ski.MarkRented(); !errors.Is(err, ErrSkiNotAvailable) {
		t.Fatalf("expected ErrSkiNotAvailable on second rental, got: %v", err)
	}

	ski.MarkReturned()
	if !ski.Available {
		t.Fatalf("expected ski to be available after return")
	}
}

func TestReservationComplete(t *testing.T) {
	res := &Reservation{
		ID:        1,
		SkiID:     1,
		SkiNumber: "SKI-101",
		SkiType:   "Allround",
		PersonID:  1,
		StartTime: time.Now().Add(-2 * time.Hour),
		Active:    true,
	}

	now := time.Now()
	if err := res.Complete(now); err != nil {
		t.Fatalf("expected complete to succeed, got: %v", err)
	}
	if res.Active || res.EndTime == nil {
		t.Fatalf("expected reservation to be inactive with EndTime set")
	}

	if err := res.Complete(now); !errors.Is(err, ErrAlreadyReturned) {
		t.Fatalf("expected ErrAlreadyReturned on second return, got: %v", err)
	}
}
