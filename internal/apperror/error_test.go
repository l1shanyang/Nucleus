package apperror_test

import (
	"errors"
	"testing"

	"nucleus/internal/apperror"
)

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("database is unavailable")
	err := apperror.Internal("failed to create note", cause)

	if !errors.Is(err, cause) {
		t.Fatal("expected application error to unwrap cause")
	}
	if err.Error() != "failed to create note: database is unavailable" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestConstructors(t *testing.T) {
	tests := []struct {
		name string
		err  *apperror.Error
		kind apperror.Kind
		code string
	}{
		{"validation", apperror.Validation("invalid input"), apperror.KindValidation, "BAD_REQUEST"},
		{"not found", apperror.NotFound("missing"), apperror.KindNotFound, "NOT_FOUND"},
		{"conflict", apperror.Conflict("conflict"), apperror.KindConflict, "CONFLICT"},
		{"unauthorized", apperror.Unauthorized("unauthorized"), apperror.KindUnauthorized, "UNAUTHORIZED"},
		{"forbidden", apperror.Forbidden("forbidden"), apperror.KindForbidden, "FORBIDDEN"},
		{"internal", apperror.Internal("internal", nil), apperror.KindInternal, "INTERNAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Kind != tt.kind {
				t.Fatalf("kind = %q, want %q", tt.err.Kind, tt.kind)
			}
			if tt.err.Code != tt.code {
				t.Fatalf("code = %q, want %q", tt.err.Code, tt.code)
			}
		})
	}
}
