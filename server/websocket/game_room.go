package websocket

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrRoomNotFound = errors.New("room not found")
	ErrRoomFull     = errors.New("room is full")
	ErrPlayerExists = errors.New("player already exists in the room")
)

type GameRoom struct {
	ID        string
	CreatorID string
	Players   map[string]*Client
	State     GameState
	CreatedAt time.Time
	mu        sync.RWMutex
}

type GameState struct {
	Status      string `json:"status"`
	PlayerCount int    `json:"player_count"`
	MaxPlayers  int    `json:"max_players"`
}
