package main

import (
	"fmt"
	"log"
	"net/http"
	"trivia-server/db"
	"trivia-server/handlers"

	"github.com/gorilla/mux"
)

const (
	port = ":8080" // Port for the server to listen on
)

func main() {
	r := mux.NewRouter()
	database, err := db.NewConnection()
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	//WebSocket Endpoint for handling multiplayer game connections
	// r.HandleFunc("/ws", handlers.WsHandler

	// Handling team routes
	teamRepo := db.NewTeamRepository(database)
	teamHandler := handlers.NewTeamHandler(teamRepo)
	teamHandler.RegisterTeamRoutes(r)

	fmt.Printf("Server running on port %s ...", port)
	log.Fatal(http.ListenAndServe(port, r))
}
