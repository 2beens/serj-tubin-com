package main

import (
	"errors"
	"strconv"
	"time"

	"github.com/2beens/serjtubincom/internal/spotify"
)

// dataRoot represents the top-level structure for unmarshalling the JSON
type dataRoot []trackWrapper

// trackWrapper is a wrapper containing a list of tracks
type trackWrapper struct {
	Track []track `json:"track"`
}

// track represents a single scrobbled track
type track struct {
	Artist     artist       `json:"artist"`
	Streamable string       `json:"streamable"`
	Image      []Image      `json:"image"`
	Mbid       string       `json:"mbid"`
	Album      album        `json:"album"`
	Name       string       `json:"name"`
	URL        string       `json:"url"`
	Date       scrobbleDate `json:"date"`
}

// artist represents the artist of the track
type artist struct {
	Mbid string `json:"mbid"`
	Name string `json:"#text"`
}

// Image represents the image of the track
type Image struct {
	Size string `json:"size"`
	URL  string `json:"#text"`
}

// album represents the album of the track
type album struct {
	Mbid string `json:"mbid"`
	Name string `json:"#text"`
}

// scrobbleDate represents the scrobble date of the track
type scrobbleDate struct {
	Uts  string `json:"uts"`
	Text string `json:"#text"`
}

func mapLastFMTrackToSpotifyTrack(lfmTrack track) (spotify.TrackDBRecord, error) {
	parseUTSToTime := func(uts string) (time.Time, error) {
		// Parse the UNIX timestamp from string to int64
		timestamp, err := strconv.ParseInt(uts, 10, 64)
		if err != nil {
			return time.Time{}, errors.New("invalid UNIX timestamp")
		}

		// Convert the UNIX timestamp to time.Time
		return time.Unix(timestamp, 0), nil
	}

	// Convert scrobble date (UNIX timestamp) to time.Time
	playedAt, err := parseUTSToTime(lfmTrack.Date.Uts)
	if err != nil {
		return spotify.TrackDBRecord{}, err
	}

	// Map LastFM images to Spotify-compatible images
	var albumImages []spotify.Image
	for _, img := range lfmTrack.Image {
		albumImages = append(albumImages, spotify.Image{
			URL: img.URL,
		})
	}

	return spotify.TrackDBRecord{
		Album:       lfmTrack.Album.Name,
		AlbumImages: albumImages,
		Artists:     []string{lfmTrack.Artist.Name},
		Name:        lfmTrack.Name,
		PlayedAt:    playedAt,
		Source:      "lastfm",
	}, nil
}
