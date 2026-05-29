package service_test

import (
	"context"
	"strings"
	"testing"

	"nucleus/internal/service"
	"nucleus/internal/store/storetest"
)

func TestNoteService_Create(t *testing.T) {
	tests := []struct {
		name    string
		input   service.CreateInput
		wantErr string
	}{
		{
			name:    "成功创建",
			input:   service.CreateInput{Title: "Test", Body: "Content"},
			wantErr: "",
		},
		{
			name:    "title 为空",
			input:   service.CreateInput{Title: "", Body: "Content"},
			wantErr: "title is required",
		},
		{
			name:    "body 为空",
			input:   service.CreateInput{Title: "Test", Body: ""},
			wantErr: "body is required",
		},
		{
			name:    "title 只有空格",
			input:   service.CreateInput{Title: "   ", Body: "Content"},
			wantErr: "title is required",
		},
		{
			name:    "title 超过 200 字符",
			input:   service.CreateInput{Title: strings.Repeat("a", 201), Body: "Content"},
			wantErr: "title must be at most 200 characters",
		},
		{
			name:    "title 恰好 200 字符",
			input:   service.CreateInput{Title: strings.Repeat("a", 200), Body: "Content"},
			wantErr: "",
		},
		{
			name:    "前后空格被清理",
			input:   service.CreateInput{Title: "  Hello  ", Body: "  World  "},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := storetest.NewMockNoteStore()
			svc := service.NewNoteService(mock)

			note, err := svc.Create(context.Background(), tt.input)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if note.Title != strings.TrimSpace(tt.input.Title) {
				t.Errorf("title = %q, want %q", note.Title, strings.TrimSpace(tt.input.Title))
			}
			if note.Body != strings.TrimSpace(tt.input.Body) {
				t.Errorf("body = %q, want %q", note.Body, strings.TrimSpace(tt.input.Body))
			}
			if note.ID == 0 {
				t.Error("expected non-zero ID")
			}
		})
	}
}

func TestNoteService_Create_StoreError(t *testing.T) {
	mock := storetest.NewMockNoteStore()
	mock.CreateErr = &testError{"db connection lost"}
	svc := service.NewNoteService(mock)

	_, err := svc.Create(context.Background(), service.CreateInput{Title: "Test", Body: "Content"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNoteService_List(t *testing.T) {
	mock := storetest.NewMockNoteStore()
	svc := service.NewNoteService(mock)

	// 预填数据
	for i := 0; i < 5; i++ {
		svc.Create(context.Background(), service.CreateInput{Title: "Note", Body: "Body"})
	}

	tests := []struct {
		name      string
		limit     int32
		offset    int32
		wantCount int
	}{
		{"默认分页", 20, 0, 5},
		{"limit=2", 2, 0, 2},
		{"offset=3", 20, 3, 2},
		{"超出范围", 20, 100, 0},
		{"limit<=0 回退到默认", 0, 0, 5},
		{"limit>100 截断", 200, 0, 5},
		{"offset<0 回退到 0", 10, -1, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes, err := svc.List(context.Background(), tt.limit, tt.offset)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(notes) != tt.wantCount {
				t.Errorf("got %d notes, want %d", len(notes), tt.wantCount)
			}
		})
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }
