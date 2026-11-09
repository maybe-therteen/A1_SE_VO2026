package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fahrrad-verleih/internal/domain"
	"fahrrad-verleih/internal/store"
)

// Server bündelt die HTTP-Handler mit dem Store-Interface.
type Server struct {
	store store.Store
	mux   *http.ServeMux
}

// NewServer erstellt einen neuen HTTP-Handler-Server inkl. Routing und Static-File-Serving.
func NewServer(s store.Store, webDir string) *Server {
	srv := &Server{store: s, mux: http.NewServeMux()}
	srv.routes(webDir)
	return srv
}

// ServeHTTP implementiert das http.Handler-Interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes(webDir string) {
	s.mux.HandleFunc("GET /api/dashboard", s.handleDashboard)
	s.mux.HandleFunc("GET /api/bike-types", s.handleBikeTypes)

	s.mux.HandleFunc("GET /api/bikes", s.handleListBikes)
	s.mux.HandleFunc("GET /api/bikes/available", s.handleAvailableBikes)
	s.mux.HandleFunc("POST /api/bikes", s.handleCreateBike)
	s.mux.HandleFunc("DELETE /api/bikes/{id}", s.handleDeleteBike)

	s.mux.HandleFunc("GET /api/persons", s.handleListPersons)
	s.mux.HandleFunc("POST /api/persons", s.handleCreatePerson)
	s.mux.HandleFunc("DELETE /api/persons/{id}", s.handleDeletePerson)

	s.mux.HandleFunc("GET /api/reservations", s.handleListReservations)
	s.mux.HandleFunc("POST /api/reservations", s.handleCreateReservation)
	s.mux.HandleFunc("POST /api/reservations/{id}/{action}", s.handleReservationAction)

	s.mux.Handle("/", http.FileServer(http.Dir(webDir)))
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.GetDashboardStats())
}

func (s *Server) handleBikeTypes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, domain.DefaultHourlyRates)
}

func (s *Server) handleListBikes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListBikes())
}

// handleAvailableBikes: GET /api/bikes/available?start=<RFC3339>&hours=<n>
func (s *Server) handleAvailableBikes(w http.ResponseWriter, r *http.Request) {
	start, err := parseOptionalTime(r.URL.Query().Get("start"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Ungültiger Startzeitpunkt")
		return
	}
	hours, err := strconv.Atoi(r.URL.Query().Get("hours"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Dauer")
		return
	}
	bikes, err := s.store.AvailableBikes(start, hours)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bikes)
}

func (s *Server) handleCreateBike(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Number      string  `json:"number"`
		Type        string  `json:"type"`
		FrameSizeCm int     `json:"frameSizeCm"`
		HourlyRate  float64 `json:"hourlyRate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Ungültiges JSON-Format")
		return
	}
	bike, err := s.store.CreateBike(req.Number, req.Type, req.FrameSizeCm, req.HourlyRate)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, bike)
}

func (s *Server) handleDeleteBike(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Fahrrad-ID")
		return
	}
	if err := s.store.DeleteBike(id); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleListPersons(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListPersons())
}

func (s *Server) handleCreatePerson(w http.ResponseWriter, r *http.Request) {
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
}

func (s *Server) handleDeletePerson(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
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

func (s *Server) handleListReservations(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListReservations())
}

func (s *Server) handleCreateReservation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BikeID   int    `json:"bikeId"`
		PersonID int    `json:"personId"`
		Start    string `json:"start"` // RFC3339, leer = sofort
		Hours    int    `json:"hours"`
		Notes    string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Ungültiges JSON-Format")
		return
	}
	start, err := parseOptionalTime(req.Start)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Ungültiger Startzeitpunkt")
		return
	}
	res, err := s.store.CreateReservation(req.BikeID, req.PersonID, start, req.Hours, req.Notes)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

// handleReservationAction: POST /api/reservations/{id}/{pickup|return|cancel}
func (s *Server) handleReservationAction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Reservierungs-ID")
		return
	}
	var res domain.Reservation
	switch r.PathValue("action") {
	case "pickup":
		res, err = s.store.PickupReservation(id)
	case "return":
		res, err = s.store.ReturnReservation(id)
	case "cancel":
		res, err = s.store.CancelReservation(id)
	default:
		writeError(w, http.StatusNotFound, "Unbekannte Aktion")
		return
	}
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func parseOptionalTime(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, v)
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
	case errors.Is(err, domain.ErrBikeNotAvailable),
		errors.Is(err, domain.ErrInvalidState),
		errors.Is(err, domain.ErrHasActiveRentals):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
