package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"nucleus/internal/apperror"
	"nucleus/internal/service"
	"nucleus/internal/store"
	"nucleus/internal/store/storetest"
)

type fakeWorkspaceTxManager struct{}

func (fakeWorkspaceTxManager) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return fn(nil)
}

func setupWorkspaceService() (*service.WorkspaceService, *storetest.MockWorkspaceStore) {
	workspaces := storetest.NewMockWorkspaceStore()
	svc := service.NewWorkspaceService(workspaces, fakeWorkspaceTxManager{})
	return svc, workspaces
}

func TestWorkspaceService_Create(t *testing.T) {
	svc, workspaces := setupWorkspaceService()

	workspace, err := svc.Create(context.Background(), service.CreateWorkspaceInput{
		UserID: 1,
		Name:   "  Support Team  ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if workspace.ID == 0 {
		t.Fatal("expected non-zero workspace ID")
	}
	if workspace.Name != "Support Team" {
		t.Fatalf("name = %q, want Support Team", workspace.Name)
	}
	if workspace.Role != store.WorkspaceRoleOwner {
		t.Fatalf("role = %q, want %q", workspace.Role, store.WorkspaceRoleOwner)
	}
	if len(workspaces.CreatedWorkspaces) != 1 {
		t.Fatalf("created workspaces = %d, want 1", len(workspaces.CreatedWorkspaces))
	}
	if len(workspaces.CreatedMembers) != 1 {
		t.Fatalf("created members = %d, want 1", len(workspaces.CreatedMembers))
	}
	member := workspaces.CreatedMembers[0]
	if member.UserID != 1 || member.WorkspaceID != workspace.ID || member.Role != store.WorkspaceRoleOwner {
		t.Fatalf("owner member = %+v, want user 1 workspace %d owner", member, workspace.ID)
	}
}

func TestWorkspaceService_Create_Validation(t *testing.T) {
	tests := []struct {
		name     string
		input    service.CreateWorkspaceInput
		wantErr  string
		wantKind apperror.Kind
	}{
		{
			name:     "未认证",
			input:    service.CreateWorkspaceInput{UserID: 0, Name: "Support"},
			wantErr:  "authentication is required",
			wantKind: apperror.KindUnauthorized,
		},
		{
			name:     "名称为空",
			input:    service.CreateWorkspaceInput{UserID: 1, Name: " "},
			wantErr:  "name is required",
			wantKind: apperror.KindValidation,
		},
		{
			name:     "名称过长",
			input:    service.CreateWorkspaceInput{UserID: 1, Name: strings.Repeat("a", 101)},
			wantErr:  "name must be at most 100 characters",
			wantKind: apperror.KindValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, workspaces := setupWorkspaceService()

			_, err := svc.Create(context.Background(), tt.input)
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}

			var appErr *apperror.Error
			if !errors.As(err, &appErr) {
				t.Fatalf("expected apperror.Error, got %T", err)
			}
			if appErr.Kind != tt.wantKind {
				t.Fatalf("kind = %q, want %q", appErr.Kind, tt.wantKind)
			}
			if appErr.Message != tt.wantErr {
				t.Fatalf("message = %q, want %q", appErr.Message, tt.wantErr)
			}
			if len(workspaces.CreatedWorkspaces) != 0 {
				t.Fatal("invalid input should not create workspace")
			}
		})
	}
}

func TestWorkspaceService_ListAndGet(t *testing.T) {
	svc, _ := setupWorkspaceService()

	first, err := svc.Create(context.Background(), service.CreateWorkspaceInput{UserID: 1, Name: "First"})
	if err != nil {
		t.Fatalf("create first workspace: %v", err)
	}
	second, err := svc.Create(context.Background(), service.CreateWorkspaceInput{UserID: 1, Name: "Second"})
	if err != nil {
		t.Fatalf("create second workspace: %v", err)
	}

	workspaces, err := svc.List(context.Background(), 1)
	if err != nil {
		t.Fatalf("list workspaces: %v", err)
	}
	if len(workspaces) != 2 {
		t.Fatalf("got %d workspaces, want 2", len(workspaces))
	}
	if workspaces[0].ID != second.ID || workspaces[1].ID != first.ID {
		t.Fatalf("workspaces are not ordered by id desc: got %d, %d", workspaces[0].ID, workspaces[1].ID)
	}

	got, err := svc.Get(context.Background(), 1, first.ID)
	if err != nil {
		t.Fatalf("get workspace: %v", err)
	}
	if got.ID != first.ID {
		t.Fatalf("id = %d, want %d", got.ID, first.ID)
	}

	_, err = svc.Get(context.Background(), 2, first.ID)
	if err == nil {
		t.Fatal("expected not found for non-member user, got nil")
	}
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected apperror.Error, got %T", err)
	}
	if appErr.Kind != apperror.KindNotFound {
		t.Fatalf("kind = %q, want %q", appErr.Kind, apperror.KindNotFound)
	}
}
