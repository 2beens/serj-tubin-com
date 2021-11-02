package auth

import (
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_NewAuthService(t *testing.T) {
	db, _ := redismock.NewClientMock()
	authService := NewAuthService(time.Hour, db)
	require.NotNil(t, authService)
	assert.NotNil(t, authService.redisClient)
	assert.Equal(t, time.Hour, authService.ttl)
}

func TestAuthService_IsLogged(t *testing.T) {
	db, mock := redismock.NewClientMock()

	authService := NewAuthService(time.Hour, db)
	require.NotNil(t, authService)

	mock.ExpectGet(sessionKeyPrefix + "invalid token").SetErr(redis.Nil)
	isLogged, err := authService.IsLogged("invalid token")
	require.Equal(t, "redis: nil", err.Error())
	assert.False(t, isLogged)

	mock.ExpectGet(sessionKeyPrefix + "invalid token").SetErr(redis.Nil)
	isLogged, err = authService.IsLogged("invalid token")
	require.Equal(t, "redis: nil", err.Error())
	assert.False(t, isLogged) // idempotent

	// token, err := authService.Login(time.Now())
	// require.NoError(t, err)
	// require.NotEmpty(t, token)

	// isLogged, err = authService.IsLogged(token)
	// require.NoError(t, err)
	// assert.True(t, isLogged)
	// isLogged, err = authService.IsLogged(token)
	// require.NoError(t, err)
	// assert.True(t, isLogged) // idempotent
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

// func TestAuthService_ScanAndClean(t *testing.T) {
// 	ttl := time.Hour
// 	now := time.Now()

// 	authService := NewAuthService(ttl, nil)
// 	require.NotNil(t, authService)

// 	oldToken, err := authService.Login(now.Add(-2 * ttl))
// 	require.NoError(t, err)
// 	goodToken, err := authService.Login(now)
// 	require.NoError(t, err)

// 	assert.Len(t, authService.sessions, 2)
// 	authService.ScanAndClean()
// 	assert.Len(t, authService.sessions, 1)

// 	assert.False(t, authService.IsLogged(oldToken))
// 	assert.True(t, authService.IsLogged(goodToken))
// }
