package grid

import (
	"database/sql"
	"fmt"
	"math/rand"
)

const minAnswersPerCell = 3
const maxGenerationAttempts = 8

// GetFavoriteTeamCriteriaID resolves a user's favorite_team_id (an MLB Stats
// API team id, e.g. 147 for the Yankees) to a row in the criteria table.
// Returns nil if the user has no favorite team set or it doesn't match
// a known team criteria.
func GetFavoriteTeamCriteriaID(db *sql.DB, userID int) (*int, error) {
	var mlbTeamID sql.NullInt64
	err := db.QueryRow(`SELECT favorite_team_id FROM users WHERE id = ?`, userID).Scan(&mlbTeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to load favorite team for user %d: %w", userID, err)
	}
	if !mlbTeamID.Valid {
		return nil, nil
	}

	var criteriaID int
	err = db.QueryRow(`SELECT id FROM criteria WHERE type = 'team' AND mlb_team_id = ?`, mlbTeamID.Int64).Scan(&criteriaID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to resolve favorite team criteria: %w", err)
	}
	return &criteriaID, nil
}

// GenerateGrid builds a fresh grid template on the fly based on difficulty
// and the two players' favorite teams, validates that every cell has at
// least minAnswersPerCell valid answers, persists it to grid_templates +
// cell_answers, and returns it ready to use.
//
// p1FavTeamCriteriaID / p2FavTeamCriteriaID may be nil if a player has no
// favorite team set — in that case a random team is used in its place.
func (s *Service) GenerateGrid(difficulty string, p1FavTeamCriteriaID, p2FavTeamCriteriaID *int) (*GridTemplate, error) {
	teamIDs, statIDs, err := s.loadCriteriaPools()
	if err != nil {
		return nil, err
	}
	if len(teamIDs) < 6 || len(statIDs) < 2 {
		return nil, fmt.Errorf("not enough criteria to generate a grid")
	}

	for attempt := 0; attempt < maxGenerationAttempts; attempt++ {
		rowIDs, colIDs, err := buildCriteriaSets(difficulty, p1FavTeamCriteriaID, p2FavTeamCriteriaID, teamIDs, statIDs)
		if err != nil {
			return nil, err
		}

		cellData, totalAnswers, ok := s.collectCellAnswers(rowIDs, colIDs)
		if !ok {
			continue // a cell didn't meet the minimum — retry with new random slots
		}

		gt, err := s.persistGeneratedGrid(rowIDs, colIDs, difficulty, totalAnswers, cellData)
		if err != nil {
			return nil, err
		}
		return gt, nil
	}

	// Fallback — couldn't build a satisfying grid with favorite teams after
	// several attempts (not enough data for that team combo). Fall back to
	// a pre-built random grid so the game can still start.
	return s.GetRandomGrid()
}

// loadCriteriaPools returns all team criteria IDs and all non-team
// (stat/award) criteria IDs.
func (s *Service) loadCriteriaPools() (teamIDs []int, statIDs []int, err error) {
	rows, err := s.db.Query(`SELECT id, type FROM criteria`)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load criteria pools: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var cType string
		if err := rows.Scan(&id, &cType); err != nil {
			continue
		}
		if cType == "team" {
			teamIDs = append(teamIDs, id)
		} else {
			statIDs = append(statIDs, id)
		}
	}
	return teamIDs, statIDs, nil
}

