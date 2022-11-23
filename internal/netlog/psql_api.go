package netlog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
)

var _ Api = (*PsqlApi)(nil)

type PsqlApi struct {
	db *pgxpool.Pool
}

func NewNetlogPsqlApi(
	ctx context.Context,
	dbHost, dbPort, dbName string,
	tracer *tracing.PgxOtelTracer,
) (*PsqlApi, error) {
	connString := fmt.Sprintf("postgres://postgres@%s:%s/%s", dbHost, dbPort, dbName)
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse netlog db config: %w", err)
	}

	poolConfig.ConnConfig.Tracer = tracer

	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	log.Debugf("netlog api connected to: %s", connString)

	return &PsqlApi{
		db: db,
	}, nil
}

func (api *PsqlApi) CloseDB() {
	if api.db != nil {
		api.db.Close()
	}
}

func (api *PsqlApi) AddVisit(ctx context.Context, visit *Visit) error {
	ctx, span := tracing.GlobalTracer.Start(ctx, "netlogPsqlApi.add")
	span.SetAttributes(attribute.String("source", visit.Source))
	defer span.End()

	if visit.URL == "" || visit.Timestamp.IsZero() {
		return errors.New("visit url or timestamp empty")
	}

	rows, err := api.db.Query(
		ctx,
		`INSERT INTO netlog.visit (title, source, url, timestamp) VALUES ($1, $2, $3, $4) RETURNING id;`,
		visit.Title, visit.Source, visit.URL, visit.Timestamp,
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

func (api *PsqlApi) GetAllVisits(ctx context.Context, fromTimestamp *time.Time) ([]*Visit, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "netlogPsqlApi.all")
	if fromTimestamp != nil {
		span.SetAttributes(attribute.String("from-time", fromTimestamp.String()))
	} else {
		span.SetAttributes(attribute.String("from-time", "nil"))
	}
	defer span.End()

	var rows pgx.Rows
	var err error
	if fromTimestamp != nil {
		rows, err = api.db.Query(
			ctx,
			`
			SELECT
				id, COALESCE(title, '') as title, COALESCE(source, '') as source, url, timestamp
			FROM netlog.visit
			WHERE timestamp >= $1;`,
			fromTimestamp,
		)
	} else {
		rows, err = api.db.Query(
			ctx,
			`
			SELECT
				id, COALESCE(title, '') as title, COALESCE(source, '') as source, url, timestamp
			FROM netlog.visit;`,
		)
	}
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

	span.SetAttributes(attribute.Int("found-visits", len(visits)))

	return visits, nil
}

func (api *PsqlApi) GetVisits(ctx context.Context, keywords []string, field string, source string, limit int) ([]*Visit, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "netlogPsqlApi.getVisits")
	span.SetAttributes(attribute.String("source", source))
	span.SetAttributes(attribute.String("field", field))
	span.SetAttributes(attribute.Int("limit", limit))
	defer span.End()

	sbQueryLike := getQueryWhereCondition(field, source, keywords)
	query := fmt.Sprintf(`
		SELECT
			id, COALESCE(title, ''), COALESCE(source, ''), url, timestamp
		FROM netlog.visit
		%s
		ORDER BY id DESC
		LIMIT $1;
	`, sbQueryLike)

	rows, err := api.db.Query(
		ctx,
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

func (api *PsqlApi) CountAll(ctx context.Context) (int, error) {
	return api.Count(ctx, []string{}, "url", "all")
}

func (api *PsqlApi) Count(ctx context.Context, keywords []string, field string, source string) (int, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "netlogPsqlApi.count")
	span.SetAttributes(attribute.String("source", source))
	span.SetAttributes(attribute.String("field", field))
	defer span.End()

	sbQueryLike := getQueryWhereCondition(field, source, keywords)
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM netlog.visit
		%s
		;
	`, sbQueryLike)

	rows, err := api.db.Query(
		ctx,
		query,
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

func (api *PsqlApi) GetVisitsPage(ctx context.Context, keywords []string, field string, source string, page int, size int) ([]*Visit, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "netlogPsqlApi.getVisitsPage")
	span.SetAttributes(attribute.String("source", source))
	span.SetAttributes(attribute.String("field", field))
	span.SetAttributes(attribute.Int("page", page))
	span.SetAttributes(attribute.Int("size", size))
	defer span.End()

	limit := size
	offset := (page - 1) * size
	allVisitsCount, err := api.CountAll(ctx)
	if err != nil {
		return nil, err
	}

	if allVisitsCount <= limit {
		return api.GetVisits(ctx, []string{}, field, source, size)
	}

	if allVisitsCount-offset < limit {
		offset = allVisitsCount - limit
	}

	log.Tracef("getting visits, all count %d, limit %d, offset %d", allVisitsCount, limit, offset)

	sbQueryLike := getQueryWhereCondition(field, source, keywords)
	query := fmt.Sprintf(`
		SELECT
			id, COALESCE(title, ''), COALESCE(source, ''), url, timestamp
		FROM netlog.visit
		%s
		ORDER BY timestamp DESC
		LIMIT $1
		OFFSET $2;
	`, sbQueryLike)

	rows, err := api.db.Query(
		ctx,
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

// getQueryWhereCondition will make a SQL WHERE condition
// keywords starting with "-" will be filtered out with `url NOT LIKE ...`
// column - the name of the column to which the "like" is applied for
// source - the source of the netlog visit
func getQueryWhereCondition(column, source string, keywords []string) string {
	var sbQueryLike strings.Builder
	if len(keywords) > 0 {
		sbQueryLike.WriteString("WHERE ")
		for i, word := range keywords {
			if strings.HasPrefix(word, "-") {
				word = strings.TrimPrefix(word, "-")
				sbQueryLike.WriteString(fmt.Sprintf("%s NOT LIKE '%%%s%%' ", column, word))
			} else {
				sbQueryLike.WriteString(fmt.Sprintf("%s LIKE '%%%s%%' ", column, word))
			}
			if i < len(keywords)-1 {
				sbQueryLike.WriteString("AND ")
			}
		}
	}

	if source != "all" && len(keywords) == 0 {
		sbQueryLike.WriteString(fmt.Sprintf("WHERE source = '%s'", source))
	} else if source != "all" {
		sbQueryLike.WriteString(fmt.Sprintf("AND source = '%s'", source))
	}

	return sbQueryLike.String()
}
