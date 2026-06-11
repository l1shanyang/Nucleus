-- name: CreateAuthSession :one
INSERT INTO auth_sessions (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, created_at;

-- name: GetAuthSessionUserByTokenHash :one
SELECT u.id, u.email, u.name, u.password_hash, u.created_at, u.updated_at
FROM auth_sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1
  AND s.expires_at > NOW();
