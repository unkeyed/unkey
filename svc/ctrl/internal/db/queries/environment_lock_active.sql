-- name: LockActiveEnvironment :one
SELECT e.id FROM environments e
JOIN apps a ON a.id = e.app_id
JOIN projects p ON p.id = e.project_id
WHERE e.id = sqlc.arg(id)
  AND e.deleting_at IS NULL
  AND a.deleting_at IS NULL
  AND p.deleting_at IS NULL
LOCK IN SHARE MODE;
