package domain

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

var (
	ErrInvalidInput     = errors.New("ungültige Eingabedaten")
	ErrNotFound         = errors.New("Eintrag nicht gefunden")
	ErrBikeNotAvailable = errors.New("Fahrrad ist im gewünschten Zeitraum nicht verfügbar")
	ErrInvalidState     = errors.New("Aktion im aktuellen Status nicht möglich")
	ErrHasActiveRentals = errors.New("Eintrag hat noch aktive oder geplante Ausleihen")
)

// Erlaubte Fahrrad-Typen inkl. Standard-Stundenpreis in Euro.
var DefaultHourlyRates = map[string]float64{
	"City":         4.00,
	"Trekking":     5.00,
	"Mountainbike": 7.00,
	"E-Bike":       9.00,
	"Rennrad":      8.00,
	"Kinder":       3.00,
}

// Grenzen für die Ausleihdauer (in Stunden).
const (
	MinRentalHours = 1
	MaxRentalHours = 14 * 24 // max. 2 Wochen
)

// Reservierungs-Status.
const (
	StatusReserved  = "Reserviert"    // geplant, Fahrrad noch nicht abgeholt
	StatusActive    = "Aktiv"         // Fahrrad ist ausgeliehen
	StatusReturned  = "Zurückgegeben" // abgeschlossen
	StatusCancelled = "Storniert"     // vor Abholung abgesagt
)

// Bike repräsentiert ein einzelnes Verleih-Fahrrad mit eindeutiger Inventarnummer und Typ.
type Bike struct {
	ID          int     `json:"id"`
	Number      string  `json:"number"`      // z.B. "BIKE-101"
	Type        string  `json:"type"`        // z.B. "City", "E-Bike", ...
	FrameSizeCm int     `json:"frameSizeCm"` // Rahmenhöhe in cm
	HourlyRate  float64 `json:"hourlyRate"`  // Preis pro angefangener Stunde in €
	Available   bool    `json:"available"`   // true = aktuell im Laden, false = gerade ausgeliehen
}

// Validate prüft die Pflichtfelder eines Fahrrads und setzt ggf. den Standardpreis.
func (b *Bike) Validate() error {
	b.Number = strings.ToUpper(strings.TrimSpace(b.Number))
	b.Type = strings.TrimSpace(b.Type)
	if b.Number == "" {
		return fmt.Errorf("Fahrrad-Nummer darf nicht leer sein: %w", ErrInvalidInput)
	}
	defaultRate, ok := DefaultHourlyRates[b.Type]
	if !ok {
		return fmt.Errorf("unbekannter Fahrrad-Typ %q: %w", b.Type, ErrInvalidInput)
	}
	if b.FrameSizeCm < 30 || b.FrameSizeCm > 65 {
		return fmt.Errorf("Rahmenhöhe muss zwischen 30 und 65 cm liegen: %w", ErrInvalidInput)
	}
	if b.HourlyRate == 0 {
		b.HourlyRate = defaultRate
	}
	if b.HourlyRate < 0 || b.HourlyRate > 100 {
		return fmt.Errorf("Stundenpreis muss zwischen 0 und 100 € liegen: %w", ErrInvalidInput)
	}
	return nil
}

// Label gibt eine lesbare Kennzeichnung aus Nummer, Typ und Rahmenhöhe zurück.
func (b *Bike) Label() string {
	return fmt.Sprintf("%s (%s, %d cm)", b.Number, b.Type, b.FrameSizeCm)
}

// MarkRented markiert das Fahrrad als ausgeliehen (bei Abholung).
func (b *Bike) MarkRented() error {
	if !b.Available {
		return fmt.Errorf("Fahrrad %s ist gerade ausgeliehen: %w", b.Number, ErrBikeNotAvailable)
	}
	b.Available = false
	return nil
}

// MarkReturned gibt das Fahrrad wieder frei.
func (b *Bike) MarkReturned() {
	b.Available = true
}

// Person repräsentiert eine registrierte Kundin bzw. einen registrierten Kunden.
type Person struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// Validate prüft die Pflichtfelder einer Person.
func (p *Person) Validate() error {
	p.Name = strings.TrimSpace(p.Name)
	p.Email = strings.TrimSpace(p.Email)
	p.Phone = strings.TrimSpace(p.Phone)
	if p.Name == "" {
		return fmt.Errorf("Name darf nicht leer sein: %w", ErrInvalidInput)
	}
	if p.Email == "" || !strings.Contains(p.Email, "@") {
		return fmt.Errorf("gültige E-Mail-Adresse erforderlich: %w", ErrInvalidInput)
	}
	return nil
}

