-- name: LockActivePortal :one
SELECT p.id
FROM portals p
JOIN projects project ON project.id = p.project_id
LEFT JOIN apps a ON a.id = p.app_id
WHERE p.id = sqlc.arg(id)
    AND p.workspace_id = sqlc.arg(workspace_id)
    AND project.deleting_at IS NULL
    AND (p.app_id IS NULL OR (a.id IS NOT NULL AND a.deleting_at IS NULL))
LOCK IN SHARE MODE;
