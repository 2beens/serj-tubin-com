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

	albumImages, err := json.Marshal(track.AlbumImages)
	if err != nil {
		return fmt.Errorf("marshal album images: %w", err)
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO spotify_track_record (
			album, album_images, release_date, artists, duration_ms, explicit,
			external_urls, href, spotify_id, name, uri, track_type, played_at, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		track.Album, albumImages, track.ReleaseDate, track.Artists, track.Duration, track.Explicit, externalURLs, track.Endpoint,
		track.SpotifyID, track.Name, track.URI, track.Type, track.PlayedAt, track.Source,
	)
	if err != nil {
		return fmt.Errorf("db exec: %w", err)
	}

	return nil
}

type GetPageParams struct {
	Page           int
	Size           int
	SearchKeywords []string
}

func (r *Repo) GetPage(ctx context.Context, params GetPageParams) (_ []TrackDBRecord, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.spotify.getPage")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	if params.Page < 1 {
		return nil, fmt.Errorf("page must be greater than 0")
	}
	if params.Size < 1 {
		return nil, fmt.Errorf("size must be greater than 0")
	}

	limit := params.Size
	offset := (params.Page - 1) * params.Size

	query := `
        SELECT
            id, album, album_images, release_date, artists, duration_ms, explicit,
            external_urls, href, spotify_id, name, uri, track_type, played_at, source
        FROM spotify_track_record
        WHERE 1=1
    `
	var args []interface{}

	// Add search keywords filtering.
	if len(params.SearchKeywords) > 0 {
		for _, keyword := range params.SearchKeywords {
			query += ` AND (ARRAY_TO_STRING(artists, ' ') ILIKE '%' || $` + fmt.Sprint(len(args)+1) + ` || '%' OR name ILIKE '%' || $` + fmt.Sprint(len(args)+2) + ` || '%')`
			args = append(args, keyword, keyword)
		}
	}

	query += ` ORDER BY played_at DESC LIMIT $` + fmt.Sprint(len(args)+1) + ` OFFSET $` + fmt.Sprint(len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var tracks []TrackDBRecord
	for rows.Next() {
		var track TrackDBRecord
		var externalURLs, albumImages []byte
		var releaseDate sql.NullTime
		if err := rows.Scan(
			&track.ID, &track.Album, &albumImages, &releaseDate, &track.Artists, &track.Duration, &track.Explicit,
			&externalURLs, &track.Endpoint, &track.SpotifyID, &track.Name, &track.URI, &track.Type, &track.PlayedAt, &track.Source,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		if err := json.Unmarshal(externalURLs, &track.ExternalURLs); err != nil {
			return nil, fmt.Errorf("unmarshal external URLs: %w", err)
		}

		if err := json.Unmarshal(albumImages, &track.AlbumImages); err != nil {
			return nil, fmt.Errorf("unmarshal album images: %w", err)
		}

		if releaseDate.Valid {
			track.ReleaseDate = releaseDate.Time
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
		SELECT
		    id, album, album_images, release_date, artists, duration_ms, explicit,
		    external_urls, href, spotify_id, name, uri, track_type, played_at, source
		FROM spotify_track_record
		WHERE id = $1
	`, id)

	var track TrackDBRecord
	var externalURLs, albumImages []byte
	var releaseDate sql.NullTime
	err = row.Scan(
		&track.ID, &track.Album, &albumImages, &releaseDate, &track.Artists, &track.Duration, &track.Explicit, &externalURLs,
		&track.Endpoint, &track.SpotifyID, &track.Name, &track.URI, &track.Type, &track.PlayedAt, &track.Source,
	)
	if err != nil {
		return TrackDBRecord{}, fmt.Errorf("scan: %w", err)
	}

	if err := json.Unmarshal(externalURLs, &track.ExternalURLs); err != nil {
		return TrackDBRecord{}, fmt.Errorf("unmarshal external URLs: %w", err)
	}

	if releaseDate.Valid {
		track.ReleaseDate = releaseDate.Time
	}

	if err := json.Unmarshal(albumImages, &track.AlbumImages); err != nil {
		return TrackDBRecord{}, fmt.Errorf("unmarshal album images: %w", err)
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

	albumImages, err := json.Marshal(track.AlbumImages)
	if err != nil {
		return fmt.Errorf("marshal album images: %w", err)
	}

	_, err = r.db.Exec(ctx, `
		UPDATE spotify_track_record SET
			album = $1, album_images = $2, release_date = $3, artists = $4, duration_ms = $5, explicit = $6,
			external_urls = $7, href = $8, spotify_id = $9, name = $10, uri = $11, track_type = $12, played_at = $13, source = $14
		WHERE id = $15`,
		track.Album, albumImages, track.ReleaseDate, track.Artists, track.Duration, track.Explicit,
		externalURLs, track.Endpoint, track.SpotifyID, track.Name, track.URI, track.Type, track.PlayedAt,
		track.Source, track.ID,
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
