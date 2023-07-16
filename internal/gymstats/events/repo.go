package events

import (
	"context"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/codes"
)

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
			tx.Rollback(ctx)
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
