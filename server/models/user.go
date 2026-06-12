package models

import (
	"time"
)

type User struct {
	ID               int       `json:"id" db:"id"`
	Username         string    `json:"username" db:"username"`
	Email            string    `json:"email" db:"email"`
	PasswordHash     string    `json:"-" db:"password_hash"` // Don't include in JSON
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
	IsActive         bool      `json:"is_active" db:"is_active"`
	GamesPlayed      int       `json:"games_played" db:"games_played"`
	GamesWon         int       `json:"games_won" db:"games_won"`
	FavoriteTeamID   *int      `json:"favorite_team_id,omitempty" db:"favorite_team_id"`
	FavoriteTeamName *string   `json:"favorite_team_name,omitempty" db:"favorite_team_name"`
}

// UserStats represents user game statistics
type UserStats struct {
	Username    string  `json:"username"`
	GamesPlayed int     `json:"games_played"`
	GamesWon    int     `json:"games_won"`
	WinRate     float64 `json:"win_rate"`
}

// GameHistoryEntry represents one past game for the profile page
type GameHistoryEntry struct {
	GameID       int       `json:"game_id"`
	OpponentName string    `json:"opponent_name"`
	Result       string    `json:"result"` // "win" | "loss" | "draw"
	Difficulty   string    `json:"difficulty"`
	PlayedAt     time.Time `json:"played_at"`
}
