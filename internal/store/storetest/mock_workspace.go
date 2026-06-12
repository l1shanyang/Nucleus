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

	CreateErr           error
	CreateMemberErr     error
	ListMembersErr      error
	GetMemberErr        error
	UpdateMemberRoleErr error
	DeleteMemberErr     error
	ListErr             error
	GetErr              error
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
	for _, member := range m.members {
		if member.WorkspaceID == input.WorkspaceID && member.UserID == input.UserID {
			return store.WorkspaceMember{}, apperror.Conflict("workspace member already exists")
		}
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

func (m *MockWorkspaceStore) ListMembers(_ context.Context, workspaceID int64) ([]store.WorkspaceMemberView, error) {
	if m.ListMembersErr != nil {
		return nil, m.ListMembersErr
	}

	members := make([]store.WorkspaceMemberView, 0)
	for _, member := range m.members {
		if member.WorkspaceID != workspaceID {
			continue
		}
		members = append(members, m.toMemberView(member))
	}

	sort.Slice(members, func(i, j int) bool {
		return members[i].ID < members[j].ID
	})
	return members, nil
}

func (m *MockWorkspaceStore) GetMember(_ context.Context, workspaceID, memberID int64) (store.WorkspaceMemberView, error) {
	if m.GetMemberErr != nil {
		return store.WorkspaceMemberView{}, m.GetMemberErr
	}

	for _, member := range m.members {
		if member.WorkspaceID == workspaceID && member.ID == memberID {
			return m.toMemberView(member), nil
		}
	}
	return store.WorkspaceMemberView{}, apperror.NotFound("workspace member not found")
}

func (m *MockWorkspaceStore) UpdateMemberRole(_ context.Context, input store.UpdateWorkspaceMemberRoleInput) (store.WorkspaceMemberView, error) {
	if m.UpdateMemberRoleErr != nil {
		return store.WorkspaceMemberView{}, m.UpdateMemberRoleErr
	}

	for i := range m.members {
		if m.members[i].WorkspaceID == input.WorkspaceID && m.members[i].ID == input.MemberID {
			m.members[i].Role = input.Role
			return m.toMemberView(m.members[i]), nil
		}
	}
	return store.WorkspaceMemberView{}, apperror.NotFound("workspace member not found")
}

func (m *MockWorkspaceStore) DeleteMember(_ context.Context, workspaceID, memberID int64) error {
	if m.DeleteMemberErr != nil {
		return m.DeleteMemberErr
	}

	for i := range m.members {
		if m.members[i].WorkspaceID == workspaceID && m.members[i].ID == memberID {
			m.members = append(m.members[:i], m.members[i+1:]...)
			return nil
		}
	}
	return apperror.NotFound("workspace member not found")
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

func (m *MockWorkspaceStore) toMemberView(member store.WorkspaceMember) store.WorkspaceMemberView {
	return store.WorkspaceMemberView{
		ID:          member.ID,
		WorkspaceID: member.WorkspaceID,
		UserID:      member.UserID,
		UserEmail:   "",
		UserName:    "",
		Role:        member.Role,
		CreatedAt:   member.CreatedAt,
	}
}

func (m *MockWorkspaceStore) WithTx(pgx.Tx) store.WorkspaceStore {
	return m
}
