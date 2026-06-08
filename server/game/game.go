package game

import (
	"errors"
	"time"
	"trivia-server/models"
)

func MakeMove(state *models.GameState, userID, row, col int, answer string) (*models.GameMove, int, error) {
	if state.Game.Status != models.GameStatusActive {
		return nil, state.Game.CurrentTurn, errors.New("game is not active")
	}

	if row < 0 || row >= 3 || col < 0 || col >= 3 {
		return nil, state.Game.CurrentTurn, errors.New("invalid grid position")
	}

	// Check if it's the player's turn
	playerIdx := state.Game.CurrentTurn % len(state.Players)
	if state.Players[playerIdx].UserID != userID {
		return nil, state.Game.CurrentTurn, errors.New("not your turn")
	}

	move := &models.GameMove{
		GameID:        state.Game.ID,
		UserID:        userID,
		GridRow:       row,
		GridCol:       col,
		PlayerAnswer:  answer,
		PlayerID:      &userID,
		IsValid:       false, // default false, set true if valid
		MoveTimestamp: time.Now(),
		Username:      state.Players[playerIdx].Username,
		PlayerName:    state.Players[playerIdx].Username,
	}

	// Always advance the turn regardless of validity
	state.Game.CurrentTurn = (state.Game.CurrentTurn + 1) % len(state.Players)

	return move, state.Game.CurrentTurn, nil
}

// CheckWin checks if the given user has won the game
func CheckWin(state *models.GameState, userID int) bool {
	grid := state.Grid

	// Check rows and columns
	for i := range 3 {
		if grid[i][0] != nil && grid[i][1] != nil && grid[i][2] != nil &&
			*grid[i][0].PlayerID == userID && *grid[i][1].PlayerID == userID && *grid[i][2].PlayerID == userID {
			return true
		}
		if grid[0][i] != nil && grid[1][i] != nil && grid[2][i] != nil &&
			*grid[0][i].PlayerID == userID && *grid[1][i].PlayerID == userID && *grid[2][i].PlayerID == userID {
			return true
		}
	}

	// Check diagonals
	if grid[0][0] != nil && grid[1][1] != nil && grid[2][2] != nil &&
		*grid[0][0].PlayerID == userID && *grid[1][1].PlayerID == userID && *grid[2][2].PlayerID == userID {
		return true
	}
	if grid[0][2] != nil && grid[1][1] != nil && grid[2][0] != nil &&
		*grid[0][2].PlayerID == userID && *grid[1][1].PlayerID == userID && *grid[2][0].PlayerID == userID {
		return true
	}

	return false
}

// NewGameState initializes a new GameState for a started game.
func NewGameState(game models.Game, players []models.GamePlayer) *models.GameState {
	var history [3][3][]models.CellAttempt
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			history[i][j] = []models.CellAttempt{}
		}
	}
	return &models.GameState{
		Game:        game,
		Players:     players,
		Moves:       []models.GameMove{},
		Grid:        [3][3]*models.GameMove{},
		CellHistory: history,
	}
}
