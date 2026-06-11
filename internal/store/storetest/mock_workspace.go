package storetest

import (
	"context"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	"nucleus/internal/apperror"
	"nucleus/internal/store"
)

type MockWorkspaceStore struct {
	workspaces map[int64]store.Workspace
	members    []store.WorkspaceMember
	nextID     int64
	nextMember int64

	CreatedWorkspaces []store.Workspace
	CreatedMembers    []store.WorkspaceMember

	CreateErr       error
	CreateMemberErr error
	ListErr         error
	GetErr          error
}

func NewMockWorkspaceStore() *MockWorkspaceStore {
	return &MockWorkspaceStore{
		workspaces: make(map[int64]store.Workspace),
		nextID:     1,
		nextMember: 1,
	}
}

func (m *MockWorkspaceStore) Create(_ context.Context, input store.CreateWorkspaceInput) (store.Workspace, error) {
	if m.CreateErr != nil {
		return store.Workspace{}, m.CreateErr
	}

	now := time.Now().Truncate(time.Second)
	workspace := store.Workspace{
		ID:        m.nextID,
		Name:      input.Name,
		CreatedBy: input.CreatedBy,
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.nextID++
	m.workspaces[workspace.ID] = workspace
	m.CreatedWorkspaces = append(m.CreatedWorkspaces, workspace)
	return workspace, nil
}

func (m *MockWorkspaceStore) CreateMember(_ context.Context, input store.CreateWorkspaceMemberInput) (store.WorkspaceMember, error) {
	if m.CreateMemberErr != nil {
		return store.WorkspaceMember{}, m.CreateMemberErr
	}
	if _, exists := m.workspaces[input.WorkspaceID]; !exists {
		return store.WorkspaceMember{}, apperror.NotFound("workspace not found")
	}

	member := store.WorkspaceMember{
		ID:          m.nextMember,
		WorkspaceID: input.WorkspaceID,
		UserID:      input.UserID,
		Role:        input.Role,
		CreatedAt:   time.Now().Truncate(time.Second),
	}
	m.nextMember++
	m.members = append(m.members, member)
	m.CreatedMembers = append(m.CreatedMembers, member)
	return member, nil
}

func (m *MockWorkspaceStore) ListForUser(_ context.Context, userID int64) ([]store.WorkspaceView, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}

	workspaces := make([]store.WorkspaceView, 0)
	for _, member := range m.members {
		if member.UserID != userID {
			continue
		}
		workspace, exists := m.workspaces[member.WorkspaceID]
		if !exists {
			continue
		}
		workspaces = append(workspaces, store.WorkspaceView{
			ID:        workspace.ID,
			Name:      workspace.Name,
			Role:      member.Role,
			CreatedAt: workspace.CreatedAt,
			UpdatedAt: workspace.UpdatedAt,
		})
	}

	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].ID > workspaces[j].ID
	})
	return workspaces, nil
}

func (m *MockWorkspaceStore) GetForUser(_ context.Context, userID, workspaceID int64) (store.WorkspaceView, error) {
	if m.GetErr != nil {
		return store.WorkspaceView{}, m.GetErr
	}

	for _, member := range m.members {
		if member.UserID != userID || member.WorkspaceID != workspaceID {
			continue
		}
		workspace, exists := m.workspaces[workspaceID]
		if !exists {
			break
		}
		return store.WorkspaceView{
			ID:        workspace.ID,
			Name:      workspace.Name,
			Role:      member.Role,
			CreatedAt: workspace.CreatedAt,
			UpdatedAt: workspace.UpdatedAt,
		}, nil
	}

	return store.WorkspaceView{}, apperror.NotFound("workspace not found")
}

func (m *MockWorkspaceStore) WithTx(pgx.Tx) store.WorkspaceStore {
	return m
}
