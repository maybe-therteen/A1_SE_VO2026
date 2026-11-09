package store

import (
	"errors"
	"testing"
	"time"

	"fahrrad-verleih/internal/domain"
)

func newSeeded(t *testing.T) *MemoryStore {
	t.Helper()
	s := NewMemoryStore()
	if _, err := s.CreateBike("BIKE-1", "City", 50, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateBike("BIKE-2", "E-Bike", 52, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreatePerson("Anna Test", "anna@example.at", ""); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCreateBikeDuplicateNumber(t *testing.T) {
	s := newSeeded(t)
	if _, err := s.CreateBike("bike-1", "City", 50, 0); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("doppelte Nummer sollte abgelehnt werden, bekommen %v", err)
	}
}

func TestImmediateRentalAndReturn(t *testing.T) {
	s := newSeeded(t)
	res, err := s.CreateReservation(1, 1, time.Time{}, 2, "sofort")
	if err != nil {
		t.Fatalf("CreateReservation: %v", err)
	}
	if res.Status != domain.StatusActive {
		t.Fatalf("Sofort-Ausleihe sollte aktiv sein, ist %s", res.Status)
	}
	if res.Price != 8 {
		t.Errorf("geschätzter Preis 2 h × 4 € = 8 € erwartet, bekommen %.2f", res.Price)
	}
	if st := s.GetDashboardStats(); st.RentedBikes != 1 || st.AvailableBikes != 1 {
		t.Errorf("Dashboard falsch: %+v", st)
	}

	// zweite Sofort-Ausleihe desselben Fahrrads muss scheitern
	if _, err := s.CreateReservation(1, 1, time.Time{}, 1, ""); !errors.Is(err, domain.ErrBikeNotAvailable) {
		t.Errorf("ErrBikeNotAvailable erwartet, bekommen %v", err)
	}

	ret, err := s.ReturnReservation(res.ID)
	if err != nil {
		t.Fatalf("Return: %v", err)
	}
	if ret.Status != domain.StatusReturned || ret.ReturnTime == nil {
		t.Errorf("Rückgabe nicht korrekt protokolliert: %+v", ret)
	}
	if ret.Price != 4 {
		t.Errorf("sofortige Rückgabe = 1 h × 4 € erwartet, bekommen %.2f", ret.Price)
	}
	if st := s.GetDashboardStats(); st.AvailableBikes != 2 || st.Revenue != 4 {
		t.Errorf("Dashboard nach Rückgabe falsch: %+v", st)
	}
}

func TestFutureReservationBlocksWindowOnly(t *testing.T) {
	s := newSeeded(t)
	start := time.Now().Add(24 * time.Hour)
	res, err := s.CreateReservation(2, 1, start, 4, "morgen")
	if err != nil {
		t.Fatalf("CreateReservation: %v", err)
	}
	if res.Status != domain.StatusReserved {
		t.Fatalf("zukünftige Reservierung sollte Reserviert sein, ist %s", res.Status)
	}

	// Überschneidung → abgelehnt
	if _, err := s.CreateReservation(2, 1, start.Add(2*time.Hour), 4, ""); !errors.Is(err, domain.ErrBikeNotAvailable) {
		t.Errorf("Überschneidung sollte abgelehnt werden, bekommen %v", err)
	}
	// direkt danach → ok
	if _, err := s.CreateReservation(2, 1, start.Add(4*time.Hour), 2, ""); err != nil {
		t.Errorf("anschließender Zeitraum sollte frei sein: %v", err)
	}
	// jetzt für 2 h → ok, da vor der Reservierung
	if _, err := s.CreateReservation(2, 1, time.Time{}, 2, ""); err != nil {
		t.Errorf("Sofort-Ausleihe vor Reservierung sollte möglich sein: %v", err)
	}

	avail, err := s.AvailableBikes(start, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(avail) != 1 || avail[0].ID != 1 {
		t.Errorf("nur BIKE-1 sollte morgen frei sein, bekommen %+v", avail)
	}
}

func TestPickupAndCancel(t *testing.T) {
	s := newSeeded(t)
	res, err := s.CreateReservation(1, 1, time.Now().Add(2*time.Hour), 1, "")
	if err != nil {
		t.Fatal(err)
	}
	picked, err := s.PickupReservation(res.ID)
	if err != nil || picked.Status != domain.StatusActive {
		t.Fatalf("Pickup fehlgeschlagen: %v %+v", err, picked)
	}
	if _, err := s.CancelReservation(res.ID); !errors.Is(err, domain.ErrInvalidState) {
		t.Errorf("aktive Ausleihe darf nicht storniert werden")
	}

	res2, err := s.CreateReservation(2, 1, time.Now().Add(5*time.Hour), 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DeletePerson(1); !errors.Is(err, domain.ErrHasActiveRentals) {
		t.Errorf("Person mit offenen Reservierungen darf nicht gelöscht werden")
	}
	if _, err := s.CancelReservation(res2.ID); err != nil {
		t.Errorf("Storno fehlgeschlagen: %v", err)
	}
	if err := s.DeleteBike(2); err != nil {
		t.Errorf("Fahrrad nach Storno sollte löschbar sein: %v", err)
	}
}

func TestOverdueRentalBlocksBike(t *testing.T) {
	s := newSeeded(t)
	start := time.Now().Add(-3 * time.Hour)
	if _, err := s.CreateReservationAt(1, 1, start, 1, "überfällig", start); err != nil {
		t.Fatal(err)
	}
	if st := s.GetDashboardStats(); st.OverdueRentals != 1 {
		t.Errorf("1 überfällige Ausleihe erwartet: %+v", st)
	}
	avail, err := s.AvailableBikes(time.Time{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(avail) != 1 || avail[0].ID != 2 {
		t.Errorf("überfälliges Fahrrad darf nicht als frei gelten: %+v", avail)
	}
	if _, err := s.CreateReservation(1, 1, time.Now().Add(48*time.Hour), 1, ""); !errors.Is(err, domain.ErrBikeNotAvailable) {
		t.Errorf("überfälliges Fahrrad sollte bis zur Rückgabe blockiert sein, bekommen %v", err)
	}
}

func TestPastStartRejected(t *testing.T) {
	s := newSeeded(t)
	if _, err := s.CreateReservation(1, 1, time.Now().Add(-time.Hour), 1, ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("Start in der Vergangenheit sollte abgelehnt werden, bekommen %v", err)
	}
	if _, err := s.CreateReservation(1, 1, time.Time{}, 0, ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("Dauer 0 sollte abgelehnt werden, bekommen %v", err)
	}
}
