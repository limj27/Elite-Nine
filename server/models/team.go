package models

import "time"

type Team struct {
	ID           int       `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	City         string    `json:"city" db:"city"`
	Abbreviation string    `json:"abbreviation" db:"abbreviation"`
	League       string    `json:"league" db:"league"` // AL or NL
	Division     string    `json:"division" db:"division"`
	FoundedYear  *int      `json:"founded_year" db:"founded_year"`
	Colors       string    `json:"colors" db:"colors"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// TeamSummary for lighter API responses
type TeamSummary struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	City         string `json:"city"`
	Abbreviation string `json:"abbreviation"`
	League       string `json:"league"`
}
