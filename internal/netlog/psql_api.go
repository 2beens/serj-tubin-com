package netlog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
)

type PsqlApi struct {
	db *pgxpool.Pool
}

func NewNetlogPsqlApi() (*PsqlApi, error) {
	ctx := context.Background()

	// TODO: place in config
	const connString = "postgres://postgres@localhost:5432/serj_blogs"
	dbpool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v\n", err)
	}

	return &PsqlApi{
		db: dbpool,
	}, nil
}

func (api *PsqlApi) CloseDB() {
	if api.db != nil {
		api.db.Close()
	}
}

func (api *PsqlApi) AddVisit(visit *Visit) error {
	if visit.URL == "" || visit.Timestamp.IsZero() {
		return errors.New("visit url or timestamp empty")
	}

	rows, err := api.db.Query(
		context.Background(),
		`INSERT INTO netlog.visit (title, source, url, timestamp) VALUES ($1, $2, $3, $4) RETURNING id;`,
		visit.Title, visit.Source, visit.URL, visit.Timestamp,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return err
	}

	if rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			visit.Id = id
			return nil
		}
	}

	return errors.New("unexpected error, failed to insert visit")
}

func (api *PsqlApi) GetVisits(keywords []string, field string, source string, limit int) ([]*Visit, error) {
	sbQueryLike := getQueryLikeCondition(field, keywords)
	sourceCondition := ""
	if source != "all" && sbQueryLike == "" {
		sourceCondition = fmt.Sprintf("WHERE source = '%s'", source)
	} else if source != "all" {
		sourceCondition = fmt.Sprintf("AND source = '%s'", source)
	}

	query := fmt.Sprintf(`
		SELECT
			id, COALESCE(title, ''), COALESCE(source, ''), url, timestamp
		FROM netlog.visit
		%s
		%s
		ORDER BY id DESC
		LIMIT $1;
	`, sbQueryLike, sourceCondition)

	rows, err := api.db.Query(
		context.Background(),
		query,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var visits []*Visit
	for rows.Next() {
		var id int
		var title string
		var source string
		var url string
		var timestamp time.Time
		if err := rows.Scan(&id, &title, &source, &url, &timestamp); err != nil {
			return nil, err
		}
		visits = append(visits, &Visit{
			Id:        id,
			Title:     title,
			URL:       url,
			Timestamp: timestamp,
		})
	}

	return visits, nil
}

func (api *PsqlApi) CountAll() (int, error) {
	return api.Count([]string{}, "url", "all")
}

func (api *PsqlApi) Count(keywords []string, field string, source string) (int, error) {
	sbQueryLike := getQueryLikeCondition(field, keywords)
	sourceCondition := ""
	if source != "all" && sbQueryLike == "" {
		sourceCondition = fmt.Sprintf("WHERE source = '%s'", source)
	} else if source != "all" {
		sourceCondition = fmt.Sprintf("AND source = '%s'", source)
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM netlog.visit
		%s
		%s
		;
	`, sbQueryLike, sourceCondition)

	rows, err := api.db.Query(
		context.Background(),
		query,
	)
	if err != nil {
		return -1, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return -1, err
	}

	if rows.Next() {
		var count int
		if err := rows.Scan(&count); err == nil {
			return count, nil
		}
	}

	return -1, errors.New("unexpected error, failed to get netlog visits count")
}

func (api *PsqlApi) GetVisitsPage(keywords []string, field string, source string, page int, size int) ([]*Visit, error) {
	limit := size
	offset := (page - 1) * size
	allVisitsCount, err := api.CountAll()
	if err != nil {
		return nil, err
	}

	if allVisitsCount <= limit {
		return api.GetVisits([]string{}, field, source, size)
	}

	if allVisitsCount-offset < limit {
		offset = allVisitsCount - limit
	}

	log.Tracef("getting visits, all count %d, limit %d, offset %d", allVisitsCount, limit, offset)

	sbQueryLike := getQueryLikeCondition(field, keywords)
	sourceCondition := ""
	if source != "all" && sbQueryLike == "" {
		sourceCondition = fmt.Sprintf("WHERE source = '%s'", source)
	} else if source != "all" {
		sourceCondition = fmt.Sprintf("AND source = '%s'", source)
	}

	query := fmt.Sprintf(`
		SELECT
			id, COALESCE(title, ''), COALESCE(source, ''), url, timestamp
		FROM netlog.visit
		%s
		%s
		ORDER BY timestamp DESC
		LIMIT $1
		OFFSET $2;
	`, sbQueryLike, sourceCondition)

	rows, err := api.db.Query(
		context.Background(),
		query,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var visits []*Visit
	for rows.Next() {
		var id int
		var title string
		var source string
		var url string
		var timestamp time.Time
		if err := rows.Scan(&id, &title, &source, &url, &timestamp); err != nil {
			return nil, err
		}
		visits = append(visits, &Visit{
			Id:        id,
			Title:     title,
			Source:    source,
			URL:       url,
			Timestamp: timestamp,
		})
	}

	return visits, nil
}