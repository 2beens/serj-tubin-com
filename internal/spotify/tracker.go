package spotify

import (
	"github.com/zmb3/spotify/v2"
)

// Tracker is a struct that fires at every interval (e.g. 24 hours, at midnight),
// fetches the logged-in user's recently played tracks and saves them to the database.
// The user is actually - me!
type Tracker struct {
	repo   *Repo
	client *spotify.Client
	// status can be: stopped, started, terminated (in case of an error)
	status string
}

func NewTracker(repo *Repo, client *spotify.Client) *Tracker {
	return &Tracker{
		repo:   repo,
		client: client,
		status: "stopped",
	}
}

func (t *Tracker) Status() string {
	return t.status
}

func (t *Tracker) IsRunning() bool {
	return t.status == "started"
}

func (t *Tracker) Stop() {
	if t.status == "stopped" {
		return
	}

	t.status = "stopped"

	// TODO:
}

func (t *Tracker) Start() {
	if t.status == "started" {
		return
	}

	t.status = "started"

	// TODO:
}
