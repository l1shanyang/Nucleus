package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"nucleus/internal/apperror"
	"nucleus/internal/http/handler"
	"nucleus/internal/service"
	"nucleus/internal/store"
	"nucleus/internal/store/storetest"
)

func setupAuthHandler() (*handler.AuthHandler, *storetest.MockUserStore, *storetest.MockSessionStore) {
	users := storetest.NewMockUserStore()
	sessions := storetest.NewMockSessionStore()
	svc := service.NewAuthService(users, sessions)
	h := handler.NewAuthHandler(svc)
	return h, users, sessions
}

func seedAuthUser(t *testing.T, users *storetest.MockUserStore, email, password string) store.User {
	t.Helper()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := store.User{
		ID:           1,
		Email:        email,
		Name:         "Alice",
		PasswordHash: string(passwordHash),
		CreatedAt:    time.Now().Truncate(time.Second),
		UpdatedAt:    time.Now().Truncate(time.Second),
	}
	users.Put(&user)
	return user
}

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "成功注册",
			body:       `{"email":"USER@example.com","name":"Alice","password":"password123"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "无效 JSON",
			body:       `{bad json`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "email 为空",
			body:       `{"email":"","name":"Alice","password":"password123"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "缺少 password",
			body:       `{"email":"user@example.com","name":"Alice"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "未知字段被拒绝",
			body:       `{"email":"user@example.com","name":"Alice","password":"password123","extra":"field"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, _ := setupAuthHandler()

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.WrapHandler(h.Register)(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantCode != "" {
				var resp handler.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}
				if resp.Error.Code != tt.wantCode {
					t.Fatalf("error code = %q, want %q", resp.Error.Code, tt.wantCode)
				}
			}
		})
	}
}

func TestAuthHandler_Register_SuccessResponse(t *testing.T) {
	h, _, _ := setupAuthHandler()

	body := `{"email":"USER@example.com","name":"Alice","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.WrapHandler(h.Register)(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp handler.SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected data to be a map, got %T", resp.Data)
	}
	if data["email"] != "user@example.com" {
		t.Fatalf("email = %v, want user@example.com", data["email"])
	}
	if data["name"] != "Alice" {
		t.Fatalf("name = %v, want Alice", data["name"])
	}
	if _, exists := data["password"]; exists {
		t.Fatal("response leaked password")
	}
	if _, exists := data["password_hash"]; exists {
		t.Fatal("response leaked password_hash")
	}
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	h, users, _ := setupAuthHandler()
	users.CreateErr = apperror.Conflict("user already exists")

	body := `{"email":"user@example.com","name":"Alice","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.WrapHandler(h.Register)(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}

	var resp handler.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error.Code != "CONFLICT" {
		t.Fatalf("error code = %q, want CONFLICT", resp.Error.Code)
	}
	if resp.Error.Message != "user already exists" {
		t.Fatalf("message = %q, want user already exists", resp.Error.Message)
	}
}

func TestAuthHandler_Login_SuccessResponse(t *testing.T) {
	h, users, _ := setupAuthHandler()
	seedAuthUser(t, users, "user@example.com", "password123")

	body := `{"email":"USER@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.WrapHandler(h.Login)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp handler.SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected data to be a map, got %T", resp.Data)
	}
	if data["access_token"] == "" {
		t.Fatal("expected access_token")
	}
	if data["token_type"] != "Bearer" {
		t.Fatalf("token_type = %v, want Bearer", data["token_type"])
	}
	if _, exists := data["password"]; exists {
		t.Fatal("response leaked password")
	}
	if _, exists := data["password_hash"]; exists {
		t.Fatal("response leaked password_hash")
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	h, users, _ := setupAuthHandler()
	seedAuthUser(t, users, "user@example.com", "password123")

	body := `{"email":"user@example.com","password":"wrong-password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.WrapHandler(h.Login)(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

func TestAuthHandler_Me(t *testing.T) {
	h, users, sessions := setupAuthHandler()
	user := seedAuthUser(t, users, "user@example.com", "password123")

	loginBody := `{"email":"user@example.com","password":"password123"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	handler.WrapHandler(h.Login)(loginW, loginReq)
	if loginW.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d, body: %s", loginW.Code, http.StatusOK, loginW.Body.String())
	}
	if len(sessions.CreatedSessions) != 1 {
		t.Fatalf("created sessions = %d, want 1", len(sessions.CreatedSessions))
	}
	sessions.AttachUser(sessions.CreatedSessions[0].TokenHash, &user)

	var loginResp handler.SuccessResponse
	if err := json.Unmarshal(loginW.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}
	loginData := loginResp.Data.(map[string]any)
	accessToken := loginData["access_token"].(string)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()

	h.RequireAuth(handler.WrapHandler(h.Me)).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp handler.SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	data := resp.Data.(map[string]any)
	if data["email"] != "user@example.com" {
		t.Fatalf("email = %v, want user@example.com", data["email"])
	}
}

func TestAuthHandler_Me_MissingToken(t *testing.T) {
	h, _, _ := setupAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", http.NoBody)
	w := httptest.NewRecorder()

	h.RequireAuth(handler.WrapHandler(h.Me)).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}
