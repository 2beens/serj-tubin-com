package netlog

import (
	"time"
)

type Visit struct {
	Id        int       `json:"id"`
	Title     string    `json:"title"`
	Source    string    `json:"source"`
	URL       string    `json:"url"`       // mandatory
	Timestamp time.Time `json:"timestamp"` // mandatory
}
