package websocket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"trivia-server/game"
	"trivia-server/grid"
	"trivia-server/models"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type wsMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type createRoomPayload struct {
	RoomID     string `json:"room_id,omitempty"`
	RoomName   string `json:"room_name,omitempty"`
	Password   string `json:"password,omitempty"`
	MaxPlayers int    `json:"max_Players"`
	Difficulty string `json:"difficulty,omitempty"`
}

type joinRoomPayload struct {
	RoomID   string `json:"room_id,omitempty"`
	RoomName string `json:"room_name,omitempty"`
	Password string `json:"password,omitempty"`
}

type makeMovePayload struct {
	RoomID         string `json:"room_id"`
	Row            int    `json:"row"`
	Col            int    `json:"col"`
	Answer         string `json:"answer"`
	PlayerID       int    `json:"player_id"`
	PlayerName     string `json:"player_name"`
	PlayerHeadshot string `json:"player_headshot"`
}

type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte
	ID          string
	userID      string
	username    string
	GameManager *GameManager
	currentRoom string
}

func NewClient(hub *Hub, conn *websocket.Conn, userID string, username string, gm *GameManager) *Client {
	return &Client{
		hub:         hub,
		conn:        conn,
		send:        make(chan []byte, 256),
		ID:          uuid.New().String(),
		userID:      userID,
		username:    username,
		GameManager: gm,
	}
}

// readPump reads messages from the WebSocket connection and handles them.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		c.hub.removeClientFromRooms(c)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("ws read error: %v", err)
			}
			break
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		var msg wsMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Println("invalid message format:", err)
			continue
		}
		c.handleMessage(msg)
	}
}

// writePump writes messages to the WebSocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("Error getting next writer for client %s: %v", c.ID, err)
				return
			}
			_, _ = w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error writing ping message to client %s: %v", c.ID, err)
				return
			}
		}
	}
}

