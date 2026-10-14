-- name: ListRatelimitOverrideIDsByNamespaceID :many
SELECT id
FROM ratelimit_overrides
WHERE workspace_id = sqlc.arg(workspace_id)
  AND namespace_id = sqlc.arg(namespace_id)
  AND deleted_at_m IS NULL
ORDER BY id
FOR UPDATE;
