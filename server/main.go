package main

import (
	"fmt"
	"log"
	"net/http"
	"trivia-server/handlers"

	"github.com/gorilla/mux"
)

const (
	port = ":8080" // Port for the server to listen on
)

func main() {
	r := mux.NewRouter()

	//WebSocket Endpoint for handling multiplayer game connections
	r.HandleFunc("/ws", handlers.WsHandler)

	fmt.Println("Server running on port %s ...", port)
	log.Fatal(http.ListenAndServe(port, r))
}
