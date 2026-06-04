package store

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"nucleus/internal/apperror"
)

func TestMapDBError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantKind    apperror.Kind
		wantMessage string
	}{
		{
			name:        "no rows",
			err:         pgx.ErrNoRows,
			wantKind:    apperror.KindNotFound,
			wantMessage: "note not found",
		},
		{
			name:        "unique violation",
			err:         &pgconn.PgError{Code: postgresUniqueViolation},
			wantKind:    apperror.KindConflict,
			wantMessage: "note already exists",
		},
		{
			name:        "foreign key violation",
			err:         &pgconn.PgError{Code: postgresForeignKeyViolation},
			wantKind:    apperror.KindConflict,
			wantMessage: "related resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapDBError(tt.err, "note")

			var appErr *apperror.Error
			if !errors.As(got, &appErr) {
				t.Fatalf("expected apperror.Error, got %T", got)
			}
			if appErr.Kind != tt.wantKind {
				t.Fatalf("kind = %q, want %q", appErr.Kind, tt.wantKind)
			}
			if appErr.Message != tt.wantMessage {
				t.Fatalf("message = %q, want %q", appErr.Message, tt.wantMessage)
			}
			if !errors.Is(got, tt.err) {
				t.Fatal("expected mapped error to wrap original database error")
			}
		})
	}
}

func TestMapDBErrorKeepsUnknownError(t *testing.T) {
	cause := errors.New("connection reset")

	got := mapDBError(cause, "note")

	if !errors.Is(got, cause) {
		t.Fatalf("error = %v, want original cause", got)
	}
}
