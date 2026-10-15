-- name: LockActiveApp :one
SELECT a.id FROM apps a
JOIN projects p ON p.id = a.project_id
WHERE a.id = sqlc.arg(id)
  AND a.deleted_at IS NULL
  AND p.deleted_at IS NULL
LOCK IN SHARE MODE;
