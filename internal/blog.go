package internal

import (
	"time"
)

type Blog struct {
	Id        int       `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Content   string    `json:"content"`
}
