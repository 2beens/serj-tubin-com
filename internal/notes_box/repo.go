package notes_box

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNoteNotFound = errors.New("note not found")

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) (*Repo, error) {
	return &Repo{
		db: db,
	}, nil
}

func (r *Repo) Add(ctx context.Context, note *Note) (*Note, error) {
	if note.Content == "" || note.CreatedAt.IsZero() {
		return nil, errors.New("note content or timestamp empty")
	}

	rows, err := r.db.Query(
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

func (r *Repo) Get(ctx context.Context, noteId int) (*Note, error) {
	rows, err := r.db.Query(
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

func (r *Repo) Update(ctx context.Context, note *Note) error {
	if note.Content == "" {
		return errors.New("note content empty")
	}

	tag, err := r.db.Exec(
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

func (r *Repo) Delete(ctx context.Context, id int) error {
	tag, err := r.db.Exec(
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

func (r *Repo) List(ctx context.Context) ([]Note, error) {
	rows, err := r.db.Query(
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
