package handlers

import "github.com/gorilla/websocket"

type Player struct {
	ID       int64           `json:"id"`
	UserName string          `json:"username"`
	Conn     *websocket.Conn `json:"-"` // Exclude from JSON serialization
}

func (p *Player) NewPlayer(id int64, user *User) *Player {
	return &Player{
		ID:       id,
		UserName: user.Username,
	}
}

type Game struct {
	ID      int64        `json:"id"`
	Players []Player     `json:"players"`
	Grid    [3][3]string `json:"grid"`
	Turn    int          `json:"turn"`
}

func (g *Game) NewGame(id int64, players []Player) *Game {
	return &Game{
		ID:      id,
		Players: players,
		Grid:    [3][3]string{},
		Turn:    0,
	}
}

func (g *Game) ValidateAnswer(x, y int, answer string) bool {
	// Validate the answer (e.g., compare with the database)
	// If correct, update the grid and return true
	g.Grid[x][y] = g.Players[g.Turn].UserName
	g.Turn = (g.Turn + 1) % len(g.Players)
	return true
}

func (g *Game) CheckWin() bool {
	return false // Placeholder for win condition check
}
