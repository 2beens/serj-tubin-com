package netlog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
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

func (b *VisitApi) CloseDB() {
	if b.db != nil {
		b.db.Close()
	}
}

func (b *VisitApi) AddVisit(visit *Visit) error {
	if visit.URL == "" || visit.Timestamp.IsZero() {
		return errors.New("visit url or timestamp empty")
	}

	rows, err := b.db.Query(
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

func (b *VisitApi) GetVisits(limit int) ([]*Visit, error) {
	rows, err := b.db.Query(
		context.Background(),
		`SELECT * FROM netlog.visit ORDER BY id DESC LIMIT $1;`,
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
		var url string
		var timestamp time.Time
		if err := rows.Scan(&id, &title, &url, &timestamp); err != nil {
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
