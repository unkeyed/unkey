-- name: UpdateProject :exec
UPDATE projects
SET
    name = sqlc.arg(name),
    delete_protection = sqlc.arg(delete_protection),
    updated_at = sqlc.arg(updated_at)
WHERE workspace_id = sqlc.arg(workspace_id)
  AND slug = sqlc.arg(slug);
