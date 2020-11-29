package internal

import (
	"time"
)

type LoginSession struct {
	Token     string
	CreatedAt time.Time
	TTL       time.Duration
}
