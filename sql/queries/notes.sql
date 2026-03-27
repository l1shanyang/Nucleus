-- name: CreateNote :one
INSERT INTO notes (title, body)
VALUES ($1, $2)
RETURNING id, title, body, created_at; -- create

-- name: ListNotes :many
SELECT id, title, body, created_at
FROM notes
ORDER BY id DESC
LIMIT $1 OFFSET $2; -- list
