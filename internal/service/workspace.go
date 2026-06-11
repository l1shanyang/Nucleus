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
	tx         WorkspaceTxManager
}

func NewWorkspaceService(workspaces store.WorkspaceStore, tx WorkspaceTxManager) *WorkspaceService {
	return &WorkspaceService{workspaces: workspaces, tx: tx}
}

type CreateWorkspaceInput struct {
	UserID int64
	Name   string
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

func wrapWorkspaceError(message string, err error) error {
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		return err
	}
	return apperror.Internal(message, err)
}
