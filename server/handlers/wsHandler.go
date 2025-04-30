package handlers

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var connectedPlayers []Player
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func WsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading:", err)
		return
	}

	go handleConnection(conn)
}

func handleConnection(conn *websocket.Conn) {
	defer conn.Close()

	var gameState *Game
	player := Player{
		ID:       int64(len(connectedPlayers) + 1),
		UserName: "Player" + fmt.Sprint(len(connectedPlayers)+1),
		Conn:     conn,
	}
	connectedPlayers = append(connectedPlayers, player)
	defer func() {
		for i, p := range connectedPlayers {
			if p.Conn == conn {
				connectedPlayers = append(connectedPlayers[:i], connectedPlayers[i+1:]...)
				break
			}
		}
	}()

	for {
		var msg struct {
			Type    string `json:"type"`
			Payload any    `json:"payload"`
		}

		if err := conn.ReadJSON(&msg); err != nil {
			fmt.Println("Error reading JSON:", err)
			break
		}

		switch msg.Type {
		case "selectCell":
			// Handle cell selection and validate the answer
			payload := msg.Payload.(map[string]interface{})
			x := int(payload["x"].(float64))
			y := int(payload["y"].(float64))
			answer := payload["answer"].(string)

			if gameState.ValidateAnswer(x, y, answer) {
				broadcastGameState(gameState)
			} else {
				conn.WriteJSON(map[string]string{
					"type":    "error",
					"message": "Invalid answer",
				})
			}
		case "updateState":
			// Broadcast updated game state to all players
		default:
			conn.WriteJSON(map[string]interface{}{
				"type":    "error",
				"message": "Unknown message type",
			})
		}
	}
}

func broadcastGameState(gameState *Game) {
	for _, player := range connectedPlayers {
		err := player.Conn.WriteJSON(map[string]interface{}{
			"type":    "gameState",
			"payload": gameState,
		})
		if err != nil {
			fmt.Printf("Error broadcasting to player %d: %v\n", player.ID, err)
		}
	}
}
