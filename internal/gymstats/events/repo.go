package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
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
		tracing.EndSpanWithErrCheck(span, err)
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

	log.Debugf("adding event: %+v", event)

	dataJson, err := json.Marshal(event.Data)
	if err != nil {
		return nil, fmt.Errorf("marshal data: %w", err)
	}

	rows, err := r.db.Query(
		ctx, `
			INSERT INTO gymstats_event (type, data, timestamp) 
			VALUES ($1, $2, $3)
			RETURNING id;`,
		event.Type,
		dataJson,
		event.Timestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("query rows: %w", err)
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

	span.SetAttributes(attribute.Int("event.id", id))

	event.ID = id
	return &event, nil
}

func (r *Repo) Get(ctx context.Context, id int) (_ *Event, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.events.get")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
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
		tracing.EndSpanWithErrCheck(span, err)
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
		params.Size, params.Size*(params.Page-1),
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

func (r *Repo) Count(ctx context.Context, params EventParams) (_ int, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.events.count")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	rows, err := r.db.Query(ctx, `
		SELECT COUNT(*) FROM gymstats_event
		WHERE ($1::text IS NULL OR type = $1)
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
