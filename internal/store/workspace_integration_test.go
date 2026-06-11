//go:build integration

package store_test

import (
	"context"
	"testing"

	"nucleus/internal/db/dbtest"
	"nucleus/internal/db/sqlc"
	"nucleus/internal/store"
)

func TestWorkspaceStore_CreateListAndGet_Integration(t *testing.T) {
	pool := dbtest.NewPool(t)
	dbtest.ApplyMigrations(t, pool)
	dbtest.TruncateTables(t, pool, "workspace_members", "workspaces", "users")

	queries := sqlc.New(pool)
	userStore := store.NewUserStore(queries)
	workspaceStore := store.NewWorkspaceStore(queries)

	user, err := userStore.Create(context.Background(), store.CreateUserInput{
		Email:        "user@example.com",
		Name:         "Alice",
		PasswordHash: "$2a$10$hashed-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	workspace, err := workspaceStore.Create(context.Background(), store.CreateWorkspaceInput{
		Name:      "Support",
		CreatedBy: user.ID,
	})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := workspaceStore.CreateMember(context.Background(), store.CreateWorkspaceMemberInput{
		WorkspaceID: workspace.ID,
		UserID:      user.ID,
		Role:        store.WorkspaceRoleOwner,
	}); err != nil {
		t.Fatalf("create workspace member: %v", err)
	}

	workspaces, err := workspaceStore.ListForUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("list workspaces: %v", err)
	}
	if len(workspaces) != 1 {
		t.Fatalf("got %d workspaces, want 1", len(workspaces))
	}
	if workspaces[0].Role != store.WorkspaceRoleOwner {
		t.Fatalf("role = %q, want %q", workspaces[0].Role, store.WorkspaceRoleOwner)
	}

	got, err := workspaceStore.GetForUser(context.Background(), user.ID, workspace.ID)
	if err != nil {
		t.Fatalf("get workspace: %v", err)
	}
	if got.ID != workspace.ID {
		t.Fatalf("id = %d, want %d", got.ID, workspace.ID)
	}
}
