package sqlc

import "context"

const createNote = `-- name: CreateNote :one
INSERT INTO notes (title, body)
VALUES ($1, $2)
RETURNING id, title, body, created_at
`

type CreateNoteParams struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (q *Queries) CreateNote(ctx context.Context, arg CreateNoteParams) (Note, error) {
	row := q.db.QueryRow(ctx, createNote, arg.Title, arg.Body)
	var note Note
	err := row.Scan(
		&note.ID,
		&note.Title,
		&note.Body,
		&note.CreatedAt,
	)
	return note, err
}

const listNotes = `-- name: ListNotes :many
SELECT id, title, body, created_at
FROM notes
ORDER BY id DESC
LIMIT $1 OFFSET $2
`

type ListNotesParams struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

func (q *Queries) ListNotes(ctx context.Context, arg ListNotesParams) ([]Note, error) {
	rows, err := q.db.Query(ctx, listNotes, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Note
	for rows.Next() {
		var note Note
		if err := rows.Scan(
			&note.ID,
			&note.Title,
			&note.Body,
			&note.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, note)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
