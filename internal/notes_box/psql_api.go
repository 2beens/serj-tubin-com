package notes_box

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
)

var ErrNoteNotFound = errors.New("note not found")

type PsqlApi struct {
	// TODO: check if DB pool connection should be shared with other components
	// e.g. netlog PSQL API
	db *pgxpool.Pool
}

func NewPsqlApi(ctx context.Context, dbHost, dbPort, dbName string) (*PsqlApi, error) {
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

func (api *PsqlApi) Add(ctx context.Context, note *Note) (*Note, error) {
	if note.Content == "" || note.CreatedAt.IsZero() {
		return nil, errors.New("note content or timestamp empty")
	}

	rows, err := api.db.Query(
		ctx,
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

	if !rows.Next() {
		return nil, errors.New("unexpected error [no rows next]")
	}

	var id int
	if err := rows.Scan(&id); err != nil {
		return nil, fmt.Errorf("rows scan: %w", err)
	}

	note.Id = id
	return note, nil
}

func (api *PsqlApi) Get(ctx context.Context, noteId int) (*Note, error) {
	rows, err := api.db.Query(
		ctx,
		`SELECT * FROM note WHERE id = $1;`,
		noteId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if !rows.Next() {
		return nil, ErrNoteNotFound
	}

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

func (api *PsqlApi) Update(ctx context.Context, note *Note) error {
	if note.Content == "" {
		return errors.New("note content empty")
	}

	tag, err := api.db.Exec(
		ctx,
		`UPDATE note SET title = $1, content = $2 WHERE id = $3;`,
		note.Title, note.Content, note.Id,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNoteNotFound
	}

	return nil
}

func (api *PsqlApi) Delete(ctx context.Context, id int) error {
	tag, err := api.db.Exec(
		ctx,
		`DELETE FROM note WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoteNotFound
	}
	return nil
}

func (api *PsqlApi) List(ctx context.Context) ([]Note, error) {
	rows, err := api.db.Query(
		ctx,
		`
			SELECT
				id, title, created_at, content
			FROM note
			ORDER BY created_at DESC;`,
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
