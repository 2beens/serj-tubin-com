package internal

import (
	"sync"
	"time"
)

type Admin struct {
	Username     string
	PasswordHash string
}

type LoginSession struct {
	Token     string
	CreatedAt time.Time
}

type AuthService struct {
	mutex    sync.Mutex
	ttl      time.Duration
	sessions map[string]*LoginSession
}

func NewAuthService(ttl time.Duration) *AuthService {
	return &AuthService{
		ttl:      ttl,
		sessions: make(map[string]*LoginSession),
	}
}

func (as *AuthService) Login(createdAt time.Time) (string, error) {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	token, err := GenerateRandomString(35)
	if err != nil {
		return "", err
	}

	// i don't care if token already exists
	as.sessions[token] = &LoginSession{
		Token:     token,
		CreatedAt: createdAt,
	}

	return token, nil
}

func (as *AuthService) Logout(token string) bool {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if s, ok := as.sessions[token]; !ok {
		return false
	} else {
		delete(as.sessions, s.Token)
	}

	return true
}

func (as *AuthService) IsLogged(token string) bool {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	for _, s := range as.sessions {
		if s.Token == token {
			return true
		}
	}
	return false
}

// will run through all sessions, check the TTL, and clean them if old
func (as *AuthService) ScanAndClean() {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	// TODO:
}
