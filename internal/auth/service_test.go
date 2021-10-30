package auth

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_NewAuthService(t *testing.T) {
	authService := NewAuthService(time.Hour, nil)
	require.NotNil(t, authService)
	require.NotNil(t, authService.sessions)
}

func TestAuthService_IsLogged(t *testing.T) {
	authService := NewAuthService(time.Hour, nil)
	require.NotNil(t, authService)

	assert.False(t, authService.IsLogged("invalid token"))
	assert.False(t, authService.IsLogged("invalid token")) // idempotent

	token, err := authService.Login(time.Now())
	require.NoError(t, err)
	require.NotEmpty(t, token)
	assert.True(t, authService.IsLogged(token))
	assert.True(t, authService.IsLogged(token)) // idempotent
}

func TestAuthService_MultiLogin_MultiAccess_Then_Logout(t *testing.T) {
	authService := NewAuthService(time.Hour, nil)
	require.NotNil(t, authService)

	loginsCount := 10

	var wg sync.WaitGroup
	wg.Add(loginsCount)

	newTokensChan := make(chan string)
	addedTokens := map[string]struct{}{}
	for i := 0; i < loginsCount; i++ {
		// simluate many logins comming at once
		go func() {
			newToken, err := authService.Login(time.Now())
			require.NoError(t, err)
			newTokensChan <- newToken
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(newTokensChan)
	}()

	for t := range newTokensChan {
		addedTokens[t] = struct{}{}
	}

	// assert we have created all different logins/tokens
	assert.Len(t, addedTokens, loginsCount)

	wg.Add(loginsCount)
	for token := range addedTokens {
		// simluate many logouts requested at once
		go func(token string) {
			assert.True(t, authService.Logout(token))
			wg.Done()
		}(token)
	}
	wg.Wait()

	assert.Empty(t, authService.sessions) // all sessions logged out
}

func TestAuthService_Login_Logout(t *testing.T) {
	authService := NewAuthService(time.Hour, nil)
	require.NotNil(t, authService)

	token1, err := authService.Login(time.Now())
	require.NoError(t, err)
	require.NotEmpty(t, token1)
	assert.True(t, authService.IsLogged(token1))
	token2, err := authService.Login(time.Now())
	require.NoError(t, err)
	require.NotEmpty(t, token2)
	assert.True(t, authService.IsLogged(token2))

	assert.NotEqual(t, token1, token2)

	assert.False(t, authService.Logout("invalid token"))
	assert.True(t, authService.Logout(token1))
	assert.True(t, authService.Logout(token2))
}

func TestAuthService_ScanAndClean(t *testing.T) {
	ttl := time.Hour
	now := time.Now()

	authService := NewAuthService(ttl, nil)
	require.NotNil(t, authService)

	oldToken, err := authService.Login(now.Add(-2 * ttl))
	require.NoError(t, err)
	goodToken, err := authService.Login(now)
	require.NoError(t, err)

	assert.Len(t, authService.sessions, 2)
	authService.ScanAndClean()
	assert.Len(t, authService.sessions, 1)

	assert.False(t, authService.IsLogged(oldToken))
	assert.True(t, authService.IsLogged(goodToken))
}
