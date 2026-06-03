package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"nucleus/internal/http/handler"
)

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    handler.Pagination
		wantErr bool
	}{
		{
			name:  "默认分页",
			query: "",
			want:  handler.Pagination{Limit: handler.DefaultPageLimit, Offset: handler.DefaultOffset},
		},
		{
			name:  "指定分页",
			query: "?limit=10&offset=20",
			want:  handler.Pagination{Limit: 10, Offset: 20},
		},
		{
			name:  "limit 超过最大值时截断",
			query: "?limit=500&offset=1",
			want:  handler.Pagination{Limit: handler.MaxPageLimit, Offset: 1},
		},
		{
			name:    "limit 不是数字",
			query:   "?limit=abc",
			wantErr: true,
		},
		{
			name:    "limit 不能小于等于 0",
			query:   "?limit=0",
			wantErr: true,
		},
		{
			name:    "offset 不能为负数",
			query:   "?offset=-1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/notes"+tt.query, http.NoBody)

			got, err := handler.ParsePagination(req)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var appErr *handler.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected handler.AppError, got %T", err)
				}
				if appErr.Code != "INVALID_PARAM" {
					t.Fatalf("code = %q, want INVALID_PARAM", appErr.Code)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("pagination = %+v, want %+v", got, tt.want)
			}
		})
	}
}
