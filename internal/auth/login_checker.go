package auth

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type LoginChecker struct {
	ttl         time.Duration
	mutex       sync.Mutex    // TODO: now with redis maybe not needed
	redisClient *redis.Client // TODO: add one more cachine layer above redis
}

func NewLoginChecker(ttl time.Duration, redisClient *redis.Client) *LoginChecker {
	return &LoginChecker{
		ttl:         ttl,
		redisClient: redisClient,
	}
}

func (as *LoginChecker) IsLogged(token string) (bool, error) {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	sessionKey := sessionKeyPrefix + token
	cmd := as.redisClient.Get(context.Background(), sessionKey)
	if err := cmd.Err(); err != nil {
		return false, err
	}

	createdAtUnixStr := cmd.Val()
	createdAtUnix, err := strconv.ParseInt(createdAtUnixStr, 10, 64)
	if err != nil {
		return false, err
	}

	createdAt := time.Unix(createdAtUnix, 0)
	sessionDuration := time.Since(createdAt)
	if sessionDuration > as.ttl {
		return false, nil
	}

	return true, nil
}
