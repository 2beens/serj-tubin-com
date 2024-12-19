package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
)

var (
	ErrLoginSessionNotFound = errors.New("login session not found")
	ErrWrongUsername        = errors.New("wrong username")
	ErrWrongPassword        = errors.New("wrong password")
)

const (
	DefaultTTL       = 90 * 24 * time.Hour // 90 days ~ 3 months
	sessionKeyPrefix = "serj-service-session||"
	tokensSetKey     = "serj-service-sessions"
)

type Admin struct {
	Username     string
	PasswordHash string
}

type Credentials struct {
	Username string
	Password string
}

type LoginSession struct {
	Token     string
	CreatedAt time.Time
}

type Service struct {
	admin       *Admin
	redisClient *redis.Client
	ttl         time.Duration
	// ability to inject random string generator func for tokens (for unit and dev testing)
	RandStringFunc func(s int) (string, error)
}

func NewAuthService(
	admin *Admin,
	ttl time.Duration,
	redisClient *redis.Client,
) *Service {
	return &Service{
		admin:          admin,
		ttl:            ttl,
		redisClient:    redisClient,
		RandStringFunc: pkg.GenerateRandomString,
	}
}

func (as *Service) Login(ctx context.Context, creds Credentials, createdAt time.Time) (string, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "authService.login")
	defer span.End()

	if creds.Username != as.admin.Username {
		return "", ErrWrongUsername
	}

	if !pkg.CheckPasswordHash(creds.Password, as.admin.PasswordHash) {
		return "", ErrWrongPassword
	}

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
	ctx, span := tracing.GlobalTracer.Start(ctx, "authService.logout")
	defer span.End()

	sessionKey := sessionKeyPrefix + token
	cmd := as.redisClient.Get(ctx, sessionKey)
	if err := cmd.Err(); err != nil {
		if err.Error() == "redis: nil" {
			return false, ErrLoginSessionNotFound
		}
		return false, fmt.Errorf("get session from redis: %w", err)
	}

	createdAtUnixStr := cmd.Val()
	createdAtUnix, err := strconv.ParseInt(createdAtUnixStr, 10, 64)
	if err != nil {
		return false, fmt.Errorf("parse created at: %w", err)
	}

	cmdSet := as.redisClient.Set(ctx, sessionKey, 0, 0)
	if err := cmdSet.Err(); err != nil {
		return false, fmt.Errorf("redis set session 0 0: %w", err)
	}

	// remove token from the list of sessions
	cmdSRem := as.redisClient.SRem(ctx, tokensSetKey, token)
	if err := cmdSRem.Err(); err != nil {
		return false, fmt.Errorf("redis remove token from set: %w", err)
	}

	return createdAtUnix > 0, nil
}

// ScanAndClean will run through all sessions, check the TTL, and clean them if old
func (as *Service) ScanAndClean(ctx context.Context) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "authService.scanAndClean")
	defer span.End()
	defer func(start time.Time) {
		span.SetAttributes(attribute.Int64("elapsed.ms", time.Since(start).Milliseconds()))
	}(time.Now())

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
			// just remove the token with invalid value
			toRemove = append(toRemove, token)
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
