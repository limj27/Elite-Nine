package websocket

import (
	"log"
	"net/http"
)

func Handler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserIDFromToken(r) // Implement this function to extract user ID from token
		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade error:", err)
			return
		}

		client := NewClient(hub, conn, userID)
		client.hub.register <- client

		// allow connection of memory referenced by the caller by doing all work in goroutines
		go client.writePump()
		go client.readPump()
	}
}

func getUserIDFromToken(r *http.Request) string {
	token := r.Header.Get("Authorization")
	if token == "" {
		return r.URL.Query().Get("user_id")
	}

	//TODO: Implement prooper JWT Validation
	// For now, return a dummy user ID
	return "dummyUserID"
}
