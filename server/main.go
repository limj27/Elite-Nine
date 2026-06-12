package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"trivia-server/handlers"
	"trivia-server/sessions"
	"trivia-server/websocket"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func setupWebSocket(db *sql.DB) *websocket.Hub {
	hub := websocket.NewHub(db)
	go hub.Run()
	return hub
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env vars")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

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
	wsHub := setupWebSocket(db)

	// Create GameManager and pass into handler along with JWT service
	gm := websocket.NewGameManager()
	router.HandleFunc("/ws", websocket.Handler(wsHub, jwtService, gm))

	// SPA fallback — serve static files if they exist, otherwise serve index.html
	// This allows /lobby and /game to work as browser URLs without 404ing
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientDir := "../client"
		path := filepath.Join(clientDir, r.URL.Path)

		// If the requested path is a real file (JS, CSS, images etc.), serve it directly
		if _, err := os.Stat(path); err == nil {
			http.FileServer(http.Dir(clientDir)).ServeHTTP(w, r)
			return
		}

		// Otherwise fall back to index.html so the frontend JS can handle routing
		http.ServeFile(w, r, filepath.Join(clientDir, "index.html"))
	})

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}).Methods("GET")

	// Start server
	log.Printf("Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
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
	protected.HandleFunc("/profile/full", userHandler.GetFullProfile).Methods("GET")
	protected.HandleFunc("/profile/check-username", userHandler.CheckUsernameAvailable).Methods("POST")
	protected.HandleFunc("/profile/username", userHandler.UpdateUsername).Methods("PUT")
	protected.HandleFunc("/profile/password", userHandler.UpdatePassword).Methods("PUT")
	protected.HandleFunc("/profile/team", userHandler.UpdateFavoriteTeam).Methods("PUT")
	protected.HandleFunc("/profile/history", userHandler.GetGameHistory).Methods("GET")
	protected.HandleFunc("/profile", userHandler.DeleteAccount).Methods("DELETE")

}
