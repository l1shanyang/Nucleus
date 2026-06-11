-- name: CreateUser :one
INSERT INTO users (email, name, password_hash)
VALUES ($1, $2, $3)
RETURNING id, email, name, password_hash, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, name, password_hash, created_at, updated_at
FROM users
WHERE email = $1;
