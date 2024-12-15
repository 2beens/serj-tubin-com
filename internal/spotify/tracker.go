package spotify

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	log "github.com/sirupsen/logrus"
	"github.com/zmb3/spotify/v2"
)

type signal struct{}

// Tracker is a struct that fires at every interval (e.g. 24 hours, at midnight),
// fetches the logged-in user's recently played tracks and saves them to the database.
// The user is actually - me!
type Tracker struct {
	// repo is the repository used to interact with the database.
	repo *Repo
	// client is the Spotify client used to interact with the Spotify API.
	client *spotify.Client
	// isRunning is a flag that indicates whether the tracker is running or not.
	isRunning bool
	// fireIntervalMinutes is the interval in minutes at which the tracker should fire.
	fireIntervalMinutes int
	wg                  sync.WaitGroup
	stopCh              chan signal
}

func NewTracker(
	repo *Repo,
	client *spotify.Client,
	fireIntervalMinutes int,
) *Tracker {
	return &Tracker{
		repo:                repo,
		client:              client,
		isRunning:           false,
		fireIntervalMinutes: fireIntervalMinutes,
		stopCh:              make(chan signal),
	}
}

func (t *Tracker) IsRunning() bool {
	return t.isRunning
}

func (t *Tracker) Status() string {
	if t.isRunning {
		return "running"
	}
	return "stopped"
}

func (t *Tracker) Stop() {
	if !t.isRunning {
		return
	}

	// send a signal to stop the tracker loop
	t.stopCh <- signal{}
	// wait for the tracker to stop
	t.wg.Wait()
	t.isRunning = false
}

func (t *Tracker) Start() {
	if t.isRunning {
		return
	}

	t.isRunning = true
	t.wg.Add(1)

	log.Debugf("starting tracker loop, next fire in %d minutes", t.fireIntervalMinutes)

	go func() {
		defer t.wg.Done()

		for {
			select {
			case <-t.stopCh:
				return
			case <-time.After(time.Duration(t.fireIntervalMinutes) * time.Minute):
				log.Debugf("tracker tick, saving recently played tracks ...")
				ctx, span := tracing.GlobalTracer.Start(context.Background(), "spotify.tracker.tick")
				err := t.SaveRecentlyPlayedTracks(ctx)
				if err != nil {
					log.Errorf("failed to save recently played tracks: %s", err)
				}
				tracing.EndSpanWithErrCheck(span, err)
			}
		}
	}()
}

func (t *Tracker) SaveRecentlyPlayedTracks(ctx context.Context) (err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "spotify.tracker.saveRecentlyPlayedTracks")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	lastPlayedTrackTime, err := t.repo.GetLastPlayedTrackTime(ctx)
	if err != nil {
		return fmt.Errorf("get last played track time: %w", err)
	}
	log.Debugf("last played track time: %s", lastPlayedTrackTime)

	// check if the last played track time is zero, if so, set it to 7 days ago
	if lastPlayedTrackTime.IsZero() {
		lastPlayedTrackTime = time.Now().Add(-7 * 24 * time.Hour)
	}

	ops := spotify.RecentlyPlayedOptions{
		// AfterEpochMs is a Unix epoch in milliseconds that describes a time after
		// which to return songs.
		AfterEpochMs: lastPlayedTrackTime.Unix() * 1000,
	}

	tracks, err := t.client.PlayerRecentlyPlayedOpt(ctx, &ops)
	if err != nil {
		return fmt.Errorf("get recently played tracks: %w", err)
	}
	log.Debugf("got %d recently played tracks", len(tracks))

	for _, track := range tracks {
		trackDBRecord := NewTrackDBRecordFromRecentlyPlayedItem(track)
		if err := t.repo.Add(ctx, trackDBRecord); err != nil {
			return fmt.Errorf("add track to db: %w", err)
		}
		log.Debugf("saved track -> %s: %s [at: %s]", trackDBRecord.Artists, trackDBRecord.Name, trackDBRecord.PlayedAt)
	}

	return nil
}
