package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"nucleus/internal/apperror"
	"nucleus/internal/service"
)

type WorkspaceHandler struct {
	svc *service.WorkspaceService
}

type createWorkspaceRequest struct {
	Name string `json:"name"`
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

func parseWorkspaceID(r *http.Request) (int64, error) {
	raw := chi.URLParam(r, "workspaceID")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, InvalidParam("workspaceID must be a positive integer")
	}
	return id, nil
}
