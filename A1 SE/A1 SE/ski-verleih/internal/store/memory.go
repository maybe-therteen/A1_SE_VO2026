package store

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"ski-verleih/internal/domain"
)

// Store definiert die Schnittstelle für die Datenhaltung des Ski-Verleihs.
type Store interface {
	GetDashboardStats() domain.DashboardStats

	ListSkis() []domain.Ski
	CreateSki(number, skiType string, lengthCm int) (domain.Ski, error)
	DeleteSki(id int) error

	ListPersons() []domain.Person
	CreatePerson(name, email, phone string) (domain.Person, error)
	DeletePerson(id int) error

	ListReservations() []domain.Reservation
	CreateReservation(skiID, personID int, notes string) (domain.Reservation, error)
	ReturnReservation(id int) (domain.Reservation, error)
}

// MemoryStore implementiert Store thread-sicher im Arbeitsspeicher.
type MemoryStore struct {
	mu            sync.RWMutex
	skis          map[int]*domain.Ski
	persons       map[int]*domain.Person
	reservations  map[int]*domain.Reservation
	nextSkiID     int
	nextPersonID  int
	nextReserveID int
}

// NewMemoryStore erzeugt einen neuen, leeren In-Memory-Store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		skis:          make(map[int]*domain.Ski),
		persons:       make(map[int]*domain.Person),
		reservations:  make(map[int]*domain.Reservation),
		nextSkiID:     1,
		nextPersonID:  1,
		nextReserveID: 1,
	}
}

// GetDashboardStats berechnet die aktuellen Dashboard-Zahlen.
func (m *MemoryStore) GetDashboardStats() domain.DashboardStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	available := 0
	rented := 0
	for _, s := range m.skis {
		if s.Available {
			available++
		} else {
			rented++
		}
	}

	activeRentals := 0
	for _, r := range m.reservations {
		if r.Active {
			activeRentals++
		}
	}

	return domain.DashboardStats{
		AvailableSkis:     available,
		RentedSkis:        rented,
		TotalSkis:         len(m.skis),
		RegisteredPersons: len(m.persons),
		ActiveRentals:     activeRentals,
	}
}

// ListSkis gibt alle Ski sortiert nach ID zurück.
func (m *MemoryStore) ListSkis() []domain.Ski {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]domain.Ski, 0, len(m.skis))
	for _, s := range m.skis {
		result = append(result, *s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

// CreateSki legt einen neuen Ski an und prüft auf eindeutige Inventarnummer.
func (m *MemoryStore) CreateSki(number, skiType string, lengthCm int) (domain.Ski, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ski := &domain.Ski{
		ID:        m.nextSkiID,
		Number:    number,
		Type:      skiType,
		LengthCm:  lengthCm,
		Available: true,
	}
	if err := ski.Validate(); err != nil {
		return domain.Ski{}, fmt.Errorf("Ski-Validierung fehlgeschlagen: %w", err)
	}

	for _, existing := range m.skis {
		if strings.EqualFold(existing.Number, ski.Number) {
			return domain.Ski{}, fmt.Errorf("Ski-Nummer %q existiert bereits: %w", ski.Number, domain.ErrInvalidInput)
		}
	}

	m.skis[ski.ID] = ski
	m.nextSkiID++
	return *ski, nil
}

// DeleteSki entfernt einen Ski, sofern er nicht gerade ausgeliehen ist.
func (m *MemoryStore) DeleteSki(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ski, ok := m.skis[id]
	if !ok {
		return fmt.Errorf("Ski #%d: %w", id, domain.ErrNotFound)
	}
	if !ski.Available {
		return fmt.Errorf("Ski %s ist aktuell ausgeliehen: %w", ski.Number, domain.ErrHasActiveRentals)
	}

	delete(m.skis, id)
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
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

// CreatePerson registriert eine neue Person im System.
func (m *MemoryStore) CreatePerson(name, email, phone string) (domain.Person, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	person := &domain.Person{
		ID:    m.nextPersonID,
		Name:  name,
		Email: email,
		Phone: phone,
	}
	if err := person.Validate(); err != nil {
		return domain.Person{}, fmt.Errorf("Personen-Validierung fehlgeschlagen: %w", err)
	}

	m.persons[person.ID] = person
	m.nextPersonID++
	return *person, nil
}

// DeletePerson entfernt eine Person, sofern sie keine aktiven Ausleihen hat.
func (m *MemoryStore) DeletePerson(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.persons[id]; !ok {
		return fmt.Errorf("Person #%d: %w", id, domain.ErrNotFound)
	}
	for _, r := range m.reservations {
		if r.PersonID == id && r.Active {
			return fmt.Errorf("Person #%d hat noch aktive Ausleihen: %w", id, domain.ErrHasActiveRentals)
		}
	}

	delete(m.persons, id)
	return nil
}

// ListReservations gibt alle Ausleihen zurück (aktive zuerst, danach neueste zuerst).
func (m *MemoryStore) ListReservations() []domain.Reservation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]domain.Reservation, 0, len(m.reservations))
	for _, r := range m.reservations {
		result = append(result, *r)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Active != result[j].Active {
			return result[i].Active
		}
		return result[i].ID > result[j].ID
	})
	return result
}

// CreateReservation erstellt eine neue Ausleihe mit einem konkreten Zeitstempel.
func (m *MemoryStore) CreateReservationWithTime(skiID, personID int, notes string, startTime time.Time) (domain.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ski, ok := m.skis[skiID]
	if !ok {
		return domain.Reservation{}, fmt.Errorf("Ski #%d: %w", skiID, domain.ErrNotFound)
	}
	person, ok := m.persons[personID]
	if !ok {
		return domain.Reservation{}, fmt.Errorf("Person #%d: %w", personID, domain.ErrNotFound)
	}

	if err := ski.MarkRented(); err != nil {
		return domain.Reservation{}, fmt.Errorf("Ausleihe nicht möglich: %w", err)
	}

	res := &domain.Reservation{
		ID:         m.nextReserveID,
		SkiID:      ski.ID,
		SkiNumber:  ski.Number,
		SkiType:    ski.Type,
		SkiLength:  ski.LengthCm,
		PersonID:   person.ID,
		PersonName: person.Name,
		StartTime:  startTime,
		Active:     true,
		Notes:      strings.TrimSpace(notes),
	}

	m.reservations[res.ID] = res
	m.nextReserveID++
	return *res, nil
}

// CreateReservation legt eine neue Ausleihe zum aktuellen Zeitpunkt an.
func (m *MemoryStore) CreateReservation(skiID, personID int, notes string) (domain.Reservation, error) {
	return m.CreateReservationWithTime(skiID, personID, notes, time.Now())
}

// ReturnReservation markiert eine Ausleihe als zurückgegeben und gibt den Ski wieder frei.
func (m *MemoryStore) ReturnReservation(id int) (domain.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	res, ok := m.reservations[id]
	if !ok {
		return domain.Reservation{}, fmt.Errorf("Reservierung #%d: %w", id, domain.ErrNotFound)
	}

	if err := res.Complete(time.Now()); err != nil {
		return domain.Reservation{}, fmt.Errorf("Rückgabe fehlgeschlagen: %w", err)
	}

	if ski, ok := m.skis[res.SkiID]; ok {
		ski.MarkReturned()
	}

	return *res, nil
}
