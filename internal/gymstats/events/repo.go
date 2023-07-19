package events

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"go.opentelemetry.io/otel/attribute"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/codes"
)

type EventParams struct {
	Type *EventType
	From *time.Time
	To   *time.Time

	OnlyProd           bool
	ExcludeTestingData bool
}

type ListParams struct {
	EventParams
	Page int
	Size int
}

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) Add(ctx context.Context, event Event) (_ *Event, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.events.add")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); err != nil {
				err = fmt.Errorf("failed to rollback transaction: %w: %w", rollbackErr, err)
			}
		} else {
			err = tx.Commit(ctx)
		}
	}()

	err = tx.QueryRow(ctx, `
		INSERT INTO gymstats_event (type, data, timestamp)
		VALUES ($1, $2, $3)
		RETURNING id
	`,
		event.Type,
		event.Data,
		event.Timestamp,
	).Scan(&event.Type, &event.Data, &event.Timestamp)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *Repo) Get(ctx context.Context, id int) (_ *Event, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.events.get")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	event := &Event{}
	err = r.db.
		QueryRow(ctx, `
			SELECT id, type, data, timestamp
			FROM gymstats_event
			WHERE id = $1
		`, id).
		Scan(&event.ID, &event.Type, &event.Data, &event.Timestamp)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (r *Repo) List(ctx context.Context, params ListParams) (_ []*Event, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.events.listall")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	if params.Type != nil {
		span.SetAttributes(attribute.String("type", string(*params.Type)))
	}
	span.SetAttributes(attribute.Bool("only-prod", params.OnlyProd))
	span.SetAttributes(attribute.Bool("exclude-testing-data", params.ExcludeTestingData))
	if params.From != nil {
		span.SetAttributes(attribute.String("from", params.From.String()))
	}
	if params.To != nil {
		span.SetAttributes(attribute.String("to", params.To.String()))
	}

	events := make([]*Event, 0)
	rows, err := r.db.Query(ctx, `
		SELECT id, type, data, timestamp
		FROM gymstats_event
		WHERE ($1::text IS NULL OR type = $1)
		  AND ($2::timestamp IS NULL OR timestamp >= $2)
		  AND ($3::timestamp IS NULL OR timestamp <= $3)
		  AND ($4::boolean IS FALSE OR data->>'env' = 'prod' OR data->>'env' = 'production')
		  AND ($5::boolean IS FALSE OR data->>'testing' != 'true' OR data->>'test' != 'true')
		ORDER BY timestamp DESC
		LIMIT $6 OFFSET $7;
	`,
		params.Type,
		params.From, params.To,
		params.OnlyProd, params.ExcludeTestingData,
		params.Size, params.Size*params.Page,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	for rows.Next() {
		event := &Event{}
		if err := rows.Scan(&event.ID, &event.Type, &event.Data, &event.Timestamp); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

func (r *Repo) Count(ctx context.Context, params EventParams) (int, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.events.count")
	defer span.End()

	rows, err := r.db.Query(ctx, `
		SELECT COUNT(*) FROM gymstats_event
			WHERE ($1::text = '' OR type = $1)
		  	AND ($2::timestamp IS NULL OR timestamp >= $2)
			AND ($3::timestamp IS NULL OR timestamp <= $3)
			AND ($4::boolean IS FALSE OR data->>'env' = 'prod' OR data->>'env' = 'production')
			AND ($5::boolean IS FALSE OR data->>'testing' != 'true' OR data->>'test' != 'true');
	`,
		params.Type,
		params.From, params.To,
		params.OnlyProd, params.ExcludeTestingData,
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

	return -1, errors.New("unexpected error, failed to get exercises count")
}
