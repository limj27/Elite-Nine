package sessions

import "github.com/go-redis/redis"

type JWTService struct {
	secretKey string
	redis     *redis.Client
}

func NewJWTService(secretKey string, redisClient *redis.Client) *JWTService {
	return &JWTService{
		secretKey: secretKey,
		redis:     redisClient,
	}
}
