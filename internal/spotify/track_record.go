package spotify

import (
	"time"

	"github.com/zmb3/spotify/v2"
)

// Image identifies an image associated with an item.
type Image struct {
	// The image height, in pixels.
	Height int `db:"height" json:"height"`
	// The image width, in pixels.
	Width int `db:"width" json:"width"`
	// The source URL of the image.
	URL string `db:"url" json:"url"`
}

type TrackDBRecord struct {
	ID           int               `db:"id" json:"id"`
	Album        string            `db:"album" json:"album"`
	AlbumImages  []Image           `db:"album_images" json:"album_images"`
	ReleaseDate  time.Time         `db:"release_date" json:"release_date"`
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

	images := make([]Image, 0, len(item.Track.Album.Images))
	for _, img := range item.Track.Album.Images {
		images = append(images, Image{
			Height: int(img.Height),
			Width:  int(img.Width),
			URL:    img.URL,
		})
	}

	return TrackDBRecord{
		Album:        item.Track.Album.Name,
		Artists:      artists,
		AlbumImages:  images,
		ReleaseDate:  item.Track.Album.ReleaseDateTime(),
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
