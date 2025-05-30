package models

import (
	"encoding/json"
	"time"
)

type GameStatus string

const (
	GameStatusWaiting   GameStatus = "waiting"
	GameStatusActive    GameStatus = "active"
	GameStatusCompleted GameStatus = "completed"
	GameStatusAbandoned GameStatus = "abandoned"
)

type Game struct {
	ID          int             `json:"id" db:"id"`
	GameUUID    string          `json:"game_uuid" db:"game_uuid"`
	Status      GameStatus      `json:"status" db:"status"`
	GridConfig  json.RawMessage `json:"grid_config" db:"grid_config"`
	MaxPlayers  int             `json:"max_players" db:"max_players"`
	CurrentTurn int             `json:"current_turn" db:"current_turn"`
	WinnerID    *int            `json:"winner_id" db:"winner_id"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	CompletedAt *time.Time      `json:"completed_at" db:"completed_at"`
}

// GridConfig represents the 3x3 grid configuration
type GridConfig struct {
	Categories [3][3]GridCell `json:"categories"`
}

// GridCell represents one cell in the grid
type GridCell struct {
	RowCategory string `json:"row_category"`
	ColCategory string `json:"col_category"`
	Criteria    string `json:"criteria"`
}

// GamePlayer represents a player in a game
type GamePlayer struct {
	ID           int       `json:"id" db:"id"`
	GameID       int       `json:"game_id" db:"game_id"`
	UserID       int       `json:"user_id" db:"user_id"`
	PlayerNumber int       `json:"player_number" db:"player_number"`
	JoinedAt     time.Time `json:"joined_at" db:"joined_at"`

	// Joined fields
	Username string `json:"username,omitempty"`
}

// GameMove represents a move in the grid
type GameMove struct {
	ID            int       `json:"id" db:"id"`
	GameID        int       `json:"game_id" db:"game_id"`
	UserID        int       `json:"user_id" db:"user_id"`
	GridRow       int       `json:"grid_row" db:"grid_row"`
	GridCol       int       `json:"grid_col" db:"grid_col"`
	PlayerAnswer  string    `json:"player_answer" db:"player_answer"`
	PlayerID      *int      `json:"player_id" db:"player_id"`
	IsValid       bool      `json:"is_valid" db:"is_valid"`
	MoveTimestamp time.Time `json:"move_timestamp" db:"move_timestamp"`

	// Joined fields
	Username   string `json:"username,omitempty"`
	PlayerName string `json:"player_name,omitempty"`
}

// GameState represents the current state of a game for real-time updates
type GameState struct {
	Game    Game            `json:"game"`
	Players []GamePlayer    `json:"players"`
	Moves   []GameMove      `json:"moves"`
	Grid    [3][3]*GameMove `json:"grid"` // 2D array showing current grid state
}
