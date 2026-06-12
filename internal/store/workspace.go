package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"nucleus/internal/db/sqlc"
)

type WorkspaceRole string

const (
	WorkspaceRoleOwner  WorkspaceRole = "owner"
	WorkspaceRoleAdmin  WorkspaceRole = "admin"
	WorkspaceRoleAgent  WorkspaceRole = "agent"
	WorkspaceRoleViewer WorkspaceRole = "viewer"
)

type Workspace struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WorkspaceMember struct {
	ID          int64         `json:"id"`
	WorkspaceID int64         `json:"workspace_id"`
	UserID      int64         `json:"user_id"`
	Role        WorkspaceRole `json:"role"`
	CreatedAt   time.Time     `json:"created_at"`
}

type WorkspaceMemberView struct {
	ID          int64         `json:"id"`
	WorkspaceID int64         `json:"workspace_id"`
	UserID      int64         `json:"user_id"`
	UserEmail   string        `json:"user_email"`
	UserName    string        `json:"user_name"`
	Role        WorkspaceRole `json:"role"`
	CreatedAt   time.Time     `json:"created_at"`
}

type WorkspaceView struct {
	ID        int64         `json:"id"`
	Name      string        `json:"name"`
	Role      WorkspaceRole `json:"role"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type CreateWorkspaceInput struct {
	Name      string
	CreatedBy int64
}

type CreateWorkspaceMemberInput struct {
	WorkspaceID int64
	UserID      int64
	Role        WorkspaceRole
}

type UpdateWorkspaceMemberRoleInput struct {
	WorkspaceID int64
	MemberID    int64
	Role        WorkspaceRole
}

type WorkspaceStore interface {
	Create(ctx context.Context, input CreateWorkspaceInput) (Workspace, error)
	CreateMember(ctx context.Context, input CreateWorkspaceMemberInput) (WorkspaceMember, error)
	ListMembers(ctx context.Context, workspaceID int64) ([]WorkspaceMemberView, error)
	GetMember(ctx context.Context, workspaceID, memberID int64) (WorkspaceMemberView, error)
	UpdateMemberRole(ctx context.Context, input UpdateWorkspaceMemberRoleInput) (WorkspaceMemberView, error)
	DeleteMember(ctx context.Context, workspaceID, memberID int64) error
	ListForUser(ctx context.Context, userID int64) ([]WorkspaceView, error)
	GetForUser(ctx context.Context, userID, workspaceID int64) (WorkspaceView, error)
	WithTx(tx pgx.Tx) WorkspaceStore
}

type workspaceStore struct {
	q *sqlc.Queries
}

func NewWorkspaceStore(q *sqlc.Queries) WorkspaceStore {
	return &workspaceStore{q: q}
}

func (s *workspaceStore) Create(ctx context.Context, input CreateWorkspaceInput) (Workspace, error) {
	row, err := s.q.CreateWorkspace(ctx, sqlc.CreateWorkspaceParams{
		Name:      input.Name,
		CreatedBy: input.CreatedBy,
	})
	if err != nil {
		return Workspace{}, mapDBError(err, "workspace")
	}
	return toWorkspace(&row), nil
}

func (s *workspaceStore) CreateMember(ctx context.Context, input CreateWorkspaceMemberInput) (WorkspaceMember, error) {
	row, err := s.q.CreateWorkspaceMember(ctx, sqlc.CreateWorkspaceMemberParams{
		WorkspaceID: input.WorkspaceID,
		UserID:      input.UserID,
		Role:        string(input.Role),
	})
	if err != nil {
		return WorkspaceMember{}, mapDBError(err, "workspace member")
	}
	return toWorkspaceMember(&row), nil
}

func (s *workspaceStore) ListMembers(ctx context.Context, workspaceID int64) ([]WorkspaceMemberView, error) {
	rows, err := s.q.ListWorkspaceMembers(ctx, workspaceID)
	if err != nil {
		return nil, mapDBError(err, "workspace member")
	}

	members := make([]WorkspaceMemberView, len(rows))
	for i := range rows {
		members[i] = toWorkspaceMemberViewFromListRow(&rows[i])
	}
	return members, nil
}

func (s *workspaceStore) GetMember(ctx context.Context, workspaceID, memberID int64) (WorkspaceMemberView, error) {
	row, err := s.q.GetWorkspaceMember(ctx, sqlc.GetWorkspaceMemberParams{
		WorkspaceID: workspaceID,
		ID:          memberID,
	})
	if err != nil {
		return WorkspaceMemberView{}, mapDBError(err, "workspace member")
	}
	return toWorkspaceMemberViewFromGetRow(&row), nil
}

func (s *workspaceStore) UpdateMemberRole(ctx context.Context, input UpdateWorkspaceMemberRoleInput) (WorkspaceMemberView, error) {
	row, err := s.q.UpdateWorkspaceMemberRole(ctx, sqlc.UpdateWorkspaceMemberRoleParams{
		WorkspaceID: input.WorkspaceID,
		ID:          input.MemberID,
		Role:        string(input.Role),
	})
	if err != nil {
		return WorkspaceMemberView{}, mapDBError(err, "workspace member")
	}
	return toWorkspaceMemberViewFromUpdateRow(&row), nil
}

func (s *workspaceStore) DeleteMember(ctx context.Context, workspaceID, memberID int64) error {
	rows, err := s.q.DeleteWorkspaceMember(ctx, sqlc.DeleteWorkspaceMemberParams{
		WorkspaceID: workspaceID,
		ID:          memberID,
	})
	if err != nil {
		return mapDBError(err, "workspace member")
	}
	if rows == 0 {
		return mapDBError(pgx.ErrNoRows, "workspace member")
	}
	return nil
}

func (s *workspaceStore) ListForUser(ctx context.Context, userID int64) ([]WorkspaceView, error) {
	rows, err := s.q.ListWorkspacesForUser(ctx, userID)
	if err != nil {
		return nil, mapDBError(err, "workspace")
	}

	workspaces := make([]WorkspaceView, len(rows))
	for i := range rows {
		workspaces[i] = toWorkspaceViewFromListRow(&rows[i])
	}
	return workspaces, nil
}

func (s *workspaceStore) GetForUser(ctx context.Context, userID, workspaceID int64) (WorkspaceView, error) {
	row, err := s.q.GetWorkspaceForUser(ctx, sqlc.GetWorkspaceForUserParams{
		UserID: userID,
		ID:     workspaceID,
	})
	if err != nil {
		return WorkspaceView{}, mapDBError(err, "workspace")
	}
	return toWorkspaceViewFromGetRow(&row), nil
}

func (s *workspaceStore) WithTx(tx pgx.Tx) WorkspaceStore {
	return &workspaceStore{q: s.q.WithTx(tx)}
}

func toWorkspace(row *sqlc.Workspace) Workspace {
	return Workspace{
		ID:        row.ID,
		Name:      row.Name,
		CreatedBy: row.CreatedBy,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func toWorkspaceMember(row *sqlc.WorkspaceMember) WorkspaceMember {
	return WorkspaceMember{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		UserID:      row.UserID,
		Role:        WorkspaceRole(row.Role),
		CreatedAt:   row.CreatedAt,
	}
}

func toWorkspaceMemberViewFromListRow(row *sqlc.ListWorkspaceMembersRow) WorkspaceMemberView {
	return WorkspaceMemberView{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		UserID:      row.UserID,
		UserEmail:   row.UserEmail,
		UserName:    row.UserName,
		Role:        WorkspaceRole(row.Role),
		CreatedAt:   row.CreatedAt,
	}
}

func toWorkspaceMemberViewFromGetRow(row *sqlc.GetWorkspaceMemberRow) WorkspaceMemberView {
	return WorkspaceMemberView{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		UserID:      row.UserID,
		UserEmail:   row.UserEmail,
		UserName:    row.UserName,
		Role:        WorkspaceRole(row.Role),
		CreatedAt:   row.CreatedAt,
	}
}

func toWorkspaceMemberViewFromUpdateRow(row *sqlc.UpdateWorkspaceMemberRoleRow) WorkspaceMemberView {
	return WorkspaceMemberView{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		UserID:      row.UserID,
		UserEmail:   row.UserEmail,
		UserName:    row.UserName,
		Role:        WorkspaceRole(row.Role),
		CreatedAt:   row.CreatedAt,
	}
}

func toWorkspaceViewFromListRow(row *sqlc.ListWorkspacesForUserRow) WorkspaceView {
	return WorkspaceView{
		ID:        row.ID,
		Name:      row.Name,
		Role:      WorkspaceRole(row.Role),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func toWorkspaceViewFromGetRow(row *sqlc.GetWorkspaceForUserRow) WorkspaceView {
	return WorkspaceView{
		ID:        row.ID,
		Name:      row.Name,
		Role:      WorkspaceRole(row.Role),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
