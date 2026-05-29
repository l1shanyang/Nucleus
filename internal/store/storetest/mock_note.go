package storetest

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"nucleus/internal/store"
)

// MockNoteStore 是 NoteStore 的内存实现，用于测试。
type MockNoteStore struct {
	notes  []store.Note
	nextID int64

	// CreateErr 如果非 nil，Create 方法会返回此错误。
	CreateErr error
	// ListErr 如果非 nil，List 方法会返回此错误。
	ListErr error
}

func NewMockNoteStore() *MockNoteStore {
	return &MockNoteStore{nextID: 1}
}

func (m *MockNoteStore) Create(_ context.Context, title, body string) (store.Note, error) {
	if m.CreateErr != nil {
		return store.Note{}, m.CreateErr
	}

	note := store.Note{
		ID:        m.nextID,
		Title:     title,
		Body:      body,
		CreatedAt: time.Now().Truncate(time.Second),
	}
	m.nextID++
	m.notes = append(m.notes, note)
	return note, nil
}

func (m *MockNoteStore) List(_ context.Context, limit, offset int32) ([]store.Note, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}

	start := int(offset)
	if start >= len(m.notes) {
		return []store.Note{}, nil
	}

	end := start + int(limit)
	if end > len(m.notes) {
		end = len(m.notes)
	}

	return m.notes[start:end], nil
}

func (m *MockNoteStore) WithTx(pgx.Tx) store.NoteStore {
	return m // mock 不需要真正的事务
}
