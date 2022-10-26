package blog

import (
	"errors"
	"time"
)

var (
	ErrBlogNotFound            = errors.New("blog not found")
	ErrBlogTitleOrContentEmpty = errors.New("blog title or content empty")
)

type Blog struct {
	Id        int       `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Content   string    `json:"content"`
	Claps     int       `json:"claps"` // basically blog likes

	// TODO: maybe also add public comments?
}
