package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidInput     = errors.New("ungültige Eingabedaten")
	ErrNotFound         = errors.New("Eintrag nicht gefunden")
	ErrSkiNotAvailable  = errors.New("Ski ist derzeit bereits ausgeliehen")
	ErrAlreadyReturned  = errors.New("Ausleihe wurde bereits zurückgegeben")
	ErrHasActiveRentals = errors.New("Eintrag hat noch aktive Ausleihen")
)

// Ski repräsentiert einen einzelnen Verleih-Ski mit eindeutiger Inventarnummer und Typ.
type Ski struct {
	ID        int    `json:"id"`
	Number    string `json:"number"`    // z.B. "SKI-101"
	Type      string `json:"type"`      // z.B. "Allround", "Race", "Slalom", "Freeride", "Touring"
	LengthCm  int    `json:"lengthCm"`  // Länge in cm, z.B. 170
	Available bool   `json:"available"` // true = Verfügbar, false = Ausgeliehen
}

// Validate prüft, ob die Pflichtfelder eines Skis gültig gesetzt sind.
func (s *Ski) Validate() error {
	s.Number = strings.TrimSpace(s.Number)
	s.Type = strings.TrimSpace(s.Type)
	if s.Number == "" {
		return fmt.Errorf("Ski-Nummer darf nicht leer sein: %w", ErrInvalidInput)
	}
	if s.Type == "" {
		return fmt.Errorf("Ski-Typ darf nicht leer sein: %w", ErrInvalidInput)
	}
	if s.LengthCm < 80 || s.LengthCm > 230 {
		return fmt.Errorf("Ski-Länge muss zwischen 80 und 230 cm liegen: %w", ErrInvalidInput)
	}
	return nil
}

// Label gibt eine lesbare Kennzeichnung aus Nummer, Typ und Länge zurück.
func (s *Ski) Label() string {
	return fmt.Sprintf("%s (%s, %d cm)", s.Number, s.Type, s.LengthCm)
}

// MarkRented markiert den Ski als ausgeliehen.
func (s *Ski) MarkRented() error {
	if !s.Available {
		return fmt.Errorf("Ski %s: %w", s.Number, ErrSkiNotAvailable)
	}
	s.Available = false
	return nil
}

// MarkReturned gibt den Ski wieder frei.
func (s *Ski) MarkReturned() {
	s.Available = true
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

// Reservation repräsentiert einen Ausleihvorgang und sorgt für volle Nachvollziehbarkeit
// (wer hat welchen Ski mit welcher Nummer und welchem Typ wann ausgeliehen und zurückgegeben).
type Reservation struct {
	ID         int        `json:"id"`
	SkiID      int        `json:"skiId"`
	SkiNumber  string     `json:"skiNumber"`
	SkiType    string     `json:"skiType"`
	SkiLength  int        `json:"skiLength"`
	PersonID   int        `json:"personId"`
	PersonName string     `json:"personName"`
	StartTime  time.Time  `json:"startTime"`
	EndTime    *time.Time `json:"endTime,omitempty"`
	Active     bool       `json:"active"`
	Notes      string     `json:"notes"`
}

// Complete beendet eine aktive Ausleihe und setzt den Rückgabezeitpunkt.
func (r *Reservation) Complete(returnTime time.Time) error {
	if !r.Active {
		return fmt.Errorf("Reservierung #%d: %w", r.ID, ErrAlreadyReturned)
	}
	r.Active = false
	r.EndTime = &returnTime
	return nil
}

// DashboardStats fasst die Kennzahlen für das Dashboard zusammen.
type DashboardStats struct {
	AvailableSkis     int `json:"availableSkis"`
	RentedSkis        int `json:"rentedSkis"`
	TotalSkis         int `json:"totalSkis"`
	RegisteredPersons int `json:"registeredPersons"`
	ActiveRentals     int `json:"activeRentals"`
}
