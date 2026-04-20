package websocket

import (
	"errors"
	"sync"
	"time"
	"trivia-server/models"
)

var (
	ErrRoomNotFound = errors.New("room not found")
	ErrRoomFull     = errors.New("room is full")
	ErrPlayerExists = errors.New("player already exists in the room")
	ErrRoomClosed   = errors.New("room is closed")
)

type GameRoom struct {
	ID          string
	Name        string
	Password    string
	CreatorID   string
	Players     map[string]*Client
	playerOrder []string
	State       GameState
	CreatedAt   time.Time
	mu          sync.RWMutex

	GameModel   *models.GameState
	GameID      int
	GameManager *GameManager
	GameStatus  string
}

type GameState struct {
	Status      string `json:"status"`
	PlayerCount int    `json:"player_count"`
	MaxPlayers  int    `json:"max_players"`
}

func NewGameRoom(id, name, password, creatorID string) *GameRoom {
	return &GameRoom{
		ID:          id,
		Name:        name,
		Password:    password,
		CreatorID:   creatorID,
		Players:     make(map[string]*Client),
		playerOrder: make([]string, 0),
		State: GameState{
			Status:      "waiting",
			PlayerCount: 0,
			MaxPlayers:  2, // default
		},
		CreatedAt: time.Now(),
	}
}

func (r *GameRoom) AddPlayer(client *Client) error {
	r.mu.Lock()

	if len(r.Players) >= r.State.MaxPlayers {
		r.mu.Unlock()
		return ErrRoomFull
	}

	if _, exists := r.Players[client.ID]; exists {
		r.mu.Unlock()
		return ErrPlayerExists
	}

	r.Players[client.ID] = client
	r.playerOrder = append(r.playerOrder, client.ID)
	r.State.PlayerCount = len(r.Players)
	isFull := r.State.PlayerCount == r.State.MaxPlayers
	if isFull {
		r.State.Status = "ready"
	}

	r.mu.Unlock() // release BEFORE broadcasting

	joinMsg := Message{
		Type: "player_joined",
		Payload: map[string]interface{}{
			"roomId":      r.ID,
			"playerId":    client.ID,
			"playerCount": r.State.PlayerCount,
			"userId":      client.userID,
		},
	}
	r.Broadcast(joinMsg.ToJSON())

	if isFull {
		readyMsg := Message{
			Type: "room_ready",
			Payload: map[string]interface{}{
				"roomId": r.ID,
			},
		}
		r.Broadcast(readyMsg.ToJSON())
	}

	return nil
}

func (r *GameRoom) StartGame(gameState *models.GameState, gameID int, gm *GameManager) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.GameModel = gameState
	r.GameID = gameID
	r.GameManager = gm
	r.State.Status = "active"
	r.GameStatus = "active" // Set game status

	startMsg := Message{
		Type: "game_started",
		Payload: map[string]interface{}{
			"roomId": r.ID,
			"gameId": r.GameID,
			"state":  r.GameModel,
		},
	}
	r.Broadcast(startMsg.ToJSON())

	r.GameManager.AddGameRoom(r.GameID, r) // Add game room to game manager
}

func (r *GameRoom) RemovePlayer(clientID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.Players[clientID]; !exists {
		return false
	}

	delete(r.Players, clientID)
	r.State.PlayerCount = len(r.Players)

	for i, id := range r.playerOrder {
		if id == clientID {
			r.playerOrder = append(r.playerOrder[:i], r.playerOrder[i+1:]...)
			break
		}
	}

	leaveMsg := Message{
		Type: "player_left",
		Payload: map[string]interface{}{
			"roomId":      r.ID,
			"playerId":    clientID,
			"playerCount": r.State.PlayerCount,
		},
	}
	r.Broadcast(leaveMsg.ToJSON())

	if r.State.PlayerCount == 0 {
		r.Close()
	}

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

func (r *GameRoom) GetOrderedClients() []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ordered := make([]*Client, 0, len(r.playerOrder))
	for _, id := range r.playerOrder {
		if client, ok := r.Players[id]; ok {
			ordered = append(ordered, client)
		}
	}
	return ordered
}

func (r *GameRoom) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.State.Status = "closed"

	closeMsg := Message{
		Type: "room_closed",
		Payload: map[string]interface{}{
			"roomId": r.ID,
		},
	}
	r.Broadcast(closeMsg.ToJSON())

	// Disconnect all players
	for _, client := range r.Players {
		client.Close()
	}
}