// buildCriteriaSets returns 3 row criteria IDs and 3 col criteria IDs
// based on the requested difficulty.
func buildCriteriaSets(difficulty string, p1Fav, p2Fav *int, teamIDs, statIDs []int) (rowIDs, colIDs [3]int, err error) {
	used := map[int]bool{}

	pickRandomTeam := func() (int, error) {
		for i := 0; i < 25; i++ {
			id := teamIDs[rand.Intn(len(teamIDs))]
			if !used[id] {
				used[id] = true
				return id, nil
			}
		}
		return 0, fmt.Errorf("could not find a unique random team")
	}

	pickRandomStat := func() (int, error) {
		for i := 0; i < 25; i++ {
			id := statIDs[rand.Intn(len(statIDs))]
			if !used[id] {
				used[id] = true
				return id, nil
			}
		}
		return 0, fmt.Errorf("could not find a unique random stat")
	}

	switch difficulty {

	case "easy":
		// Row 1: player 1's favorite team (or random team if unset)
		// Row 2/3: stat criteria
		// Col 1: player 2's favorite team (or random team if unset/duplicate)
		// Col 2/3: stat criteria
		r0, err := resolveFavoriteOrRandomTeam(p1Fav, used, pickRandomTeam)
		if err != nil {
			return rowIDs, colIDs, err
		}
		rowIDs[0] = r0

		r1, err := pickRandomStat()
		if err != nil {
			return rowIDs, colIDs, err
		}
		rowIDs[1] = r1

		r2, err := pickRandomStat()
		if err != nil {
			return rowIDs, colIDs, err
		}
		rowIDs[2] = r2

		c0, err := resolveFavoriteOrRandomTeam(p2Fav, used, pickRandomTeam)
		if err != nil {
			return rowIDs, colIDs, err
		}
		colIDs[0] = c0

		c1, err := pickRandomStat()
		if err != nil {
			return rowIDs, colIDs, err
		}
		colIDs[1] = c1

		c2, err := pickRandomStat()
		if err != nil {
			return rowIDs, colIDs, err
		}
		colIDs[2] = c2

	case "regular":
		// Row 1: player 1's favorite team   | Col 1: player 2's favorite team
		// Row 2: random team                | Col 2: random team
		// Row 3: stat criteria              | Col 3: stat criteria
		r0, err := resolveFavoriteOrRandomTeam(p1Fav, used, pickRandomTeam)
		if err != nil {
			return rowIDs, colIDs, err
		}
		rowIDs[0] = r0

		r1, err := pickRandomTeam()
		if err != nil {
			return rowIDs, colIDs, err
		}
		rowIDs[1] = r1

		r2, err := pickRandomStat()
		if err != nil {
			return rowIDs, colIDs, err
		}
		rowIDs[2] = r2

		c0, err := resolveFavoriteOrRandomTeam(p2Fav, used, pickRandomTeam)
		if err != nil {
			return rowIDs, colIDs, err
		}
		colIDs[0] = c0

		c1, err := pickRandomTeam()
		if err != nil {
			return rowIDs, colIDs, err
		}
		colIDs[1] = c1

		c2, err := pickRandomStat()
		if err != nil {
			return rowIDs, colIDs, err
		}
		colIDs[2] = c2

	default: // "hard" — fully random, no favorite teams
		grid_type := rand.Intn(4)
		var nTeamsRow, nTeamsCol int
		if grid_type == 0 {
			// all teams
			nTeamsRow, nTeamsCol = 3, 3
		} else {
			// mixed — between 1 and 2 teams per side
			nTeamsRow = 1 + rand.Intn(2)
			nTeamsCol = 1 + rand.Intn(2)
		}

		fillSide := func(nTeams int) ([3]int, error) {
			var side [3]int
			for i := 0; i < 3; i++ {
				if i < nTeams {
					id, err := pickRandomTeam()
					if err != nil {
						return side, err
					}
					side[i] = id
				} else {
					id, err := pickRandomStat()
					if err != nil {
						return side, err
					}
					side[i] = id
				}
			}
			return side, nil
		}

		rowIDs, err = fillSide(nTeamsRow)
		if err != nil {
			return rowIDs, colIDs, err
		}
		colIDs, err = fillSide(nTeamsCol)
		if err != nil {
			return rowIDs, colIDs, err
		}
	}

	return rowIDs, colIDs, nil
}

// resolveFavoriteOrRandomTeam returns the given favorite team criteria ID if
// it's set and not already used; otherwise picks a random unused team.
func resolveFavoriteOrRandomTeam(fav *int, used map[int]bool, pickRandomTeam func() (int, error)) (int, error) {
	if fav != nil && !used[*fav] {
		used[*fav] = true
		return *fav, nil
	}
	return pickRandomTeam()
}

// cellAnswerRow mirrors a row from cell_answers before insertion.
type cellAnswerRow struct {
	mlbID       int
	playerName  string
	headshotURL string
	rarity      float64
}

