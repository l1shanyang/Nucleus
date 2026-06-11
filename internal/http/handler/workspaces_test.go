package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"nucleus/internal/http/handler"
	"nucleus/internal/service"
	"nucleus/internal/store/storetest"
)

type fakeWorkspaceHandlerTxManager struct{}

func (fakeWorkspaceHandlerTxManager) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return fn(nil)
}

func setupWorkspaceHandler() (*handler.AuthHandler, *handler.WorkspaceHandler, *storetest.MockUserStore, *storetest.MockSessionStore) {
	users := storetest.NewMockUserStore()
	sessions := storetest.NewMockSessionStore()
	workspaces := storetest.NewMockWorkspaceStore()

	authSvc := service.NewAuthService(users, sessions)
	workspaceSvc := service.NewWorkspaceService(workspaces, fakeWorkspaceHandlerTxManager{})

	return handler.NewAuthHandler(authSvc), handler.NewWorkspaceHandler(workspaceSvc), users, sessions
}

func loginWorkspaceUser(t *testing.T, authHandler *handler.AuthHandler, users *storetest.MockUserStore, sessions *storetest.MockSessionStore) string {
	t.Helper()

	user := seedAuthUser(t, users, "user@example.com", "password123")

	body := `{"email":"user@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.WrapHandler(authHandler.Login)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if len(sessions.CreatedSessions) != 1 {
		t.Fatalf("created sessions = %d, want 1", len(sessions.CreatedSessions))
	}
	sessions.AttachUser(sessions.CreatedSessions[0].TokenHash, &user)

	var resp handler.SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}
	data := resp.Data.(map[string]any)
	return data["access_token"].(string)
}

func TestWorkspaceHandler_Create(t *testing.T) {
	authHandler, workspaceHandler, users, sessions := setupWorkspaceHandler()
	token := loginWorkspaceUser(t, authHandler, users, sessions)

	body := `{"name":" Support Team "}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	authHandler.RequireAuth(handler.WrapHandler(workspaceHandler.Create)).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp handler.SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	data := resp.Data.(map[string]any)
	if data["name"] != "Support Team" {
		t.Fatalf("name = %v, want Support Team", data["name"])
	}
	if data["role"] != "owner" {
		t.Fatalf("role = %v, want owner", data["role"])
	}
}

func TestWorkspaceHandler_Create_MissingToken(t *testing.T) {
	authHandler, workspaceHandler, _, _ := setupWorkspaceHandler()

	body := `{"name":"Support Team"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	authHandler.RequireAuth(handler.WrapHandler(workspaceHandler.Create)).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

func TestWorkspaceHandler_ListAndGet(t *testing.T) {
	authHandler, workspaceHandler, users, sessions := setupWorkspaceHandler()
	token := loginWorkspaceUser(t, authHandler, users, sessions)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(authHandler.RequireAuth)
		r.Post("/api/v1/workspaces", handler.WrapHandler(workspaceHandler.Create))
		r.Get("/api/v1/workspaces", handler.WrapHandler(workspaceHandler.List))
		r.Get("/api/v1/workspaces/{workspaceID}", handler.WrapHandler(workspaceHandler.Get))
	})

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBufferString(`{"name":"Support"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d, body: %s", createW.Code, http.StatusCreated, createW.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", http.NoBody)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body: %s", listW.Code, http.StatusOK, listW.Body.String())
	}

	var listResp handler.SuccessResponse
	if err := json.Unmarshal(listW.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to parse list response: %v", err)
	}
	items := listResp.Data.([]any)
	if len(items) != 1 {
		t.Fatalf("got %d workspaces, want 1", len(items))
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/1", http.NoBody)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d, body: %s", getW.Code, http.StatusOK, getW.Body.String())
	}
}

func TestWorkspaceHandler_Get_InvalidID(t *testing.T) {
	authHandler, workspaceHandler, users, sessions := setupWorkspaceHandler()
	token := loginWorkspaceUser(t, authHandler, users, sessions)

	r := chi.NewRouter()
	r.With(authHandler.RequireAuth).Get("/api/v1/workspaces/{workspaceID}", handler.WrapHandler(workspaceHandler.Get))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/not-a-number", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}
