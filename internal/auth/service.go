package auth

import (
	"context"
	"strconv"
	"time"

	"github.com/2beens/serjtubincom/pkg"
	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
)

const (
	DefaultTTL       = 24 * 7 * time.Hour
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
	redisClient *redis.Client
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

func (as *Service) Login(ctx context.Context, createdAt time.Time) (string, error) {
	token, err := as.RandStringFunc(35)
	if err != nil {
		return "", err
	}

	sessionKey := sessionKeyPrefix + token
	cmdSet := as.redisClient.Set(ctx, sessionKey, createdAt.Unix(), 0)
	if err := cmdSet.Err(); err != nil {
		return "", err
	}

	// add token to list of sessions
	cmdSAdd := as.redisClient.SAdd(ctx, tokensSetKey, token)
	if err := cmdSAdd.Err(); err != nil {
		return "", err
	}

	return token, nil
}

func (as *Service) Logout(ctx context.Context, token string) (bool, error) {
	sessionKey := sessionKeyPrefix + token
	cmd := as.redisClient.Get(ctx, sessionKey)
	if err := cmd.Err(); err != nil {
		return false, err
	}

	createdAtUnixStr := cmd.Val()
	createdAtUnix, err := strconv.ParseInt(createdAtUnixStr, 10, 64)
	if err != nil {
		return false, err
	}

	cmdSet := as.redisClient.Set(ctx, sessionKey, 0, 0)
	if err := cmdSet.Err(); err != nil {
		return false, err
	}

	// remove token from the list of sessions
	cmdSRem := as.redisClient.SRem(ctx, tokensSetKey, token)
	if err := cmdSRem.Err(); err != nil {
		return false, err
	}

	return createdAtUnix > 0, nil
}

// ScanAndClean will run through all sessions, check the TTL, and clean them if old
func (as *Service) ScanAndClean(ctx context.Context) {
	cmd := as.redisClient.SMembers(ctx, tokensSetKey)
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
		cmd := as.redisClient.Get(ctx, sessionKey)
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
		cmdSet := as.redisClient.Del(ctx, sessionKey)
		if err := cmdSet.Err(); err != nil {
			log.Errorf("=> auth service, clean token %s: %s", token, err)
			continue
		}

		// remove token from the list of sessions
		cmdSRem := as.redisClient.SRem(ctx, tokensSetKey, token)
		if err := cmdSRem.Err(); err != nil {
			log.Errorf("=> auth service, clean token %s: %s", token, err)
			continue
		}
	}
}
