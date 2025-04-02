package main

import (
	"fmt"
	"net/http"
	"trivia-server/handlers"
)

func main() {
	http.HandleFunc("/ws", handlers.WsHandler)

	fmt.Println("Websocket server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server: ", err)
	}
}
