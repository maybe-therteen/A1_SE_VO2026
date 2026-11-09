package store

import (
	"testing"
)

func TestMemoryStoreRentalLifecycle(t *testing.T) {
	s := NewMemoryStore()

	ski, err := s.CreateSki("SKI-101", "Race", 175)
	if err != nil {
		t.Fatalf("CreateSki failed: %v", err)
	}

	person, err := s.CreatePerson("Anna Hofer", "anna.hofer@example.at", "+43 660 1112233")
	if err != nil {
		t.Fatalf("CreatePerson failed: %v", err)
	}

	res, err := s.CreateReservation(ski.ID, person.ID, "Tagesski")
	if err != nil {
		t.Fatalf("CreateReservation failed: %v", err)
	}

	if res.SkiNumber != "SKI-101" || res.SkiType != "Race" {
		t.Fatalf("expected reservation to track ski number and type, got %s / %s", res.SkiNumber, res.SkiType)
	}

	stats := s.GetDashboardStats()
	if stats.AvailableSkis != 0 || stats.RentedSkis != 1 || stats.TotalSkis != 1 || stats.RegisteredPersons != 1 {
		t.Fatalf("unexpected dashboard stats during rental: %+v", stats)
	}

	returned, err := s.ReturnReservation(res.ID)
	if err != nil {
		t.Fatalf("ReturnReservation failed: %v", err)
	}
	if returned.Active || returned.EndTime == nil {
		t.Fatalf("expected reservation to be marked returned with timestamp")
	}

	statsAfter := s.GetDashboardStats()
	if statsAfter.AvailableSkis != 1 || statsAfter.RentedSkis != 0 {
		t.Fatalf("unexpected dashboard stats after return: %+v", statsAfter)
	}
}
