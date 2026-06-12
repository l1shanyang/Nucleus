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

func setupWorkspaceService() (*service.WorkspaceService, *storetest.MockWorkspaceStore, *storetest.MockUserStore) {
	workspaces := storetest.NewMockWorkspaceStore()
	users := storetest.NewMockUserStore()
	svc := service.NewWorkspaceService(workspaces, users, fakeWorkspaceTxManager{})
	return svc, workspaces, users
}

func TestWorkspaceService_Create(t *testing.T) {
	svc, workspaces, _ := setupWorkspaceService()

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
			svc, workspaces, _ := setupWorkspaceService()

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
	svc, _, _ := setupWorkspaceService()

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

func TestWorkspaceService_MemberManagement(t *testing.T) {
	svc, _, users := setupWorkspaceService()
	users.Put(&store.User{ID: 2, Email: "agent@example.com", Name: "Agent"})

	workspace, err := svc.Create(context.Background(), service.CreateWorkspaceInput{UserID: 1, Name: "Support"})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	member, err := svc.AddMember(context.Background(), service.AddWorkspaceMemberInput{
		ActorUserID: 1,
		WorkspaceID: workspace.ID,
		Email:       " Agent@Example.com ",
		Role:        store.WorkspaceRoleAgent,
	})
	if err != nil {
		t.Fatalf("add member: %v", err)
	}
	if member.UserID != 2 || member.UserEmail != "agent@example.com" || member.Role != store.WorkspaceRoleAgent {
		t.Fatalf("member = %+v, want user 2 agent@example.com agent", member)
	}

	members, err := svc.ListMembers(context.Background(), 1, workspace.ID)
	if err != nil {
		t.Fatalf("list members: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("members = %d, want 2", len(members))
	}

	updated, err := svc.UpdateMemberRole(context.Background(), service.UpdateWorkspaceMemberRoleInput{
		ActorUserID: 1,
		WorkspaceID: workspace.ID,
		MemberID:    member.ID,
		Role:        store.WorkspaceRoleViewer,
	})
	if err != nil {
		t.Fatalf("update role: %v", err)
	}
	if updated.Role != store.WorkspaceRoleViewer {
		t.Fatalf("role = %q, want %q", updated.Role, store.WorkspaceRoleViewer)
	}

	if err := svc.RemoveMember(context.Background(), service.RemoveWorkspaceMemberInput{
		ActorUserID: 1,
		WorkspaceID: workspace.ID,
		MemberID:    member.ID,
	}); err != nil {
		t.Fatalf("remove member: %v", err)
	}

	members, err = svc.ListMembers(context.Background(), 1, workspace.ID)
	if err != nil {
		t.Fatalf("list members after remove: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("members after remove = %d, want 1", len(members))
	}
}

func TestWorkspaceService_MemberManagementRequiresOwnerOrAdmin(t *testing.T) {
	svc, _, users := setupWorkspaceService()
	users.Put(&store.User{ID: 2, Email: "agent@example.com", Name: "Agent"})
	users.Put(&store.User{ID: 3, Email: "viewer@example.com", Name: "Viewer"})

	workspace, err := svc.Create(context.Background(), service.CreateWorkspaceInput{UserID: 1, Name: "Support"})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := svc.AddMember(context.Background(), service.AddWorkspaceMemberInput{
		ActorUserID: 1,
		WorkspaceID: workspace.ID,
		Email:       "agent@example.com",
		Role:        store.WorkspaceRoleAgent,
	}); err != nil {
		t.Fatalf("add agent: %v", err)
	}

	_, err = svc.AddMember(context.Background(), service.AddWorkspaceMemberInput{
		ActorUserID: 2,
		WorkspaceID: workspace.ID,
		Email:       "viewer@example.com",
		Role:        store.WorkspaceRoleViewer,
	})
	assertWorkspaceServiceError(t, err, apperror.KindForbidden, "workspace member management requires owner or admin role")
}

func TestWorkspaceService_OwnerRoleCannotBeManaged(t *testing.T) {
	svc, workspaces, _ := setupWorkspaceService()

	workspace, err := svc.Create(context.Background(), service.CreateWorkspaceInput{UserID: 1, Name: "Support"})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	ownerMemberID := workspaces.CreatedMembers[0].ID

	_, err = svc.UpdateMemberRole(context.Background(), service.UpdateWorkspaceMemberRoleInput{
		ActorUserID: 1,
		WorkspaceID: workspace.ID,
		MemberID:    ownerMemberID,
		Role:        store.WorkspaceRoleAdmin,
	})
	assertWorkspaceServiceError(t, err, apperror.KindForbidden, "workspace owner role cannot be changed")

	err = svc.RemoveMember(context.Background(), service.RemoveWorkspaceMemberInput{
		ActorUserID: 1,
		WorkspaceID: workspace.ID,
		MemberID:    ownerMemberID,
	})
	assertWorkspaceServiceError(t, err, apperror.KindForbidden, "workspace owner cannot be removed")
}

func TestWorkspaceService_MemberRoleValidation(t *testing.T) {
	svc, _, users := setupWorkspaceService()
	users.Put(&store.User{ID: 2, Email: "owner@example.com", Name: "Owner"})

	workspace, err := svc.Create(context.Background(), service.CreateWorkspaceInput{UserID: 1, Name: "Support"})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	_, err = svc.AddMember(context.Background(), service.AddWorkspaceMemberInput{
		ActorUserID: 1,
		WorkspaceID: workspace.ID,
		Email:       "owner@example.com",
		Role:        store.WorkspaceRoleOwner,
	})
	assertWorkspaceServiceError(t, err, apperror.KindValidation, "role must be one of admin, agent, viewer")
}

func assertWorkspaceServiceError(t *testing.T, err error, wantKind apperror.Kind, wantMessage string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected %s error, got nil", wantKind)
	}
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected apperror.Error, got %T", err)
	}
	if appErr.Kind != wantKind {
		t.Fatalf("kind = %q, want %q", appErr.Kind, wantKind)
	}
	if appErr.Message != wantMessage {
		t.Fatalf("message = %q, want %q", appErr.Message, wantMessage)
	}
}
