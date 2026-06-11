package handler

import (
	"net/http"

	"nucleus/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

type registerRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) error {
	var req registerRequest
	if err := DecodeJSON(r, &req); err != nil {
		return err
	}

	user, err := h.svc.Register(r.Context(), service.RegisterInput{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	WriteSuccess(w, http.StatusCreated, user)
	return nil
}
