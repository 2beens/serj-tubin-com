package spotify

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"

	log "github.com/sirupsen/logrus"
	"github.com/zmb3/spotify/v2"
)

//go:generate mockgen -source=$GOFILE -destination=tracker_mocks_test.go -package=spotify_test

type tracksRepo interface {
	GetLastPlayedTrackTime(context.Context) (time.Time, error)
	Add(context.Context, TrackDBRecord) error
}

type spotifyClient interface {
	PlayerRecentlyPlayedOpt(ctx context.Context, opt *spotify.RecentlyPlayedOptions) ([]spotify.RecentlyPlayedItem, error)
}

type signal struct{}

// Tracker is a struct that fires at every interval (e.g. 24 hours, at midnight),
// fetches the logged-in user's recently played tracks and saves them to the database.
// The user is actually - me!
type Tracker struct {
	// tracksRepo is the repository used to interact with the database.
	repo tracksRepo
	// client is the Spotify client used to interact with the Spotify API.
	client spotifyClient
	// isRunning is a flag that indicates whether the tracker is running or not.
	isRunning bool
	// fireIntervalMinutes is the interval in minutes at which the tracker should fire.
	fireIntervalMinutes int
	wg                  sync.WaitGroup
	stopCh              chan signal
}

func NewTracker(
	repo tracksRepo,
	client spotifyClient,
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
	log.Debugf("tracker stopped")
}

// Start starts the tracker loop. The tracker will fire at the interval specified
// in the fireIntervalMinutes field. It will also start the first iteration immediately.
func (t *Tracker) Start() {
	if t.isRunning {
		return
	}

	t.isRunning = true
	t.wg.Add(1)

	log.Debugf("starting tracker loop and first iteration, next fire in %d minutes", t.fireIntervalMinutes)
	if err := t.SaveRecentlyPlayedTracks(context.Background()); err != nil {
		log.Errorf("failed to save (some) recently played tracks: %s", err)
	}

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
					log.Errorf("failed to save (some) recently played tracks: %s", err)
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

	// also add a few seconds to the last played track time to avoid fetching the same track again
	lastPlayedTrackTime = lastPlayedTrackTime.Add(5 * time.Second)

	ops := spotify.RecentlyPlayedOptions{
		// AfterEpochMs is a Unix epoch in milliseconds that describes a time after
		// which to return songs.
		AfterEpochMs: lastPlayedTrackTime.Unix() * 1000,
		Limit:        35,
	}

	tracks, err := t.client.PlayerRecentlyPlayedOpt(ctx, &ops)
	if err != nil {
		return fmt.Errorf("get recently played tracks: %w", err)
	}
	log.Debugf("got %d recently played tracks", len(tracks))

	for _, track := range tracks {
		trackDBRecord := NewTrackDBRecordFromRecentlyPlayedItem(track)
		if addErr := t.repo.Add(ctx, trackDBRecord); addErr != nil {
			err = errors.Join(err, fmt.Errorf(
				"add track [spid:%s song:%s-%s] to db: %w",
				trackDBRecord.SpotifyID, trackDBRecord.Artists, trackDBRecord.Name, addErr,
			))
		} else {
			log.Debugf("saved track -> %s: %s [at: %s]", trackDBRecord.Artists, trackDBRecord.Name, trackDBRecord.PlayedAt)
		}
	}

	if err != nil {
		return fmt.Errorf("save recently played tracks: %w", err)
	}

	return nil
}
