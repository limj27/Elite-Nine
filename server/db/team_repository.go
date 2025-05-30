package db

import (
	"database/sql"
	"fmt"
	"trivia-server/models"
)

// TeamRepository handles team data operations
type TeamRepository struct {
	db *sql.DB
}

// NewTeamRepository creates a new team repository
func NewTeamRepository(db *sql.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

// CreateTeam inserts a new team into the database
func (r *TeamRepository) CreateTeam(team *models.Team) error {
	query := `
		INSERT INTO teams (name, city, abbreviation, league, division, founded, active)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query, team.Name, team.City, team.Abbreviation,
		team.League, team.Division, team.FoundedYear, team.IsActive)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get team ID: %w", err)
	}

	team.ID = int(id)
	return nil
}

// GetTeamByID retrieves a team by its ID
func (r *TeamRepository) GetTeamByID(id int) (*models.Team, error) {
	query := `
		SELECT id, name, city, abbreviation, league, division, founded, active, created_at, updated_at
		FROM teams
		WHERE id = ?
	`

	team := &models.Team{}
	err := r.db.QueryRow(query, id).Scan(
		&team.ID, &team.Name, &team.City, &team.Abbreviation,
		&team.League, &team.Division, &team.FoundedYear, &team.IsActive,
		&team.CreatedAt, &team.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("team with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	return team, nil
}

// GetTeamByName retrieves a team by its name
func (r *TeamRepository) GetTeamByName(name string) (*models.Team, error) {
	query := `
		SELECT id, name, city, abbreviation, league, division, founded, active, created_at, updated_at
		FROM teams
		WHERE name = ? OR abbreviation = ?
	`

	team := &models.Team{}
	err := r.db.QueryRow(query, name, name).Scan(
		&team.ID, &team.Name, &team.City, &team.Abbreviation,
		&team.League, &team.Division, &team.FoundedYear, &team.IsActive,
		&team.CreatedAt, &team.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("team '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	return team, nil
}

// GetAllTeams retrieves all teams
func (r *TeamRepository) GetAllTeams() ([]*models.Team, error) {
	query := `
		SELECT id, name, city, abbreviation, league, division, founded, active, created_at, updated_at
		FROM teams
		ORDER BY league, division, name
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query teams: %w", err)
	}
	defer rows.Close()

	var teams []*models.Team
	for rows.Next() {
		team := &models.Team{}
		err := rows.Scan(
			&team.ID, &team.Name, &team.City, &team.Abbreviation,
			&team.League, &team.Division, &team.FoundedYear, &team.IsActive,
			&team.CreatedAt, &team.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team: %w", err)
		}
		teams = append(teams, team)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating teams: %w", err)
	}

	return teams, nil
}

// GetActiveTeams retrieves only active teams
func (r *TeamRepository) GetActiveTeams() ([]*models.Team, error) {
	query := `
		SELECT id, name, city, abbreviation, league, division, founded, active, created_at, updated_at
		FROM teams
		WHERE active = true
		ORDER BY league, division, name
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active teams: %w", err)
	}
	defer rows.Close()

	var teams []*models.Team
	for rows.Next() {
		team := &models.Team{}
		err := rows.Scan(
			&team.ID, &team.Name, &team.City, &team.Abbreviation,
			&team.League, &team.Division, &team.FoundedYear, &team.IsActive,
			&team.CreatedAt, &team.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team: %w", err)
		}
		teams = append(teams, team)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active teams: %w", err)
	}

	return teams, nil
}

// GetTeamsByLeague retrieves teams by league (AL or NL)
func (r *TeamRepository) GetTeamsByLeague(league string) ([]*models.Team, error) {
	query := `
		SELECT id, name, city, abbreviation, league, division, founded, active, created_at, updated_at
		FROM teams
		WHERE league = ? AND active = true
		ORDER BY division, name
	`

	rows, err := r.db.Query(query, league)
	if err != nil {
		return nil, fmt.Errorf("failed to query teams by league: %w", err)
	}
	defer rows.Close()

	var teams []*models.Team
	for rows.Next() {
		team := &models.Team{}
		err := rows.Scan(
			&team.ID, &team.Name, &team.City, &team.Abbreviation,
			&team.League, &team.Division, &team.FoundedYear, &team.IsActive,
			&team.CreatedAt, &team.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team: %w", err)
		}
		teams = append(teams, team)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating teams by league: %w", err)
	}

	return teams, nil
}

// GetTeamsByDivision retrieves teams by division
func (r *TeamRepository) GetTeamsByDivision(league, division string) ([]*models.Team, error) {
	query := `
		SELECT id, name, city, abbreviation, league, division, founded, active, created_at, updated_at
		FROM teams
		WHERE league = ? AND division = ? AND active = true
		ORDER BY name
	`

	rows, err := r.db.Query(query, league, division)
	if err != nil {
		return nil, fmt.Errorf("failed to query teams by division: %w", err)
	}
	defer rows.Close()

	var teams []*models.Team
	for rows.Next() {
		team := &models.Team{}
		err := rows.Scan(
			&team.ID, &team.Name, &team.City, &team.Abbreviation,
			&team.League, &team.Division, &team.FoundedYear, &team.IsActive,
			&team.CreatedAt, &team.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team: %w", err)
		}
		teams = append(teams, team)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating teams by division: %w", err)
	}

	return teams, nil
}

// UpdateTeam updates an existing team
func (r *TeamRepository) UpdateTeam(team *models.Team) error {
	query := `
		UPDATE teams 
		SET name = ?, city = ?, abbreviation = ?, league = ?, division = ?, 
		    founded = ?, active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.Exec(query, team.Name, team.City, team.Abbreviation,
		team.League, team.Division, team.FoundedYear, team.IsActive, team.ID)
	if err != nil {
		return fmt.Errorf("failed to update team: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("team with ID %d not found", team.ID)
	}

	return nil
}

// DeleteTeam deletes a team by ID
func (r *TeamRepository) DeleteTeam(id int) error {
	query := `DELETE FROM teams WHERE id = ?`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("team with ID %d not found", id)
	}

	return nil
}

// TeamExists checks if a team exists by name or abbreviation
func (r *TeamRepository) TeamExists(name string) (bool, error) {
	query := `SELECT COUNT(*) FROM teams WHERE name = ? OR abbreviation = ?`

	var count int
	err := r.db.QueryRow(query, name, name).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check team existence: %w", err)
	}

	return count > 0, nil
}
