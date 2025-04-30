package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"trivia-server/handlers"
	"trivia-server/sessions"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

func main() {
	// Load environment variables
	// tlsKeyPath := getEnv("TLSKEY")
	// tlsCertPath := getEnv("TLSCERT")
	// sessionKey := getEnv("SESSIONKEY")
	redisaddr := getEnv("REDISADDR")
	hour, _ := time.ParseDuration("1h")

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisaddr,
		Password: "",
		DB:       0,
	})

	// Initialize RedisStore
	redisStore := sessions.NewRedisStore(redisClient, hour)

	// Initialize GameServer
	gameServer := handlers.NewGameServer(redisStore)

	// Set up router
	router := mux.NewRouter()
	gameServer.SetupRoutes(router)

	// Start the server
	fmt.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
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
