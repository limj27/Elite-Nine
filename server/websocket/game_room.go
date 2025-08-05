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

func NewGameRoom(id, creatorID string) *GameRoom {
	return &GameRoom{
		ID:        id,
		CreatorID: creatorID,
		Players:   make(map[string]*Client),
		State: GameState{
			Status:      "waiting",
			PlayerCount: 0,
			MaxPlayers:  2, // Example max players
		},
		CreatedAt: time.Now(),
	}
}

func (r *GameRoom) AddPlayer(client *Client) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.Players) >= r.State.MaxPlayers {
		return ErrRoomFull
	}

	if _, exists := r.Players[client.ID]; exists {
		return ErrPlayerExists
	}

	r.Players[client.ID] = client
	r.State.PlayerCount = len(r.Players)

	joinMsg := Message{
		Type: "player_joined",
		Data: map[string]interface{}{
			"roomId":      r.ID,
			"playerId":    client.ID,
			"playerCount": r.State.PlayerCount,
			"userId":      client.userID,
		},
	}
	r.Broadcast(joinMsg.ToJSON())
	return nil
}

func (r *GameRoom) RemovePlayer(clientID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.Players[clientID]; !exists {
		return false
	}

	delete(r.Players, clientID)
	r.State.PlayerCount = len(r.Players)

	return true
}

func (r *GameRoom) Broadcast(message []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, client := range r.Players {
		select {
		case client.send <- message:
		default:
			// Client's send channel is full, skip
		}
	}
}

func (r *GameRoom) GetPlayers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players := make([]string, 0, len(r.Players))
	for id := range r.Players {
		players = append(players, id)
	}

	return players
}
