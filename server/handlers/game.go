package handlers

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

type Game struct {
	ID      string    `json:"id"`
	Board   [3][3]int `json:"board"`
	Players []string  `json:"players"`
	Turn    string    `jsons:"currentTurn"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity; adjust as needed
	},
}

func WsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading:", err)
		return
	}

	defer conn.Close()

	go handleConnection(conn)
}

func handleConnection(conn *websocket.Conn) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}
		fmt.Println("Received message:", string(message))
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			fmt.Println("Error writing message:", err)
			break
		}
	}
}
