package websocket

import (
	"log"
	"trivia-server/game"
	"trivia-server/models"
)

// onTurnTimeout fires when a player's turn timer elapses without a move
// being made. It is passed as the callback to GameRoom.StartTurnTimer,
// so it accepts (room, turnAtStart) and restarts the timer for the next
// player once the turn has been skipped.
func onTurnTimeout(room *GameRoom, turnAtStart int) {
	room.mu.Lock()
	if room.GameModel == nil || room.GameModel.Game.Status != models.GameStatusActive {
		room.mu.Unlock()
		return
	}
	if room.GameModel.Game.CurrentTurn != turnAtStart {
		// A move already happened and advanced the turn — this timer
		// is stale, ignore it.
		room.mu.Unlock()
		return
	}

	game.SkipTurn(room.GameModel)
	log.Printf("Turn timed out in room %s, skipping to turn %d", room.ID, room.GameModel.Game.CurrentTurn)
	room.mu.Unlock()

	room.Broadcast(mustMarshal(map[string]interface{}{
		"type":    "turn_timeout",
		"payload": map[string]interface{}{"roomId": room.ID},
	}))

	room.Broadcast(mustMarshal(map[string]interface{}{
		"type":    "game_state",
		"payload": room.GameModel,
	}))

	// Start the timer again for whoever's turn it is now
	room.StartTurnTimer(onTurnTimeout)
}