// Reservation repräsentiert einen Ausleihvorgang mit flexiblem Beginn, geplanter Dauer
// und stundenweiser Abrechnung. Fahrrad-Daten werden kopiert, damit die Historie
// auch nach Löschen des Fahrrads nachvollziehbar bleibt.
type Reservation struct {
	ID           int        `json:"id"`
	BikeID       int        `json:"bikeId"`
	BikeNumber   string     `json:"bikeNumber"`
	BikeType     string     `json:"bikeType"`
	BikeFrameCm  int        `json:"bikeFrameCm"`
	PersonID     int        `json:"personId"`
	PersonName   string     `json:"personName"`
	PlannedStart time.Time  `json:"plannedStart"` // gewünschter Ausleihbeginn
	PlannedHours int        `json:"plannedHours"` // gebuchte Dauer in Stunden
	PickupTime   *time.Time `json:"pickupTime,omitempty"`
	ReturnTime   *time.Time `json:"returnTime,omitempty"`
	HourlyRate   float64    `json:"hourlyRate"`
	Price        float64    `json:"price"` // geschätzt (geplant) bzw. final (nach Rückgabe)
	Status       string     `json:"status"`
	Notes        string     `json:"notes"`
}

// PlannedEnd liefert das geplante Ende der Ausleihe.
func (r *Reservation) PlannedEnd() time.Time {
	return r.PlannedStart.Add(time.Duration(r.PlannedHours) * time.Hour)
}

// IsOpen gibt an, ob die Reservierung noch Kapazität blockiert (geplant oder aktiv).
func (r *Reservation) IsOpen() bool {
	return r.Status == StatusReserved || r.Status == StatusActive
}

// IsOverdue gibt an, ob eine aktive Ausleihe das geplante Ende überschritten hat.
func (r *Reservation) IsOverdue(now time.Time) bool {
	return r.Status == StatusActive && !now.Before(r.PlannedEnd())
}

// Overlaps prüft, ob sich das Zeitfenster [start, end) mit dieser Reservierung überschneidet.
// Überfällige aktive Ausleihen blockieren das Fahrrad bis zur tatsächlichen Rückgabe.
func (r *Reservation) Overlaps(start, end, now time.Time) bool {
	if !r.IsOpen() {
		return false
	}
	if r.IsOverdue(now) {
		return r.PlannedStart.Before(end)
	}
	return start.Before(r.PlannedEnd()) && r.PlannedStart.Before(end)
}

// Pickup startet die Ausleihe (Fahrrad wird abgeholt).
func (r *Reservation) Pickup(t time.Time) error {
	if r.Status != StatusReserved {
		return fmt.Errorf("Reservierung #%d ist %s: %w", r.ID, r.Status, ErrInvalidState)
	}
	r.Status = StatusActive
	r.PickupTime = &t
	return nil
}

// Complete beendet eine aktive Ausleihe und berechnet den Endpreis.
func (r *Reservation) Complete(t time.Time) error {
	if r.Status != StatusActive {
		return fmt.Errorf("Reservierung #%d ist %s: %w", r.ID, r.Status, ErrInvalidState)
	}
	start := r.PlannedStart
	if r.PickupTime != nil {
		start = *r.PickupTime
	}
	r.Status = StatusReturned
	r.ReturnTime = &t
	r.Price = CalculatePrice(r.HourlyRate, BillableHours(start, t))
	return nil
}

// Cancel storniert eine noch nicht abgeholte Reservierung.
func (r *Reservation) Cancel() error {
	if r.Status != StatusReserved {
		return fmt.Errorf("Reservierung #%d ist %s: %w", r.ID, r.Status, ErrInvalidState)
	}
	r.Status = StatusCancelled
	r.Price = 0
	return nil
}

// BillableHours berechnet die abzurechnenden Stunden: jede angefangene Stunde zählt, mind. 1.
func BillableHours(start, end time.Time) int {
	d := end.Sub(start)
	if d <= 0 {
		return MinRentalHours
	}
	h := int(math.Ceil(d.Hours()))
	if h < MinRentalHours {
		return MinRentalHours
	}
	return h
}

// CalculatePrice berechnet den Preis (auf Cent gerundet).
func CalculatePrice(hourlyRate float64, hours int) float64 {
	return math.Round(hourlyRate*float64(hours)*100) / 100
}

// ValidateHours prüft die gewünschte Ausleihdauer.
func ValidateHours(hours int) error {
	if hours < MinRentalHours || hours > MaxRentalHours {
		return fmt.Errorf("Dauer muss zwischen %d und %d Stunden liegen: %w", MinRentalHours, MaxRentalHours, ErrInvalidInput)
	}
	return nil
}

// DashboardStats fasst die Kennzahlen für das Dashboard zusammen.
type DashboardStats struct {
	AvailableBikes    int     `json:"availableBikes"`
	RentedBikes       int     `json:"rentedBikes"`
	TotalBikes        int     `json:"totalBikes"`
	RegisteredPersons int     `json:"registeredPersons"`
	ActiveRentals     int     `json:"activeRentals"`
	UpcomingReserved  int     `json:"upcomingReserved"`
	OverdueRentals    int     `json:"overdueRentals"`
	Revenue           float64 `json:"revenue"`
}
