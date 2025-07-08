package sessions

import (
	"database/sql"
	"fmt"
	"trivia-server/models"

	"github.com/go-redis/redis"
	"golang.org/x/crypto/bcrypt"
)

type UserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

type UserService struct {
	db    *sql.DB
	redis *redis.Client
}

func NewUserService(db *sql.DB, redis *redis.Client) *UserService {
	return &UserService{
		db:    db,
		redis: redis,
	}
}

func (us *UserService) CreateUser(username, email, password string) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	query := `INSERT INTO users (username, email, password_hash, created_at, updated_at, is_active)
			  VALUES (?, ?, ?, NOW(), NOW(), TRUE)
			  `

	result, err := us.db.Exec(query, username, email, string(hashedPassword))
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	return us.GetUserByID(int(userID))
}

func (us *UserService) GetUserByID(userID int) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at, games_played, games_won
		FROM users WHERE id = ? AND is_active = TRUE`

	err := us.db.QueryRow(query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.GamesPlayed,
		&user.GamesWon,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return user, nil
}

func (us *UserService) GetUserByUserName(username string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at, games_played, games_won
		FROM users WHERE username = ? AND is_active = TRUE`

	err := us.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.GamesPlayed,
		&user.GamesWon,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return user, nil
}

func (us *UserService) AuthenticateUser(username, password string) (*models.User, error) {
	user, err := us.GetUserByUserName(username)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid username or password: %w", err)
	}

	return user, nil
}

func (us *UserService) UpdateUserStats(userID int, won bool) error {
	var query string
	if won {
		query = `UPDATE users SET games_won = games_won + 1, games_played = games_played + 1, updated_at = NOW() WHERE id = ?`
	} else {
		query = `UPDATE users SET games_played = games_played + 1, updated_at = NOW() WHERE id = ?`
	}

	_, err := us.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to update user stats: %w", err)
	}
	return nil
}
