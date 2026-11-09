package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"fahrrad-verleih/internal/handler"
	"fahrrad-verleih/internal/store"
)

func main() {
	memStore := store.NewMemoryStore()
	if err := seedInitialData(memStore); err != nil {
		log.Fatalf("Fehler beim Initialisieren der Beispieldaten: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := handler.NewServer(memStore, "./web")
	fmt.Printf("Fahrradverleih-Server läuft auf http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatalf("Server-Fehler: %v", err)
	}
}

// seedInitialData befüllt den Store mit Beispieldaten:
// 20 Fahrräder, 10 Personen, 1 abgeschlossene, 3 aktive und 2 geplante Ausleihen.
func seedInitialData(s *store.MemoryStore) error {
	bikeSeeds := []struct {
		number string
		bType  string
		frame  int
	}{
		{"BIKE-101", "City", 50},
		{"BIKE-102", "E-Bike", 52},
		{"BIKE-103", "Mountainbike", 46},
		{"BIKE-104", "Trekking", 56},
		{"BIKE-105", "Rennrad", 54},
		{"BIKE-106", "Kinder", 32},
		{"BIKE-107", "City", 45},
		{"BIKE-108", "E-Bike", 56},
		{"BIKE-109", "Mountainbike", 50},
		{"BIKE-110", "Trekking", 52},
		{"BIKE-111", "E-Bike", 48},
		{"BIKE-112", "City", 55},
		{"BIKE-113", "Rennrad", 58},
		{"BIKE-114", "Kinder", 36},
		{"BIKE-115", "Mountainbike", 54},
		{"BIKE-116", "Trekking", 60},
		{"BIKE-117", "E-Bike", 54},
		{"BIKE-118", "City", 48},
		{"BIKE-119", "Rennrad", 52},
		{"BIKE-120", "Kinder", 40},
	}
	for _, b := range bikeSeeds {
		if _, err := s.CreateBike(b.number, b.bType, b.frame, 0); err != nil {
			return fmt.Errorf("Seed Fahrrad %s: %w", b.number, err)
		}
	}

	personSeeds := []struct{ name, email, phone string }{
		{"Lukas Gruber", "lukas.gruber@example.at", "+43 664 1012030"},
		{"Sarah Leitner", "sarah.leitner@example.at", "+43 660 2023040"},
		{"Maximilian Hofer", "m.hofer@example.at", "+43 676 3034050"},
		{"Hannah Winkler", "hannah.winkler@example.at", "+43 650 4045060"},
		{"Tobias Moser", "tobias.moser@example.at", "+43 664 5056070"},
		{"Elena Steiner", "elena.steiner@example.at", "+43 660 6067080"},
		{"David Berger", "david.berger@example.at", "+43 676 7078090"},
		{"Clara Fuchs", "clara.fuchs@example.at", "+43 650 8089010"},
		{"Felix Eder", "felix.eder@example.at", "+43 664 9091020"},
		{"Nina Schwarz", "nina.schwarz@example.at", "+43 660 1122334"},
	}
	for _, p := range personSeeds {
		if _, err := s.CreatePerson(p.name, p.email, p.phone); err != nil {
			return fmt.Errorf("Seed Person %s: %w", p.name, err)
		}
	}

	now := time.Now().Truncate(time.Minute)

	// Abgeschlossene Ausleihe (Historie + Umsatz): 3 h gebucht, nach 2 h 40 min zurück → 3 h verrechnet.
	past := now.Add(-6 * time.Hour)
	resPast, err := s.CreateReservationAt(5, 4, past, 3, "Vormittagsrunde Salzachradweg", past)
	if err != nil {
		return fmt.Errorf("Seed Historie: %w", err)
	}
	if _, err := s.ReturnReservationAt(resPast.ID, past.Add(160*time.Minute)); err != nil {
		return fmt.Errorf("Seed Rückgabe: %w", err)
	}

	// Aktive Ausleihen.
	active := []struct {
		bike, person, hours int
		ago                 time.Duration
		notes               string
	}{
		{1, 1, 8, 3 * time.Hour, "Ganztags inkl. Helm"},
		{2, 2, 4, 2 * time.Hour, "E-Bike Tour Gaisberg"},
		{3, 3, 1, 75 * time.Minute, "Trail-Session (überfällig)"},
	}
	for _, a := range active {
		start := now.Add(-a.ago)
		if _, err := s.CreateReservationAt(a.bike, a.person, start, a.hours, a.notes, start); err != nil {
			return fmt.Errorf("Seed aktive Ausleihe: %w", err)
		}
	}

	// Geplante Reservierungen (flexibler Beginn in der Zukunft).
	if _, err := s.CreateReservationAt(8, 6, now.Add(24*time.Hour), 48, "Wochenend-Tour", now); err != nil {
		return fmt.Errorf("Seed Reservierung: %w", err)
	}
	if _, err := s.CreateReservationAt(4, 7, now.Add(3*time.Hour), 2, "Feierabendrunde", now); err != nil {
		return fmt.Errorf("Seed Reservierung: %w", err)
	}
	return nil
}
