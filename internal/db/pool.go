package db

import (
	"context"
	"fmt"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NewDBPoolParams struct {
	DBHost         string
	DBPort         string
	DBName         string
	TracingEnabled bool
}

func NewDBPool(ctx context.Context, params NewDBPoolParams) (*pgxpool.Pool, error) {
	connString := fmt.Sprintf(
		"postgres://postgres@%s:%s/%s",
		params.DBHost, params.DBPort, params.DBName,
	)
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	if params.TracingEnabled {
		poolConfig.ConnConfig.Tracer = otelpgx.NewTracer()
	}

	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	return db, nil
}
