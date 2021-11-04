package auth

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/2beens/serjtubincom/pkg"
	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
)

const (
	sessionKeyPrefix = "serj-service-session||"
	tokensSetKey     = "serj-service-sessions"
)

type Admin struct {
	Username     string
	PasswordHash string
}

type LoginSession struct {
	Token     string
	CreatedAt time.Time
}

type Service struct {
	mutex       sync.Mutex    // TODO: now with redis maybe not needed
	redisClient *redis.Client // TODO: add one more cachine layer above redis
	ttl         time.Duration
	// ability to inject random string generator func for tokens (for unit and dev testing)
	RandStringFunc func(s int) (string, error)
}

func NewAuthService(
	ttl time.Duration,
	redisClient *redis.Client,
) *Service {
	return &Service{
		ttl:            ttl,
		redisClient:    redisClient,
		RandStringFunc: pkg.GenerateRandomString,
	}
}

func (as *Service) Login(createdAt time.Time) (string, error) {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	token, err := as.RandStringFunc(35)
	if err != nil {
		return "", err
	}

	sessionKey := sessionKeyPrefix + token
	cmdSet := as.redisClient.Set(context.Background(), sessionKey, createdAt.Unix(), 0)
	if err := cmdSet.Err(); err != nil {
		return "", err
	}

	// add token to list of sessions
	cmdSAdd := as.redisClient.SAdd(context.Background(), tokensSetKey, token)
	if err := cmdSAdd.Err(); err != nil {
		return "", err
	}

	return token, nil
}

func (as *Service) Logout(token string) (bool, error) {
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

	cmdSet := as.redisClient.Set(context.Background(), sessionKey, 0, 0)
	if err := cmdSet.Err(); err != nil {
		return false, err
	}

	// remove token from the list of sessions
	cmdSRem := as.redisClient.SRem(context.Background(), tokensSetKey, token)
	if err := cmdSRem.Err(); err != nil {
		return false, err
	}

	return createdAtUnix > 0, nil
}

func (as *Service) IsLogged(token string) (bool, error) {
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

// will run through all sessions, check the TTL, and clean them if old
func (as *Service) ScanAndClean() {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	cmd := as.redisClient.SMembers(context.Background(), tokensSetKey)
	if err := cmd.Err(); err != nil {
		log.Errorf("!!! auth service, scan and clean, get sessions: %s", err)
		return
	}

	sessionTokens := cmd.Val()
	if len(sessionTokens) == 0 {
		log.Warnln("=> auth service, scan and clean abort, no sessions")
		return
	}

	log.Warnf("=> auth service, scan and clean [%d sessions] start ...", len(sessionTokens))
	var toRemove []string
	for _, token := range sessionTokens {
		sessionKey := sessionKeyPrefix + token
		cmd := as.redisClient.Get(context.Background(), sessionKey)
		if err := cmd.Err(); err != nil {
			log.Errorf("=> auth service, scan and clean token %s: %s", token, err)
			continue
		}

		createdAtUnixStr := cmd.Val()
		createdAtUnix, err := strconv.ParseInt(createdAtUnixStr, 10, 64)
		if err != nil {
			log.Errorf("=> auth service, scan and clean token %s: %s", token, err)
			continue
		}

		createdAt := time.Unix(createdAtUnix, 0)
		sessionDuration := time.Since(createdAt)
		if sessionDuration > as.ttl {
			log.Warnf("=>\twill clean the session with token: %s", token)
			toRemove = append(toRemove, token)
		}
	}

	for _, token := range toRemove {
		sessionKey := sessionKeyPrefix + token
		cmdSet := as.redisClient.Del(context.Background(), sessionKey)
		if err := cmdSet.Err(); err != nil {
			log.Errorf("=> auth service, clean token %s: %s", token, err)
			continue
		}

		// remove token from the list of sessions
		cmdSRem := as.redisClient.SRem(context.Background(), tokensSetKey, token)
		if err := cmdSRem.Err(); err != nil {
			log.Errorf("=> auth service, clean token %s: %s", token, err)
			continue
		}
	}
}
