-- name: LockRoleByIDOrNameAndWorkspaceID :one
-- LockRoleByIDOrNameAndWorkspaceID locks IDs across a workspace while names
-- resolve only in the selected project because name uniqueness is project-scoped.
-- An ID match wins when search also matches a name because ORDER BY ranks IDs
-- first. For example, search "role_admin" locks the row with that ID instead
-- of a selected-project role named "role_admin".
SELECT id, project_id, name
FROM roles
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (
    id = sqlc.arg('search')
    OR (project_id = sqlc.arg('project_id') AND name = sqlc.arg('search'))
  )
ORDER BY id = sqlc.arg('search') DESC
LIMIT 1
FOR UPDATE;
