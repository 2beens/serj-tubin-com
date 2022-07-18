package auth

import (
	"context"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type LoginChecker struct {
	ttl         time.Duration
	redisClient *redis.Client
}

func NewLoginChecker(ttl time.Duration, redisClient *redis.Client) *LoginChecker {
	return &LoginChecker{
		ttl:         ttl,
		redisClient: redisClient,
	}
}

func (as *LoginChecker) IsLogged(token string) (bool, error) {
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
