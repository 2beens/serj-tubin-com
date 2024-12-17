package spotify

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) Add(ctx context.Context, track TrackDBRecord) (err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.spotify.add")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	externalURLs, err := json.Marshal(track.ExternalURLs)
	if err != nil {
		return fmt.Errorf("marshal external URLs: %w", err)
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO spotify_track_record (
			album, artists, duration_ms, explicit, external_urls, href, spotify_id, name, uri, track_type, played_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		track.Album, track.Artists, track.Duration, track.Explicit, externalURLs, track.Endpoint,
		track.SpotifyID, track.Name, track.URI, track.Type, track.PlayedAt,
	)
	return err
}

// GetPage returns a page of tracks from the database.
func (r *Repo) GetPage(ctx context.Context, page, size int) (_ []TrackDBRecord, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.spotify.getPage")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	if page < 1 {
		return nil, fmt.Errorf("page must be greater than 0")
	}
	if size < 1 {
		return nil, fmt.Errorf("size must be greater than 0")
	}

	limit := size
	offset := (page - 1) * size

	rows, err := r.db.Query(ctx, `
		SELECT * FROM spotify_track_record
		ORDER BY played_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var tracks []TrackDBRecord
	for rows.Next() {
		var track TrackDBRecord
		var externalURLs []byte
		if err := rows.Scan(
			&track.ID, &track.Album, &track.Artists, &track.Duration, &track.Explicit, &externalURLs,
			&track.Endpoint, &track.SpotifyID, &track.Name, &track.URI, &track.Type, &track.PlayedAt,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		if err := json.Unmarshal(externalURLs, &track.ExternalURLs); err != nil {
			return nil, fmt.Errorf("unmarshal external URLs: %w", err)
		}

		tracks = append(tracks, track)
	}

	return tracks, nil
}

func (r *Repo) GetByID(ctx context.Context, id int) (_ TrackDBRecord, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.spotify.getByID")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	row := r.db.QueryRow(ctx, `
		SELECT * FROM spotify_track_record 
		WHERE id = $1
	`, id)

	var track TrackDBRecord
	var externalURLs []byte
	err = row.Scan(
		&track.ID, &track.Album, &track.Artists, &track.Duration, &track.Explicit, &externalURLs,
		&track.Endpoint, &track.SpotifyID, &track.Name, &track.URI, &track.Type, &track.PlayedAt,
	)
	if err != nil {
		return TrackDBRecord{}, fmt.Errorf("scan: %w", err)
	}

	if err := json.Unmarshal(externalURLs, &track.ExternalURLs); err != nil {
		return TrackDBRecord{}, fmt.Errorf("unmarshal external URLs: %w", err)
	}

	return track, nil
}

func (r *Repo) GetLastPlayedTrackTime(ctx context.Context) (playedAt time.Time, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.spotify.getLastPlayedTrackTime")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	var nullPlayedAt sql.NullTime
	row := r.db.QueryRow(ctx, `
		SELECT MAX(played_at) FROM spotify_track_record
	`)

	if err := row.Scan(&nullPlayedAt); err != nil {
		return time.Time{}, fmt.Errorf("scan row: %w", err)
	}

	if nullPlayedAt.Valid {
		return nullPlayedAt.Time, nil
	}

	return time.Time{}, nil
}

func (r *Repo) Update(ctx context.Context, track TrackDBRecord) (err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.spotify.update")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	externalURLs, err := json.Marshal(track.ExternalURLs)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
		UPDATE spotify_track_record SET
			album = $1, artists = $2, duration_ms = $3, explicit = $4, external_urls = $5, href = $6, spotify_id = $7, name = $8, uri = $9, track_type = $10, played_at = $11
		WHERE id = $12`,
		track.Album, track.Artists, track.Duration, track.Explicit, externalURLs, track.Endpoint,
		track.SpotifyID, track.Name, track.URI, track.Type, track.PlayedAt, track.ID,
	)
	if err != nil {
		return fmt.Errorf("update track: %w", err)
	}
	return nil
}

func (r *Repo) Delete(ctx context.Context, id int) (err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.spotify.delete")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	_, err = r.db.Exec(ctx, `DELETE FROM spotify_track_record WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete track: %w", err)
	}

	return nil
}
