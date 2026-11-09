package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"ski-verleih/internal/handler"
	"ski-verleih/internal/store"
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
	addr := ":" + port
	fmt.Printf("Ski-Verleih Server läuft auf http://localhost:%s\n", port)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("Server-Fehler: %v", err)
	}
}

// seedInitialData befüllt den Store exakt passend zum Vorlesungs-Dashboard:
// 20 Ski gesamt (17 verfügbar, 3 ausgeliehen) und 10 registrierte Personen.
func seedInitialData(s *store.MemoryStore) error {
	skiSeeds := []struct {
		number string
		sType  string
		length int
	}{
		{"SKI-101", "Allround", 168},
		{"SKI-102", "Race", 175},
		{"SKI-103", "Slalom", 165},
		{"SKI-104", "Freeride", 184},
		{"SKI-105", "Allround", 170},
		{"SKI-106", "Touring", 172},
		{"SKI-107", "Race", 178},
		{"SKI-108", "Slalom", 160},
		{"SKI-109", "Allround", 166},
		{"SKI-110", "Freeride", 188},
		{"SKI-111", "Touring", 174},
		{"SKI-112", "Allround", 172},
		{"SKI-113", "Race", 180},
		{"SKI-114", "Slalom", 163},
		{"SKI-115", "Allround", 169},
		{"SKI-116", "Freeride", 182},
		{"SKI-117", "Touring", 176},
		{"SKI-118", "Race", 174},
		{"SKI-119", "Allround", 164},
		{"SKI-120", "Slalom", 158},
	}

	for _, item := range skiSeeds {
		if _, err := s.CreateSki(item.number, item.sType, item.length); err != nil {
			return fmt.Errorf("Seed Ski %s: %w", item.number, err)
		}
	}

	personSeeds := []struct {
		name  string
		email string
		phone string
	}{
		{"Lukas Gruber", "lukas.gruber@example.at", "+43 664 1012030"},
		{"Sarah Leitner", "sarah.leitner@example.at", "+43 660 2023040"},
		{"Maximilian Hofer", "m.hofer@example.at", "+43 676 3034050"},
		{"Hannah Winkler", "hannah.winkler@example.at", "+43 650 4045060"},
		{"Tobias Moser", "tobias.moser@example.at", "+43 664 5056070"},
		{"Elena Steiner", "elena.steiner@example.at", "+43 660 6067080"},
		{"David Berger", "david.berger@example.at", "+43 676 7078090"},
		{"Clara Fuchs", "clara.fuchs@example.at", "+43 650 8089010"},
		{"felix.eder@example.at", "felix.eder@example.at", "+43 664 9091020"},
		{"Nina Schwarz", "nina.schwarz@example.at", "+43 660 1122334"},
	}
	// Fix name of 9th person cleanly
	personSeeds[8].name = "Felix Eder"

	for _, p := range personSeeds {
		if _, err := s.CreatePerson(p.name, p.email, p.phone); err != nil {
			return fmt.Errorf("Seed Person %s: %w", p.name, err)
		}
	}

	now := time.Now()
	// 1 abgeschlossene Ausleihe zur Demonstration der Historie
	resPast, err := s.CreateReservationWithTime(5, 4, "Vormittagstraining – pünktlich zurückgegeben", now.Add(-6*time.Hour))
	if err != nil {
		return err
	}
	if _, err := s.ReturnReservation(resPast.ID); err != nil {
		return err
	}

	// 3 aktive Ausleihen (damit 17 verfügbar & 3 ausgeliehen sind)
	if _, err := s.CreateReservationWithTime(1, 1, "Ganztags-Ausleihe inkl. Stöcke", now.Add(-3*time.Hour)); err != nil {
		return err
	}
	if _, err := s.CreateReservationWithTime(2, 2, "Rennlauf-Training", now.Add(-2*time.Hour)); err != nil {
		return err
	}
	if _, err := s.CreateReservationWithTime(4, 3, "Tiefschnee-Tour Gaisberg", now.Add(-75*time.Minute)); err != nil {
		return err
	}

	return nil
}
