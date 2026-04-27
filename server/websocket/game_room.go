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

	readyPlayers map[string]bool

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
		ID:           id,
		Name:         name,
		Password:     password,
		CreatorID:    creatorID,
		Players:      make(map[string]*Client),
		playerOrder:  make([]string, 0),
		readyPlayers: make(map[string]bool),
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
			"username":    client.username,
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
	r.GameModel = gameState
	r.GameID = gameID
	r.GameManager = gm
	r.State.Status = "active"
	r.GameStatus = "active"
	r.mu.Unlock()

	// Remove the startMsg broadcast — handleStartGame sends game_started
	// with playerIndex to each client individually instead
	r.GameManager.AddGameRoom(r.GameID, r)
}

func (r *GameRoom) RemovePlayer(clientID string) bool {
	r.mu.Lock()

	if _, exists := r.Players[clientID]; !exists {
		r.mu.Unlock()
		return false
	}

	delete(r.Players, clientID)
	delete(r.readyPlayers, clientID)
	r.State.PlayerCount = len(r.Players)
	isEmpty := r.State.PlayerCount == 0

	// Reset status back to waiting if game hasn't started
	if r.State.Status == "ready" {
		r.State.Status = "waiting"
	}

	for i, id := range r.playerOrder {
		if id == clientID {
			r.playerOrder = append(r.playerOrder[:i], r.playerOrder[i+1:]...)
			break
		}
	}

	r.mu.Unlock() // release BEFORE broadcasting

	leaveMsg := Message{
		Type: "player_left",
		Payload: map[string]interface{}{
			"roomId":      r.ID,
			"playerId":    clientID,
			"playerCount": r.State.PlayerCount,
		},
	}
	r.Broadcast(leaveMsg.ToJSON())

	if isEmpty {
		r.Close()
	}

	return true
}

// SetReady marks a player as ready or not ready
// Returns true if ALL players in the room are now ready
func (r *GameRoom) SetReady(clientID string, ready bool) bool {
	r.mu.Lock()
	r.readyPlayers[clientID] = ready
	playerCount := len(r.Players)

	// Count how many are ready
	readyCount := 0
	for _, isReady := range r.readyPlayers {
		if isReady {
			readyCount++
		}
	}
	r.mu.Unlock()

	return readyCount == playerCount && playerCount == r.State.MaxPlayers
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
	r.State.Status = "closed"
	// collect clients before unlocking
	clients := make([]*Client, 0, len(r.Players))
	for _, client := range r.Players {
		clients = append(clients, client)
	}
	r.mu.Unlock() // release BEFORE broadcasting

	closeMsg := Message{
		Type: "room_closed",
		Payload: map[string]interface{}{
			"roomId": r.ID,
		},
	}
	r.Broadcast(closeMsg.ToJSON())

	for _, client := range clients {
		client.Close()
	}
}
