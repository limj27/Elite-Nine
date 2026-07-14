package websocket

import (
	"errors"
	"log"
	"strconv"
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

	GameModel      *models.GameState
	GameID         int
	GameManager    *GameManager
	GameStatus     string
	GridTemplateID int
	Difficulty     string // "easy" | "regular" | "hard"

	RematchRequests map[string]bool // playerID -> accepted
	rematchMu       sync.Mutex

	// Turn timer
	turnTimer   *time.Timer
	turnTimerMu sync.Mutex
}

type GameState struct {
	Status      string `json:"status"`
	PlayerCount int    `json:"player_count"`
	MaxPlayers  int    `json:"max_players"`
	Difficulty  string `json:"difficulty"`
}

type RematchRequest struct {
	PlayerID string
	Accepted bool
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
		Difficulty:   "regular",
		State: GameState{
			Status:      "waiting",
			PlayerCount: 0,
			MaxPlayers:  2, // default
			Difficulty:  "regular",
		},
		CreatedAt:       time.Now(),
		RematchRequests: make(map[string]bool),
	}
}

// turnDurationForDifficulty returns the per-turn time limit for a
// given room difficulty. A duration of 0 means "no timer".
func turnDurationForDifficulty(difficulty string) time.Duration {
	switch difficulty {
	case "hard":
		return 30 * time.Second
	case "regular":
		return 60 * time.Second
	default: // "easy" — untimed
		return 0
	}
}

// StartTurnTimer (re)starts the per-turn countdown for this room based
// on its difficulty. Stops any existing timer first. onTimeout is
// invoked in its own goroutine if the timer elapses without a move
// being made — the caller is responsible for verifying the turn is
// still the same one the timer was started for (turnAtStart).
func (r *GameRoom) StartTurnTimer(onTimeout func(room *GameRoom, turnAtStart int)) {
	duration := turnDurationForDifficulty(r.Difficulty)

	r.turnTimerMu.Lock()
	if r.turnTimer != nil {
		r.turnTimer.Stop()
		r.turnTimer = nil
	}

	if duration <= 0 {
		r.turnTimerMu.Unlock()
		// No timer for this difficulty — tell clients to hide any UI
		r.Broadcast(mustMarshal(map[string]interface{}{
			"type":    "turn_timer",
			"payload": map[string]interface{}{"duration": 0},
		}))
		return
	}

	r.mu.RLock()
	turnAtStart := 0
	active := r.GameModel != nil && r.GameModel.Game.Status == models.GameStatusActive
	if r.GameModel != nil {
		turnAtStart = r.GameModel.Game.CurrentTurn
	}
	r.mu.RUnlock()

	if !active {
		r.turnTimerMu.Unlock()
		return
	}

	deadline := time.Now().Add(duration)
	r.turnTimer = time.AfterFunc(duration, func() {
		onTimeout(r, turnAtStart)
	})
	r.turnTimerMu.Unlock()

	r.Broadcast(mustMarshal(map[string]interface{}{
		"type": "turn_timer",
		"payload": map[string]interface{}{
			"deadline": deadline.UnixMilli(),
			"duration": int(duration.Seconds()),
		},
	}))
}

// StopTurnTimer cancels any active turn timer for this room.
func (r *GameRoom) StopTurnTimer() {
	r.turnTimerMu.Lock()
	defer r.turnTimerMu.Unlock()
	if r.turnTimer != nil {
		r.turnTimer.Stop()
		r.turnTimer = nil
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
		r.StopTurnTimer()
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

func (r *GameRoom) EndGame(winnerID int) {
	r.StopTurnTimer()

	r.mu.Lock()
	r.GameStatus = "completed"
	r.State.Status = "completed"

	var winnerUsername string
	var isDraw bool

	if winnerID == 0 {
		isDraw = true
		winnerUsername = "Draw"
	} else {
		// Get winner info
		for _, client := range r.Players {
			uid, _ := strconv.Atoi(client.userID)
			if uid == winnerID {
				winnerUsername = client.username
				break
			}
		}
	}
	r.mu.Unlock()

	// Broadcast game ended
	payload := map[string]interface{}{
		"room_id":     r.ID,
		"final_state": r.GameModel,
		"is_draw":     isDraw,
	}

	if !isDraw {
		payload["winner_id"] = winnerID
		payload["winner_username"] = winnerUsername
	}

	r.Broadcast(mustMarshal(map[string]interface{}{
		"type":    "game_ended",
		"payload": payload,
	}))

	if isDraw {
		log.Printf("Game ended in draw in room %s", r.ID)
	} else {
		log.Printf("Game ended in room %s, winner: %s (ID: %d)", r.ID, winnerUsername, winnerID)
	}
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
	r.StopTurnTimer()

	r.mu.Lock()
	r.State.Status = "closed"

	closeMsg := Message{
		Type: "room_closed",
		Payload: map[string]interface{}{
			"roomId": r.ID,
		},
	}

	// Get client list before broadcasting
	clients := make([]*Client, 0, len(r.Players))
	for _, client := range r.Players {
		clients = append(clients, client)
	}
	r.mu.Unlock()

	// Broadcast without holding lock
	msgBytes := closeMsg.ToJSON()
	for _, client := range clients {
		select {
		case client.send <- msgBytes:
		default:
		}
	}

	// Note: Don't close clients here - let them disconnect naturally
	// Closing the client connection should be handled by the hub
	log.Printf("Room %s closed", r.ID)
}

func (r *GameRoom) RequestRematch(clientID string) (bool, error) {
	r.rematchMu.Lock()
	defer r.rematchMu.Unlock()

	r.mu.RLock()
	playerCount := len(r.Players)
	gameStatus := r.GameStatus
	r.mu.RUnlock()

	if gameStatus != "completed" && gameStatus != "finished" {
		return false, errors.New("game not finished")
	}

	// Mark this player as accepting rematch
	r.RematchRequests[clientID] = true

	// Check if all players have accepted
	if len(r.RematchRequests) == playerCount {
		return true, nil // All players ready for rematch
	}

	return false, nil // Waiting for other players
}

func (r *GameRoom) ResetForRematch() {
	r.StopTurnTimer()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.rematchMu.Lock()
	r.RematchRequests = make(map[string]bool)
	r.rematchMu.Unlock()

	r.GameModel = nil
	r.GameID = 0
	r.State.Status = "ready"
	r.GameStatus = "waiting"

	log.Printf("Room %s reset for rematch", r.ID)
}
