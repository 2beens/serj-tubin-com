package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

var (
	testUsername     = "testuser"
	testPassword     = "testpass"
	testPasswordHash = "$2a$14$6Gmhg85si2etd3K9oB8nYu1cxfbrdmhkg6wI6OXsa88IF4L2r/L9i" // testpass
	testAdmin        = &Admin{
		Username:     testUsername,
		PasswordHash: testPasswordHash,
	}
	testCredentials = Credentials{
		Username: testUsername,
		Password: testPassword,
	}
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// INFO: https://github.com/go-redis/redis/issues/1029
		goleak.IgnoreTopFunction(
			"github.com/go-redis/redis/v8/internal/pool.(*ConnPool).reaper",
		),
	)
}

func TestAuthService_NewAuthService(t *testing.T) {
	db, mock := redismock.NewClientMock()
	defer db.Close()

	authService := NewAuthService(testAdmin, time.Hour, db)
	require.NotNil(t, authService)
	assert.NotNil(t, authService.redisClient)
	assert.Equal(t, time.Hour, authService.ttl)

	testToken := "test_token"
	randStringFunc := func(s int) (string, error) {
		return testToken, nil
	}
	authService.RandStringFunc = randStringFunc

	now := time.Now()
	sessionKey := sessionKeyPrefix + testToken
	mock.ExpectSet(sessionKey, now.Unix(), 0).SetVal(fmt.Sprintf("%d", now.Unix()))
	mock.ExpectSAdd(tokensSetKey, testToken).SetVal(1)
	token, err := authService.Login(context.Background(), testCredentials, now)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// test failed login again
	token, err = authService.Login(context.Background(), Credentials{
		Username: testUsername,
		Password: "invalid_pass",
	}, now)
	assert.ErrorIs(t, err, ErrWrongPassword)
	assert.Empty(t, token)
}

func TestAuthService_ScanAndClean(t *testing.T) {
	ttl := time.Hour
	now := time.Now()
	then := now.Add(-2 * time.Hour)

	rdb, mock := redismock.NewClientMock()

	authService := NewAuthService(testAdmin, ttl, rdb)
	require.NotNil(t, authService)

	// expected calls
	t1, t2 := "token1", "token2"
	mock.ExpectSMembers(tokensSetKey).SetVal([]string{t1, t2})
	mock.ExpectGet(t1).SetVal(fmt.Sprintf("%d", then.Unix()))
	mock.ExpectGet(t2).SetVal(fmt.Sprintf("%d", now.Unix()))
	// expect deleted only t2, old life
	mock.ExpectDel(t2)
	mock.ExpectSRem(tokensSetKey, t2)
}
