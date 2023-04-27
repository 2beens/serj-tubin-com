package auth

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	testingpkg "github.com/2beens/serjtubincom/pkg/testing"
	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
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

	authService := NewAuthService(time.Hour, db)
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
	token, err := authService.Login(context.Background(), now)
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestAuthService_ScanAndClean(t *testing.T) {
	ttl := time.Hour
	now := time.Now()
	then := now.Add(-2 * time.Hour)

	db, mock := redismock.NewClientMock()

	authService := NewAuthService(ttl, db)
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

// integration kinda test (uses real redis connection)
func TestAuthService_MultiLogin_MultiAccess_Then_Logout(t *testing.T) {
	os.Setenv("REDIS_PASS", "<remove>")
	rdb := testingpkg.GetRedisClientAndCtx(t)
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	authService := NewAuthService(time.Hour, rdb)
	require.NotNil(t, authService)
	loginChecker := NewLoginChecker(time.Hour, rdb)
	require.NotNil(t, loginChecker)

	loginsCount := 10

	var wg sync.WaitGroup
	wg.Add(loginsCount)

	newTokensChan := make(chan string)
	for i := 0; i < loginsCount; i++ {
		// simluate many logins comming at once
		go func() {
			newToken, err := authService.Login(ctx, time.Now())
			require.NoError(t, err)
			newTokensChan <- newToken
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(newTokensChan)
	}()

	addedTokens := map[string]struct{}{}
	for t := range newTokensChan {
		addedTokens[t] = struct{}{}
	}

	// assert we have created all different logins/tokens
	assert.Len(t, addedTokens, loginsCount)
	for token := range addedTokens {
		isLogged, err := loginChecker.IsLogged(ctx, token)
		require.NoError(t, err)
		assert.True(t, isLogged)
	}

	wg.Add(loginsCount)
	for token := range addedTokens {
		// simulate many logouts requested at once
		go func(token string) {
			loggedOut, err := authService.Logout(ctx, token)
			assert.NoError(t, err)
			assert.True(t, loggedOut)
			wg.Done()
		}(token)
	}
	wg.Wait()

	// assert all sessions logged out
	for token := range addedTokens {
		isLogged, err := loginChecker.IsLogged(ctx, token)
		require.NoError(t, err)
		assert.False(t, isLogged)
	}
}

func TestAuthService_Login_Logout(t *testing.T) {
	os.Setenv("REDIS_PASS", "<remove>")
	rdb := testingpkg.GetRedisClientAndCtx(t)
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now()

	authService := NewAuthService(time.Hour, rdb)
	require.NotNil(t, authService)
	loginChecker := NewLoginChecker(time.Hour, rdb)
	require.NotNil(t, loginChecker)

	token1, err := authService.Login(ctx, now)
	require.NoError(t, err)
	require.NotEmpty(t, token1)
	isLogged1, err := loginChecker.IsLogged(ctx, token1)
	require.NoError(t, err)
	assert.True(t, isLogged1)

	token2, err := authService.Login(ctx, now)
	require.NoError(t, err)
	require.NotEmpty(t, token2)
	isLogged2, err := loginChecker.IsLogged(ctx, token1)
	require.NoError(t, err)
	assert.True(t, isLogged2)

	assert.NotEqual(t, token1, token2)

	loggedOut, err := authService.Logout(ctx, "invalid token")
	require.ErrorIs(t, err, ErrLoginSessionNotFound)
	assert.False(t, loggedOut)
	loggedOut, err = authService.Logout(ctx, token1)
	require.NoError(t, err, fmt.Sprintf("received err [%T]: %+v", err, err))
	assert.True(t, loggedOut)
	loggedOut, err = authService.Logout(ctx, token2)
	require.NoError(t, err, fmt.Sprintf("received err [%T]: %+v", err, err))
	assert.True(t, loggedOut)
}
