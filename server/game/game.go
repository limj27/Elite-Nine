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

	// Check bounds
	if row < 0 || row >= 3 || col < 0 || col >= 3 {
		return nil, state.Game.CurrentTurn, errors.New("invalid grid position")
	}

	// Check if cell already filled
	if state.Grid[row][col] != nil {
		return nil, state.Game.CurrentTurn, errors.New("cell already filled")
	}

	// Check if it's the player's turn
	playerIdx := state.Game.CurrentTurn % len(state.Players)
	if state.Players[playerIdx].UserID != userID {
		return nil, state.Game.CurrentTurn, errors.New("not your turn")
	}

	// Validate answer (stubbed here, implement actual validation)
	isValid := true //TODO: implement real answer validation

	move := &models.GameMove{
		GameID:        state.Game.ID,
		UserID:        userID,
		GridRow:       row,
		GridCol:       col,
		PlayerAnswer:  answer,
		PlayerID:      &userID,
		IsValid:       isValid,
		MoveTimestamp: time.Now(),
		Username:      state.Players[playerIdx].Username,
		PlayerName:    state.Players[playerIdx].Username,
	}
	state.Moves = append(state.Moves, *move)
	state.Grid[row][col] = move

	if isValid {
		if CheckWin(state, userID) {
			state.Game.Status = models.GameStatusCompleted
			state.Game.WinnerID = &userID
		} else {
			state.Game.CurrentTurn = (state.Game.CurrentTurn + 1) % len(state.Players)
		}
	}

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
