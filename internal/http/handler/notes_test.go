package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"context"

	"nucleus/internal/http/handler"
	"nucleus/internal/service"
	"nucleus/internal/store/storetest"
)

func setupNoteHandler() (*handler.NoteHandler, *storetest.MockNoteStore) {
	mock := storetest.NewMockNoteStore()
	svc := service.NewNoteService(mock)
	h := handler.NewNoteHandler(svc)
	return h, mock
}

func TestNoteHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "成功创建",
			body:       `{"title":"Test","body":"Content"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "无效 JSON",
			body:       `{bad json`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "title 为空",
			body:       `{"title":"","body":"Content"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "缺少 body",
			body:       `{"title":"Test"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "未知字段被拒绝",
			body:       `{"title":"Test","body":"Content","extra":"field"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := setupNoteHandler()

			req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.WrapHandler(h.Create)(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantCode != "" {
				var resp handler.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}
				if resp.Error.Code != tt.wantCode {
					t.Errorf("error code = %q, want %q", resp.Error.Code, tt.wantCode)
				}
			}
		})
	}
}

func TestNoteHandler_Create_Success(t *testing.T) {
	h, _ := setupNoteHandler()

	body := `{"title":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.WrapHandler(h.Create)(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp handler.SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected data to be a map, got %T", resp.Data)
	}
	if data["title"] != "Hello" {
		t.Errorf("title = %v, want Hello", data["title"])
	}
}

func TestNoteHandler_List(t *testing.T) {
	h, mock := setupNoteHandler()

	// 预填数据
	for i := 0; i < 3; i++ {
		mock.Create(context.Background(), "Note", "Body")
	}

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantCount  int
	}{
		{"默认", "", http.StatusOK, 3},
		{"limit=1", "?limit=1", http.StatusOK, 1},
		{"offset=2", "?offset=2", http.StatusOK, 1},
		{"无效 limit", "?limit=abc", http.StatusBadRequest, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/notes"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.WrapHandler(h.List)(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantCount > 0 {
				var resp handler.ListResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}
				data, ok := resp.Data.([]any)
				if !ok {
					t.Fatalf("expected data to be an array, got %T", resp.Data)
				}
				if len(data) != tt.wantCount {
					t.Errorf("got %d items, want %d", len(data), tt.wantCount)
				}
			}
		})
	}
}
