package service

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"nucleus/internal/apperror"
	"nucleus/internal/store"
)

const maxWorkspaceNameLength = 100

type WorkspaceTxManager interface {
	WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error
}

type WorkspaceService struct {
	workspaces store.WorkspaceStore
	users      store.UserStore
	tx         WorkspaceTxManager
}

func NewWorkspaceService(workspaces store.WorkspaceStore, users store.UserStore, tx WorkspaceTxManager) *WorkspaceService {
	return &WorkspaceService{workspaces: workspaces, users: users, tx: tx}
}

type CreateWorkspaceInput struct {
	UserID int64
	Name   string
}

type AddWorkspaceMemberInput struct {
	ActorUserID int64
	WorkspaceID int64
	Email       string
	Role        store.WorkspaceRole
}

type UpdateWorkspaceMemberRoleInput struct {
	ActorUserID int64
	WorkspaceID int64
	MemberID    int64
	Role        store.WorkspaceRole
}

type RemoveWorkspaceMemberInput struct {
	ActorUserID int64
	WorkspaceID int64
	MemberID    int64
}

func (s *WorkspaceService) Create(ctx context.Context, input CreateWorkspaceInput) (store.WorkspaceView, error) {
	name := strings.TrimSpace(input.Name)
	if err := validateCreateWorkspaceInput(input.UserID, name); err != nil {
		return store.WorkspaceView{}, err
	}

	var result store.WorkspaceView
	err := s.tx.WithTx(ctx, func(tx pgx.Tx) error {
		workspaceStore := s.workspaces.WithTx(tx)

		workspace, err := workspaceStore.Create(ctx, store.CreateWorkspaceInput{
			Name:      name,
			CreatedBy: input.UserID,
		})
		if err != nil {
			return err
		}

		if _, err := workspaceStore.CreateMember(ctx, store.CreateWorkspaceMemberInput{
			WorkspaceID: workspace.ID,
			UserID:      input.UserID,
			Role:        store.WorkspaceRoleOwner,
		}); err != nil {
			return err
		}

		result = store.WorkspaceView{
			ID:        workspace.ID,
			Name:      workspace.Name,
			Role:      store.WorkspaceRoleOwner,
			CreatedAt: workspace.CreatedAt,
			UpdatedAt: workspace.UpdatedAt,
		}
		return nil
	})
	if err != nil {
		return store.WorkspaceView{}, wrapWorkspaceError("failed to create workspace", err)
	}

	return result, nil
}

func (s *WorkspaceService) List(ctx context.Context, userID int64) ([]store.WorkspaceView, error) {
	if userID <= 0 {
		return nil, apperror.Unauthorized("authentication is required")
	}

	workspaces, err := s.workspaces.ListForUser(ctx, userID)
	if err != nil {
		return nil, wrapWorkspaceError("failed to list workspaces", err)
	}
	return workspaces, nil
}

func (s *WorkspaceService) Get(ctx context.Context, userID, workspaceID int64) (store.WorkspaceView, error) {
	if userID <= 0 {
		return store.WorkspaceView{}, apperror.Unauthorized("authentication is required")
	}
	if workspaceID <= 0 {
		return store.WorkspaceView{}, apperror.Validation("workspace id must be positive")
	}

	workspace, err := s.workspaces.GetForUser(ctx, userID, workspaceID)
	if err != nil {
		return store.WorkspaceView{}, wrapWorkspaceError("failed to get workspace", err)
	}
	return workspace, nil
}

func (s *WorkspaceService) AddMember(ctx context.Context, input AddWorkspaceMemberInput) (store.WorkspaceMemberView, error) {
	email := normalizeEmail(input.Email)
	if err := validateAddWorkspaceMemberInput(input, email); err != nil {
		return store.WorkspaceMemberView{}, err
	}
	if err := s.ensureCanManageMembers(ctx, input.ActorUserID, input.WorkspaceID); err != nil {
		return store.WorkspaceMemberView{}, err
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return store.WorkspaceMemberView{}, wrapWorkspaceError("failed to find workspace member user", err)
	}

	member, err := s.workspaces.CreateMember(ctx, store.CreateWorkspaceMemberInput{
		WorkspaceID: input.WorkspaceID,
		UserID:      user.ID,
		Role:        input.Role,
	})
	if err != nil {
		return store.WorkspaceMemberView{}, wrapWorkspaceError("failed to add workspace member", err)
	}

	return store.WorkspaceMemberView{
		ID:          member.ID,
		WorkspaceID: member.WorkspaceID,
		UserID:      member.UserID,
		UserEmail:   user.Email,
		UserName:    user.Name,
		Role:        member.Role,
		CreatedAt:   member.CreatedAt,
	}, nil
}

func (s *WorkspaceService) ListMembers(ctx context.Context, userID, workspaceID int64) ([]store.WorkspaceMemberView, error) {
	if err := validateWorkspaceAccessInput(userID, workspaceID); err != nil {
		return nil, err
	}
	if _, err := s.workspaceForUser(ctx, userID, workspaceID); err != nil {
		return nil, err
	}

	members, err := s.workspaces.ListMembers(ctx, workspaceID)
	if err != nil {
		return nil, wrapWorkspaceError("failed to list workspace members", err)
	}
	return members, nil
}

