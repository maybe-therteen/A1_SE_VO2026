# Ski-Verleih (Aufgabe 1a – SDD implementiert)

Ein kundenfreundliches Verleihsystem für Ski mit Dashboard, Ski-Verwaltung, Personen-Verwaltung und lückenlos nachvollziehbarer Ausleihe (inkl. Ski-Inventarnummer und Ski-Typ). Implementiert mittels Spec-Driven Development (`CLAUDE.md`) in Go (Standard-Library) und reinem HTML/CSS/JS.

## Features (Aufgabe 1a)
1. **Dashboard**:
   - Echtzeit-Kennzahlen: *Verfügbare Ski*, *Ausgeliehene Ski*, *Gesamt Ski*, *Registrierte Personen* (initial vorbelegt mit 20 Ski – davon 17 verfügbar, 3 ausgeliehen – und 10 registrierten Personen, analog zur Vorlesungsvorlage).
   - Schnellzugriff-Aktionen (*Ski verwalten*, *Personen verwalten*, *Neue Ausleihe*) sowie eine Tabelle der aktuell aktiven Ausleihen inkl. direkter Rückgabe-Funktion.
2. **Ski-Verwaltung**:
   - Übersicht aller Ski mit Inventarnummer (`Nr.`), Typ (`Allround`, `Race`, `Freeride`, `Slalom`, `Touring`), Länge in cm und aktuellem Status.
   - Filterung nach Status/Suchbegriff sowie Anlegen und Löschen von Ski.
3. **Personen-Verwaltung**:
   - Übersicht aller registrierten Kunden inkl. Kontaktdaten und Anzahl aktiver Ausleihen.
   - Anlegen neuer Kunden und Löschen bestehender Kunden (sofern keine aktive Ausleihe vorliegt).
4. **Nachvollziehbare Ausleihe & Reservierungen**:
   - Neue Ausleihe über ein kundenfreundliches Modal oder direkt im Reiter *Reservierungen*.
   - Jede Ausleihe speichert und zeigt **Ski-Nummer + Ski-Typ + Länge**, die ausleihende Person, den Ausleihzeitpunkt (`StartTime`), den Rückgabezeitpunkt (`EndTime`) sowie den Status (`Aktiv` / `Zurückgegeben`).
   - Rückgabe per Klick gibt den Ski sofort wieder im Bestand frei.

## Projektstruktur
```text
ski-verleih/
├── CLAUDE.md                  # SDD-Spezifikation & Architektur-Regeln
├── README.md                  # Projektdokumentation
├── go.mod                     # Go-Moduldefinition (keine externen Dependencies)
├── main.go                    # Einstiegspunkt, Seed-Daten (20 Ski, 10 Personen, 3 Ausleihen), Server
├── internal/
│   ├── domain/
│   │   ├── models.go          # Kern-Structs (Ski, Person, Reservation, DashboardStats) & Methoden
│   │   └── models_test.go     # Unit-Tests für Domain-Logik
│   ├── store/
│   │   ├── memory.go          # Thread-sicherer In-Memory-Store (sync.RWMutex)
│   │   └── memory_test.go     # Unit-Tests für Store & Nachvollziehbarkeit der Ausleihe
│   └── handler/
│       └── api.go             # REST-API-Handler (/api/dashboard, /api/skis, /api/persons, /api/reservations)
└── web/
    ├── index.html             # Single-Page UI im Vorlesungs-Design
    ├── style.css              # Styling inkl. Gradient-Cards und responsivem Layout
    └── app.js                 # Vanilla-JS Frontend-Logik gegen die Go-REST-Endpoints
```

## Starten & Testen

### Voraussetzungen
- Go 1.21 oder neuer

### Server starten
```bash
go run .
```
Anschließend im Browser öffnen: **[http://localhost:8080](http://localhost:8080)**

### Tests ausführen
```bash
go test -v ./...
```
