package websocket

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"trivia-server/sessions"
)

func Handler(hub *Hub, jwtService *sessions.JWTService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, username, err := getUserIDFromToken(r, jwtService) // Implement this function to extract user ID from token
		if userID == "" {
			log.Printf("WebSocket authentication failed: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade error:", err)
			return
		}

		client := NewClient(hub, conn, userID, username)
		client.hub.register <- client

		// allow connection of memory referenced by the caller by doing all work in goroutines
		go client.writePump()
		go client.readPump()
	}
}

func getUserIDFromToken(r *http.Request, jwtService *sessions.JWTService) (string, string, error) {
	// First try Authorization header (Bearer token)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenString := parts[1]
			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				return "", "", err
			}
			return strconv.Itoa(claims.UserID), claims.Username, nil
		}
	}

	// Fallback: Try token from query parameter (useful for WebSocket connections from browsers)
	tokenString := r.URL.Query().Get("token")
	if tokenString != "" {
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			return "", "", err
		}
		return strconv.Itoa(claims.UserID), claims.Username, nil
	}

	// For development/testing: allow user_id query parameter
	if userID := r.URL.Query().Get("user_id"); userID != "" {
		log.Printf("WARNING: Using test user_id parameter: %s", userID)
		return userID, "test_user", nil
	}

	return "", "", fmt.Errorf("no valid authentication found")
}
