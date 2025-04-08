package sessions

import (
	"encoding/json"
	"time"

	"github.com/go-redis/redis"
)

// RedisStore is the equivalent to session.Store backed by Redis
type RedisStore struct {
	//Redis client used to talk to redis server
	Client *redis.Client
	//Used for key expiry time on Redis
	SessionDuration time.Duration
}

// Constructor for RedisStore
func NewRedisStore(client *redis.Client, sessionDuration time.Duration) *RedisStore {
	return &RedisStore{
		Client:          client,
		SessionDuration: sessionDuration,
	}
}

func (sid SessionID) getRedisKey() string {
	return "sid:" + sid.String()
}

// Store Implementation of Redis (Exactly the same as session.Store but refactored for Redis)
func (rs *RedisStore) Save(sid SessionID, sessionState interface{}) error {
	j, err := json.Marshal(sessionState)
	if err != nil {
		return err
	}
	rs.Client.Set(sid.getRedisKey(), j, 0)
	return nil
}

func (rs *RedisStore) Get(sid SessionID, sessionState interface{}) error {
	val, err := rs.Client.Get(sid.getRedisKey()).Bytes()
	if err != nil {
		return ErrStateNotFound
	}
	rs.Client.Expire(sid.getRedisKey(), rs.SessionDuration)
	return json.Unmarshal(val, sessionState)
}

func (rs *RedisStore) Delete(sid SessionID) error {
	val := rs.Client.Del(sid.getRedisKey())
	if val == nil {
		return ErrStateNotFound
	}
	return nil
}
