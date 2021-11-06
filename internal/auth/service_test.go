package auth

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_NewAuthService(t *testing.T) {
	db, mock := redismock.NewClientMock()
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
	token, err := authService.Login(now)
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

// func TestAuthService_MultiLogin_MultiAccess_Then_Logout(t *testing.T) {
// 	authService := NewAuthService(time.Hour, nil)
// 	require.NotNil(t, authService)

// 	loginsCount := 10

// 	var wg sync.WaitGroup
// 	wg.Add(loginsCount)

// 	newTokensChan := make(chan string)
// 	addedTokens := map[string]struct{}{}
// 	for i := 0; i < loginsCount; i++ {
// 		// simluate many logins comming at once
// 		go func() {
// 			newToken, err := authService.Login(time.Now())
// 			require.NoError(t, err)
// 			newTokensChan <- newToken
// 			wg.Done()
// 		}()
// 	}

// 	go func() {
// 		wg.Wait()
// 		close(newTokensChan)
// 	}()

// 	for t := range newTokensChan {
// 		addedTokens[t] = struct{}{}
// 	}

// 	// assert we have created all different logins/tokens
// 	assert.Len(t, addedTokens, loginsCount)

// 	wg.Add(loginsCount)
// 	for token := range addedTokens {
// 		// simluate many logouts requested at once
// 		go func(token string) {
// 			loggedOut, err := authService.Logout(token)
// 			assert.NoError(t, err)
// 			assert.True(t, loggedOut)
// 			wg.Done()
// 		}(token)
// 	}
// 	wg.Wait()

// 	assert.Empty(t, authService.sessions) // all sessions logged out
// }

// func TestAuthService_Login_Logout(t *testing.T) {
// 	authService := NewAuthService(time.Hour, nil)
// 	require.NotNil(t, authService)

// 	token1, err := authService.Login(time.Now())
// 	require.NoError(t, err)
// 	require.NotEmpty(t, token1)
// 	assert.True(t, authService.IsLogged(token1))
// 	token2, err := authService.Login(time.Now())
// 	require.NoError(t, err)
// 	require.NotEmpty(t, token2)
// 	assert.True(t, authService.IsLogged(token2))

// 	assert.NotEqual(t, token1, token2)

// 	assert.False(t, authService.Logout("invalid token"))
// 	assert.True(t, authService.Logout(token1))
// 	assert.True(t, authService.Logout(token2))
// }
