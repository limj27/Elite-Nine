package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"trivia-server/handlers"
	"trivia-server/sessions"
	"trivia-server/websocket"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

const (
	port = ":8080" // Port for the server to listen on
)

func setupWebSocket() *websocket.Hub {
	hub := websocket.NewHub()
	go hub.Run() // Start the WebSocket hub in a goroutine
	return hub
}

func main() {
	// Database connection
	db, err := sql.Open("mysql", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Redis connection
	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	// Services
	userService := sessions.NewUserService(db, redisClient)
	jwtService := sessions.NewJWTService(os.Getenv("JWT_SECRET"), redisClient)
	userHandler := handlers.NewUserHandler(userService, jwtService)

	// Router
	router := mux.NewRouter()
	SetupUserRoutes(router, userHandler, jwtService)

	// WebSocket Hub
	wsHub := setupWebSocket()
	http.HandleFunc("/ws", websocket.Handler(wsHub, jwtService))

	// Start server
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func SetupUserRoutes(router *mux.Router, userHandler *handlers.UserHandler, jwtService *sessions.JWTService) {
	// Public routes
	router.HandleFunc("/register", userHandler.Register).Methods("POST")
	router.HandleFunc("/login", userHandler.Login).Methods("POST")

	// Protected routes
	protected := router.PathPrefix("/api").Subrouter()
	protected.Use(sessions.AuthMiddleware(jwtService))
	protected.HandleFunc("/profile", userHandler.GetProfile).Methods("GET")
	protected.HandleFunc("/logout", userHandler.Logout).Methods("POST")
}
