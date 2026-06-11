-- name: CreateWorkspace :one
INSERT INTO workspaces (name, created_by)
VALUES ($1, $2)
RETURNING id, name, created_by, created_at, updated_at;

-- name: CreateWorkspaceMember :one
INSERT INTO workspace_members (workspace_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING id, workspace_id, user_id, role, created_at;

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
