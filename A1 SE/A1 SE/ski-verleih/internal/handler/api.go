package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"ski-verleih/internal/domain"
	"ski-verleih/internal/store"
)

// Server bündelt die HTTP-Handler mit dem Store-Interface.
type Server struct {
	store store.Store
	mux   *http.ServeMux
}

// NewServer erstellt einen neuen HTTP-Handler-Server inkl. Routing und Static-File-Serving.
func NewServer(s store.Store, webDir string) *Server {
	srv := &Server{
		store: s,
		mux:   http.NewServeMux(),
	}
	srv.routes(webDir)
	return srv
}

// ServeHTTP implementiert das http.Handler-Interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes(webDir string) {
	s.mux.HandleFunc("/api/dashboard", s.handleDashboard)
	s.mux.HandleFunc("/api/skis", s.handleSkis)
	s.mux.HandleFunc("/api/skis/", s.handleSkiByID)
	s.mux.HandleFunc("/api/persons", s.handlePersons)
	s.mux.HandleFunc("/api/persons/", s.handlePersonByID)
	s.mux.HandleFunc("/api/reservations", s.handleReservations)
	s.mux.HandleFunc("/api/reservations/", s.handleReservationAction)

	fs := http.FileServer(http.Dir(webDir))
	s.mux.Handle("/", fs)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Methode nicht erlaubt")
		return
	}
	stats := s.store.GetDashboardStats()
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleSkis(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.store.ListSkis())
	case http.MethodPost:
		var req struct {
			Number   string `json:"number"`
			Type     string `json:"type"`
			LengthCm int    `json:"lengthCm"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Ungültiges JSON-Format")
			return
		}
		ski, err := s.store.CreateSki(req.Number, req.Type, req.LengthCm)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, ski)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Methode nicht erlaubt")
	}
}

func (s *Server) handleSkiByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "Methode nicht erlaubt")
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/api/skis/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Ski-ID")
		return
	}
	if err := s.store.DeleteSki(id); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handlePersons(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.store.ListPersons())
	case http.MethodPost:
		var req struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Phone string `json:"phone"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Ungültiges JSON-Format")
			return
		}
		person, err := s.store.CreatePerson(req.Name, req.Email, req.Phone)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, person)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Methode nicht erlaubt")
	}
}

func (s *Server) handlePersonByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "Methode nicht erlaubt")
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/api/persons/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Personen-ID")
		return
	}
	if err := s.store.DeletePerson(id); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleReservations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.store.ListReservations())
	case http.MethodPost:
		var req struct {
			SkiID    int    `json:"skiId"`
			PersonID int    `json:"personId"`
			Notes    string `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Ungültiges JSON-Format")
			return
		}
		res, err := s.store.CreateReservation(req.SkiID, req.PersonID, req.Notes)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, res)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Methode nicht erlaubt")
	}
}

func (s *Server) handleReservationAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Methode nicht erlaubt")
		return
	}
	// Erwartet Pfad: /api/reservations/{id}/return
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/reservations/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[1] != "return" {
		writeError(w, http.StatusNotFound, "Unbekannter Endpoint")
		return
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Reservierungs-ID")
		return
	}
	res, err := s.store.ReturnReservation(id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrSkiNotAvailable),
		errors.Is(err, domain.ErrAlreadyReturned),
		errors.Is(err, domain.ErrHasActiveRentals):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
