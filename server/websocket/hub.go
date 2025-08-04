package websocket

import (
	"log"
	"sync"
)

type Hub struct {
	//Registerd clients
	clients map[*Client]bool
	//Inbound messages from the clients
	broadcast chan []byte
	//Register requests frrom the clients
	register chan *Client
	//Unregister requests from the clients
	unregister chan *Client

	//Game rooms
	rooms map[string]*GameRoom
	mu    sync.RWMutex
}

// Creates a new WebSocket hub instance
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]*GameRoom),
	}
}

// Run starts the hub and handles client registration, unregistration, and message broadcasting.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("Client registered: %s", client.ID)

			//Send welcome message
			welcomeMsg := Message{
				Type: "connected",
				Data: map[string]interface{}{
					"message":  "Connected to server",
					"clientId": client.ID,
				},
			}
			client.send <- welcomeMsg.ToJSON()

		case client <- h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Client unregistered: %s", client.ID)

				h.removeClientFromRooms(client)
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// removeClientFromRooms removes a client from all game rooms they are part of.
func (h *Hub) removeClientFromRooms(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for roomID, room := range h.rooms {
		if room.RemovePlayer(client.id) {
			leaveMsg := Message{
				Type: "player_left",
				Data: map[string]interface{}{
					"roomId":   roomID,
					"playerId": client.id,
				},
			}
			room.Broadcast(leaveMsg.ToJSON())

			if len(room.Players) == 0 {
				delete(h.rooms, roomID)
				log.Printf("Room %s removed due to no players", roomID)
			}
		}
	}
}

// CreateRoom creates a new game room
func (h *Hub) CreateRoom(roomID string, creatorID string) *GameRoom {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.rooms[roomID]; exists {
		log.Printf("Room %s already exists", roomID)
		return nil
	}

	room := NewGameRoom(roomID, creatorID)
	h.rooms[roomID] = room
	log.Printf("Room %s created by %s", roomID, creatorID)

	return room
}

// JoinRoom adds a player to a game room
func (h *Hub) JoinRoom(roomID string, client *Client) error {
	h.mu.RLock()
	room, exists := h.rooms[roomID]
	h.mu.RUnlock()

	if !exists {
		return ErrRoomNotFound
	}

	return room.AddPlayer(client)
}

func (h *Hub) GetRoom(roomID string) (*GameRoom, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	room, exists := h.rooms[roomID]
	return room, exists
}
