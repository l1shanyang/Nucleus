-- name: CreateWorkspace :one
INSERT INTO workspaces (name, created_by)
VALUES ($1, $2)
RETURNING id, name, created_by, created_at, updated_at;

-- name: CreateWorkspaceMember :one
INSERT INTO workspace_members (workspace_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING id, workspace_id, user_id, role, created_at;

-- name: ListWorkspaceMembers :many
SELECT
    wm.id,
    wm.workspace_id,
    wm.user_id,
    u.email AS user_email,
    u.name AS user_name,
    wm.role,
    wm.created_at
FROM workspace_members wm
JOIN users u ON u.id = wm.user_id
WHERE wm.workspace_id = $1
ORDER BY wm.id ASC;

-- name: GetWorkspaceMember :one
SELECT
    wm.id,
    wm.workspace_id,
    wm.user_id,
    u.email AS user_email,
    u.name AS user_name,
    wm.role,
    wm.created_at
FROM workspace_members wm
JOIN users u ON u.id = wm.user_id
WHERE wm.workspace_id = $1
  AND wm.id = $2;

-- name: UpdateWorkspaceMemberRole :one
UPDATE workspace_members wm
SET role = $3
FROM users u
WHERE u.id = wm.user_id
  AND wm.workspace_id = $1
  AND wm.id = $2
RETURNING
    wm.id,
    wm.workspace_id,
    wm.user_id,
    u.email AS user_email,
    u.name AS user_name,
    wm.role,
    wm.created_at;

-- name: DeleteWorkspaceMember :execrows
DELETE FROM workspace_members
WHERE workspace_id = $1
  AND id = $2;

-- name: ListWorkspacesForUser :many
SELECT
    w.id,
    w.name,
    wm.role,
    w.created_at,
    w.updated_at
FROM workspace_members wm
JOIN workspaces w ON w.id = wm.workspace_id
WHERE wm.user_id = $1
ORDER BY w.id DESC;

-- name: GetWorkspaceForUser :one
SELECT
    w.id,
    w.name,
    wm.role,
    w.created_at,
    w.updated_at
FROM workspace_members wm
JOIN workspaces w ON w.id = wm.workspace_id
WHERE wm.user_id = $1
  AND w.id = $2;
