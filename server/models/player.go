package models

import (
	"time"
)

type Player struct {
	ID              int        `json:"id" db:"id"`
	FirstName       string     `json:"first_name" db:"first_name"`
	LastName        string     `json:"last_name" db:"last_name"`
	BirthDate       *time.Time `json:"birth_date" db:"birth_date"`
	BirthCity       string     `json:"birth_city" db:"birth_city"`
	BirthState      string     `json:"birth_state" db:"birth_state"`
	BirthCountry    string     `json:"birth_country" db:"birth_country"`
	DebutDate       *time.Time `json:"debut_date" db:"debut_date"`
	FinalGameDate   *time.Time `json:"final_game_date" db:"final_game_date"`
	PrimaryPosition string     `json:"primary_position" db:"primary_position"`
	Bats            string     `json:"bats" db:"bats"`
	Throws          string     `json:"throws" db:"throws"`
	HeightInches    *int       `json:"height_inches" db:"height_inches"`
	WeightLbs       *int       `json:"weight_lbs" db:"weight_lbs"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// PlayerSummary for API responses
type PlayerSummary struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	FullName  string `json:"full_name"`
	Position  string `json:"position"`
}

// FullName returns the player's full name
func (p *Player) FullName() string {
	return p.FirstName + " " + p.LastName
}

// PlayerTeam represents a player's association with a team
type PlayerTeam struct {
	ID            int    `json:"id" db:"id"`
	PlayerID      int    `json:"player_id" db:"player_id"`
	TeamID        int    `json:"team_id" db:"team_id"`
	StartYear     int    `json:"start_year" db:"start_year"`
	EndYear       *int   `json:"end_year" db:"end_year"`
	Position      string `json:"position" db:"position"`
	IsPrimaryTeam bool   `json:"is_primary_team" db:"is_primary_team"`

	// Joined fields
	PlayerName string `json:"player_name,omitempty"`
	TeamName   string `json:"team_name,omitempty"`
}
