package spotify_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/spotify"

	"github.com/stretchr/testify/assert"
	spotifyclient "github.com/zmb3/spotify/v2"
	"go.uber.org/mock/gomock"
)

func TestTracker_SaveRecentlyPlayedTracks_NoTracks(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTracksRepo := NewMocktracksRepo(ctrl)
	mockSpotifyClient := NewMockspotifyClient(ctrl)
	tracker := spotify.NewTracker(mockTracksRepo, mockSpotifyClient, 1)

	now := time.Now()
	mockTracksRepo.EXPECT().
		GetLastPlayedTrackTime(gomock.Any()).
		Return(now, nil)

	mockSpotifyClient.EXPECT().
		PlayerRecentlyPlayedOpt(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, ops *spotifyclient.RecentlyPlayedOptions) ([]spotifyclient.RecentlyPlayedItem, error) {
			lastPlayedTrackTime := now.Add(5 * time.Second)
			expectedOps := spotifyclient.RecentlyPlayedOptions{
				AfterEpochMs: lastPlayedTrackTime.Unix() * 1000,
				Limit:        35,
			}
			assert.Equal(t, expectedOps, *ops)
			return []spotifyclient.RecentlyPlayedItem{}, nil
		})

	assert.NoError(t, tracker.SaveRecentlyPlayedTracks(context.Background()))
}

func TestTracker_SaveRecentlyPlayedTracks_TracksFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTracksRepo := NewMocktracksRepo(ctrl)
	mockSpotifyClient := NewMockspotifyClient(ctrl)
	tracker := spotify.NewTracker(mockTracksRepo, mockSpotifyClient, 1)

	now := time.Now()
	mockTracksRepo.EXPECT().
		GetLastPlayedTrackTime(gomock.Any()).
		Return(now, nil)

	mockSpotifyClient.EXPECT().
		PlayerRecentlyPlayedOpt(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, ops *spotifyclient.RecentlyPlayedOptions) ([]spotifyclient.RecentlyPlayedItem, error) {
			lastPlayedTrackTime := now.Add(5 * time.Second)
			expectedOps := spotifyclient.RecentlyPlayedOptions{
				AfterEpochMs: lastPlayedTrackTime.Unix() * 1000,
				Limit:        35,
			}
			assert.Equal(t, expectedOps, *ops)
			return []spotifyclient.RecentlyPlayedItem{
				{
					Track: spotifyclient.SimpleTrack{
						Album: spotifyclient.SimpleAlbum{
							Name: "Test Album",
							Images: []spotifyclient.Image{
								{Height: 640, Width: 640, URL: "https://example.com/image1.jpg"},
								{Height: 300, Width: 300, URL: "https://example.com/image2.jpg"},
								{Height: 64, Width: 64, URL: "https://example.com/image3.jpg"},
							},
							ReleaseDate: "2023-01-01",
						},
						Artists: []spotifyclient.SimpleArtist{
							{Name: "Artist 1"},
							{Name: "Artist 2"},
						},
						Duration:     180000,
						Explicit:     true,
						ExternalURLs: map[string]string{"spotify": "https://open.spotify.com/track/123"},
						Endpoint:     "https://api.spotify.com/v1/tracks/123",
						ID:           spotifyclient.ID("123"),
						Name:         "Test Track",
						URI:          spotifyclient.URI("spotify:track:123"),
						Type:         "track",
					},
					PlayedAt: now.Add(-1 * time.Hour),
				},
			}, nil
		})

	mockTracksRepo.EXPECT().
		Add(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, track spotify.TrackDBRecord) error {
			assert.Equal(t, "Test Album", track.Album)
			assert.Equal(t, 2, len(track.Artists))
			assert.Equal(t, "Artist 1", track.Artists[0])
			assert.Equal(t, "Artist 2", track.Artists[1])
			assert.Equal(t, 180000, track.Duration)
			assert.True(t, track.Explicit)
			assert.Equal(t, "https://open.spotify.com/track/123", track.ExternalURLs["spotify"])
			assert.Equal(t, "https://api.spotify.com/v1/tracks/123", track.Endpoint)
			assert.Equal(t, "123", track.SpotifyID)
			assert.Equal(t, "Test Track", track.Name)
			assert.Equal(t, "spotify:track:123", track.URI)
			assert.Equal(t, "track", track.Type)
			assert.Equal(t, now.Add(-1*time.Hour).Truncate(time.Second), track.PlayedAt.Truncate(time.Second))
			return nil
		})

	assert.NoError(t, tracker.SaveRecentlyPlayedTracks(context.Background()))
}

