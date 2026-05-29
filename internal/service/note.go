package service

import (
	"context"
	"strings"

	"nucleus/internal/store"
)

// NoteService 封装笔记相关的业务逻辑。
type NoteService struct {
	store store.NoteStore
}

func NewNoteService(s store.NoteStore) *NoteService {
	return &NoteService{store: s}
}

// CreateInput 创建笔记的输入参数。
type CreateInput struct {
	Title string
	Body  string
}

// Create 创建笔记，执行业务校验。
func (s *NoteService) Create(ctx context.Context, input CreateInput) (store.Note, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)

	if input.Title == "" {
		return store.Note{}, validationError("title is required")
	}
	if input.Body == "" {
		return store.Note{}, validationError("body is required")
	}
	if len(input.Title) > 200 {
		return store.Note{}, validationError("title must be at most 200 characters")
	}

	return s.store.Create(ctx, input.Title, input.Body)
}

// List 查询笔记列表。
func (s *NoteService) List(ctx context.Context, limit, offset int32) ([]store.Note, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return s.store.List(ctx, limit, offset)
}
