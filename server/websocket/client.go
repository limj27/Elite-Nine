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
		//Modify this function to check for proper origin checking
		//For example, you can check against a list of allowed origins
		//For now, we allow all origins for simplicity
		return true // Allow all origins for simplicity; adjust as needed,
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
}

type joinRoomPayload struct {
	RoomID   string `json:"room_id,omitempty"`
	RoomName string `json:"room_name,omitempty"`
	Password string `json:"password,omitempty"`
}

type makeMovePayload struct {
	RoomID string `json:"room_id"`
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Answer string `json:"answer"`
}

type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte
	ID          string
	userID      string //From JWT Token
	username    string //From JWT Token
	GameManager *GameManager

	currentRoom string
}

func NewClient(hub *Hub, conn *websocket.Conn, userID string, username string, gm *GameManager) *Client {
	return &Client{
		hub:         hub,
		conn:        conn,
		send:        make(chan []byte, 256), // Buffered channel to prevent blocking
		ID:          uuid.New().String(),    // Use remote address as client ID
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

		//normalize
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
				// hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("Error getting next writer for client %s: %v", c.ID, err)
				return
			}
			_, _ = w.Write(message)

			// send queued messages
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

// handleMesasage processes incoming messages from the client.
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
	default:
		c.sendError("unknown message type")
	}
}

func (c *Client) sendJSON(data interface{}) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	select {
	case c.send <- bytes:
	default:
		// drop if send buffer full
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

	// Announce newly created room to all connected clients and refresh room lists
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
	c.sendJSON(map[string]interface{}{"type": "joined_room", "payload": map[string]interface{}{"room_id": room.ID, "room_name": room.Name}})
	c.hub.BroadcastRoomList()
	log.Printf("Client %s joined room %s", c.ID, room.ID)
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

	gameModel := models.Game{
		Status:      models.GameStatusActive,
		CurrentTurn: 0,
	}

	gs := game.NewGameState(gameModel, players)

	if c.GameManager != nil {
		c.GameManager.Create(gs)
	}

	room.StartGame(gs, 0, c.GameManager)

	// Broadcast initial game state for both clients explicitly
	room.Broadcast(mustMarshal(map[string]interface{}{"type": "game_state", "payload": room.GameModel}))
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

	move, _, err := game.MakeMove(room.GameModel, uid, p.Row, p.Col, p.Answer)
	if err != nil {
		c.sendError(err.Error())
		return
	}

	// persist move using gamemanager or DB as needed

	// broadcast updated game state to room
	room.Broadcast(mustMarshal(map[string]interface{}{"type": "game_state", "payload": room.GameModel}))

	// also broadcast the single move event if desired
	room.Broadcast(mustMarshal(map[string]interface{}{"type": "move_made", "payload": move}))
}

// handleLeaveRoom handles a client leaving a game room.
func (c *Client) handleLeaveRoom() {
	if c.currentRoom == "" {
		return
	}
	room, exists := c.hub.GetRoom(c.currentRoom)
	if exists {
		room.RemovePlayer(c.ID)
		room.Broadcast(mustMarshal(map[string]interface{}{"type": "player_left", "payload": map[string]interface{}{"playerId": c.ID}}))
	}
	c.currentRoom = ""
	c.hub.BroadcastRoomList()
}

func (c *Client) Close() {
	close(c.send)
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
