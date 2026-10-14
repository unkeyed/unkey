-- name: SoftDeleteRatelimitOverridesByNamespaceID :exec
UPDATE ratelimit_overrides
SET deleted_at_m = sqlc.arg(now)
WHERE workspace_id = sqlc.arg(workspace_id)
  AND namespace_id = sqlc.arg(namespace_id)
  AND deleted_at_m IS NULL;
