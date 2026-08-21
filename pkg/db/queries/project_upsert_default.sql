-- name: UpsertDefaultProject :exec
-- transactional-batch-statement
INSERT INTO projects (
    id,
    workspace_id,
    name,
    slug,
    delete_protection,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    'Default',
    'default',
    true,
    sqlc.arg(created_at),
    NULL
)
ON DUPLICATE KEY UPDATE slug = slug;

-- name: FindDefaultProjectForBatch :one
-- transactional-batch-statement
SELECT id
FROM projects
WHERE workspace_id = sqlc.arg(workspace_id)
  AND BINARY slug = sqlc.arg(slug)
LIMIT 1;
