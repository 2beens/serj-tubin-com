package netlog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
)

type Visit struct {
	Id        int       `json:"id"`
	Title     string    `json:"title"`
	Source    string    `json:"source"`
	Device    string    `json:"device"`
	URL       string    `json:"url"`       // mandatory
	Timestamp time.Time `json:"timestamp"` // mandatory
}

var _ netlogRepo = (*Repo)(nil)

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) AddVisit(ctx context.Context, visit *Visit) (err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "netlogPsqlApi.add")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	span.SetAttributes(attribute.String("visit.source", visit.Source))
	span.SetAttributes(attribute.String("visit.device", visit.Device))

	if visit.URL == "" || visit.Timestamp.IsZero() {
		span.SetStatus(codes.Error, "visit url or timestamp empty")
		return errors.New("visit url or timestamp empty")
	}

	rows, err := r.db.Query(
		ctx,
		`INSERT INTO netlog.visit (title, source, device, url, timestamp) VALUES ($1, $2, $3, $4, $5) RETURNING id;`,
		visit.Title, visit.Source, visit.Device, visit.URL, visit.Timestamp,
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

	return fmt.Errorf("unexpected error, failed to insert visit: %+v", *visit)
}

func (r *Repo) GetAllVisits(ctx context.Context, fromTimestamp *time.Time) (_ []*Visit, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "netlogPsqlApi.all")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	if fromTimestamp != nil {
		span.SetAttributes(attribute.String("visit.from-time", fromTimestamp.String()))
	} else {
		span.SetAttributes(attribute.String("visit.from-time", "nil"))
	}

	var rows pgx.Rows
	if fromTimestamp != nil {
		rows, err = r.db.Query(
			ctx,
			`
			SELECT
				id, COALESCE(title, '') as title, COALESCE(source, '') as source, COALESCE(device, '') as device, url, timestamp
			FROM netlog.visit
			WHERE timestamp >= $1;`,
			fromTimestamp,
		)
	} else {
		rows, err = r.db.Query(
			ctx,
			`
			SELECT
				id, COALESCE(title, '') as title, COALESCE(source, '') as source, COALESCE(device, '') as device, url, timestamp
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

	visits := visitsFromRows(rows)
	span.SetAttributes(attribute.Int("found-visits", len(visits)))
	return visits, nil
}

func (r *Repo) GetVisits(ctx context.Context, keywords []string, field string, source string, limit int) (_ []*Visit, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "netlogPsqlApi.getVisits")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	span.SetAttributes(attribute.String("visit.source", source))
	span.SetAttributes(attribute.String("visit.field", field))
	span.SetAttributes(attribute.Int("limit", limit))

	sbQueryLike := getQueryWhereCondition(field, source, keywords)
	query := fmt.Sprintf(`
		SELECT
			id, COALESCE(title, ''), COALESCE(source, ''), COALESCE(device, '') as device, url, timestamp
		FROM netlog.visit
		%s
		ORDER BY id DESC
		LIMIT $1;
	`, sbQueryLike)

	rows, err := r.db.Query(
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

	visits := visitsFromRows(rows)
	span.SetAttributes(attribute.Int("found-visits", len(visits)))
	return visits, nil
}

func (r *Repo) CountAll(ctx context.Context) (int, error) {
	return r.Count(ctx, []string{}, "url", "all")
}

func (r *Repo) Count(ctx context.Context, keywords []string, field string, source string) (_ int, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "netlogPsqlApi.count")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	span.SetAttributes(attribute.String("visit.source", source))
	span.SetAttributes(attribute.String("visit.field", field))

	sbQueryLike := getQueryWhereCondition(field, source, keywords)
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM netlog.visit
		%s
		;
	`, sbQueryLike)

	rows, err := r.db.Query(
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

func (r *Repo) GetVisitsPage(ctx context.Context, keywords []string, field string, source string, page int, size int) (_ []*Visit, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "netlogPsqlApi.getVisitsPage")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	span.SetAttributes(attribute.String("visit.source", source))
	span.SetAttributes(attribute.String("visit.field", field))
	span.SetAttributes(attribute.Int("page", page))
	span.SetAttributes(attribute.Int("size", size))

	limit := size
	offset := (page - 1) * size
	allVisitsCount, err := r.CountAll(ctx)
	if err != nil {
		return nil, err
	}

	if allVisitsCount <= limit {
		return r.GetVisits(ctx, []string{}, field, source, size)
	}

	if allVisitsCount-offset < limit {
		offset = allVisitsCount - limit
	}

	log.Tracef("getting visits, all count %d, limit %d, offset %d", allVisitsCount, limit, offset)

	sbQueryLike := getQueryWhereCondition(field, source, keywords)
	query := fmt.Sprintf(`
		SELECT
			id, COALESCE(title, ''), COALESCE(source, ''), COALESCE(device, '') as device, url, timestamp
		FROM netlog.visit
		%s
		ORDER BY timestamp DESC
		LIMIT $1
		OFFSET $2;
	`, sbQueryLike)

	rows, err := r.db.Query(
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

	visits := visitsFromRows(rows)
	span.SetAttributes(attribute.Int("found-visits", len(visits)))
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

func visitsFromRows(rows pgx.Rows) []*Visit {
	var visits []*Visit
	for rows.Next() {
		var id int
		var title string
		var source string
		var device string
		var url string
		var timestamp time.Time
		if err := rows.Scan(&id, &title, &source, &device, &url, &timestamp); err != nil {
			return nil
		}
		visits = append(visits, &Visit{
			Id:        id,
			Title:     title,
			Source:    source,
			Device:    device,
			URL:       url,
			Timestamp: timestamp,
		})
	}
	return visits
}
