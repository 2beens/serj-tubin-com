package notes_box

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"

	log "github.com/sirupsen/logrus"
)

type PsqlApi struct {
	// TODO: check if DB pool connection should be shared with other components
	// e.g. netlog PSQL API
	db *pgxpool.Pool
}

func NewPsqlApi(dbHost, dbPort, dbName string) (*PsqlApi, error) {
	ctx := context.Background()

	connString := fmt.Sprintf("postgres://postgres@%s:%s/%s", dbHost, dbPort, dbName)
	dbPool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("notes api unable to connect to database: %w", err)
	}

	log.Debugf("notes api connected to: %s", connString)

	return &PsqlApi{
		db: dbPool,
	}, nil
}

func (api *PsqlApi) CloseDB() {
	if api.db != nil {
		api.db.Close()
	}
}

func (api *PsqlApi) Add(note *Note) (*Note, error) {
	if note.Content == "" || note.CreatedAt.IsZero() {
		return nil, errors.New("note content or timestamp empty")
	}

	rows, err := api.db.Query(
		context.Background(),
		`INSERT INTO note (title, created_at, content) VALUES ($1, $2, $3) RETURNING id;`,
		note.Title, note.CreatedAt, note.Content,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			note.Id = id
			return note, nil
		}
	}

	return nil, errors.New("unexpected error, failed to insert note")
}

func (api *PsqlApi) Get(id int) (*Note, error) {
	rows, err := api.db.Query(
		context.Background(),
		`SELECT * FROM note WHERE id = $1;`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if rows.Next() {
		var id int
		var title string
		var createdAt time.Time
		var content string
		if err := rows.Scan(&id, &title, &createdAt, &content); err != nil {
			return nil, err
		}
		return &Note{
			Id:        id,
			Title:     title,
			CreatedAt: createdAt,
			Content:   content,
		}, nil
	}

	return nil, errors.New("unexpected error, failed to get note")
}

func (api *PsqlApi) Remove(id int) (bool, error) {
	tag, err := api.db.Exec(
		context.Background(),
		`DELETE FROM note WHERE id = $1`,
		id,
	)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	return true, nil
}

func (api *PsqlApi) List() ([]Note, error) {
	rows, err := api.db.Query(
		context.Background(),
		`
			SELECT
				id, title, created_at, content
			FROM note;`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var notes []Note
	for rows.Next() {
		var id int
		var title string
		var createdAt time.Time
		var content string
		if err := rows.Scan(&id, &title, &createdAt, &content); err != nil {
			return nil, err
		}
		notes = append(notes, Note{
			Id:        id,
			Title:     title,
			CreatedAt: createdAt,
			Content:   content,
		})
	}

	return notes, nil
}
