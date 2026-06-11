package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"nucleus/internal/apperror"
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

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authUserContextKey struct{}

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

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	var req loginRequest
	if err := DecodeJSON(r, &req); err != nil {
		return err
	}

	token, err := h.svc.Login(r.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	WriteSuccess(w, http.StatusOK, token)
	return nil
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) error {
	user, ok := authUserFromContext(r.Context())
	if !ok {
		return apperror.Unauthorized("authentication is required")
	}

	WriteSuccess(w, http.StatusOK, user)
	return nil
}

func (h *AuthHandler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := parseBearerToken(r.Header.Get("Authorization"))
		if !ok {
			WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authorization bearer token is required")
			return
		}

		user, err := h.svc.Authenticate(r.Context(), token)
		if err != nil {
			writeAppError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), authUserContextKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func authUserFromContext(ctx context.Context) (service.AuthUser, bool) {
	user, ok := ctx.Value(authUserContextKey{}).(service.AuthUser)
	return user, ok
}

func parseBearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	return token, token != ""
}

func writeAppError(w http.ResponseWriter, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		WriteError(w, appErr.Status, appErr.Code, appErr.Message)
		return
	}

	var serviceErr *apperror.Error
	if errors.As(err, &serviceErr) {
		status, code, message := mapAppError(serviceErr)
		WriteError(w, status, code, message)
		return
	}

	WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
}
