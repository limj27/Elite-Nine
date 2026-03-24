package websocket

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"trivia-server/sessions"
)

func Handler(hub *Hub, jwtService *sessions.JWTService, gm *GameManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := r.URL.Query().Get("token")
		if tokenStr == "" {
			// Fallback to checking Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenStr = parts[1]
				}
			}
		}

		if tokenStr == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		claims, err := jwtService.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		userID := strconv.Itoa(claims.UserID)
		username := claims.Username

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "userID", userID)
		ctx = context.WithValue(ctx, "username", username)

		client := NewClient(hub, conn, userID, username, gm)

		hub.register <- client

		go client.writePump()
		go client.readPump()
	}
}
