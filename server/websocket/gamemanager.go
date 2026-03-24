package websocket

import (
	"sync"
	"trivia-server/models"
)

type GameManager struct {
	mu       sync.Mutex
	games    map[int]*models.GameState
	rooms    map[int]*GameRoom // Track active game rooms by game ID
	nextID   int
	roomsMux sync.RWMutex
}

func NewGameManager() *GameManager {
	return &GameManager{
		games:  make(map[int]*models.GameState),
		rooms:  make(map[int]*GameRoom),
		nextID: 1,
	}
}

func (gm *GameManager) Create(state *models.GameState) int {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	id := gm.nextID
	gm.games[id] = state
	gm.nextID++
	return id
}

func (gm *GameManager) AddGameRoom(gameID int, room *GameRoom) {
	gm.roomsMux.Lock()
	defer gm.roomsMux.Unlock()
	gm.rooms[gameID] = room
}

func (gm *GameManager) GetGameRoom(gameID int) (*GameRoom, bool) {
	gm.roomsMux.RLock()
	defer gm.roomsMux.RUnlock()
	room, ok := gm.rooms[gameID]
	return room, ok
}

func (gm *GameManager) RemoveGameRoom(gameID int) {
	gm.roomsMux.Lock()
	defer gm.roomsMux.Unlock()
	delete(gm.rooms, gameID)
}
