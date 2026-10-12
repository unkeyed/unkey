-- name: UpdateRole :exec
UPDATE roles
SET
    name = sqlc.arg('name'),
    description = sqlc.narg('description'),
    updated_at_m = sqlc.arg('updated_at_m')
WHERE workspace_id = sqlc.arg('workspace_id')
  AND id = sqlc.arg('id');
