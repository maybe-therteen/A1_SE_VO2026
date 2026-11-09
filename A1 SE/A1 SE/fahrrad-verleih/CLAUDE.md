# Prototype Project: Fahrradverleih (Aufgabe 1c)

Nur Prototypen. Priorität: schnell verständlich, leicht änderbar.
Basis: Ski-Verleih aus Aufgabe 1a inkl. Erweiterung 1b (flexibler Beginn/Dauer, stundenweise Preise), fachlich auf Fahrräder umgestellt.

## Stack
- Backend: Go ≥ 1.22 ('objektorientiert': Structs + Methoden, Interfaces wo sinnvoll)
- DB: In-Memory (kein Postgres/SQLite, kein ORM)
- Frontend: reines HTML/CSS/JS (ausgeliefert über den Go-Server unter `/`)
- Versionierung: lokales Git, bei Bedarf GitHub (public Repo für Abgabe)

## Commands
- Backend starten: `go run .` (Standard: `http://localhost:8080`, Port via `PORT`)
- Tests: `go test ./...`
- Format: `gofmt -w .` bzw. `go fmt ./...`
- Vet: `go vet ./...`

## Architektur (Go)
- `main.go` → Einstieg, Seed-Daten, HTTP-Server
- `internal/domain/` → Kern-Structs (`Bike`, `Person`, `Reservation`, `DashboardStats`) + Business-Logik (Validierung, Statusübergänge, Preisberechnung, Zeitfenster-Überschneidung)
- `internal/store/` → Thread-sicherer In-Memory-Store (`sync.RWMutex` + Maps) hinter einem `Store`-Interface
- `internal/handler/` → HTTP-Handler (Go-1.22-Routing-Patterns `GET /api/...`)
- Keine globalen Variablen außer Konstanten/Konfiguration
- Fehler mit `fmt.Errorf("%w", err)` wrappen, nicht verschlucken

## Konventionen
- Methoden an Structs statt freier Funktionen, wo sinnvoll
- Interfaces nur bei echtem Bedarf (`Store`)
- Keine externen Dependencies (nur Go Standard Library)
- Keine Secrets im Code

## Fachliche Anforderungen (Aufgabe 1c: Fahrradverleih)
- **Fahrrad**: eindeutige Inventarnummer (z.B. `BIKE-101`), Typ (`City`, `Trekking`, `Mountainbike`, `E-Bike`, `Rennrad`, `Kinder`), Rahmenhöhe 30–65 cm, Stundenpreis (Standard je Typ, überschreibbar), Status `Verfügbar` / `Ausgeliehen`.
- **Standard-Stundenpreise**: City 4 €, Trekking 5 €, Mountainbike 7 €, E-Bike 9 €, Rennrad 8 €, Kinder 3 €.
- **Personen**: Name, E-Mail (Pflicht, muss `@` enthalten), Telefon.
- **Reservierung / Ausleihe (aus 1b)**:
  - Beginn **sofort** oder zu einem **frei wählbaren späteren Zeitpunkt** (nicht in der Vergangenheit).
  - **Dauer** in Stunden, 1 h bis 14 Tage.
  - Ein Fahrrad darf sich nicht in überschneidenden Zeitfenstern befinden (offene Reservierungen = `Reserviert` oder `Aktiv`; überfällige aktive Ausleihen blockieren bis zur Rückgabe).
  - Status-Übergänge: `Reserviert → Aktiv` (Abholung), `Aktiv → Zurückgegeben` (Rückgabe), `Reserviert → Storniert`. Sofort-Buchung startet direkt als `Aktiv`.
  - **Preis**: geplant = Stunden × Stundenpreis; final bei Rückgabe = **jede angefangene Stunde** der tatsächlichen Nutzung (ab Abholung), mindestens 1 h.
  - Fahrrad-Daten (Nr., Typ, Rahmen) und Stundenpreis werden in die Reservierung kopiert → Historie bleibt nachvollziehbar.
- **Dashboard**: Karten *Verfügbare Fahrräder*, *Ausgeliehene Fahrräder*, *Gesamt Fahrräder*, *Registrierte Personen*; zusätzlich *Geplante Reservierungen*, *Überfällige Rückgaben*, *Umsatz*. Tabellen „Aktuell ausgeliehen“ und „Nächste Abholungen“.
- Löschen von Fahrrad/Person nur ohne offene Reservierungen.

## API
| Methode | Pfad | Zweck |
|---|---|---|
| GET | `/api/dashboard` | Kennzahlen |
| GET | `/api/bike-types` | Typen + Standardpreise |
| GET/POST | `/api/bikes` | Liste / anlegen |
| GET | `/api/bikes/available?start=RFC3339&hours=n` | freie Fahrräder im Zeitraum |
| DELETE | `/api/bikes/{id}` | löschen |
| GET/POST | `/api/persons`, DELETE `/api/persons/{id}` | Personen |
| GET/POST | `/api/reservations` | Liste / buchen (`bikeId, personId, start, hours, notes`) |
| POST | `/api/reservations/{id}/pickup\|return\|cancel` | Statuswechsel |

## Frontend
- Vanilla JS, `fetch` gegen `/api/...`, nie direkt gegen In-Memory-Strukturen
- Benutzerfreundlich: Schnellwahl-Chips für Dauer, Live-Preisvorschau, nur im Zeitraum freie Fahrräder wählbar, nächste Inventarnummer als Vorschlag

## Regeln
- Keine generierten Dateien von Hand editieren
- Commit-Messages: kurz und imperativ
- Bei Unklarheit fragen statt raten
- Prototyp-Fokus: lieber einfach und klar als elegant und komplex
