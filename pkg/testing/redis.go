package testing

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func GetRedisClientAndCtx(t *testing.T) (context.Context, *redis.Client) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	go func() {
		<-ctx.Done()
		cancel()
	}()

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	t.Logf("using redis host: [%s]", redisHost)

	redisPass := os.Getenv("REDIS_PASS")
	if redisPass == "" {
		redisPass = "todo"
	} else if redisPass == "<remove>" {
		redisPass = ""
	}
	t.Logf("using redis pass: [%s]", redisPass)

	rdb := redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(redisHost, "6379"),
		Password: redisPass,
		DB:       0, // use default DB
	})

	pingRes, err := rdb.Ping(ctx).Result()
	require.NoError(t, err)
	t.Logf("redis ping res: %s", pingRes)

	return ctx, rdb
}
