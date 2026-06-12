package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"nucleus/internal/apperror"
	"nucleus/internal/service"
	"nucleus/internal/store"
)

type WorkspaceHandler struct {
	svc *service.WorkspaceService
}

type createWorkspaceRequest struct {
	Name string `json:"name"`
}

type addWorkspaceMemberRequest struct {
	Email string              `json:"email"`
	Role  store.WorkspaceRole `json:"role"`
}

type updateWorkspaceMemberRoleRequest struct {
	Role store.WorkspaceRole `json:"role"`
}

func NewWorkspaceHandler(svc *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{svc: svc}
}

func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) error {
	user, ok := authUserFromContext(r.Context())
	if !ok {
		return apperror.Unauthorized("authentication is required")
	}

	var req createWorkspaceRequest
	if err := DecodeJSON(r, &req); err != nil {
		return err
	}

	workspace, err := h.svc.Create(r.Context(), service.CreateWorkspaceInput{
		UserID: user.ID,
		Name:   req.Name,
	})
	if err != nil {
		return err
	}

	WriteSuccess(w, http.StatusCreated, workspace)
	return nil
}

func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) error {
	user, ok := authUserFromContext(r.Context())
	if !ok {
		return apperror.Unauthorized("authentication is required")
	}

	workspaces, err := h.svc.List(r.Context(), user.ID)
	if err != nil {
		return err
	}

	WriteSuccess(w, http.StatusOK, workspaces)
	return nil
}

func (h *WorkspaceHandler) Get(w http.ResponseWriter, r *http.Request) error {
	user, ok := authUserFromContext(r.Context())
	if !ok {
		return apperror.Unauthorized("authentication is required")
	}

	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		return err
	}

	workspace, err := h.svc.Get(r.Context(), user.ID, workspaceID)
	if err != nil {
		return err
	}

	WriteSuccess(w, http.StatusOK, workspace)
	return nil
}

func (h *WorkspaceHandler) AddMember(w http.ResponseWriter, r *http.Request) error {
	user, ok := authUserFromContext(r.Context())
	if !ok {
		return apperror.Unauthorized("authentication is required")
	}

	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		return err
	}

	var req addWorkspaceMemberRequest
	if err := DecodeJSON(r, &req); err != nil {
		return err
	}

	member, err := h.svc.AddMember(r.Context(), service.AddWorkspaceMemberInput{
		ActorUserID: user.ID,
		WorkspaceID: workspaceID,
		Email:       req.Email,
		Role:        req.Role,
	})
	if err != nil {
		return err
	}

	WriteSuccess(w, http.StatusCreated, member)
	return nil
}

func (h *WorkspaceHandler) ListMembers(w http.ResponseWriter, r *http.Request) error {
	user, ok := authUserFromContext(r.Context())
	if !ok {
		return apperror.Unauthorized("authentication is required")
	}

	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		return err
	}

	members, err := h.svc.ListMembers(r.Context(), user.ID, workspaceID)
	if err != nil {
		return err
	}

	WriteSuccess(w, http.StatusOK, members)
	return nil
}

func (h *WorkspaceHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) error {
	user, ok := authUserFromContext(r.Context())
	if !ok {
		return apperror.Unauthorized("authentication is required")
	}

	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		return err
	}
	memberID, err := parseMemberID(r)
	if err != nil {
		return err
	}

	var req updateWorkspaceMemberRoleRequest
	if err := DecodeJSON(r, &req); err != nil {
		return err
	}

	member, err := h.svc.UpdateMemberRole(r.Context(), service.UpdateWorkspaceMemberRoleInput{
		ActorUserID: user.ID,
		WorkspaceID: workspaceID,
		MemberID:    memberID,
		Role:        req.Role,
	})
	if err != nil {
		return err
	}

	WriteSuccess(w, http.StatusOK, member)
	return nil
}

func (h *WorkspaceHandler) RemoveMember(w http.ResponseWriter, r *http.Request) error {
	user, ok := authUserFromContext(r.Context())
	if !ok {
		return apperror.Unauthorized("authentication is required")
	}

	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		return err
	}
	memberID, err := parseMemberID(r)
	if err != nil {
		return err
	}

	if err := h.svc.RemoveMember(r.Context(), service.RemoveWorkspaceMemberInput{
		ActorUserID: user.ID,
		WorkspaceID: workspaceID,
		MemberID:    memberID,
	}); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func parseWorkspaceID(r *http.Request) (int64, error) {
	raw := chi.URLParam(r, "workspaceID")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, InvalidParam("workspaceID must be a positive integer")
	}
	return id, nil
}

func parseMemberID(r *http.Request) (int64, error) {
	raw := chi.URLParam(r, "memberID")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, InvalidParam("memberID must be a positive integer")
	}
	return id, nil
}
