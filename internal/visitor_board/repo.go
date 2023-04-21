package visitor_board

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
)

var (
	ErrMessageNotFound = errors.New("visitor board message not found")
)

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(
	ctx context.Context,
	dbHost, dbPort, dbName string,
	tracingEnabled bool,
) (*Repo, error) {
	connString := fmt.Sprintf("postgres://postgres@%s:%s/%s", dbHost, dbPort, dbName)
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse netlog db config: %w", err)
	}

	if tracingEnabled {
		poolConfig.ConnConfig.Tracer = otelpgx.NewTracer()
	}

	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	log.Debugf("notes api connected to: %s", connString)

	return &Repo{
		db: db,
	}, nil
}

func (r *Repo) CloseDB() {
	if r.db != nil {
		r.db.Close()
	}
}

func (r *Repo) Add(ctx context.Context, message Message) (int, error) {
	rows, err := r.db.Query(
		ctx,
		`INSERT INTO visitor_board_message (author, message, timestamp) VALUES ($1, $2, $3) RETURNING id;`,
		message.Author, message.Message, message.Timestamp,
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

func (r *Repo) List(ctx context.Context) ([]Message, error) {
	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				id, author, message, timestamp
			FROM visitor_board_message
			ORDER BY timestamp DESC;`,
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
		var timestamp time.Time
		if err := rows.Scan(&id, &author, &message, &timestamp); err != nil {
			return nil, err
		}
		messages = append(messages, Message{
			ID:        id,
			Author:    author,
			Message:   message,
			Timestamp: timestamp.Unix(),
		})
	}
	return messages, nil
}
