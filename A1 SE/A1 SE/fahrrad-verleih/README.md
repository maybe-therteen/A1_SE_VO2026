# Fahrradverleih (Aufgabe 1c – SDD implementiert)

Verleihsystem für Fahrräder mit Dashboard, Fahrrad- und Personenverwaltung, **flexiblem Ausleihbeginn, frei wählbarer Dauer und stundenweiser Abrechnung**. Spezifiziert in [`CLAUDE.md`](CLAUDE.md), implementiert in Go (nur Standard-Library) und reinem HTML/CSS/JS.

## Features
1. **Dashboard** – Verfügbare / ausgeliehene / gesamte Fahrräder, registrierte Personen, geplante Reservierungen, überfällige Rückgaben, Umsatz; Tabellen „Aktuell ausgeliehen“ (mit laufendem Preis) und „Nächste Abholungen“.
2. **Fahrrad-Verwaltung** – Nr. (z.B. `BIKE-101`), Typ (City, Trekking, Mountainbike, E-Bike, Rennrad, Kinder), Rahmenhöhe, Stundenpreis (Standard je Typ), Status, aktuelle bzw. nächste Reservierung; Filter nach Typ/Status/Suche; direkt „Ausleihen“.
3. **Personen-Verwaltung** – Anlegen, Löschen (nur ohne offene Ausleihen), Anzahl offener Ausleihen.
4. **Reservierungen** (aus Aufgabe 1b)
   - Beginn *sofort* oder *später* (Datum/Uhrzeit), Dauer in Stunden oder Tagen (1 h – 14 Tage), Schnellwahl-Chips.
   - Nur Fahrräder, die im gewählten Zeitraum frei sind, werden angeboten (Überschneidungsprüfung im Backend).
   - Live-Preisvorschau; Endpreis bei Rückgabe = jede angefangene Stunde × Stundenpreis.
   - Status: Reserviert → Aktiv (Abholen) → Zurückgegeben, oder Storniert. Überfällige Ausleihen werden markiert.

## Projektstruktur
```text
fahrrad-verleih/
├── CLAUDE.md                  # SDD-Spezifikation
├── README.md
├── go.mod
├── main.go                    # Einstieg + Seed (20 Fahrräder, 10 Personen, Beispiel-Reservierungen)
├── internal/
│   ├── domain/                # Bike, Person, Reservation, Preis- & Zeitlogik (+ Tests)
│   ├── store/                 # Thread-sicherer In-Memory-Store (+ Tests)
│   └── handler/api.go         # REST-API
└── web/                       # index.html, style.css, app.js
```

## Starten & Testen
Voraussetzung: Go 1.22 oder neuer.

```bash
go run .        # → http://localhost:8080
go test ./...
```
