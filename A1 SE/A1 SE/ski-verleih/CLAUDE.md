# Prototype Project: Ski-Verleih (Aufgabe 1a)

Nur Prototypen. Priorität: schnell verständlich, leicht änderbar.

## Stack
- Backend: Go ('objektorientiert': Structs + Methoden, Interfaces wo sinnvoll)
- DB: In-Memory (kein Postgres/SQLite, kein Object-Relational Mapping (ORM) nötig)
- Frontend: reines HTML/CSS/JS (ausgeliefert über den Go-Server unter `/`)
- Versionierung: lokales Git, bei Bedarf GitHub

## Commands
- Backend starten: `go run .` (läuft standardmäßig auf `http://localhost:8080`)
- Tests: `go test ./...`
- Format: `gofmt -w .` bzw. `go fmt ./...`
- Lint (falls eingerichtet): `golangci-lint run`
- Frontend (HTML): direkt über `http://localhost:8080` im Browser öffnen

## Architektur (Go)
- `main.go` → Einstieg, Initialisierung mit Beispieldaten (Seed), HTTP-Server, Routing
- `internal/domain/` → Kern-Structs (`Ski`, `Person`, `Reservation`, `DashboardStats`) + Methoden (Business-Logik & Validierung)
- `internal/store/` → Thread-sicherer In-Memory-Store (`sync.RWMutex` + Maps) hinter einem `Store`-Interface
- `internal/handler/` → HTTP-Handler, die Domain-Objekte und Store-Methoden aufrufen
- Keine globalen Variablen außer bei Bedarf im Einstiegspunkt
- Fehler mit `fmt.Errorf("%w", err)` wrappen, nicht verschlucken

## Konventionen
- Methoden an Structs hängen statt freier Funktionen, wo es Sinn ergibt
- Interfaces nur bei echtem Bedarf (z.B. `Store`-Interface für spätere Austauschbarkeit und Tests)
- Kein Dependency Injection Framework
- Keine externen Dependencies (nur Go Standard Library)
- In-Memory-Store muss thread-safe sein (`sync.RWMutex`)
- Keine Secrets im Code (auch bei Prototypen nicht)

## Fachliche Anforderungen (Aufgabe 1a: Ski-Verleih)
- **Dashboard**: Übersichtskarten für *Verfügbare Ski*, *Ausgeliehene Ski*, *Gesamt Ski* und *Registrierte Personen* sowie Schnellzugriff-Buttons.
- **Ski-Verwaltung**: Ski mit eindeutiger Inventarnummer (z.B. `SKI-101`), Typ (`Allround`, `Race`, `Freeride`, `Slalom`, `Touring`), Länge (cm) und Status (`Verfügbar` / `Ausgeliehen`).
- **Personen-Verwaltung**: Registrierung und Verwaltung von Kunden (Name, E-Mail, Telefon).
- **Reservierungen / Ausleihe (Nachvollziehbarkeit)**:
  - Zuordnung einer Ausleihe zu einer Person und einem spezifischen Ski (inkl. Anzeige von Ski-Nummer + Ski-Typ).
  - Aktive Ausleihen und abgeschlossene Rückgaben sind jederzeit nachvollziehbar (Zeitstempel für Ausleihbeginn und Rückgabe).

## Frontend
- HTML-Variante: minimales Vanilla JS, klare Trennung von Backend-API (`fetch` gegen `/api/...`)
- API-Calls immer gegen die Go-Endpoints, nie direkt gegen In-Memory-Strukturen

## Regeln
- Keine generierten Dateien von Hand editieren
- Commit-Messages: kurz und imperativ
- Bei Unklarheit fragen statt raten
- Prototyp-Fokus: lieber einfach und klar als elegant und komplex
