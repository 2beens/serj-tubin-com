package spotify

import (
	"time"

	"github.com/zmb3/spotify/v2"
)

type TrackDBRecord struct {
	ID           int               `db:"id" json:"id"`
	Album        string            `db:"album" json:"album"`
	Artists      []string          `db:"artists" json:"artists"`
	Duration     int               `db:"duration_ms" json:"duration_ms"`
	Explicit     bool              `db:"explicit" json:"explicit"`
	ExternalURLs map[string]string `db:"external_urls" json:"external_urls"`
	Endpoint     string            `db:"href" json:"href"`
	SpotifyID    string            `db:"spotify_id" json:"spotify_id"`
	Name         string            `db:"name" json:"name"`
	URI          string            `db:"uri" json:"uri"`
	Type         string            `db:"type" json:"type"`
	PlayedAt     time.Time         `db:"played_at" json:"played_at"`
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
