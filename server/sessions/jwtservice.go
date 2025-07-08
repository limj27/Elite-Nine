package sessions

import (
	"fmt"
	"log"
	"strconv"
	"time"
	"trivia-server/models"

	"github.com/go-redis/redis"
	"github.com/golang-jwt/jwt/v4"
)

type JWTService struct {
	secretKey string
	redis     *redis.Client
}

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func NewJWTService(secretKey string, redisClient *redis.Client) *JWTService {
	return &JWTService{
		secretKey: secretKey,
		redis:     redisClient,
	}
}

func (js *JWTService) GenerateToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Token valid for 24 hours
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   strconv.Itoa(user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(js.secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	err = js.redis.Set(fmt.Sprintf("token:%d", user.ID), tokenString, 24*time.Hour).Err()
	if err != nil {
		log.Printf("Warning: Failed to store tokein in Redis: %v", err)
	}

	return tokenString, nil
}

func (js *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(js.secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (js *JWTService) RevokeToken(userID int) error {
	return js.redis.Del(fmt.Sprintf("token:%d", userID)).Err()
}
