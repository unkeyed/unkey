-- name: LockEnvironmentForUpdate :one
SELECT e.id FROM environments e
JOIN apps a ON a.id = e.app_id
JOIN projects p ON p.id = e.project_id
WHERE e.id = sqlc.arg(id)
  AND e.deleted_at IS NULL
  AND a.deleted_at IS NULL
  AND p.deleted_at IS NULL
FOR UPDATE;
