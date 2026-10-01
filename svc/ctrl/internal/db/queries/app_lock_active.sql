-- name: LockActiveApp :one
SELECT a.id FROM apps a
JOIN projects p ON p.id = a.project_id
WHERE a.id = sqlc.arg(id)
  AND a.deleting_at IS NULL
  AND p.deleting_at IS NULL
LOCK IN SHARE MODE;
