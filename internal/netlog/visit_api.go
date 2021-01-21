package netlog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
)

type VisitApi struct {
	db *pgxpool.Pool
}

func NewVisitApi() (*VisitApi, error) {
	ctx := context.Background()

	// TODO: place in config
	const connString = "postgres://postgres@localhost:5432/serj_blogs"
	dbpool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v\n", err)
	}

	visitApi := &VisitApi{
		db: dbpool,
	}

	return visitApi, nil
}

func (api *VisitApi) CloseDB() {
	if api.db != nil {
		api.db.Close()
	}
}

func (api *VisitApi) AddVisit(visit *Visit) error {
	if visit.URL == "" || visit.Timestamp.IsZero() {
		return errors.New("visit url or timestamp empty")
	}

	rows, err := api.db.Query(
		context.Background(),
		`INSERT INTO netlog.visit (title, url, timestamp) VALUES ($1, $2, $3) RETURNING id;`,
		visit.Title, visit.URL, visit.Timestamp,
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

func (api *VisitApi) GetVisits(keywords []string, limit int) ([]*Visit, error) {
	sbQueryLike := getQueryLikeCondition(keywords)
	query := fmt.Sprintf(`
		SELECT
			id, COALESCE(title, ''), COALESCE(source, ''), url, timestamp
		FROM netlog.visit
		%s
		ORDER BY id DESC
		LIMIT $1;
	`, sbQueryLike)

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

func (api *VisitApi) Count() (int, error) {
	rows, err := api.db.Query(
		context.Background(),
		`SELECT COUNT(*) FROM netlog.visit;`,
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

func (api *VisitApi) GetVisitsPage(keywords []string, page, size int) ([]*Visit, error) {
	limit := size
	offset := (page - 1) * size
	allVisitsCount, err := api.Count()
	if err != nil {
		return nil, err
	}

	if allVisitsCount <= limit {
		return api.GetVisits([]string{}, size)
	}

	if allVisitsCount-offset < limit {
		offset = allVisitsCount - limit
	}

	log.Tracef("getting visits, all count %d, limit %d, offset %d", allVisitsCount, limit, offset)

	sbQueryLike := getQueryLikeCondition(keywords)
	query := fmt.Sprintf(`
		SELECT
			id, COALESCE(title, ''), COALESCE(source, ''), url, timestamp
		FROM netlog.visit
		%s
		ORDER BY id DESC
		LIMIT $1
		OFFSET $2;
	`, sbQueryLike)

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
