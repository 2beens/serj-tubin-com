package spotify

import (
	"time"

	"github.com/zmb3/spotify/v2"
)

type TrackDBRecord struct {
	ID           int               `db:"id"`
	Album        string            `db:"album"`
	Artists      []string          `db:"artists"`
	Duration     int               `db:"duration_ms"`
	Explicit     bool              `db:"explicit"`
	ExternalURLs map[string]string `db:"external_urls"`
	Endpoint     string            `db:"href"`
	SpotifyID    string            `db:"spotify_id"`
	Name         string            `db:"name"`
	URI          string            `db:"uri"`
	Type         string            `db:"type"`
	PlayedAt     time.Time         `db:"played_at"`
}

func NewTrackDBRecordFromRecentlyPlayedItem(item spotify.RecentlyPlayedItem) TrackDBRecord {
	artists := make([]string, 0, len(item.Track.Artists))
	for _, artist := range item.Track.Artists {
		artists = append(artists, artist.Name)
	}

	return TrackDBRecord{
		Album:        item.Track.Album.Name,
		Artists:      artists,
		Duration:     int(item.Track.Duration),
		Explicit:     item.Track.Explicit,
		ExternalURLs: item.Track.ExternalURLs,
		Endpoint:     item.Track.Endpoint,
		SpotifyID:    string(item.Track.ID),
		Name:         item.Track.Name,
		URI:          string(item.Track.URI),
		Type:         item.Track.Type,
		PlayedAt:     item.PlayedAt,
	}
}
