package grid

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
)

// ═══════════════════════════════════════════════════════════
// TYPES
// ═══════════════════════════════════════════════════════════

type Criteria struct {
	ID         int    `json:"id"`
	Type       string `json:"type"`
	Label      string `json:"label"`
	ShortLabel string `json:"short_label"`
	MlbTeamID  *int   `json:"mlb_team_id,omitempty"` // optional, only for team-based criteria
}

type GridTemplate struct {
	ID          int        `json:"id"`
	RowCriteria []Criteria `json:"row_criteria"` // 3 items
	ColCriteria []Criteria `json:"col_criteria"` // 3 items
	Difficulty  string     `json:"difficulty"`
}

type CellAnswer struct {
	MlbID       int     `json:"mlb_id"`
	PlayerName  string  `json:"player_name"`
	HeadshotURL string  `json:"headshot_url"`
	RarityScore float64 `json:"rarity_score"`
}

type ValidationResult struct {
	Valid       bool       `json:"valid"`
	Answer      CellAnswer `json:"answer"`
	RarityScore float64    `json:"rarity_score"`
	Message     string     `json:"message"`
}

// ═══════════════════════════════════════════════════════════
// SERVICE
// ═══════════════════════════════════════════════════════════

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// GetRandomGrid picks a random active grid template from the database
func (s *Service) GetRandomGrid() (*GridTemplate, error) {
	// Get count of available grids
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM grid_templates WHERE active = TRUE").Scan(&count)
	if err != nil || count == 0 {
		return nil, errors.New("no grid templates available")
	}

	// Pick a random offset
	offset := rand.Intn(count)

	var gt GridTemplate
	var rowIDs [3]int
	var colIDs [3]int

	err = s.db.QueryRow(`
		SELECT id, row_criteria_1, row_criteria_2, row_criteria_3,
		       col_criteria_1, col_criteria_2, col_criteria_3, difficulty
		FROM grid_templates
		WHERE active = TRUE
		LIMIT 1 OFFSET ?
	`, offset).Scan(
		&gt.ID,
		&rowIDs[0], &rowIDs[1], &rowIDs[2],
		&colIDs[0], &colIDs[1], &colIDs[2],
		&gt.Difficulty,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grid template: %w", err)
	}

	// Fetch criteria details
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

	return &gt, nil
}

func (s *Service) getCriteria(id int) (*Criteria, error) {
	c := &Criteria{}
	err := s.db.QueryRow(`
		SELECT id, type, label, COALESCE(short_label, label),
				mlb_team_id
		FROM criteria WHERE id = ?
	`, id).Scan(&c.ID, &c.Type, &c.Label, &c.ShortLabel, &c.MlbTeamID)
	if err != nil {
		return nil, fmt.Errorf("criteria %d not found: %w", id, err)
	}
	return c, nil
}

// ValidateAnswer checks if a player is a valid answer for a given cell
// Returns the answer with rarity score if valid
func (s *Service) ValidateAnswer(gridTemplateID, rowIndex, colIndex, mlbID int, playerName string) (*ValidationResult, error) {
	result := &ValidationResult{}

	// Look up the answer in pre-computed cell_answers table
	var answer CellAnswer
	err := s.db.QueryRow(`
		SELECT mlb_id, player_name, COALESCE(headshot_url, ''), rarity_score
		FROM cell_answers
		WHERE grid_template_id = ?
		  AND row_index = ?
		  AND col_index = ?
		  AND (mlb_id = ? OR LOWER(player_name) = LOWER(?))
	`, gridTemplateID, rowIndex, colIndex, mlbID, playerName).Scan(
		&answer.MlbID,
		&answer.PlayerName,
		&answer.HeadshotURL,
		&answer.RarityScore,
	)

	if err == sql.ErrNoRows {
		result.Valid = false
		result.Message = "That player doesn't satisfy both criteria for this cell"
		return result, nil
	}

	if err != nil {
		return nil, fmt.Errorf("validation query failed: %w", err)
	}

	// Update answer frequency for rarity tracking
	s.db.Exec(`
		INSERT INTO answer_frequency (mlb_id, use_count)
		VALUES (?, 1)
		ON DUPLICATE KEY UPDATE use_count = use_count + 1
	`, answer.MlbID)

	result.Valid = true
	result.Answer = answer
	result.RarityScore = answer.RarityScore
	result.Message = "Valid answer!"
	return result, nil
}

// GetCellAnswers returns all valid answers for a cell (for debugging/admin)
func (s *Service) GetCellAnswers(gridTemplateID, rowIndex, colIndex int) ([]CellAnswer, error) {
	rows, err := s.db.Query(`
		SELECT mlb_id, player_name, COALESCE(headshot_url, ''), rarity_score
		FROM cell_answers
		WHERE grid_template_id = ? AND row_index = ? AND col_index = ?
		ORDER BY rarity_score ASC
	`, gridTemplateID, rowIndex, colIndex)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var answers []CellAnswer
	for rows.Next() {
		var a CellAnswer
		if err := rows.Scan(&a.MlbID, &a.PlayerName, &a.HeadshotURL, &a.RarityScore); err != nil {
			continue
		}
		answers = append(answers, a)
	}
	return answers, nil
}
