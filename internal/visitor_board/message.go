package visitor_board

import (
	"time"
)

type Message struct {
	ID        int       `json:"id"`
	Author    string    `json:"author"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
