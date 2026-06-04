//go:build integration

package store_test

import (
	"context"
	"testing"

	"nucleus/internal/db/dbtest"
	"nucleus/internal/db/sqlc"
	"nucleus/internal/store"
)

func TestNoteStore_CreateAndList_Integration(t *testing.T) {
	pool := dbtest.NewPool(t)
	dbtest.ApplyMigrations(t, pool)
	dbtest.TruncateTables(t, pool, "notes")

	noteStore := store.NewNoteStore(sqlc.New(pool))

	first, err := noteStore.Create(context.Background(), "First", "Body")
	if err != nil {
		t.Fatalf("create first note: %v", err)
	}
	second, err := noteStore.Create(context.Background(), "Second", "Body")
	if err != nil {
		t.Fatalf("create second note: %v", err)
	}

	notes, err := noteStore.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}

	if len(notes) != 2 {
		t.Fatalf("got %d notes, want 2", len(notes))
	}
	if notes[0].ID != second.ID || notes[1].ID != first.ID {
		t.Fatalf("notes are not ordered by id desc: got ids %d, %d", notes[0].ID, notes[1].ID)
	}
}