func TestTracker_SaveRecentlyPlayedTracks_FetchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTracksRepo := NewMocktracksRepo(ctrl)
	mockSpotifyClient := NewMockspotifyClient(ctrl)
	tracker := spotify.NewTracker(mockTracksRepo, mockSpotifyClient, 1)

	now := time.Now()
	mockTracksRepo.EXPECT().
		GetLastPlayedTrackTime(gomock.Any()).
		Return(now, nil)

	mockSpotifyClient.EXPECT().
		PlayerRecentlyPlayedOpt(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("spotify API error"))

	err := tracker.SaveRecentlyPlayedTracks(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "spotify API error")
}

func TestTracker_SaveRecentlyPlayedTracks_AddTrackError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTracksRepo := NewMocktracksRepo(ctrl)
	mockSpotifyClient := NewMockspotifyClient(ctrl)
	tracker := spotify.NewTracker(mockTracksRepo, mockSpotifyClient, 1)

	now := time.Now()
	mockTracksRepo.EXPECT().
		GetLastPlayedTrackTime(gomock.Any()).
		Return(now, nil)

	mockSpotifyClient.EXPECT().
		PlayerRecentlyPlayedOpt(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, ops *spotifyclient.RecentlyPlayedOptions) ([]spotifyclient.RecentlyPlayedItem, error) {
			lastPlayedTrackTime := now.Add(5 * time.Second)
			expectedOps := spotifyclient.RecentlyPlayedOptions{
				AfterEpochMs: lastPlayedTrackTime.Unix() * 1000,
				Limit:        35,
			}
			assert.Equal(t, expectedOps, *ops)
			return []spotifyclient.RecentlyPlayedItem{
				{
					Track: spotifyclient.SimpleTrack{
						Album: spotifyclient.SimpleAlbum{
							Name: "Test Album",
							Images: []spotifyclient.Image{
								{Height: 640, Width: 640, URL: "https://example.com/image1.jpg"},
								{Height: 300, Width: 300, URL: "https://example.com/image2.jpg"},
								{Height: 64, Width: 64, URL: "https://example.com/image3.jpg"},
							},
							ReleaseDate: "2023-01-01",
						},
						Artists: []spotifyclient.SimpleArtist{
							{Name: "Artist 1"},
							{Name: "Artist 2"},
						},
						Duration:     180000,
						Explicit:     true,
						ExternalURLs: map[string]string{"spotify": "https://open.spotify.com/track/123"},
						Endpoint:     "https://api.spotify.com/v1/tracks/123",
						ID:           spotifyclient.ID("123"),
						Name:         "Test Track",
						URI:          spotifyclient.URI("spotify:track:123"),
						Type:         "track",
					},
					PlayedAt: now.Add(-1 * time.Hour),
				},
			}, nil
		})

	mockTracksRepo.EXPECT().
		Add(gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("database error"))

	err := tracker.SaveRecentlyPlayedTracks(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestTracker_Start_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockTracksRepo := NewMocktracksRepo(ctrl)
	mockSpotifyClient := NewMockspotifyClient(ctrl)
	tracker := spotify.NewTracker(mockTracksRepo, mockSpotifyClient, time.Duration(50)*time.Millisecond)

	assert.Equal(t, "stopped", tracker.Status())
	assert.False(t, tracker.IsRunning())

	mockTracksRepo.EXPECT().
		GetLastPlayedTrackTime(gomock.Any()).
		Return(time.Now(), nil).
		Times(2)

	mockSpotifyClient.EXPECT().
		PlayerRecentlyPlayedOpt(gomock.Any(), gomock.Any()).
		Return([]spotifyclient.RecentlyPlayedItem{}, nil).
		Times(2)

	tracker.Start()
	assert.True(t, tracker.IsRunning())
	tracker.Start() // consecutive start calls should be no-op
	assert.True(t, tracker.IsRunning())
	assert.Equal(t, "running", tracker.Status())

	time.Sleep(50 * time.Millisecond)
	tracker.Stop()
	assert.False(t, tracker.IsRunning())

	assert.Equal(t, "stopped", tracker.Status())
}
