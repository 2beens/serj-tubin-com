package mcp

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SchemaRepo provides gymstats DB schema (information_schema) data.
type SchemaRepo interface {
	GetGymstatsColumns(ctx context.Context) ([]SchemaColumn, error)
}

// SchemaColumn represents one row from information_schema.columns for gymstats tables.
type SchemaColumn struct {
	TableSchema string
	TableName   string
	ColumnName  string
	DataType    string
	IsNullable  string
	ColumnDef   *string
}

var gymstatsTables = []string{"exercise", "exercise_type", "exercise_image", "gymstats_event"}

type poolSchemaRepo struct {
	pool *pgxpool.Pool
}

// NewPoolSchemaRepo returns a SchemaRepo that uses the given pool.
func NewPoolSchemaRepo(pool *pgxpool.Pool) SchemaRepo {
	return &poolSchemaRepo{pool: pool}
}

// GetGymstatsColumns returns column metadata for gymstats-related tables.
func (r *poolSchemaRepo) GetGymstatsColumns(ctx context.Context) ([]SchemaColumn, error) {
	query := `
		SELECT table_schema, table_name, column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = ANY($1)
		ORDER BY table_name, ordinal_position`
	rows, err := r.pool.Query(ctx, query, gymstatsTables)
	if err != nil {
		return nil, fmt.Errorf("query information_schema: %w", err)
	}
	defer rows.Close()

	var cols []SchemaColumn
	for rows.Next() {
		var c SchemaColumn
		var def *string
		if err := rows.Scan(&c.TableSchema, &c.TableName, &c.ColumnName, &c.DataType, &c.IsNullable, &def); err != nil {
			return nil, fmt.Errorf("scan column row: %w", err)
		}
		c.ColumnDef = def
		cols = append(cols, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating columns: %w", err)
	}

	return cols, nil
}
