package store

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"fahrrad-verleih/internal/domain"
)

// immediateTolerance: Startzeitpunkte bis zu dieser Spanne in der Zukunft gelten als "sofort".
const immediateTolerance = time.Minute

// pastTolerance: so weit darf ein gewünschter Beginn in der Vergangenheit liegen (z.B. Formular-Verzögerung).
const pastTolerance = 5 * time.Minute

// Store definiert die Schnittstelle für die Datenhaltung des Fahrradverleihs.
type Store interface {
	GetDashboardStats() domain.DashboardStats

	ListBikes() []domain.Bike
	AvailableBikes(start time.Time, hours int) ([]domain.Bike, error)
	CreateBike(number, bikeType string, frameSizeCm int, hourlyRate float64) (domain.Bike, error)
	DeleteBike(id int) error

	ListPersons() []domain.Person
	CreatePerson(name, email, phone string) (domain.Person, error)
	DeletePerson(id int) error

	ListReservations() []domain.Reservation
	CreateReservation(bikeID, personID int, start time.Time, hours int, notes string) (domain.Reservation, error)
	PickupReservation(id int) (domain.Reservation, error)
	ReturnReservation(id int) (domain.Reservation, error)
	CancelReservation(id int) (domain.Reservation, error)
}

// MemoryStore implementiert Store thread-sicher im Arbeitsspeicher.
type MemoryStore struct {
	mu            sync.RWMutex
	bikes         map[int]*domain.Bike
	persons       map[int]*domain.Person
	reservations  map[int]*domain.Reservation
	nextBikeID    int
	nextPersonID  int
	nextReserveID int
}

// NewMemoryStore erzeugt einen neuen, leeren In-Memory-Store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		bikes:         make(map[int]*domain.Bike),
		persons:       make(map[int]*domain.Person),
		reservations:  make(map[int]*domain.Reservation),
		nextBikeID:    1,
		nextPersonID:  1,
		nextReserveID: 1,
	}
}

// GetDashboardStats berechnet die aktuellen Dashboard-Zahlen.
func (m *MemoryStore) GetDashboardStats() domain.DashboardStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	stats := domain.DashboardStats{
		TotalBikes:        len(m.bikes),
		RegisteredPersons: len(m.persons),
	}
	for _, b := range m.bikes {
		if b.Available {
			stats.AvailableBikes++
		} else {
			stats.RentedBikes++
		}
	}
	for _, r := range m.reservations {
		switch r.Status {
		case domain.StatusActive:
			stats.ActiveRentals++
			if r.IsOverdue(now) {
				stats.OverdueRentals++
			}
		case domain.StatusReserved:
			stats.UpcomingReserved++
		case domain.StatusReturned:
			stats.Revenue += r.Price
		}
	}
	return stats
}

