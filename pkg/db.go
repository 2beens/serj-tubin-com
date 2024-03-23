package pkg

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// https://www.postgresql.org/docs/8.2/errcodes-appendix.html

// IsUniqueViolationError checks if the error is a unique violation error
func IsUniqueViolationError(err error) bool {
	var pqErr *pgconn.PgError
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}

// IsForeignKeyViolationError checks if the error is a foreign key violation error
func IsForeignKeyViolationError(err error) bool {
	var pqErr *pgconn.PgError
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23503"
	}
	return false
}
