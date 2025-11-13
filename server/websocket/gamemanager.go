package websocket

import (
	"sync"
	"trivia-server/models"
)

type GameManager struct {
	mu     sync.Mutex
	games  map[int]*models.GameState
	nextID int
}

func NewGameManager() *GameManager {
	return &GameManager{
		games:  make(map[int]*models.GameState),
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

func (gm *GameManager) Get(id int) (*models.GameState, bool) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	state, exists := gm.games[id]
	return state, exists
}

func (gm *GameManager) Delete(id int) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	delete(gm.games, id)
}
