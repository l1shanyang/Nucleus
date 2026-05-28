package store

import (
	"context"
	"time"

	"nucleus/internal/db/sqlc"
)

// Note 是 store 层的领域模型，与数据库 schema 解耦。
type Note struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// NoteStore 定义笔记的数据访问接口。
type NoteStore interface {
	Create(ctx context.Context, title, body string) (Note, error)
	List(ctx context.Context, limit, offset int32) ([]Note, error)
}

// noteStore 基于 sqlc 的实现。
type noteStore struct {
	q *sqlc.Queries
}

func NewNoteStore(q *sqlc.Queries) NoteStore {
	return &noteStore{q: q}
}

func (s *noteStore) Create(ctx context.Context, title, body string) (Note, error) {
	row, err := s.q.CreateNote(ctx, sqlc.CreateNoteParams{
		Title: title,
		Body:  body,
	})
	if err != nil {
		return Note{}, err
	}
	return toNote(row), nil
}

func (s *noteStore) List(ctx context.Context, limit, offset int32) ([]Note, error) {
	rows, err := s.q.ListNotes(ctx, sqlc.ListNotesParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	notes := make([]Note, len(rows))
	for i, row := range rows {
		notes[i] = toNote(row)
	}
	return notes, nil
}

func toNote(row sqlc.Note) Note {
	return Note{
		ID:        row.ID,
		Title:     row.Title,
		Body:      row.Body,
		CreatedAt: row.CreatedAt,
	}
}
