package store

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"nucleus/internal/apperror"
)

const (
	postgresUniqueViolation     = "23505"
	postgresForeignKeyViolation = "23503"
)

func mapDBError(err error, resource string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NotFound(resource+" not found", err)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case postgresUniqueViolation:
			return apperror.Conflict(resource+" already exists", err)
		case postgresForeignKeyViolation:
			return apperror.Conflict("related resource not found", err)
		}
	}

	return err
}
