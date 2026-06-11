//go:build integration

package store_test

import (
	"context"
	"errors"
	"testing"

	"nucleus/internal/apperror"
	"nucleus/internal/db/dbtest"
	"nucleus/internal/db/sqlc"
	"nucleus/internal/store"
)

func TestUserStore_CreateAndGetByEmail_Integration(t *testing.T) {
	pool := dbtest.NewPool(t)
	dbtest.ApplyMigrations(t, pool)
	dbtest.TruncateTables(t, pool, "users")

	userStore := store.NewUserStore(sqlc.New(pool))

	created, err := userStore.Create(context.Background(), store.CreateUserInput{
		Email:        "user@example.com",
		Name:         "Alice",
		PasswordHash: "$2a$10$hashed-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	got, err := userStore.GetByEmail(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("get user by email: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("id = %d, want %d", got.ID, created.ID)
	}
	if got.PasswordHash != "$2a$10$hashed-password" {
		t.Fatalf("password hash = %q, want stored hash", got.PasswordHash)
	}

	_, err = userStore.Create(context.Background(), store.CreateUserInput{
		Email:        "user@example.com",
		Name:         "Alice",
		PasswordHash: "$2a$10$another-hash",
	})
	if err == nil {
		t.Fatal("expected duplicate email error, got nil")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected apperror.Error, got %T", err)
	}
	if appErr.Kind != apperror.KindConflict {
		t.Fatalf("kind = %q, want %q", appErr.Kind, apperror.KindConflict)
	}
}
