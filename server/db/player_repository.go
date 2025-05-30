package db

import (
	"database/sql"
	"fmt"
	"strings"
	"trivia-server/models"
)

type PlayerRepository struct {
	db *DB
}

func NewPlayerRepository(db *DB) *PlayerRepository {
	return &PlayerRepository{db: db}
}

// CreatePlayer creates a new player
func (pr *PlayerRepository) CreatePlayer(player *models.Player) error {
	query := `
		INSERT INTO players (first_name, last_name, birth_date, birth_city, birth_state, 
		                    birth_country, debut_date, final_game_date, primary_position, 
		                    bats, throws, height_inches, weight_lbs)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := pr.db.Exec(query,
		player.FirstName, player.LastName, player.BirthDate, player.BirthCity,
		player.BirthState, player.BirthCountry, player.DebutDate, player.FinalGameDate,
		player.PrimaryPosition, player.Bats, player.Throws, player.HeightInches, player.WeightLbs)

	if err != nil {
		return fmt.Errorf("error creating player: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error getting player ID: %v", err)
	}

	player.ID = int(id)
	return nil
}

// GetPlayerByID retrieves a player by ID
func (pr *PlayerRepository) GetPlayerByID(id int) (*models.Player, error) {
	query := `
		SELECT id, first_name, last_name, birth_date, birth_city, birth_state, 
		       birth_country, debut_date, final_game_date, primary_position, 
		       bats, throws, height_inches, weight_lbs, created_at, updated_at
		FROM players WHERE id = ?
	`

	player := &models.Player{}
	err := pr.db.QueryRow(query, id).Scan(
		&player.ID, &player.FirstName, &player.LastName, &player.BirthDate,
		&player.BirthCity, &player.BirthState, &player.BirthCountry,
		&player.DebutDate, &player.FinalGameDate, &player.PrimaryPosition,
		&player.Bats, &player.Throws, &player.HeightInches, &player.WeightLbs,
		&player.CreatedAt, &player.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("player not found")
		}
		return nil, fmt.Errorf("error getting player: %v", err)
	}

	return player, nil
}

// SearchPlayers searches for players by name
func (pr *PlayerRepository) SearchPlayers(query string) ([]models.PlayerSummary, error) {
	searchQuery := `
		SELECT id, first_name, last_name, primary_position
		FROM players 
		WHERE first_name LIKE ? OR last_name LIKE ?
		ORDER BY last_name, first_name
		LIMIT 20
	`

	searchTerm := "%" + strings.ToLower(query) + "%"
	rows, err := pr.db.Query(searchQuery, searchTerm, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("error searching players: %v", err)
	}
	defer rows.Close()

	var players []models.PlayerSummary
	for rows.Next() {
		var player models.PlayerSummary
		err := rows.Scan(&player.ID, &player.FirstName, &player.LastName, &player.Position)
		if err != nil {
			return nil, fmt.Errorf("error scanning player: %v", err)
		}
		player.FullName = player.FirstName + " " + player.LastName
		players = append(players, player)
	}

	return players, nil
}

// GetPlayersByTeam gets all players who played for a specific team
func (pr *PlayerRepository) GetPlayersByTeam(teamID int) ([]models.PlayerSummary, error) {
	query := `
		SELECT DISTINCT p.id, p.first_name, p.last_name, p.primary_position
		FROM players p
		JOIN player_teams pt ON p.id = pt.player_id
		WHERE pt.team_id = ?
		ORDER BY p.last_name, p.first_name
	`

	rows, err := pr.db.Query(query, teamID)
	if err != nil {
		return nil, fmt.Errorf("error getting players by team: %v", err)
	}
	defer rows.Close()

	var players []models.PlayerSummary
	for rows.Next() {
		var player models.PlayerSummary
		err := rows.Scan(&player.ID, &player.FirstName, &player.LastName, &player.Position)
		if err != nil {
			return nil, fmt.Errorf("error scanning player: %v", err)
		}
		player.FullName = player.FirstName + " " + player.LastName
		players = append(players, player)
	}

	return players, nil
}

// ValidatePlayerForGrid checks if a player satisfies grid criteria
func (pr *PlayerRepository) ValidatePlayerForGrid(playerName string, rowCriteria, colCriteria string) (bool, *models.Player, error) {
	// This is a simplified version - you'll need to implement specific validation logic
	// based on your grid criteria (team, awards, stats, etc.)

	// First, find the player
	searchQuery := `
		SELECT id, first_name, last_name, primary_position
		FROM players 
		WHERE CONCAT(first_name, ' ', last_name) LIKE ?
		LIMIT 1
	`

	var player models.Player
	err := pr.db.QueryRow(searchQuery, "%"+playerName+"%").Scan(
		&player.ID, &player.FirstName, &player.LastName, &player.PrimaryPosition)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil, fmt.Errorf("player not found")
		}
		return false, nil, fmt.Errorf("error finding player: %v", err)
	}

	// TODO: Implement specific validation logic based on criteria
	// For now, return true if player exists
	return true, &player, nil
}
