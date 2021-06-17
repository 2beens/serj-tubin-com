package internal

import (
	"time"
)

type Admin struct {
	Username     string
	PasswordHash string
}

type LoginSession struct {
	Token     string
	CreatedAt time.Time
	// TODO: make use of this, it's still not used
	TTL time.Duration
}
