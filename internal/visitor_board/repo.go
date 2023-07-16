package visitor_board

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
)

var (
	ErrMessageNotFound = errors.New("visitor board message not found")
)

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) Add(ctx context.Context, message Message) (int, error) {
	rows, err := r.db.Query(
		ctx,
		`INSERT INTO visitor_board_message (author, message, created_at) VALUES ($1, $2, $3) RETURNING id;`,
		message.Author, message.Message, message.CreatedAt,
	)
	if err != nil {
		return -1, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return -1, err
	}

	if !rows.Next() {
		return -1, errors.New("unexpected error [no rows next]")
	}

	var id int
	if err := rows.Scan(&id); err != nil {
		return -1, fmt.Errorf("rows scan: %w", err)
	}

	return id, nil
}

func (r *Repo) Delete(ctx context.Context, id int) error {
	tag, err := r.db.Exec(
		ctx,
		`DELETE FROM visitor_board_message WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

type ListOptions struct {
	Limit int
}

func ListWithLimit(limit int) func(*ListOptions) {
	return func(opts *ListOptions) {
		opts.Limit = limit
	}
}

func (r *Repo) List(ctx context.Context, options ...func(*ListOptions)) ([]Message, error) {
	opts := &ListOptions{}
	for _, option := range options {
		option(opts)
	}

	limitClause := ""
	var params []interface{}
	if opts.Limit > 0 {
		limitClause = "LIMIT $1"
		params = append(params, opts.Limit)
	}

	query := `
		SELECT
			id, author, message, created_at
		FROM visitor_board_message
		ORDER BY created_at ASC ` + limitClause + ";"
	rows, err := r.db.Query(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return r.rows2messages(rows)
}

func (r *Repo) GetMessagesPage(ctx context.Context, page, size int) ([]Message, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "boardMessagesRepo.page")
	span.SetAttributes(attribute.Int("page", page))
	span.SetAttributes(attribute.Int("size", size))
	defer span.End()

	limit := size
	offset := (page - 1) * size
	allMessagesCount, err := r.AllMessagesCount(ctx)
	if err != nil {
		return nil, err
	}

	if allMessagesCount <= limit {
		return r.List(ctx)
	}

	if allMessagesCount-offset < limit {
		offset = allMessagesCount - limit
	}

	log.Tracef("getting board messages, count %d, limit %d, offset %d", allMessagesCount, limit, offset)

	rows, err := r.db.Query(
		ctx,
		`
			SELECT * FROM visitor_board_message
			ORDER BY id DESC
			LIMIT $1
			OFFSET $2;
		`,
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

	return r.rows2messages(rows)
}

func (r *Repo) AllMessagesCount(ctx context.Context) (int, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "boardMessagesRepo.allCount")
	defer span.End()

	rows, err := r.db.Query(ctx, `SELECT COUNT(*) FROM visitor_board_message`)
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

	return -1, errors.New("unexpected error, failed to get visitor board messages count")
}

func (r *Repo) rows2messages(rows pgx.Rows) ([]Message, error) {
	var messages []Message
	for rows.Next() {
		var id int
		var author string
		var message string
		var createdAt time.Time
		if err := rows.Scan(&id, &author, &message, &createdAt); err != nil {
			return nil, err
		}
		messages = append(messages, Message{
			ID:        id,
			Author:    author,
			Message:   message,
			CreatedAt: createdAt,
		})
	}
	return messages, nil
}
