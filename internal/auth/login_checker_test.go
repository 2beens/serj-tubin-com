package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_IsLogged(t *testing.T) {
	db, mock := redismock.NewClientMock()

	loginChecker := NewLoginChecker(time.Hour, db)
	require.NotNil(t, loginChecker)

	ctx := context.Background()

	mock.ExpectGet(sessionKeyPrefix + "invalid token").SetErr(redis.Nil)
	isLogged, err := loginChecker.IsLogged(ctx, "invalid token")
	require.Equal(t, "redis: nil", err.Error())
	assert.False(t, isLogged)

	mock.ExpectGet(sessionKeyPrefix + "invalid token").SetErr(redis.Nil)
	isLogged, err = loginChecker.IsLogged(ctx, "invalid token")
	require.Equal(t, "redis: nil", err.Error())
	assert.False(t, isLogged) // idempotent

	testToken := "test-token"
	now := time.Now()
	sessionKey := sessionKeyPrefix + testToken

	mock.ExpectGet(sessionKey).SetVal(fmt.Sprintf("%d", now.Unix()))
	isLogged, err = loginChecker.IsLogged(ctx, testToken)
	require.NoError(t, err)
	assert.True(t, isLogged)
	mock.ExpectGet(sessionKey).SetVal(fmt.Sprintf("%d", now.Unix()))
	isLogged, err = loginChecker.IsLogged(ctx, testToken)
	require.NoError(t, err)
	assert.True(t, isLogged) // idempotent
}