// ListBikes gibt alle Fahrräder sortiert nach ID zurück.
func (m *MemoryStore) ListBikes() []domain.Bike {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]domain.Bike, 0, len(m.bikes))
	for _, b := range m.bikes {
		result = append(result, *b)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

// AvailableBikes liefert alle Fahrräder, die im Zeitraum [start, start+hours) frei sind.
func (m *MemoryStore) AvailableBikes(start time.Time, hours int) ([]domain.Bike, error) {
	if err := domain.ValidateHours(hours); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	start = normalizeStart(start, now)
	end := start.Add(time.Duration(hours) * time.Hour)

	result := make([]domain.Bike, 0)
	for _, b := range m.bikes {
		if m.isFreeLocked(b.ID, start, end, now, 0) {
			result = append(result, *b)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// CreateBike legt ein neues Fahrrad an und prüft auf eindeutige Inventarnummer.
func (m *MemoryStore) CreateBike(number, bikeType string, frameSizeCm int, hourlyRate float64) (domain.Bike, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	bike := &domain.Bike{
		ID:          m.nextBikeID,
		Number:      number,
		Type:        bikeType,
		FrameSizeCm: frameSizeCm,
		HourlyRate:  hourlyRate,
		Available:   true,
	}
	if err := bike.Validate(); err != nil {
		return domain.Bike{}, fmt.Errorf("Fahrrad-Validierung fehlgeschlagen: %w", err)
	}
	for _, existing := range m.bikes {
		if strings.EqualFold(existing.Number, bike.Number) {
			return domain.Bike{}, fmt.Errorf("Fahrrad-Nummer %q existiert bereits: %w", bike.Number, domain.ErrInvalidInput)
		}
	}

	m.bikes[bike.ID] = bike
	m.nextBikeID++
	return *bike, nil
}

// DeleteBike entfernt ein Fahrrad, sofern es weder ausgeliehen noch reserviert ist.
func (m *MemoryStore) DeleteBike(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	bike, ok := m.bikes[id]
	if !ok {
		return fmt.Errorf("Fahrrad #%d: %w", id, domain.ErrNotFound)
	}
	for _, r := range m.reservations {
		if r.BikeID == id && r.IsOpen() {
			return fmt.Errorf("Fahrrad %s: %w", bike.Number, domain.ErrHasActiveRentals)
		}
	}
	delete(m.bikes, id)
	return nil
}

// ListPersons gibt alle registrierten Personen sortiert nach ID zurück.
func (m *MemoryStore) ListPersons() []domain.Person {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]domain.Person, 0, len(m.persons))
	for _, p := range m.persons {
		result = append(result, *p)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

// CreatePerson registriert eine neue Person im System.
func (m *MemoryStore) CreatePerson(name, email, phone string) (domain.Person, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	person := &domain.Person{ID: m.nextPersonID, Name: name, Email: email, Phone: phone}
	if err := person.Validate(); err != nil {
		return domain.Person{}, fmt.Errorf("Personen-Validierung fehlgeschlagen: %w", err)
	}
	m.persons[person.ID] = person
	m.nextPersonID++
	return *person, nil
}

// DeletePerson entfernt eine Person, sofern sie keine offenen Ausleihen/Reservierungen hat.
func (m *MemoryStore) DeletePerson(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.persons[id]; !ok {
		return fmt.Errorf("Person #%d: %w", id, domain.ErrNotFound)
	}
	for _, r := range m.reservations {
		if r.PersonID == id && r.IsOpen() {
			return fmt.Errorf("Person #%d: %w", id, domain.ErrHasActiveRentals)
		}
	}
	delete(m.persons, id)
	return nil
}

// ListReservations gibt alle Reservierungen zurück: Aktiv, dann Reserviert (nach Beginn),
// danach abgeschlossene (neueste zuerst).
func (m *MemoryStore) ListReservations() []domain.Reservation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]domain.Reservation, 0, len(m.reservations))
	for _, r := range m.reservations {
		result = append(result, *r)
	}
	rank := map[string]int{
		domain.StatusActive:    0,
		domain.StatusReserved:  1,
		domain.StatusReturned:  2,
		domain.StatusCancelled: 2,
	}
	sort.Slice(result, func(i, j int) bool {
		ri, rj := rank[result[i].Status], rank[result[j].Status]
		if ri != rj {
			return ri < rj
		}
		if result[i].Status == domain.StatusReserved {
			return result[i].PlannedStart.Before(result[j].PlannedStart)
		}
		return result[i].ID > result[j].ID
	})
	return result
}

// CreateReservation legt eine Reservierung mit flexiblem Beginn und Dauer an.
// Liegt der Beginn "jetzt", wird das Fahrrad sofort ausgegeben (Status Aktiv).
func (m *MemoryStore) CreateReservation(bikeID, personID int, start time.Time, hours int, notes string) (domain.Reservation, error) {
	return m.CreateReservationAt(bikeID, personID, start, hours, notes, time.Now())
}

// CreateReservationAt wie CreateReservation, aber mit explizitem "Jetzt" (für Seed-Daten und Tests).
func (m *MemoryStore) CreateReservationAt(bikeID, personID int, start time.Time, hours int, notes string, now time.Time) (domain.Reservation, error) {
	if err := domain.ValidateHours(hours); err != nil {
		return domain.Reservation{}, err
	}
	if start.IsZero() {
		start = now
	}
	if start.Before(now.Add(-pastTolerance)) {
		return domain.Reservation{}, fmt.Errorf("Ausleihbeginn darf nicht in der Vergangenheit liegen: %w", domain.ErrInvalidInput)
	}
	start = normalizeStart(start, now)

	m.mu.Lock()
	defer m.mu.Unlock()

	bike, ok := m.bikes[bikeID]
	if !ok {
		return domain.Reservation{}, fmt.Errorf("Fahrrad #%d: %w", bikeID, domain.ErrNotFound)
	}
	person, ok := m.persons[personID]
	if !ok {
		return domain.Reservation{}, fmt.Errorf("Person #%d: %w", personID, domain.ErrNotFound)
	}

	end := start.Add(time.Duration(hours) * time.Hour)
	if !m.isFreeLocked(bike.ID, start, end, now, 0) {
		return domain.Reservation{}, fmt.Errorf("Fahrrad %s: %w", bike.Number, domain.ErrBikeNotAvailable)
	}

	res := &domain.Reservation{
		ID:           m.nextReserveID,
		BikeID:       bike.ID,
		BikeNumber:   bike.Number,
		BikeType:     bike.Type,
		BikeFrameCm:  bike.FrameSizeCm,
		PersonID:     person.ID,
		PersonName:   person.Name,
		PlannedStart: start,
		PlannedHours: hours,
		HourlyRate:   bike.HourlyRate,
		Price:        domain.CalculatePrice(bike.HourlyRate, hours),
		Status:       domain.StatusReserved,
		Notes:        strings.TrimSpace(notes),
	}

	// Sofort-Ausleihe: Fahrrad direkt ausgeben.
	if !start.After(now.Add(immediateTolerance)) {
		if err := bike.MarkRented(); err != nil {
			return domain.Reservation{}, fmt.Errorf("Ausleihe nicht möglich: %w", err)
		}
		_ = res.Pickup(start)
	}

	m.reservations[res.ID] = res
	m.nextReserveID++
	return *res, nil
}

// PickupReservation markiert eine geplante Reservierung als abgeholt.
func (m *MemoryStore) PickupReservation(id int) (domain.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	res, ok := m.reservations[id]
	if !ok {
		return domain.Reservation{}, fmt.Errorf("Reservierung #%d: %w", id, domain.ErrNotFound)
	}
	bike, ok := m.bikes[res.BikeID]
	if !ok {
		return domain.Reservation{}, fmt.Errorf("Fahrrad #%d: %w", res.BikeID, domain.ErrNotFound)
	}
	if res.Status != domain.StatusReserved {
		return domain.Reservation{}, fmt.Errorf("Abholung nicht möglich, Status %s: %w", res.Status, domain.ErrInvalidState)
	}

	now := time.Now()
	end := res.PlannedEnd()
	if now.Add( time.Duration(res.PlannedHours)*time.Hour ).After(end) {
		end = now.Add(time.Duration(res.PlannedHours) * time.Hour)
	}
	if !m.isFreeLocked(bike.ID, now, end, now, res.ID) {
		return domain.Reservation{}, fmt.Errorf("Fahrrad %s im Zeitraum belegt: %w", bike.Number, domain.ErrBikeNotAvailable)
	}

	if err := bike.MarkRented(); err != nil {
		return domain.Reservation{}, fmt.Errorf("Abholung nicht möglich: %w", err)
	}
	_ = res.Pickup(now)
	return *res, nil
}
// ReturnReservation beendet eine aktive Ausleihe, berechnet den Endpreis und gibt das Fahrrad frei.
func (m *MemoryStore) ReturnReservation(id int) (domain.Reservation, error) {
	return m.ReturnReservationAt(id, time.Now())
}

// ReturnReservationAt wie ReturnReservation, aber mit explizitem Rückgabezeitpunkt.
func (m *MemoryStore) ReturnReservationAt(id int, t time.Time) (domain.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	res, ok := m.reservations[id]
	if !ok {
		return domain.Reservation{}, fmt.Errorf("Reservierung #%d: %w", id, domain.ErrNotFound)
	}
	if err := res.Complete(t); err != nil {
		return domain.Reservation{}, fmt.Errorf("Rückgabe fehlgeschlagen: %w", err)
	}
	if bike, ok := m.bikes[res.BikeID]; ok {
		bike.MarkReturned()
	}
	return *res, nil
}

// CancelReservation storniert eine noch nicht abgeholte Reservierung.
func (m *MemoryStore) CancelReservation(id int) (domain.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	res, ok := m.reservations[id]
	if !ok {
		return domain.Reservation{}, fmt.Errorf("Reservierung #%d: %w", id, domain.ErrNotFound)
	}
	if err := res.Cancel(); err != nil {
		return domain.Reservation{}, fmt.Errorf("Stornierung fehlgeschlagen: %w", err)
	}
	return *res, nil
}

// isFreeLocked prüft (bei gehaltenem Lock), ob ein Fahrrad im Zeitfenster frei ist.
// excludeID erlaubt das Ignorieren einer bestimmten Reservierung (0 = keine).
func (m *MemoryStore) isFreeLocked(bikeID int, start, end, now time.Time, excludeID int) bool {
	for _, r := range m.reservations {
		if r.BikeID != bikeID || r.ID == excludeID {
			continue
		}
		if r.Overlaps(start, end, now) {
			return false
		}
	}
	return true
}

// normalizeStart setzt leicht vergangene Startzeitpunkte auf "jetzt".
func normalizeStart(start, now time.Time) time.Time {
	if start.IsZero() || start.Before(now) {
		return now
	}
	return start
}
