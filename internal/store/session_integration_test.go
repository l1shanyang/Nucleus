//go:build integration

package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"nucleus/internal/apperror"
	"nucleus/internal/db/dbtest"
	"nucleus/internal/db/sqlc"
	"nucleus/internal/store"
)

func TestSessionStore_CreateAndGetUserByTokenHash_Integration(t *testing.T) {
	pool := dbtest.NewPool(t)
	dbtest.ApplyMigrations(t, pool)
	dbtest.TruncateTables(t, pool, "auth_sessions", "users")

	queries := sqlc.New(pool)
	userStore := store.NewUserStore(queries)
	sessionStore := store.NewSessionStore(queries)

	user, err := userStore.Create(context.Background(), store.CreateUserInput{
		Email:        "user@example.com",
		Name:         "Alice",
		PasswordHash: "$2a$10$hashed-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if _, err := sessionStore.Create(context.Background(), store.CreateSessionInput{
		UserID:    user.ID,
		TokenHash: "valid-token-hash",
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	got, err := sessionStore.GetUserByTokenHash(context.Background(), "valid-token-hash")
	if err != nil {
		t.Fatalf("get user by token hash: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("user id = %d, want %d", got.ID, user.ID)
	}

	if _, err := sessionStore.Create(context.Background(), store.CreateSessionInput{
		UserID:    user.ID,
		TokenHash: "expired-token-hash",
		ExpiresAt: time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("create expired session: %v", err)
	}

	_, err = sessionStore.GetUserByTokenHash(context.Background(), "expired-token-hash")
	if err == nil {
		t.Fatal("expected expired session error, got nil")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected apperror.Error, got %T", err)
	}
	if appErr.Kind != apperror.KindNotFound {
		t.Fatalf("kind = %q, want %q", appErr.Kind, apperror.KindNotFound)
	}
}
