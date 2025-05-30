package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"trivia-server/db"
	"trivia-server/models"

	"github.com/gorilla/mux"
)

type TeamHandler struct {
	teamRepo *db.TeamRepository
}

func NewTeamHandler(teamRepo *db.TeamRepository) *TeamHandler {
	return &TeamHandler{teamRepo: teamRepo}
}

// GetAllTeams handles GET /api/teams
func (h *TeamHandler) GetAllTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := h.teamRepo.GetAllTeams()
	if err != nil {
		http.Error(w, "Failed to fetch teams", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"teams": teams,
		"count": len(teams),
	})
}

// GetActiveTeams handles GET /api/teams/active
func (h *TeamHandler) GetActiveTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := h.teamRepo.GetActiveTeams()
	if err != nil {
		http.Error(w, "Failed to fetch active teams", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"teams": teams,
		"count": len(teams),
	})
}

// GetTeamByID handles GET /api/teams/{id}
func (h *TeamHandler) GetTeamByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid team ID", http.StatusBadRequest)
		return
	}

	team, err := h.teamRepo.GetTeamByID(id)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

// GetTeamByName handles GET /api/teams/name/{name}
func (h *TeamHandler) GetTeamByName(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	team, err := h.teamRepo.GetTeamByName(name)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

// GetTeamsByLeague handles GET /api/teams/league/{league}
func (h *TeamHandler) GetTeamsByLeague(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	league := vars["league"]

	if league != "AL" && league != "NL" {
		http.Error(w, "Invalid league. Must be AL or NL", http.StatusBadRequest)
		return
	}

	teams, err := h.teamRepo.GetTeamsByLeague(league)
	if err != nil {
		http.Error(w, "Failed to fetch teams", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"league": league,
		"teams":  teams,
		"count":  len(teams),
	})
}

// GetTeamsByDivision handles GET /api/teams/league/{league}/division/{division}
func (h *TeamHandler) GetTeamsByDivision(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	league := vars["league"]
	division := vars["division"]

	if league != "AL" && league != "NL" {
		http.Error(w, "Invalid league. Must be AL or NL", http.StatusBadRequest)
		return
	}

	if division != "East" && division != "Central" && division != "West" {
		http.Error(w, "Invalid division. Must be East, Central, or West", http.StatusBadRequest)
		return
	}

	teams, err := h.teamRepo.GetTeamsByDivision(league, division)
	if err != nil {
		http.Error(w, "Failed to fetch teams", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"league":   league,
		"division": division,
		"teams":    teams,
		"count":    len(teams),
	})
}

// CreateTeam handles POST /api/teams
func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var team models.Team
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if team.Name == "" || team.City == "" || team.Abbreviation == "" {
		http.Error(w, "Name, city, and abbreviation are required", http.StatusBadRequest)
		return
	}

	if team.League != "AL" && team.League != "NL" {
		http.Error(w, "League must be AL or NL", http.StatusBadRequest)
		return
	}

	if team.Division != "East" && team.Division != "Central" && team.Division != "West" {
		http.Error(w, "Division must be East, Central, or West", http.StatusBadRequest)
		return
	}

	// Check if team already exists
	exists, err := h.teamRepo.TeamExists(team.Name)
	if err != nil {
		http.Error(w, "Failed to check team existence", http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, "Team already exists", http.StatusConflict)
		return
	}

	// Set default active status
	team.IsActive = true

	if err := h.teamRepo.CreateTeam(&team); err != nil {
		http.Error(w, "Failed to create team", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(team)
}

// UpdateTeam handles PUT /api/teams/{id}
func (h *TeamHandler) UpdateTeam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid team ID", http.StatusBadRequest)
		return
	}

	var team models.Team
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	team.ID = id

	if err := h.teamRepo.UpdateTeam(&team); err != nil {
		http.Error(w, "Failed to update team", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

// DeleteTeam handles DELETE /api/teams/{id}
func (h *TeamHandler) DeleteTeam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid team ID", http.StatusBadRequest)
		return
	}

	if err := h.teamRepo.DeleteTeam(id); err != nil {
		http.Error(w, "Failed to delete team", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RegisterTeamRoutes sets up all team-related routes
func (h *TeamHandler) RegisterTeamRoutes(r *mux.Router) {
	// Team routes
	r.HandleFunc("/api/teams", h.GetAllTeams).Methods("GET")
	r.HandleFunc("/api/teams/active", h.GetActiveTeams).Methods("GET")
	r.HandleFunc("/api/teams/{id:[0-9]+}", h.GetTeamByID).Methods("GET")
	r.HandleFunc("/api/teams/name/{name}", h.GetTeamByName).Methods("GET")
	r.HandleFunc("/api/teams/league/{league}", h.GetTeamsByLeague).Methods("GET")
	r.HandleFunc("/api/teams/league/{league}/division/{division}", h.GetTeamsByDivision).Methods("GET")
	r.HandleFunc("/api/teams", h.CreateTeam).Methods("POST")
	r.HandleFunc("/api/teams/{id:[0-9]+}", h.UpdateTeam).Methods("PUT")
	r.HandleFunc("/api/teams/{id:[0-9]+}", h.DeleteTeam).Methods("DELETE")
}
