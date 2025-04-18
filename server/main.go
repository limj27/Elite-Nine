package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"trivia-server/handlers"

	"github.com/go-redis/redis"
)

func main() {
	tlsKeyPath := getEnv("TLSKEY")
	tlsCertPath := getEnv("TLSCERT")
	sessionKey := getEnv("SESSIONKEY")
	redisaddr := getEnv("REDISADDR")
	dsn := getEnv("DSN")
	hour, _ := time.ParseDuration("1h")
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisaddr,
		Password: "",
		DB:       0,
	})

	http.HandleFunc("/ws", handlers.WsHandler)

	fmt.Println("Websocket server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server: ", err)
	}
}

func getEnv(name string) string {
	result := os.Getenv(name)
	if len(result) == 0 {
		envNotFound(name)
	}
	return result
}

func envNotFound(name string) {
	log.Fatalf("%s not set or not found", name)
}