// handleMessage processes incoming messages from the client.
func (c *Client) handleMessage(msg wsMessage) {
	switch msg.Type {
	case "create_room":
		var p createRoomPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			c.sendError("invalid create_room payload")
			return
		}
		c.handleCreateRoom(p)
	case "join_room":
		var p joinRoomPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			c.sendError("invalid join_room payload")
			return
		}
		p.RoomID = strings.TrimSpace(p.RoomID)
		p.RoomName = strings.TrimSpace(p.RoomName)
		if p.RoomID == "" && p.RoomName == "" {
			c.sendError("room_id or room_name is required")
			return
		}
		c.handleJoinRoom(p)
	case "start_game":
		c.handleStartGame()
	case "make_move":
		var p makeMovePayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			c.sendError("invalid make_move payload")
			return
		}
		c.handleMakeMove(p)
	case "list_rooms":
		rooms := c.hub.ListRooms()
		log.Printf("list_rooms request %d rooms", len(rooms))
		c.sendJSON(map[string]interface{}{"type": "rooms_list", "payload": map[string]interface{}{"rooms": rooms}})
	case "leave_room":
		c.handleLeaveRoom()
	case "player_ready":
		var p struct {
			Ready bool `json:"ready"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			c.sendError("invalid player_ready payload")
			return
		}
		c.handlePlayerReady(p.Ready)
	case "rematch":
		c.handleRematch()
	default:
		c.sendError("unknown message type")
	}
}

func (c *Client) sendJSON(data interface{}) {
	bytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("sendJSON marshal error: %v", err)
		return
	}
	select {
	case c.send <- bytes:
	default:
		log.Printf("sendJSON DROPPED - send buffer full for client %s", c.ID)
	}
}

func (c *Client) sendError(msg string) {
	c.sendJSON(map[string]interface{}{
		"type":    "error",
		"message": msg,
	})
}

// handleCreateRoom handles the creation of a new game room.
func (c *Client) handleCreateRoom(p createRoomPayload) {
	if c.currentRoom != "" {
		if existingRoom, exists := c.hub.GetRoom(c.currentRoom); exists {
			existingRoom.RemovePlayer(c.ID)
			existingRoom.Broadcast(mustMarshal(map[string]interface{}{
				"type": "player_left",
				"payload": map[string]interface{}{
					"playerId": c.ID,
					"roomId":   c.currentRoom,
				},
			}))
		}
		c.currentRoom = ""
	}

	requestedRoomID := strings.TrimSpace(p.RoomID)
	if requestedRoomID == "" {
		requestedRoomID = uuid.New().String()
	}

	roomName := strings.TrimSpace(p.RoomName)
	roomPassword := strings.TrimSpace(p.Password)

	if roomName == "" {
		c.sendError("room_name is required")
		return
	}

	if _, exists := c.hub.GetRoom(requestedRoomID); exists {
		c.sendError("room ID already exists")
		return
	}

	if existingRoom, nameExists := c.hub.GetRoomByName(roomName); nameExists && existingRoom.ID != requestedRoomID {
		c.sendError("room name already exists")
		return
	}

	if p.MaxPlayers <= 0 {
		p.MaxPlayers = 2
	}

	room := NewGameRoom(requestedRoomID, roomName, roomPassword, c.userID)
	room.State.MaxPlayers = p.MaxPlayers

	difficulty := strings.ToLower(strings.TrimSpace(p.Difficulty))
	switch difficulty {
	case "easy", "regular", "hard":
		room.Difficulty = difficulty
		room.State.Difficulty = difficulty
	default:
		room.Difficulty = "regular"
		room.State.Difficulty = "regular"
	}

	c.hub.AddRoom(room)
	if err := room.AddPlayer(c); err != nil {
		c.sendError(fmt.Sprintf("failed to join created room: %v", err))
		return
	}
	c.currentRoom = requestedRoomID
	c.sendJSON(map[string]interface{}{
		"type": "room_created",
		"payload": map[string]interface{}{
			"room_id":   requestedRoomID,
			"room_name": roomName,
		},
	})
	c.sendJSON(map[string]interface{}{
		"type": "joined_room",
		"payload": map[string]interface{}{
			"room_id":   requestedRoomID,
			"room_name": roomName,
		},
	})

	c.hub.BroadcastRoomList()
}

// handleJoinRoom handles a client joining an existing game room.
func (c *Client) handleJoinRoom(p joinRoomPayload) {
	roomID := strings.TrimSpace(p.RoomID)
	roomName := strings.TrimSpace(p.RoomName)
	password := strings.TrimSpace(p.Password)

	var room *GameRoom
	var exists bool

	if roomID != "" {
		log.Printf("handleJoinRoom called for client %s by roomId=%s", c.ID, roomID)
		room, exists = c.hub.GetRoom(roomID)
	}

	if !exists && roomName != "" {
		log.Printf("handleJoinRoom called for client %s by roomName=%s", c.ID, roomName)
		room, exists = c.hub.GetRoomByName(roomName)
	}

	if !exists {
		c.sendError("room not found")
		return
	}

	if room.Password != "" && room.Password != password {
		c.sendError("incorrect room password")
		return
	}

	if err := room.AddPlayer(c); err != nil {
		c.sendError(err.Error())
		return
	}

	c.currentRoom = room.ID
	c.sendJSON(map[string]interface{}{
		"type": "joined_room",
		"payload": map[string]interface{}{
			"room_id":   room.ID,
			"room_name": room.Name,
		},
	})

	// Send existing players to the newly joined client
	for _, existingClient := range room.GetOrderedClients() {
		if existingClient.ID == c.ID {
			continue
		}
		c.sendJSON(map[string]interface{}{
			"type": "player_joined",
			"payload": map[string]interface{}{
				"roomId":      room.ID,
				"playerId":    existingClient.ID,
				"playerCount": room.State.PlayerCount,
				"userId":      existingClient.userID,
				"username":    existingClient.username,
			},
		})
	}

	log.Printf("Client %s joined room %s", c.ID, room.ID)
	c.hub.BroadcastRoomList()

	// Send ready status of existing players to the newly joined client
	room.mu.RLock()
	for clientID, isReady := range room.readyPlayers {
		if clientID == c.ID {
			continue
		}
		existingClient, exists := room.Players[clientID]
		if !exists {
			continue
		}
		c.sendJSON(map[string]interface{}{
			"type": "player_ready",
			"payload": map[string]interface{}{
				"playerId": clientID,
				"username": existingClient.username,
				"ready":    isReady,
			},
		})
	}
	room.mu.RUnlock()
}

func (c *Client) handleStartGame() {
	if c.currentRoom == "" {
		c.sendError("not in a room")
		return
	}
	room, exists := c.hub.GetRoom(c.currentRoom)
	if !exists {
		c.sendError("room not found")
		return
	}

	// Safety check — verify all players are actually ready
	room.mu.RLock()
	playerCount := len(room.Players)
	readyCount := 0
	for _, isReady := range room.readyPlayers {
		if isReady {
			readyCount++
		}
	}
	room.mu.RUnlock()

	if readyCount < playerCount || playerCount < room.State.MaxPlayers {
		c.sendError("not all players are ready")
		return
	}

	players := make([]models.GamePlayer, 0, len(room.Players))
	for _, cl := range room.GetOrderedClients() {
		uid, _ := strconv.Atoi(cl.userID)
		players = append(players, models.GamePlayer{
			UserID:   uid,
			Username: cl.username,
		})
	}

	if len(players) < 2 {
		c.sendError("need 2 players to start")
		return
	}

	// Pick a grid based on room difficulty and players' favorite teams
	gridSvc := grid.NewService(c.hub.DB)

	var gridTemplate *grid.GridTemplate
	var err error

	if room.Difficulty == "hard" {
		gridTemplate, err = gridSvc.GenerateGrid("hard", nil, nil)
	} else {
		ordered := room.GetOrderedClients()
		var p1Fav, p2Fav *int

		if len(ordered) >= 1 {
			if uid, convErr := strconv.Atoi(ordered[0].userID); convErr == nil {
				p1Fav, _ = grid.GetFavoriteTeamCriteriaID(c.hub.DB, uid)
			}
		}
		if len(ordered) >= 2 {
			if uid, convErr := strconv.Atoi(ordered[1].userID); convErr == nil {
				p2Fav, _ = grid.GetFavoriteTeamCriteriaID(c.hub.DB, uid)
			}
		}

		gridTemplate, err = gridSvc.GenerateGrid(room.Difficulty, p1Fav, p2Fav)
	}

	if err != nil {
		log.Printf("Failed to get grid template: %v", err)
		c.sendError("failed to load grid — make sure the database is populated")
		return
	}

	room.GridTemplateID = gridTemplate.ID

	gameModel := models.Game{
		Status:      models.GameStatusActive,
		CurrentTurn: 0,
	}

	gs := game.NewGameState(gameModel, players)

	if c.GameManager != nil {
		c.GameManager.Create(gs)
	}

	room.StartGame(gs, 0, c.GameManager)

	// Tell each player their index and the grid template
	for i, cl := range room.GetOrderedClients() {
		cl.sendJSON(map[string]interface{}{
			"type": "game_started",
			"payload": map[string]interface{}{
				"playerIndex":    i,
				"roomId":         room.ID,
				"rowCriteria":    gridTemplate.RowCriteria,
				"colCriteria":    gridTemplate.ColCriteria,
				"difficulty":     gridTemplate.Difficulty,
				"roomDifficulty": room.Difficulty,
			},
		})
	}

	// Notify both players grid is being generated
	room.Broadcast(mustMarshal(map[string]interface{}{
		"type":    "grid_generating",
		"payload": map[string]interface{}{"roomId": room.ID},
	}))

	// Broadcast initial game state
	room.Broadcast(mustMarshal(map[string]interface{}{
		"type":    "game_state",
		"payload": room.GameModel,
	}))
}

func (c *Client) handleMakeMove(p makeMovePayload) {
	room, exists := c.hub.GetRoom(p.RoomID)
	if !exists {
		c.sendError("room not found")
		return
	}
	if room.GameModel == nil {
		c.sendError("game not started")
		return
	}

	uid, err := strconv.Atoi(c.userID)
	if err != nil {
		c.sendError("invalid user id")
		return
	}

	// Validate the answer against the grid template
	gridSvc := grid.NewService(c.hub.DB)
	result, err := gridSvc.ValidateAnswer(room.GridTemplateID, p.Row, p.Col, p.PlayerID, p.Answer)
	if err != nil {
		log.Printf("Validation error: %v", err)
		c.sendError("validation error")
		return
	}

	// Always make the move — turn advances regardless of answer validity
	move, newTurn, err := game.MakeMove(room.GameModel, uid, p.Row, p.Col, p.Answer)
	if err != nil {
		c.sendError(err.Error())
		return
	}

	// Record attempt in cell history regardless of validity
	playerName := p.Answer
	if result.Valid {
		playerName = result.Answer.PlayerName
	}
	attempt := models.CellAttempt{
		UserID:     uid,
		Username:   c.username,
		PlayerName: playerName,
		Valid:      result.Valid,
	}
	room.GameModel.CellHistory[p.Row][p.Col] = append(
		room.GameModel.CellHistory[p.Row][p.Col],
		attempt,
	)

	if result.Valid {
		move.IsValid = true
		move.PlayerName = result.Answer.PlayerName
		move.Headshot = result.Answer.HeadshotURL

		existingMove := room.GameModel.Grid[p.Row][p.Col]

		if existingMove == nil {
			// Empty cell — place it and check for win
			room.GameModel.Grid[p.Row][p.Col] = move

			if game.CheckWin(room.GameModel, uid) {
				room.GameModel.Game.Status = models.GameStatusCompleted
				room.GameModel.Game.WinnerID = &uid
			}

		} else {
			// Cell occupied — check rarity to determine overtake
			existingResult, err := gridSvc.ValidateAnswer(
				room.GridTemplateID, p.Row, p.Col,
				*existingMove.PlayerID, existingMove.PlayerAnswer,
			)

			canOvertake := false
			if err != nil || !existingResult.Valid {
				canOvertake = true
			} else {
				// Lower rarity score = rarer = can overtake higher score
				canOvertake = result.RarityScore < existingResult.RarityScore
			}

			if canOvertake {
				room.GameModel.Grid[p.Row][p.Col] = move

				if game.CheckWin(room.GameModel, uid) {
					room.GameModel.Game.Status = models.GameStatusCompleted
					room.GameModel.Game.WinnerID = &uid
				}

				room.Broadcast(mustMarshal(map[string]interface{}{
					"type": "cell_overtaken",
					"payload": map[string]interface{}{
						"row":         p.Row,
						"col":         p.Col,
						"newPlayer":   result.Answer.PlayerName,
						"oldPlayer":   existingMove.PlayerName,
						"rarityScore": result.RarityScore,
					},
				}))
			} else {
				// Valid answer but not rare enough — turn still lost
				c.sendJSON(map[string]interface{}{
					"type": "overtake_failed",
					"payload": map[string]interface{}{
						"message":        "Your answer isn't rarer than the existing one",
						"yourRarity":     result.RarityScore,
						"existingRarity": existingResult.RarityScore,
					},
				})
			}
		}
	} else {
		// Invalid answer — notify player, turn already advanced
		log.Printf("Invalid move by user %d ('%s'), turn lost", uid, p.Answer)
		c.sendJSON(map[string]interface{}{
			"type": "invalid_move",
			"payload": map[string]interface{}{
				"message": result.Message,
				"answer":  p.Answer,
			},
		})
	}

	log.Printf("Move by user %d, valid=%v, new turn: %d", uid, result.Valid, newTurn)

	// Broadcast updated game state to both players regardless of outcome
	room.Broadcast(mustMarshal(map[string]interface{}{
		"type":    "game_state",
		"payload": room.GameModel,
	}))
}

// handleLeaveRoom handles a client leaving a game room.
func (c *Client) handleLeaveRoom() {
	if c.currentRoom == "" {
		return
	}
	room, exists := c.hub.GetRoom(c.currentRoom)
	if exists {
		room.RemovePlayer(c.ID)
		room.Broadcast(mustMarshal(map[string]interface{}{
			"type":    "player_left",
			"payload": map[string]interface{}{"playerId": c.ID},
		}))

		// Delete room from hub if empty
		room.mu.RLock()
		playerCount := len(room.Players)
		room.mu.RUnlock()

		if playerCount == 0 {
			c.hub.mu.Lock()
			delete(c.hub.rooms, c.currentRoom)
			c.hub.mu.Unlock()
			log.Printf("Room %s deleted after last player left", c.currentRoom)
		}
	}
	c.currentRoom = ""
	c.hub.BroadcastRoomList()
}

func (c *Client) handlePlayerReady(ready bool) {
	if c.currentRoom == "" {
		c.sendError("not in a room")
		return
	}

	room, exists := c.hub.GetRoom(c.currentRoom)
	if !exists {
		c.sendError("room not found")
		return
	}

	room.Broadcast(mustMarshal(map[string]interface{}{
		"type": "player_ready",
		"payload": map[string]interface{}{
			"playerId": c.ID,
			"username": c.username,
			"ready":    ready,
		},
	}))

	allReady := room.SetReady(c.ID, ready)
	log.Printf("handlePlayerReady: client %s ready=%v allReady=%v", c.ID, ready, allReady)
	if allReady {
		room.Broadcast(mustMarshal(map[string]interface{}{
			"type":    "room_ready",
			"payload": map[string]interface{}{"roomId": room.ID},
		}))
	}
}

func (c *Client) handleRematch() {
	if c.currentRoom == "" {
		c.sendError("not in a room")
		return
	}

	room, exists := c.hub.GetRoom(c.currentRoom)
	if !exists {
		c.sendError("room not found")
		return
	}

	// Reset ready states and game model
	room.mu.Lock()
	for k := range room.readyPlayers {
		room.readyPlayers[k] = false
	}
	room.State.Status = "waiting"
	room.GameModel = nil
	room.GameStatus = ""
	room.mu.Unlock()

	// Notify all players to go back to ready screen
	room.Broadcast(mustMarshal(map[string]interface{}{
		"type":    "rematch",
		"payload": map[string]interface{}{"roomId": room.ID},
	}))

	log.Printf("Rematch requested in room %s", room.ID)
}

func (c *Client) Close() {
	close(c.send)
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
