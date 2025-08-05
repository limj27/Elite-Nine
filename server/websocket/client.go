package websocket

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
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

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	ID     string
	userID string
}

func NewClient(hub *Hub, conn *websocket.Conn, userID string) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256), // Buffered channel to prevent blocking
		ID:     uuid.New().String(),    // Use remote address as client ID
		userID: userID,
	}
}

// readPump reads messages from the WebSocket connection and handles them.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message from client %s: %v", c.ID, err)
			}
			break
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.handleMesasage(message)
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
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("Error getting next writer for client %s: %v", c.ID, err)
				return
			}
			w.Write(message)

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
func (c *Client) handleMesasage(data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Error unmarshalling message from client %s: %v", c.ID, err)
		return
	}

	switch msg.Type {
	case "create_room":
		c.handleCreateRoom(msg)
	case "join_room":
		c.handleJoinRoom(msg)
	case "leave_room":
		c.handleLeaveRoom(msg)
	case "ping":
		c.handlePing()
	default:
		log.Printf("Unknown message type from client: %s", msg.Type)
	}
}

// handleCreateRoom handles the creation of a new game room.
func (c *Client) handleCreateRoom(msg Message) {
	roomID, ok := msg.Data["roomId"].(string)
	if !ok {
		roomID = uuid.New().String()[:8] // Generate a new room ID if not provided
	}

	room := c.hub.CreateRoom(roomID, c.userID)
	if room == nil {
		response := Message{
			Type: "error",
			Data: map[string]interface{}{
				"message": "Room already exists",
			},
		}
		c.send <- response.ToJSON()
		return
	}

	if err := room.AddPlayer(c); err != nil {
		response := Message{
			Type: "error",
			Data: map[string]interface{}{
				"message": err.Error(),
			},
		}
		c.send <- response.ToJSON()
		return
	}

	response := Message{
		Type: "room_created",
		Data: map[string]interface{}{
			"roomId":  room.ID,
			"message": "Room created successfully",
		},
	}
	c.send <- response.ToJSON()
}

// handleJoinRoom handles a client joining an existing game room.
func (c *Client) handleJoinRoom(msg Message) {
	roomID, ok := msg.Data["roomId"].(string)
	if !ok {
		response := Message{
			Type: "error",
			Data: map[string]interface{}{
				"message": "Room ID is required",
			},
		}
		c.send <- response.ToJSON()
		return
	}

	if err := c.hub.JoinRoom(roomID, c); err != nil {
		response := Message{
			Type: "error",
			Data: map[string]interface{}{
				"message": err.Error(),
			},
		}
		c.send <- response.ToJSON()
		return
	}

	response := Message{
		Type: "joined_room",
		Data: map[string]interface{}{
			"roomId":  roomID,
			"message": "Joined room successfully",
		},
	}
	c.send <- response.ToJSON()
}

// handleLeaveRoom handles a client leaving a game room.
func (c *Client) handleLeaveRoom(msg Message) {
	roomID, ok := msg.Data["roomId"].(string)
	if !ok {
		return
	}

	room, exists := c.hub.GetRoom(roomID)
	if !exists {
		return
	}

	room.RemovePlayer(c.ID)
}

func (c *Client) handlePing() {
	response := Message{
		Type: "pong",
		Data: map[string]interface{}{
			"timestamp": time.Now().Unix(),
		},
	}
	c.send <- response.ToJSON()
}
