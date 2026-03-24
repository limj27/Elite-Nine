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

			// Send current room list to newly connected client
			msg := &Message{Type: "rooms_list", Data: map[string]interface{}{"rooms": h.ListRooms()}}
			client.send <- msg.ToJSON()

		case client := <-h.unregister:
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
		if room.RemovePlayer(client.ID) {
			leaveMsg := Message{
				Type: "player_left",
				Data: map[string]interface{}{
					"roomId":   roomID,
					"playerId": client.ID,
				},
			}
			room.Broadcast(leaveMsg.ToJSON())

			if len(room.Players) == 0 {
				delete(h.rooms, roomID)
				log.Printf("Room %s removed due to no players", roomID)
			}
		}
	}

	// Update room list for all clients when room membership changes
	h.BroadcastRoomList()
}

func (h *Hub) AddRoom(room *GameRoom) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.rooms[room.ID]; exists {
		log.Printf("Room %s already exists", room.ID)
	}
	h.rooms[room.ID] = room
	log.Printf("Room %s added", room.ID)
}

type RoomSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PlayerCount int    `json:"player_count"`
	MaxPlayers  int    `json:"max_players"`
	Status      string `json:"status"`
}

func (h *Hub) GetRoom(roomID string) (*GameRoom, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	room, exists := h.rooms[roomID]
	if !exists {
		log.Printf("GetRoom: room not found: %s, existing rooms: %v", roomID, h.ListRooms())
	} else {
		log.Printf("GetRoom: found room %s", roomID)
	}
	return room, exists
}

func (h *Hub) GetRoomByName(name string) (*GameRoom, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, room := range h.rooms {
		if room.Name == name {
			return room, true
		}
	}
	return nil, false
}

func (h *Hub) ListRooms() []RoomSummary {
	h.mu.RLock()
	defer h.mu.RUnlock()
	rooms := make([]RoomSummary, 0, len(h.rooms))
	for _, room := range h.rooms {
		rooms = append(rooms, RoomSummary{
			ID:          room.ID,
			Name:        room.Name,
			PlayerCount: room.State.PlayerCount,
			MaxPlayers:  room.State.MaxPlayers,
			Status:      room.State.Status,
		})
	}
	return rooms
}

func (h *Hub) BroadcastRoomList() {
	rooms := h.ListRooms()
	msg := Message{Type: "rooms_list", Data: map[string]interface{}{"rooms": rooms}}
	h.broadcast <- msg.ToJSON()
}

func (h *Hub) FindRoomByID(roomID string) (*GameRoom, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	room, ok := h.rooms[roomID]
	return room, ok
}

func (h *Hub) FindRoomByName(roomName string) (*GameRoom, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, room := range h.rooms {
		if room.Name == roomName {
			return room, true
		}
	}
	return nil, false
}

func (h *Hub) GetClient(clientID string) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if client.ID == clientID {
			return client, true
		}
	}
	return nil, false
}