// collectCellAnswers fetches valid answers for every cell in the proposed
// grid. Returns ok=false if any cell falls below minAnswersPerCell.
func (s *Service) collectCellAnswers(rowIDs, colIDs [3]int) (map[[2]int][]cellAnswerRow, int, bool) {
	cellData := make(map[[2]int][]cellAnswerRow)
	total := 0

	for ri, rowC := range rowIDs {
		for ci, colC := range colIDs {
			answers, err := s.getValidAnswersWithRarity(rowC, colC)
			if err != nil || len(answers) < minAnswersPerCell {
				return nil, 0, false
			}
			cellData[[2]int{ri, ci}] = answers
			total += len(answers)
		}
	}

	return cellData, total, true
}

// getValidAnswersWithRarity finds players satisfying both criteria and
// includes their existing rarity score (computed by rebuild_cell_answers.py
// based on career accomplishments).
func (s *Service) getValidAnswersWithRarity(rowCriteriaID, colCriteriaID int) ([]cellAnswerRow, error) {
	rows, err := s.db.Query(`
		SELECT p.mlb_id, p.full_name, COALESCE(p.headshot_url, ''),
		       COALESCE((SELECT rarity_score FROM cell_answers WHERE mlb_id = p.mlb_id LIMIT 1), 0.5)
		FROM mlb_players p
		JOIN player_criteria pc1 ON p.mlb_id = pc1.mlb_id AND pc1.criteria_id = ?
		JOIN player_criteria pc2 ON p.mlb_id = pc2.mlb_id AND pc2.criteria_id = ?
	`, rowCriteriaID, colCriteriaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []cellAnswerRow
	for rows.Next() {
		var r cellAnswerRow
		if err := rows.Scan(&r.mlbID, &r.playerName, &r.headshotURL, &r.rarity); err != nil {
			continue
		}
		results = append(results, r)
	}
	return results, nil
}

// persistGeneratedGrid writes the generated grid template and its cell
// answers to the database and returns it fully populated.
func (s *Service) persistGeneratedGrid(rowIDs, colIDs [3]int, difficulty string, totalAnswers int, cellData map[[2]int][]cellAnswerRow) (*GridTemplate, error) {
	dbDifficulty := difficulty
	if dbDifficulty != "easy" && dbDifficulty != "medium" && dbDifficulty != "hard" {
		dbDifficulty = "medium" // "regular" maps to the medium column value
	}

	res, err := s.db.Exec(`
		INSERT INTO grid_templates
		(row_criteria_1, row_criteria_2, row_criteria_3,
		 col_criteria_1, col_criteria_2, col_criteria_3,
		 min_answers, difficulty, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, TRUE)
	`, rowIDs[0], rowIDs[1], rowIDs[2], colIDs[0], colIDs[1], colIDs[2], totalAnswers, dbDifficulty)
	if err != nil {
		return nil, fmt.Errorf("failed to insert generated grid template: %w", err)
	}

	gridID64, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get generated grid id: %w", err)
	}
	gridID := int(gridID64)

	for cell, answers := range cellData {
		ri, ci := cell[0], cell[1]
		for _, a := range answers {
			_, err := s.db.Exec(`
				INSERT IGNORE INTO cell_answers
				(grid_template_id, row_index, col_index, mlb_id, player_name, headshot_url, rarity_score)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, gridID, ri, ci, a.mlbID, a.playerName, a.headshotURL, a.rarity)
			if err != nil {
				// Non-fatal — skip individual bad rows
				continue
			}
		}
	}

	gt := &GridTemplate{
		ID:         gridID,
		Difficulty: difficulty,
	}

	gt.RowCriteria = make([]Criteria, 3)
	gt.ColCriteria = make([]Criteria, 3)
	for i, id := range rowIDs {
		c, err := s.getCriteria(id)
		if err != nil {
			return nil, err
		}
		gt.RowCriteria[i] = *c
	}
	for i, id := range colIDs {
		c, err := s.getCriteria(id)
		if err != nil {
			return nil, err
		}
		gt.ColCriteria[i] = *c
	}

	return gt, nil
}
