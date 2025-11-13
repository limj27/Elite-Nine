package handlers

import (
	"net/http"
	"trivia-server/websocket"
)

type GameHandler struct {
	GM *websocket.GameManager
}

func NewGameHandler(gm *websocket.GameManager) *GameHandler {
	return &GameHandler{GM: gm}
}

func (gh *GameHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Implementation for creating a new game using gh.GM
}

// Might not use http handler directly for other actions,
// as they could be handled via WebSocket messages.