func (s *WorkspaceService) UpdateMemberRole(ctx context.Context, input UpdateWorkspaceMemberRoleInput) (store.WorkspaceMemberView, error) {
	if err := validateUpdateWorkspaceMemberRoleInput(input); err != nil {
		return store.WorkspaceMemberView{}, err
	}
	if err := s.ensureCanManageMembers(ctx, input.ActorUserID, input.WorkspaceID); err != nil {
		return store.WorkspaceMemberView{}, err
	}

	member, err := s.workspaces.GetMember(ctx, input.WorkspaceID, input.MemberID)
	if err != nil {
		return store.WorkspaceMemberView{}, wrapWorkspaceError("failed to get workspace member", err)
	}
	if member.Role == store.WorkspaceRoleOwner {
		return store.WorkspaceMemberView{}, apperror.Forbidden("workspace owner role cannot be changed")
	}

	updated, err := s.workspaces.UpdateMemberRole(ctx, store.UpdateWorkspaceMemberRoleInput{
		WorkspaceID: input.WorkspaceID,
		MemberID:    input.MemberID,
		Role:        input.Role,
	})
	if err != nil {
		return store.WorkspaceMemberView{}, wrapWorkspaceError("failed to update workspace member role", err)
	}
	return updated, nil
}

func (s *WorkspaceService) RemoveMember(ctx context.Context, input RemoveWorkspaceMemberInput) error {
	if err := validateRemoveWorkspaceMemberInput(input); err != nil {
		return err
	}
	if err := s.ensureCanManageMembers(ctx, input.ActorUserID, input.WorkspaceID); err != nil {
		return err
	}

	member, err := s.workspaces.GetMember(ctx, input.WorkspaceID, input.MemberID)
	if err != nil {
		return wrapWorkspaceError("failed to get workspace member", err)
	}
	if member.Role == store.WorkspaceRoleOwner {
		return apperror.Forbidden("workspace owner cannot be removed")
	}

	if err := s.workspaces.DeleteMember(ctx, input.WorkspaceID, input.MemberID); err != nil {
		return wrapWorkspaceError("failed to remove workspace member", err)
	}
	return nil
}

func validateCreateWorkspaceInput(userID int64, name string) error {
	if userID <= 0 {
		return apperror.Unauthorized("authentication is required")
	}
	if name == "" {
		return apperror.Validation("name is required")
	}
	if len(name) > maxWorkspaceNameLength {
		return apperror.Validation("name must be at most 100 characters")
	}
	return nil
}

func validateWorkspaceAccessInput(userID, workspaceID int64) error {
	if userID <= 0 {
		return apperror.Unauthorized("authentication is required")
	}
	if workspaceID <= 0 {
		return apperror.Validation("workspace id must be positive")
	}
	return nil
}

func validateAddWorkspaceMemberInput(input AddWorkspaceMemberInput, email string) error {
	if err := validateWorkspaceAccessInput(input.ActorUserID, input.WorkspaceID); err != nil {
		return err
	}
	if email == "" {
		return apperror.Validation("email is required")
	}
	if len(email) > maxEmailLength {
		return apperror.Validation("email must be at most 320 characters")
	}
	if !isValidEmail(email) {
		return apperror.Validation("email is invalid")
	}
	if !isAssignableWorkspaceMemberRole(input.Role) {
		return apperror.Validation("role must be one of admin, agent, viewer")
	}
	return nil
}

func validateUpdateWorkspaceMemberRoleInput(input UpdateWorkspaceMemberRoleInput) error {
	if err := validateWorkspaceAccessInput(input.ActorUserID, input.WorkspaceID); err != nil {
		return err
	}
	if input.MemberID <= 0 {
		return apperror.Validation("member id must be positive")
	}
	if !isAssignableWorkspaceMemberRole(input.Role) {
		return apperror.Validation("role must be one of admin, agent, viewer")
	}
	return nil
}

func validateRemoveWorkspaceMemberInput(input RemoveWorkspaceMemberInput) error {
	if err := validateWorkspaceAccessInput(input.ActorUserID, input.WorkspaceID); err != nil {
		return err
	}
	if input.MemberID <= 0 {
		return apperror.Validation("member id must be positive")
	}
	return nil
}

func (s *WorkspaceService) ensureCanManageMembers(ctx context.Context, userID, workspaceID int64) error {
	workspace, err := s.workspaceForUser(ctx, userID, workspaceID)
	if err != nil {
		return err
	}
	if !canManageWorkspaceMembers(workspace.Role) {
		return apperror.Forbidden("workspace member management requires owner or admin role")
	}
	return nil
}

func (s *WorkspaceService) workspaceForUser(ctx context.Context, userID, workspaceID int64) (store.WorkspaceView, error) {
	workspace, err := s.workspaces.GetForUser(ctx, userID, workspaceID)
	if err != nil {
		return store.WorkspaceView{}, wrapWorkspaceError("failed to get workspace", err)
	}
	return workspace, nil
}

func canManageWorkspaceMembers(role store.WorkspaceRole) bool {
	return role == store.WorkspaceRoleOwner || role == store.WorkspaceRoleAdmin
}

func isAssignableWorkspaceMemberRole(role store.WorkspaceRole) bool {
	return role == store.WorkspaceRoleAdmin ||
		role == store.WorkspaceRoleAgent ||
		role == store.WorkspaceRoleViewer
}

func wrapWorkspaceError(message string, err error) error {
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		return err
	}
	return apperror.Internal(message, err)
}
