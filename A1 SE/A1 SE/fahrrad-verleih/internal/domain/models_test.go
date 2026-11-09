package domain

import (
	"errors"
	"testing"
	"time"
)

func TestBikeValidateSetsDefaultRate(t *testing.T) {
	b := Bike{Number: " bike-1 ", Type: "E-Bike", FrameSizeCm: 52}
	if err := b.Validate(); err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if b.Number != "BIKE-1" {
		t.Errorf("Nummer nicht normalisiert: %q", b.Number)
	}
	if b.HourlyRate != DefaultHourlyRates["E-Bike"] {
		t.Errorf("Standardpreis erwartet, bekommen %.2f", b.HourlyRate)
	}
}

func TestBikeValidateRejectsInvalid(t *testing.T) {
	cases := []Bike{
		{Number: "", Type: "City", FrameSizeCm: 50},
		{Number: "B-1", Type: "Einrad", FrameSizeCm: 50},
		{Number: "B-1", Type: "City", FrameSizeCm: 10},
		{Number: "B-1", Type: "City", FrameSizeCm: 50, HourlyRate: -1},
	}
	for i, c := range cases {
		if err := c.Validate(); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("Fall %d: ErrInvalidInput erwartet, bekommen %v", i, err)
		}
	}
}

func TestBillableHours(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		end  time.Time
		want int
	}{
		{start, 1},
		{start.Add(10 * time.Minute), 1},
		{start.Add(60 * time.Minute), 1},
		{start.Add(61 * time.Minute), 2},
		{start.Add(160 * time.Minute), 3},
	}
	for _, c := range cases {
		if got := BillableHours(start, c.end); got != c.want {
			t.Errorf("BillableHours(%v) = %d, erwartet %d", c.end.Sub(start), got, c.want)
		}
	}
}

func TestReservationLifecycleAndPrice(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	r := Reservation{ID: 1, PlannedStart: start, PlannedHours: 2, HourlyRate: 7, Status: StatusReserved}

	if err := r.Pickup(start.Add(5 * time.Minute)); err != nil {
		t.Fatalf("Pickup: %v", err)
	}
	if err := r.Pickup(start); !errors.Is(err, ErrInvalidState) {
		t.Errorf("doppelte Abholung sollte fehlschlagen")
	}
	if err := r.Complete(start.Add(5*time.Minute + 2*time.Hour + time.Minute)); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if r.Price != 21 {
		t.Errorf("Preis 3 h × 7 € = 21 € erwartet, bekommen %.2f", r.Price)
	}
	if err := r.Cancel(); !errors.Is(err, ErrInvalidState) {
		t.Errorf("Storno nach Rückgabe sollte fehlschlagen")
	}
}

func TestOverlaps(t *testing.T) {
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	r := Reservation{PlannedStart: base, PlannedHours: 2, Status: StatusReserved}
	now := base.Add(-time.Hour)

	if !r.Overlaps(base.Add(time.Hour), base.Add(3*time.Hour), now) {
		t.Error("Überschneidung erwartet")
	}
	if r.Overlaps(base.Add(2*time.Hour), base.Add(3*time.Hour), now) {
		t.Error("direkt anschließender Zeitraum darf nicht kollidieren")
	}
	r.Status = StatusCancelled
	if r.Overlaps(base, base.Add(time.Hour), now) {
		t.Error("stornierte Reservierung darf nicht blockieren")
	}
}
