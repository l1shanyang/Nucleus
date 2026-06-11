package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nucleus/internal/apperror"
	"nucleus/internal/http/handler"
	"nucleus/internal/service"
	"nucleus/internal/store/storetest"
)

func setupAuthHandler() (*handler.AuthHandler, *storetest.MockUserStore) {
	mock := storetest.NewMockUserStore()
	svc := service.NewAuthService(mock)
	h := handler.NewAuthHandler(svc)
	return h, mock
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
			h, _ := setupAuthHandler()

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
	h, _ := setupAuthHandler()

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
	h, mock := setupAuthHandler()
	mock.CreateErr = apperror.Conflict("user already exists")

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
